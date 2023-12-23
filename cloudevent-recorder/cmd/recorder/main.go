/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/kelseyhightower/envconfig"
)

type envConfig struct {
	Port    int    `envconfig:"PORT" default:"8080" required:"true"`
	LogPath string `envconfig:"LOG_PATH" required:"true"`
}

func main() {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Panicf("failed to process env var: %s", err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	c, err := cloudevents.NewClientHTTP(cloudevents.WithPort(env.Port))
	if err != nil {
		log.Panicf("failed to create event client, %v", err)
	}
	if err := c.StartReceiver(ctx, func(ctx context.Context, event cloudevents.Event) error {
		dir := filepath.Join(env.LogPath, event.Type())
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		filename := filepath.Join(dir, event.ID())
		return os.WriteFile(filename, event.Data(), 0600)
	}); err != nil {
		log.Panic(err)
	}
}
