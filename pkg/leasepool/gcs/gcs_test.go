/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gcs_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool"
	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool/gcs"
	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool/internal/contract"
)

// objectAPI models only the GCS requests this adapter uses. In particular it
// enforces generation preconditions, returns generations on media reads, and
// never turns an absent generation into a matching one. TestRealGCS drives the
// identical public contract against the service to check this model's limits.
type objectAPI struct {
	mu      sync.Mutex
	objects map[string]object
	next    int64
}
type object struct {
	data []byte
	gen  int64
}

func (s *objectAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.Method == http.MethodGet {
		_, name, ok := strings.Cut(r.URL.Path, "/o/")
		o, exists := s.objects[name]
		if !ok || !exists {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("X-Goog-Generation", strconv.FormatInt(o.gen, 10))
		w.Header().Set("Content-Length", strconv.Itoa(len(o.data)))
		_, _ = w.Write(o.data)
		return
	}
	if r.Method != http.MethodPost || r.URL.Query().Get("uploadType") != "multipart" {
		http.Error(w, "unsupported operation", http.StatusBadRequest)
		return
	}
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	parts := multipart.NewReader(r.Body, params["boundary"])
	meta, err := parts.NextPart()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var metadata struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(meta).Decode(&metadata); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	body, err := parts.NextPart()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	data, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	expected, err := strconv.ParseInt(r.URL.Query().Get("ifGenerationMatch"), 10, 64)
	if err != nil {
		http.Error(w, "missing generation precondition", http.StatusBadRequest)
		return
	}
	if s.objects[metadata.Name].gen != expected {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPreconditionFailed)
		fmt.Fprint(w, `{"error":{"code":412,"message":"generation mismatch"}}`)
		return
	}
	s.next++
	s.objects[metadata.Name] = object{data: data, gen: s.next}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"name": metadata.Name, "bucket": "leases", "generation": strconv.FormatInt(s.next, 10), "size": strconv.Itoa(len(data)), "crc32c": base64.StdEncoding.EncodeToString(binary.BigEndian.AppendUint32(nil, crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli))))})
}

func TestContract(t *testing.T) {
	server := httptest.NewServer(&objectAPI{objects: make(map[string]object)})
	defer server.Close()
	client, err := storage.NewClient(t.Context(), option.WithEndpoint(server.URL), option.WithHTTPClient(server.Client()), option.WithoutAuthentication(), storage.WithJSONReads())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	contract.Test(t, func(t *testing.T) leasepool.Interface {
		p, err := gcs.New(client.Bucket("leases"), "contract", leasepool.Config{})
		if err != nil {
			t.Fatal(err)
		}
		return p
	})
}

func TestRealGCS(t *testing.T) {
	bucket := os.Getenv("LEASEPOOL_TEST_BUCKET")
	if bucket == "" {
		t.Skip("set LEASEPOOL_TEST_BUCKET to run the contract against a dedicated GCS bucket")
	}
	client, err := storage.NewClient(t.Context(), storage.WithJSONReads())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	prefix := "leasepool-test-" + rand.Text()
	// Remove only this test's namespace, after all lease callbacks have stopped.
	defer func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 30*time.Second)
		defer cancel()
		iter := client.Bucket(bucket).Objects(ctx, &storage.Query{Prefix: base64.RawURLEncoding.EncodeToString([]byte(prefix)) + "/"})
		for {
			attrs, err := iter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				t.Error(err)
				break
			}
			if err := client.Bucket(bucket).Object(attrs.Name).If(storage.Conditions{GenerationMatch: attrs.Generation}).Delete(ctx); err != nil {
				t.Error(err)
			}
		}
	}()
	contract.Test(t, func(t *testing.T) leasepool.Interface {
		p, err := gcs.New(client.Bucket(bucket), prefix, leasepool.Config{})
		if err != nil {
			t.Fatal(err)
		}
		return p
	})
}
