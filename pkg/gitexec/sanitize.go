/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gitexec

import (
	"net/url"
	"strings"
)

// stripUserinfo removes the userinfo portion of a URL (e.g. tokens embedded as
// https://x-access-token:TOKEN@github.com/...). Inputs that don't parse as URLs
// are returned unchanged.
func stripUserinfo(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return raw
	}
	if u.User == nil {
		return raw
	}
	u.User = nil
	return u.String()
}

// sanitizeArgs returns a copy of args safe to log: URL userinfo is stripped,
// and the value half of any "git -c key=value" argument is masked. Git config
// values are a routine carrier of credentials (e.g. http.<url>.extraHeader
// holds an Authorization header), and argv is logged verbatim otherwise, so we
// redact every config value structurally rather than pattern-matching for
// secrets. Everything else passes through.
func sanitizeArgs(args []string) []string {
	out := make([]string, len(args))
	prev := ""
	for i, a := range args {
		switch {
		case prev == "-c":
			out[i] = redactConfigValue(a)
		case looksLikeURL(a):
			out[i] = stripUserinfo(a)
		default:
			out[i] = a
		}
		prev = a
	}
	return out
}

// redactConfigValue masks the value half of a "key=value" git config argument,
// preserving the key so logs still show which setting was applied. A bare key
// with no "=" carries no value (git reads it as "key=true"), so it is returned
// unchanged.
func redactConfigValue(s string) string {
	key, _, ok := strings.Cut(s, "=")
	if !ok {
		return s
	}
	return key + "=<redacted>"
}

func looksLikeURL(s string) bool {
	// http(s)://… and ssh://… cover the cases where userinfo can be embedded.
	// SCP-style refs (git@host:path) carry no secret in the user portion, so we
	// leave them untouched.
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "ssh://") ||
		strings.HasPrefix(s, "git://")
}

// repoFromArgs picks the first URL-shaped argument and returns host and path.
// path has any leading slash and trailing ".git" trimmed so common forms like
// "chainguard-dev/mono" come out clean. If no URL is present, both returns are
// empty.
func repoFromArgs(args []string) (host, path string) {
	for _, a := range args {
		if !looksLikeURL(a) {
			continue
		}
		u, err := url.Parse(a)
		if err != nil {
			continue
		}
		host = u.Host
		path = strings.TrimSuffix(strings.TrimPrefix(u.Path, "/"), ".git")
		return host, path
	}
	return "", ""
}
