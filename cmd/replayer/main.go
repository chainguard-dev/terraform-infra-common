/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/pubsub/v2"
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
	if srcSub == "" {
		log.Fatal("--source is required")
	}
	if dstTop == "" {
		log.Fatal("--dest is required")
	}

	if prjID == "" {
		log.Fatal("--projectID is required")
	}
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, prjID)
	if err != nil {
		log.Fatalf("pubsub.NewClient: %v", err)
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
