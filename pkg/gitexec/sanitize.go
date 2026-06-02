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

// sanitizeArgs returns a copy of args with any URL userinfo stripped.
// We only inspect tokens that look like URLs; everything else passes through.
func sanitizeArgs(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if looksLikeURL(a) {
			out[i] = stripUserinfo(a)
			continue
		}
		out[i] = a
	}
	return out
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
