/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"github.com/chainguard-dev/clog"
)

const PollTimeout = 10 * time.Second

// Pulls messages from a pull subscription and replays them to a topic.
// This is useful for replaying messages from a pull subscription of a dead-letter topic
// to the original topic.
//
// Usage:
//
//	replayer --source=dead-letter-pull-sub --dest=original-topic --projectID=project-id
func main() {
	var srcSub, dstTop, prjID string
	flag.StringVar(&srcSub, "source", "", "source subscription")
	flag.StringVar(&dstTop, "dest", "", "destination topic")
	flag.StringVar(&prjID, "projectID", "", "project id")

	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if srcSub == "" {
		clog.FatalContextf(ctx, "--source is required")
	}
	if dstTop == "" {
		clog.FatalContextf(ctx, "--dest is required")
	}

	if prjID == "" {
		clog.FatalContextf(ctx, "--projectID is required")
	}
	client, err := pubsub.NewClient(ctx, prjID)
	if err != nil {
		clog.FatalContextf(ctx, "pubsub.NewClient: %v", err)
	}
	defer client.Close()

	sub := client.Subscriber(srcSub)
	top := client.Publisher(dstTop)

	fmt.Println("Listening for messages.")

	lastReceived := time.Now()
	go exitOnIdling(ctx, &lastReceived)

	// Receive blocks until the context is cancelled or an error occurs.
	_ = sub.Receive(ctx, func(_ context.Context, msg *pubsub.Message) {
		lastReceived = time.Now()
		fmt.Println("Found message:", string(msg.Data))

		// TODO: supporting a filter, either based on message content or attributes.
		// if filter(msg) {
		//     msg.Nack()
		// 	   return
		// }
		result := top.Publish(ctx, msg)
		if _, err := result.Get(ctx); err == nil {
			fmt.Printf("Replayed message: %s\n", string(msg.Data))
			msg.Ack()
		} else {
			fmt.Printf("Failed to publish message: %v\n", err)
			msg.Nack()
		}
	})
}

// exitOnIdling exits the program if no messages are received in the last PollTimeout.
func exitOnIdling(_ context.Context, lastReceived *time.Time) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	// nolint:all for { select {} } is the recommended way.
	for {
		select {
		case <-ticker.C:
			if time.Since(*lastReceived) > PollTimeout {
				fmt.Println("No messages received in the last", PollTimeout, ". Exiting.")
				// nolint:all We can exit without running ticker.Stop()
				os.Exit(0)
			}
		}
	}
}
