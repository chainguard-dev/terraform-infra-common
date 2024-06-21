/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/chainguard-dev/clog"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/api/iterator"

	"github.com/chainguard-dev/terraform-infra-common/pkg/octosts"
)

type envConfig struct {
	OctoSTSPolicy     string `envconfig:"OCTOSTS_POLICY" required:"true"`
	GitHubOrg         string `envconfig:"GITHUB_ORG" required:"true"`
	GitHubRepo        string `envconfig:"GITHUB_REPO" required:"true"`
	GitHubTokenSecret string `envconfig:"GITHUB_TOKEN_SECRET" required:"true"`
}

func main() {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		panic(fmt.Errorf("failed to parse env: %w", err))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	c, err := secretmanager.NewClient(ctx)
	if err != nil {
		clog.FatalContextf(ctx, "failed to create gcp secret client: %v", err)
	}

	token, err := octosts.Token(ctx, env.OctoSTSPolicy, env.GitHubOrg, env.GitHubRepo)
	if err != nil {
		clog.FromContext(ctx).Fatal("failed to get octosts token: %w", err)
	}

	// Add a new secret version with the new token.
	createdSecretVersion, err := c.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent: env.GitHubTokenSecret,
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(token),
		},
	})
	if err != nil {
		clog.FatalContextf(ctx, "failed to add secret version: %v", err)
	}
	clog.FromContext(ctx).Info("successfully rotated the GitHub token")

	// Destroy older secret versions.
	it := c.ListSecretVersions(ctx, &secretmanagerpb.ListSecretVersionsRequest{
		Parent: env.GitHubTokenSecret,
		Filter: "state:ENABLED",
	})
	for {
		version, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			clog.FatalContextf(ctx, "failed to list secret versions: %v", err)
		}

		if version.Name == createdSecretVersion.Name {
			// Don't delete the newly created secret version.
			continue
		}

		if _, err := c.DestroySecretVersion(ctx, &secretmanagerpb.DestroySecretVersionRequest{
			Name: version.Name,
		}); err != nil {
			clog.FatalContextf(ctx, "failed to destroy secret version: %v", err)
		}
	}
	clog.FromContext(ctx).Info("successfully deleted older secret versions")
}
