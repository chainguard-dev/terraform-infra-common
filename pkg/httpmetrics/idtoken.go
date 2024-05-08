/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/
package httpmetrics

import (
	"context"
	"net/http"

	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"
	"google.golang.org/api/option/internaloption"
	htransport "google.golang.org/api/transport/http"
)

// newIdTokenClient creates a new http.Client, similar to that of idtoken.NewClient(),
// but with understanding of how we wrap the DefaultTransport for metrics.
//
// Based on
// https://github.com/googleapis/google-api-go-client/blob/v0.178.0/idtoken/idtoken.go#L46
// but skipping the opts validation that can't be performed here because private types.
//
// We can't just use the option.WithHTTPClient and call upstream idtoken.NewClient()
// because that code always reads from the http.DefaultTransport anyway.
func newIdTokenClient(ctx context.Context, audience string, opts ...idtoken.ClientOption) (*http.Client, error) {
	// unwrap the transport from the metrics transport
	innerTransport := ExtractInnerTransport(http.DefaultTransport)
	httpTransport := innerTransport.(*http.Transport).Clone()

	// Everything else after this point is based on
	// https://github.com/googleapis/google-api-go-client/blob/v0.178.0/idtoken/idtoken.go#L46
	ts, err := idtoken.NewTokenSource(ctx, audience, opts...)
	if err != nil {
		return nil, err
	}
	// Skip DialSettings validation so added TokenSource will not conflict with user
	// provided credentials.
	opts = append(opts, option.WithTokenSource(ts), internaloption.SkipDialSettingsValidation())

	httpTransport.MaxIdleConnsPerHost = 100
	t, err := htransport.NewTransport(ctx, httpTransport, opts...)
	if err != nil {
		return nil, err
	}
	return &http.Client{Transport: t}, nil
}

// NewIdTokenClient creates a new http.Client based on idtoken.Client, with metrics.
func NewIdTokenClient(ctx context.Context, audience string, opts ...idtoken.ClientOption) (*http.Client, error) {
	c, err := newIdTokenClient(ctx, audience, opts...)
	if err != nil {
		return nil, err
	}
	c.Transport = WrapTransport(c.Transport)
	return c, nil
}
