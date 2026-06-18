/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gogit

import (
	"context"
	"errors"

	"github.com/chainguard-dev/terraform-infra-common/pkg/gitexec"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/storage"
)

// Option re-exports gitexec.Option so callers can tune an observation without
// importing gitexec alongside this package.
type Option = gitexec.Option

// WithRepoURL re-exports gitexec.WithRepoURL. The wrapped operations already
// derive repo_host and repo_path from the clone options or remote config, so
// callers rarely need this.
func WithRepoURL(rawURL string) Option { return gitexec.WithRepoURL(rawURL) }

// Repository wraps *git.Repository so its network operations emit the same
// observation as gitexec.Run. Every other method is promoted unchanged from the
// embedded value, so a *Repository is a drop-in for callers that read or mutate
// only local state.
type Repository struct {
	*git.Repository
}

// Remote wraps *git.Remote with instrumented fetch and push.
type Remote struct {
	*git.Remote
}

// Wrap adapts a *git.Repository obtained elsewhere — e.g. git.Open over a
// custom storer — so its network operations are observed. It returns nil for a
// nil repo so callers can wrap a (repo, err) pair without a nil check.
func Wrap(r *git.Repository) *Repository {
	if r == nil {
		return nil
	}
	return &Repository{Repository: r}
}

// NewRemote mirrors git.NewRemote, returning a *Remote whose fetch and push are
// observed.
func NewRemote(s storage.Storer, c *config.RemoteConfig) *Remote {
	return &Remote{Remote: git.NewRemote(s, c)}
}

// PlainCloneContext mirrors git.PlainCloneContext and records one "clone"
// observation. repo_host and repo_path are derived from o.URL.
func PlainCloneContext(ctx context.Context, path string, isBare bool, o *git.CloneOptions, opts ...Option) (*Repository, error) {
	var url string
	if o != nil {
		url = o.URL
	}
	var repo *git.Repository
	err := observe(ctx, "clone", url, opts, func() error {
		var cloneErr error
		repo, cloneErr = git.PlainCloneContext(ctx, path, isBare, o)
		return cloneErr
	})
	return Wrap(repo), err
}

// FetchContext mirrors (*git.Repository).FetchContext and records one "fetch"
// observation, deriving repo fields from the named remote (o.RemoteName,
// defaulting to origin).
func (r *Repository) FetchContext(ctx context.Context, o *git.FetchOptions, opts ...Option) error {
	name := git.DefaultRemoteName
	if o != nil && o.RemoteName != "" {
		name = o.RemoteName
	}
	return observe(ctx, "fetch", r.remoteURL(name), opts, func() error {
		return r.Repository.FetchContext(ctx, o)
	})
}

// PushContext mirrors (*git.Repository).PushContext and records one "push"
// observation.
func (r *Repository) PushContext(ctx context.Context, o *git.PushOptions, opts ...Option) error {
	name := git.DefaultRemoteName
	if o != nil && o.RemoteName != "" {
		name = o.RemoteName
	}
	return observe(ctx, "push", r.remoteURL(name), opts, func() error {
		return r.Repository.PushContext(ctx, o)
	})
}

// FetchContext mirrors (*git.Remote).FetchContext and records one "fetch"
// observation, deriving repo fields from the remote's configured URL.
func (r *Remote) FetchContext(ctx context.Context, o *git.FetchOptions, opts ...Option) error {
	return observe(ctx, "fetch", firstURL(r.Config()), opts, func() error {
		return r.Remote.FetchContext(ctx, o)
	})
}

// PushContext mirrors (*git.Remote).PushContext and records one "push"
// observation.
func (r *Remote) PushContext(ctx context.Context, o *git.PushOptions, opts ...Option) error {
	return observe(ctx, "push", firstURL(r.Config()), opts, func() error {
		return r.Remote.PushContext(ctx, o)
	})
}

// observe records one observation around fn. A NoErrAlreadyUpToDate result is
// recorded as a successful no-op — the round-trip happened and found nothing to
// transfer — but is returned unchanged so callers keep go-git's control flow.
func observe(ctx context.Context, op, repoURL string, opts []Option, fn func() error) error {
	all := opts
	if repoURL != "" {
		all = append([]Option{gitexec.WithRepoURL(repoURL)}, opts...)
	}
	var opErr error
	_ = gitexec.Observe(ctx, op, func() error {
		opErr = fn()
		if errors.Is(opErr, git.NoErrAlreadyUpToDate) {
			return nil
		}
		return opErr
	}, all...)
	return opErr
}

// remoteURL returns the first configured URL of the named remote, or "" if it
// cannot be resolved. It only enriches observations, so failures are silent.
func (r *Repository) remoteURL(name string) string {
	rem, err := r.Remote(name)
	if err != nil {
		return ""
	}
	return firstURL(rem.Config())
}

func firstURL(c *config.RemoteConfig) string {
	if c == nil || len(c.URLs) == 0 {
		return ""
	}
	return c.URLs[0]
}
