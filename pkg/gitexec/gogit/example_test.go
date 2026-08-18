/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gogit_test

import (
	"context"
	"fmt"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/storage/memory"

	"github.com/chainguard-dev/terraform-infra-common/pkg/gitexec/gogit"
)

func ExamplePlainCloneContext() {
	ctx := context.Background()
	repoURL := "https://github.com/example/repo"
	repo, err := gogit.PlainCloneContext(ctx, "/tmp/repo", false, &git.CloneOptions{
		URL:   repoURL,
		Depth: 1,
	})
	if err != nil {
		fmt.Println("clone failed:", err)
		return
	}
	_ = repo
}

func ExampleCloneContext() {
	ctx := context.Background()
	// CloneContext clones into a caller-provided storer, e.g. an in-memory store.
	store := memory.NewStorage()
	repo, err := gogit.CloneContext(ctx, store, nil, &git.CloneOptions{
		URL: "https://github.com/example/repo",
	})
	if err != nil {
		fmt.Println("clone failed:", err)
		return
	}
	_ = repo
}

func ExampleWrap() {
	// Wrap adapts a *git.Repository obtained elsewhere so its network
	// operations are observed.
	r, err := git.PlainOpen("/tmp/repo")
	if err != nil {
		fmt.Println("open failed:", err)
		return
	}
	repo := gogit.Wrap(r)
	_ = repo
}

func ExampleNewRemote() {
	store := memory.NewStorage()
	remote := gogit.NewRemote(store, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://github.com/example/repo"},
	})
	_ = remote
}

func ExampleWithRepoURL() {
	ctx := context.Background()
	repoURL := "https://github.com/example/repo"
	repo, err := gogit.PlainCloneContext(ctx, "/tmp/repo", false, &git.CloneOptions{
		URL: repoURL,
	}, gogit.WithRepoURL(repoURL))
	if err != nil {
		fmt.Println("clone failed:", err)
		return
	}
	_ = repo
}

func ExampleRepository_FetchContext() {
	ctx := context.Background()
	repo, err := gogit.PlainCloneContext(ctx, "/tmp/repo", false, &git.CloneOptions{
		URL: "https://github.com/example/repo",
	})
	if err != nil {
		fmt.Println("clone failed:", err)
		return
	}
	if err := repo.FetchContext(ctx, &git.FetchOptions{RemoteName: "origin"}); err != nil {
		fmt.Println("fetch failed:", err)
	}
}

func ExampleRepository_PushContext() {
	ctx := context.Background()
	repo, err := gogit.PlainCloneContext(ctx, "/tmp/repo", false, &git.CloneOptions{
		URL: "https://github.com/example/repo",
	})
	if err != nil {
		fmt.Println("clone failed:", err)
		return
	}
	if err := repo.PushContext(ctx, &git.PushOptions{RemoteName: "origin"}); err != nil {
		fmt.Println("push failed:", err)
	}
}

func ExampleRemote_FetchContext() {
	ctx := context.Background()
	store := memory.NewStorage()
	remote := gogit.NewRemote(store, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://github.com/example/repo"},
	})
	if err := remote.FetchContext(ctx, &git.FetchOptions{}); err != nil {
		fmt.Println("fetch failed:", err)
	}
}

func ExampleRemote_PushContext() {
	ctx := context.Background()
	store := memory.NewStorage()
	remote := gogit.NewRemote(store, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://github.com/example/repo"},
	})
	if err := remote.PushContext(ctx, &git.PushOptions{}); err != nil {
		fmt.Println("push failed:", err)
	}
}
