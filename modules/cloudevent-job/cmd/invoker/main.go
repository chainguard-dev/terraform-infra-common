/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"cloud.google.com/go/compute/metadata"
	run "cloud.google.com/go/run/apiv2"
	runpb "cloud.google.com/go/run/apiv2/runpb"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/sethvargo/go-envconfig"

	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	mce "github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics/cloudevents"
	"github.com/chainguard-dev/terraform-infra-common/pkg/profiler"
)

var env = envconfig.MustProcess(context.Background(), &struct {
	Port      int    `env:"PORT, default=8080"`
	JobName   string `env:"JOB_NAME, required"`
	JobRegion string `env:"JOB_REGION, required"`
}{})

var projectID string

func init() {
	var err error
	projectID, err = metadata.ProjectID()
	if err != nil {
		log.Fatalf("failed to get project ID: %v", err) //nolint:gocritic
	}
}

func main() {
	profiler.SetupProfiler()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go httpmetrics.ServeMetrics()
	defer httpmetrics.SetupTracer(ctx)()

	jobsClient, err := run.NewJobsClient(ctx)
	if err != nil {
		log.Fatalf("failed to create jobs client: %v", err) //nolint:gocritic
	}
	defer jobsClient.Close()

	c, err := mce.NewClientHTTP("ce-job", cloudevents.WithPort(env.Port))
	if err != nil {
		clog.Fatalf("failed to create event client, %v", err)
	}
	if err := c.StartReceiver(ctx, func(ctx context.Context, event cloudevents.Event) error {
		log := clog.FromContext(ctx)

		eventJSON, err := event.MarshalJSON()
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}

		op, err := jobsClient.RunJob(ctx, &runpb.RunJobRequest{
			Name: fmt.Sprintf("projects/%s/locations/%s/jobs/%s", projectID, env.JobRegion, env.JobName),
			Overrides: &runpb.RunJobRequest_Overrides{
				ContainerOverrides: []*runpb.RunJobRequest_Overrides_ContainerOverride{{
					Args: []string{"--event", string(eventJSON)},
				}},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to run job: %w", err)
		}
		ex, err := op.Metadata()
		if err != nil {
			return fmt.Errorf("failed to get job metadata: %w", err)
		}
		log.Infof("started job execution %s", ex.GetUid())
		return nil
	}); err != nil {
		clog.Fatalf("failed to start event receiver, %v", err)
	}
}
