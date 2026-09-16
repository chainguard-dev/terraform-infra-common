/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gitenv

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/chainguard-dev/terraform-infra-common/pkg/gitexec"
	"golang.org/x/oauth2"
)

// MinMajor and MinMinor name the oldest git Environ's variables all reach:
// GIT_CONFIG_GLOBAL arrived in 2.32, after GIT_CONFIG_COUNT in 2.31.
const (
	MinMajor = 2
	MinMinor = 32
)

// Config is one git configuration entry passed through the environment.
type Config struct {
	Key, Value string
}

// Options describe the subprocess an environment is built for.
type Options struct {
	// Remote is the URL the subprocess contacts (a clone, fetch, push or
	// ls-remote). Empty for a purely local run, which then carries no
	// credential and cannot fail on a token refresh.
	Remote string

	// TokenSource mints the token presented to Remote as HTTP basic auth
	// in an Authorization header scoped to Remote's URL. Nil, or an empty
	// token, sends no credential. Not consulted when Remote is empty.
	TokenSource oauth2.TokenSource

	// AllowedProtocols is the GIT_ALLOW_PROTOCOL value, the colon-separated
	// transports git may use. Empty pins https alone, which stops a
	// url.<base>.insteadOf rewrite from redirecting a fetch to file://
	// (attacker-chosen local objects), ssh, ext or another remote helper.
	AllowedProtocols string

	// Config is further configuration to pass through the environment, in
	// order, ahead of the entries this package sets. git reads the last
	// value of a single-valued key and an empty credential.helper clears the
	// list, so a key this package pins keeps its value whatever Config
	// says about it.
	Config []Config
}

// Environ returns the environment for the subprocess: the parent
// environment less the git variables listed in inherited, then the
// variables that disable system and global config, prompting and unlisted
// protocols, then the configuration entries. The token, when there is one,
// appears only in a GIT_CONFIG_VALUE_<n> entry for http.<Remote>.extraheader.
func Environ(opts Options) ([]string, error) {
	cfg := slices.Clone(opts.Config)
	cfg = append(cfg,
		// Hooks live under a repository's .git/, which the process may not
		// have written itself, and would run with the credential in the
		// environment; point git at a path that can never contain one.
		// http.proxy and credential.helper are pinned empty too:
		// configuration through the environment is command scope, which
		// outranks (proxy, last-wins) or clears (helper list) anything in
		// .git/config.
		Config{"core.hooksPath", os.DevNull},
		Config{"http.proxy", ""},
		Config{"credential.helper", ""},
		// Auto gc must not fork into the background. A forked gc can still
		// be repacking objects after the command returns and the caller
		// reopens a go-git handle on the repository, so a later go-git read
		// can land on a pack gc already removed. Auto gc itself stays on;
		// only the detach is forbidden, so a triggered run finishes inside
		// the command that triggered it.
		Config{"gc.autoDetach", "false"},
	)
	if opts.Remote != "" {
		// URL-scoped so they outrank a same-specificity .git/config pin, and
		// token-independent so an uncredentialed fetch is covered too.
		cfg = append(cfg,
			Config{"http." + opts.Remote + ".proxy", ""},
			Config{"url." + opts.Remote + ".insteadOf", opts.Remote},
		)
		if opts.TokenSource != nil {
			token, err := opts.TokenSource.Token()
			if err != nil {
				return nil, fmt.Errorf("getting token: %w", err)
			}
			if token.AccessToken != "" {
				basic := base64.StdEncoding.EncodeToString([]byte("unused-when-using-access-tokens:" + token.AccessToken))
				cfg = append(cfg, Config{"http." + opts.Remote + ".extraheader", "Authorization: Basic " + basic})
			}
		}
	}
	protocols := opts.AllowedProtocols
	if protocols == "" {
		protocols = "https"
	}
	var env []string
	for _, kv := range os.Environ() {
		if inherited(kv) {
			env = append(env, kv)
		}
	}
	env = append(env,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ALLOW_PROTOCOL="+protocols,
		fmt.Sprintf("GIT_CONFIG_COUNT=%d", len(cfg)),
	)
	for i, kv := range cfg {
		env = append(env,
			fmt.Sprintf("GIT_CONFIG_KEY_%d=%s", i, kv.Key),
			fmt.Sprintf("GIT_CONFIG_VALUE_%d=%s", i, kv.Value),
		)
	}
	return env, nil
}

// inherited reports whether a parent-environment variable reaches the
// subprocess. Every GIT_* variable is dropped, along with SSH_ASKPASS, so
// the git-specific environment the subprocess runs under is exactly what
// Environ sets. Each family git reads from the environment would otherwise
// let the host reshape the command from outside the options:
//
//   - Configuration: GIT_CONFIG_PARAMETERS and GIT_CONFIG_COUNT with its
//     keys and values apply a parent git's -c options at command scope,
//     level with the defences here; GIT_CONFIG_SYSTEM names the system file
//     and GIT_CONFIG a file the config builtin reads in place of the
//     repository's.
//   - Repository location: GIT_DIR, GIT_COMMON_DIR, GIT_WORK_TREE,
//     GIT_INDEX_FILE, GIT_OBJECT_DIRECTORY, GIT_ALTERNATE_OBJECT_DIRECTORIES
//     and GIT_NAMESPACE move the config, refs and objects the command
//     operates on away from the repository named by its working directory.
//   - Code git runs: GIT_EXEC_PATH names the directory git's subcommands are
//     loaded from; GIT_SSH, GIT_SSH_COMMAND and GIT_PROXY_COMMAND name
//     transport programs; GIT_ASKPASS and SSH_ASKPASS name a credential
//     prompt, and with prompting disabled git fails instead.
//   - Trust on the wire: GIT_SSL_NO_VERIFY, GIT_SSL_CAINFO and GIT_SSL_CAPATH
//     decide which server the credential is sent to.
//   - Tracing: GIT_TRACE* and GIT_CURL_VERBOSE write packets and HTTP
//     headers to stderr, which a caller may log, and GIT_CURL_VERBOSE turns
//     on by being present at all, so it cannot be pinned off, only removed.
//     With no override, git's own GIT_TRACE_REDACT default stays in force.
//
// Identity, editor and pager variables go with the rest: a subprocess here
// never commits or pages interactively, and a caller that needs an identity
// passes user.name and user.email through Options.Config.
func inherited(kv string) bool {
	name, _, _ := strings.Cut(kv, "=")
	return name != "SSH_ASKPASS" && !strings.HasPrefix(name, "GIT_")
}

// versionRegex extracts the major.minor pair from `git version` output,
// e.g. "git version 2.39.5 (Apple Git-154)".
var versionRegex = regexp.MustCompile(`git version (\d+)\.(\d+)`)

// CheckVersion runs `git version` and fails when the binary is missing or
// predates MinMajor.MinMinor, so an environment git would silently ignore
// is caught where the caller can refuse to run instead of failing open on
// its first fetch.
func CheckVersion(ctx context.Context) error {
	out, err := gitexec.Output(ctx, "version", gitexec.CommandContext(ctx, "version"))
	if err != nil {
		return fmt.Errorf("running git version (git >= %d.%d is required on PATH): %w", MinMajor, MinMinor, err)
	}
	major, minor, err := parseVersion(out)
	if err != nil {
		return err
	}
	if major > MinMajor || (major == MinMajor && minor >= MinMinor) {
		return nil
	}
	return fmt.Errorf("git %d.%d is too old: config-over-environment needs git >= %d.%d (older git ignores it silently, disabling the hook and config defences)", major, minor, MinMajor, MinMinor)
}

// parseVersion reads the major and minor version from `git version` output.
func parseVersion(out []byte) (major, minor int, err error) {
	m := versionRegex.FindSubmatch(out)
	if m == nil {
		return 0, 0, fmt.Errorf("parsing git version output %q", out)
	}
	major, _ = strconv.Atoi(string(m[1]))
	minor, _ = strconv.Atoi(string(m[2]))
	return major, minor, nil
}
