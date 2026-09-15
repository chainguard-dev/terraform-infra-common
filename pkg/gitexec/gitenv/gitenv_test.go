/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gitenv

import (
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

// resolve reads one configuration key as git sees it under env, in a fresh
// repository, so the assertions are about git's own resolution rather than
// the shape of the variables.
func resolve(t *testing.T, env []string, args ...string) string {
	t.Helper()
	dir := t.TempDir()
	init := exec.CommandContext(t.Context(), "git", "init", "-q", dir)
	init.Env = env
	if out, err := init.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir, "config"}, args...)...)
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			return "" // the key is unset
		}
		t.Fatalf("git config %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

// TestEnviron_TokenReachesOnlyItsRemote pins the credential's scope. The token
// must arrive as an Authorization header when git contacts the remote it
// was minted for, and must not follow a request to any other URL: a
// url.<base>.insteadOf rewrite planted in .git/config would otherwise carry
// the token to a host the attacker chose. It must appear nowhere in the
// environment but that one scoped value.
func TestEnviron_TokenReachesOnlyItsRemote(t *testing.T) {
	const remote = "https://github.test/o/r.git"
	const token = "ghs_sentinel_0123456789"
	env, err := Environ(Options{Remote: remote, TokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})})
	if err != nil {
		t.Fatal(err)
	}

	want := "Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte("unused-when-using-access-tokens:"+token))
	if got := resolve(t, env, "--get-urlmatch", "http.extraheader", remote); got != want {
		t.Errorf("extraheader for the remote: got %q, want %q", got, want)
	}
	if got := resolve(t, env, "--get-urlmatch", "http.extraheader", "https://attacker.test/o/r.git"); got != "" {
		t.Errorf("extraheader for another host: got %q, want none", got)
	}

	var carriers []string
	for _, kv := range env {
		if strings.Contains(kv, token) || strings.Contains(kv, want) {
			carriers = append(carriers, kv)
		}
	}
	if len(carriers) != 1 || !strings.HasPrefix(carriers[0], "GIT_CONFIG_VALUE_") {
		t.Errorf("token carriers in the environment: got %v, want exactly one GIT_CONFIG_VALUE_<n>", carriers)
	}
}

// TestEnviron_LocalRunCarriesNoCredential pins that a run naming no remote
// never mints a token: a purely local git command has nothing to present
// it to, and must not fail on a token source that cannot refresh.
func TestEnviron_LocalRunCarriesNoCredential(t *testing.T) {
	failing := oauth2.ReuseTokenSource(nil, failingTokenSource{})
	env, err := Environ(Options{TokenSource: failing})
	if err != nil {
		t.Fatalf("local run consulted the token source: %v", err)
	}
	for _, kv := range env {
		if strings.Contains(kv, "extraheader") {
			t.Errorf("local run carries a credential header: %s", kv)
		}
	}
}

type failingTokenSource struct{}

func (failingTokenSource) Token() (*oauth2.Token, error) {
	return nil, errors.New("token source must not be consulted")
}

// TestEnviron_Defences pins what git resolves under the environment: hooks
// pointed at a path that can hold none, the credential helper cleared, the
// proxy pinned empty, auto gc kept in the foreground, https the only
// protocol unless the caller widens it, and the caller's own entries
// readable after the defaults.
func TestEnviron_Defences(t *testing.T) {
	env, err := Environ(Options{Config: []Config{{"protocol.version", "2"}, {"gc.auto", "0"}}})
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"core.hooksPath":    "/dev/null",
		"credential.helper": "",
		"http.proxy":        "",
		"gc.autoDetach":     "false",
		"protocol.version":  "2",
		"gc.auto":           "0",
	} {
		if got := resolve(t, env, "--get", key); got != want {
			t.Errorf("%s: got %q, want %q", key, got, want)
		}
	}
	if !contains(env, "GIT_ALLOW_PROTOCOL=https") {
		t.Errorf("default GIT_ALLOW_PROTOCOL missing or widened: %v", filter(env, "GIT_ALLOW_PROTOCOL="))
	}
	for _, v := range []string{"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0"} {
		if !contains(env, v) {
			t.Errorf("environment lacks %s", v)
		}
	}

	widened, err := Environ(Options{AllowedProtocols: "https:file"})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(widened, "GIT_ALLOW_PROTOCOL=https:file") {
		t.Errorf("widened allowlist not applied: %v", filter(widened, "GIT_ALLOW_PROTOCOL="))
	}
}

// TestEnviron_DropsInheritedGitVariables pins that the host cannot reshape
// the subprocess from outside the options: tracing that would write HTTP
// headers to a logged stderr, a parent git's -c options and config files, a
// redirected repository, transport and prompt programs, and TLS trust
// settings are all dropped, while an unrelated variable passes through.
// The poisoned config file names a hooks path, so a variable that still
// reached git would show up as that path when git resolves the key.
func TestEnviron_DropsInheritedGitVariables(t *testing.T) {
	poisoned := filepath.Join(t.TempDir(), "poisoned.cfg")
	if err := os.WriteFile(poisoned, []byte("[core]\n\thooksPath = /evil/hooks\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, kv := range [][2]string{
		{"GIT_TRACE_CURL", "1"},
		{"GIT_TRACE", "1"},
		{"GIT_CURL_VERBOSE", ""},
		{"GIT_TRACE_REDACT", "0"},
		{"GIT_CONFIG_PARAMETERS", "'credential.helper=evil'"},
		{"GIT_CONFIG_COUNT", "1"},
		{"GIT_CONFIG_KEY_0", "core.hooksPath"},
		{"GIT_CONFIG_VALUE_0", "/evil/hooks"},
		{"GIT_CONFIG", poisoned},
		{"GIT_CONFIG_SYSTEM", poisoned},
		{"GIT_DIR", "/elsewhere/.git"},
		{"GIT_COMMON_DIR", "/elsewhere/.git"},
		{"GIT_WORK_TREE", "/elsewhere"},
		{"GIT_NAMESPACE", "elsewhere"},
		{"GIT_EXEC_PATH", "/evil/libexec"},
		{"GIT_SSH_COMMAND", "/evil/ssh"},
		{"GIT_SSL_NO_VERIFY", "1"},
		{"GIT_SSL_CAINFO", "/evil/ca.pem"},
		{"GIT_ASKPASS", "/evil/askpass"},
		{"SSH_ASKPASS", "/evil/askpass"},
		{"GITENV_UNRELATED", "kept"},
	} {
		t.Setenv(kv[0], kv[1])
	}
	env, err := Environ(Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if name == "SSH_ASKPASS" || (strings.HasPrefix(name, "GIT_") && !ours(name)) {
			t.Errorf("%s reached the subprocess", kv)
		}
	}
	if got := resolve(t, env, "--get", "core.hooksPath"); got != "/dev/null" {
		t.Errorf("an inherited variable outranked the defence: core.hooksPath = %q", got)
	}
	if !contains(env, "GITENV_UNRELATED=kept") {
		t.Error("unrelated variable dropped")
	}
}

// ours reports whether a GIT_* variable is one Environ itself sets.
func ours(name string) bool {
	switch name {
	case "GIT_CONFIG_NOSYSTEM", "GIT_CONFIG_GLOBAL", "GIT_TERMINAL_PROMPT", "GIT_ALLOW_PROTOCOL", "GIT_CONFIG_COUNT":
		return true
	}
	return strings.HasPrefix(name, "GIT_CONFIG_KEY_") || strings.HasPrefix(name, "GIT_CONFIG_VALUE_")
}

// TestEnviron_CallerConfigCannotOverrideDefences pins that Options.Config
// extends the environment but cannot loosen it: a caller naming a key this
// package pins gets the pinned value, because the pins are read last, while
// a key of the caller's own resolves as given.
func TestEnviron_CallerConfigCannotOverrideDefences(t *testing.T) {
	env, err := Environ(Options{Config: []Config{
		{"core.hooksPath", "/evil/hooks"},
		{"credential.helper", "!/evil/helper"},
		{"protocol.version", "2"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got := resolve(t, env, "--get", "core.hooksPath"); got != "/dev/null" {
		t.Errorf("caller overrode core.hooksPath: %q", got)
	}
	// git's credential code clears the helpers it has read so far when it
	// reads an empty value, so the pin holds as long as it is the last value.
	if got := resolve(t, env, "--get", "credential.helper"); got != "" {
		t.Errorf("caller's credential helper is the last value: %q", got)
	}
	if got := resolve(t, env, "--get-all", "credential.helper"); !strings.HasPrefix(got, "!/evil/helper") {
		t.Errorf("caller's entry missing ahead of the pin (the key may be unset rather than reset): %q", got)
	}
	if got := resolve(t, env, "--get", "protocol.version"); got != "2" {
		t.Errorf("caller's own key lost: protocol.version = %q", got)
	}
}

func contains(env []string, v string) bool {
	for _, kv := range env {
		if kv == v {
			return true
		}
	}
	return false
}

func filter(env []string, prefix string) []string {
	var out []string
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			out = append(out, kv)
		}
	}
	return out
}

// TestParseVersion_ReadsMajorMinor pins the version probe's reading of `git version`
// output, including the vendor suffixes some builds append: a misread
// would either refuse a good git or, worse, admit one that ignores the
// environment.
func TestParseVersion_ReadsMajorMinor(t *testing.T) {
	for _, tc := range []struct {
		out          string
		major, minor int
		wantErr      bool
	}{
		{"git version 2.39.5 (Apple Git-154)\n", 2, 39, false},
		{"git version 2.50.1\n", 2, 50, false},
		{"git version 3.0.0-rc1\n", 3, 0, false},
		{"not git\n", 0, 0, true},
	} {
		major, minor, err := parseVersion([]byte(tc.out))
		if (err != nil) != tc.wantErr {
			t.Errorf("parseVersion(%q): err = %v, wantErr = %v", tc.out, err, tc.wantErr)
			continue
		}
		if major != tc.major || minor != tc.minor {
			t.Errorf("parseVersion(%q): got %d.%d, want %d.%d", tc.out, major, minor, tc.major, tc.minor)
		}
	}
}

// TestCheckVersion_AcceptsHostGit runs the probe against the git on PATH, which the rest of
// this package's tests already require to be new enough.
func TestCheckVersion_AcceptsHostGit(t *testing.T) {
	if err := CheckVersion(t.Context()); err != nil {
		t.Fatalf("CheckVersion against the test host's git: %v", err)
	}
}
