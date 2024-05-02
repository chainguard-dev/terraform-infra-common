/*
Copyright 2022 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package cloudevents

import (
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"

	metrics "github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
)

func NewClientHTTP(opts ...cehttp.Option) (cloudevents.Client, error) {
	// If we don't specify a client, NewClientHTTP will use http.DefaultClient
	// and may clobber its Transport. To avoid so, we pass a client with the
	// the metrics transport instead.
	metricsClient := http.Client{
		Transport: metrics.Transport,
	}
	copt := append([]cehttp.Option{
		cehttp.WithClient(metricsClient),
		cloudevents.WithMiddleware(func(next http.Handler) http.Handler {
			return metrics.Handler("cloudevents", next)
		})}, opts...)
	return cloudevents.NewClientHTTP(copt...)
}
