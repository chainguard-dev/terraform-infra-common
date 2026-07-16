/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package valkey connects to the Memorystore for Valkey instances the
// terraform valkey module creates, under the same opinions the module
// hardcodes: IAM_AUTH as the workload identity (a fresh token per reconnect),
// TLS pinned to the managed server CA, and the PSC connect endpoint. Resolve
// the instance's full resource name once at boot, then dial clients from the
// resolved Endpoint.
package valkey

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"strconv"

	memorystore "cloud.google.com/go/memorystore/apiv1"
	"cloud.google.com/go/memorystore/apiv1/memorystorepb"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2/google"
)

// Memorystore IAM_AUTH accepts "default" as the only username; the password is
// a cloud-platform-scoped OAuth2 access token.
// https://cloud.google.com/memorystore/docs/valkey/manage-iam-auth
const (
	iamAuthScope = "https://www.googleapis.com/auth/cloud-platform"
	iamAuthUser  = "default"
)

// Endpoint is a resolved Memorystore for Valkey instance: the PSC connect
// address, the managed CA pool clients pin, and whether the instance speaks
// the cluster protocol.
type Endpoint struct {
	Addr    string
	Roots   *x509.CertPool
	Cluster bool
}

// Resolve looks up instance's connect address and managed server CA over the
// Memorystore API; instance is the full resource name
// (projects/{p}/locations/{l}/instances/{i}). A cluster-mode instance resolves
// to its discovery endpoint, a standalone one to its primary. Resolving once
// at boot pins the CA bundle: all currently-active CAs, which outlive the
// weekly server-cert rotation they sign. The caller's workload identity needs
// memorystore.instances.get alongside its connect grant.
func Resolve(ctx context.Context, instance string) (Endpoint, error) {
	// The service answers GetCertificateAuthority over gRPC with a server-side
	// Internal error; only the REST surface serves it.
	c, err := memorystore.NewRESTClient(ctx)
	if err != nil {
		return Endpoint{}, fmt.Errorf("valkey: memorystore client: %w", err)
	}
	defer c.Close()

	inst, err := c.GetInstance(ctx, &memorystorepb.GetInstanceRequest{Name: instance})
	if err != nil {
		return Endpoint{}, fmt.Errorf("valkey: get instance %q: %w", instance, err)
	}
	cluster := inst.GetMode() == memorystorepb.Instance_CLUSTER
	want := memorystorepb.ConnectionType_CONNECTION_TYPE_PRIMARY
	if cluster {
		want = memorystorepb.ConnectionType_CONNECTION_TYPE_DISCOVERY
	}
	var addr string
	for _, ep := range inst.GetEndpoints() {
		for _, conn := range ep.GetConnections() {
			auto := conn.GetPscAutoConnection()
			if auto.GetConnectionType() != want {
				continue
			}
			if addr != "" {
				// More than one connect endpoint would dial arbitrarily.
				return Endpoint{}, fmt.Errorf("valkey: instance %q has multiple %v endpoints", instance, want)
			}
			addr = net.JoinHostPort(auto.GetIpAddress(), strconv.Itoa(int(auto.GetPort())))
		}
	}
	if addr == "" {
		return Endpoint{}, fmt.Errorf("valkey: instance %q (state %v) has no %v endpoint", instance, inst.GetState(), want)
	}

	// The request doc asks for name = "{instance}/certificateAuthority", but
	// the REST binding appends the /certificateAuthority segment itself —
	// passing the suffixed form doubles it and 404s.
	ca, err := c.GetCertificateAuthority(ctx, &memorystorepb.GetCertificateAuthorityRequest{
		Name: instance,
	})
	if err != nil {
		return Endpoint{}, fmt.Errorf("valkey: get certificate authority for %q: %w", instance, err)
	}
	roots := x509.NewCertPool()
	ok := false
	for _, chain := range ca.GetManagedServerCa().GetCaCerts() {
		for _, cert := range chain.GetCertificates() {
			ok = roots.AppendCertsFromPEM([]byte(cert)) || ok
		}
	}
	if !ok {
		return Endpoint{}, fmt.Errorf("valkey: no certificates parsed from instance %q CA", instance)
	}
	return Endpoint{Addr: addr, Roots: roots, Cluster: cluster}, nil
}

// Option adjusts the client options (pool sizing, timeouts, hooks) before the
// client is built. The endpoint address, IAM credentials, and pinned TLS roots
// are already set when an Option runs; overwriting them defeats the package.
type Option func(*redis.UniversalOptions)

// NewClient dials ep as the workload identity — a standalone client against a
// primary endpoint, a cluster client against a discovery endpoint. Callers
// sharing one instance under several key namespaces should dial one client
// per namespace so heavy tenants (a Pub/Sub fan, a bulk writer) never contend
// on a shared pool.
func NewClient(ctx context.Context, ep Endpoint, opts ...Option) (redis.UniversalClient, error) {
	if ep.Addr == "" || ep.Roots == nil {
		return nil, fmt.Errorf("valkey: endpoint is unresolved")
	}

	ts, err := google.DefaultTokenSource(ctx, iamAuthScope)
	if err != nil {
		return nil, fmt.Errorf("valkey: workload token source: %w", err)
	}

	uo := &redis.UniversalOptions{
		Addrs: []string{ep.Addr},
		// Called before each (re)connect; the token source auto-refreshes, so
		// reconnects always authenticate with a current token despite hourly
		// expiry.
		CredentialsProviderContext: func(context.Context) (string, string, error) {
			tok, err := ts.Token()
			if err != nil {
				return "", "", fmt.Errorf("valkey: mint IAM_AUTH token: %w", err)
			}
			return iamAuthUser, tok.AccessToken, nil
		},
		// SAN carries the endpoint address, so default verification holds
		// without a ServerName override.
		TLSConfig: &tls.Config{
			RootCAs:    ep.Roots,
			MinVersion: tls.VersionTLS12,
		},
	}
	for _, opt := range opts {
		opt(uo)
	}

	// Not [redis.NewUniversalClient]: it selects the cluster client by len(Addrs),
	// and a cluster instance here has exactly one address — the discovery
	// endpoint. The instance's mode is the authority.
	if ep.Cluster {
		return redis.NewClusterClient(uo.Cluster()), nil
	}
	return redis.NewClient(uo.Simple()), nil
}
