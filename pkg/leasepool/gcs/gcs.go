/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gcs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool"
	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool/internal/engine"
)

// New uses bucket to coordinate participants sharing prefix and lease name.
// prefix is an exact namespace, not a directory: use the same value everywhere.
// It may contain at most 255 bytes, reserving room for any valid lease name.
// The caller supplies a client using strongly consistent, uncached reads.
func New(bucket *storage.BucketHandle, prefix string, cfg leasepool.Config) (leasepool.Interface, error) {
	if bucket == nil {
		return nil, errors.New("lease bucket is required")
	}
	// GCS object names permit 1024 bytes; reserve room for the separator and
	// the interface's largest (512-byte) name after base64 encoding.
	if base64.RawURLEncoding.EncodedLen(len(prefix))+1+base64.RawURLEncoding.EncodedLen(512) > 1024 {
		return nil, errors.New("lease prefix exceeds the GCS object name limit")
	}
	return engine.New(&store{bucket: bucket, prefix: base64.RawURLEncoding.EncodeToString([]byte(prefix)) + "/"}, cfg)
}

type store struct {
	bucket *storage.BucketHandle
	prefix string
}

func (s *store) object(name string) *storage.ObjectHandle {
	return s.bucket.Object(s.prefix + base64.RawURLEncoding.EncodeToString([]byte(name)))
}

func (s *store) Read(ctx context.Context, name string) (engine.Record, int64, error) {
	r, err := s.object(name).NewReader(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return engine.Record{}, 0, nil
	}
	if err != nil {
		return engine.Record{}, 0, err
	}
	defer r.Close()
	data, err := io.ReadAll(io.LimitReader(r, 4097))
	if err != nil {
		return engine.Record{}, 0, err
	}
	if len(data) > 4096 {
		return engine.Record{}, 0, errors.New("lease record too large")
	}
	var rec engine.Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return engine.Record{}, 0, fmt.Errorf("decode lease record: %w", err)
	}
	if r.Attrs.Generation == 0 {
		return engine.Record{}, 0, errors.New("lease record has no generation")
	}
	return rec, r.Attrs.Generation, nil
}

func (s *store) Write(ctx context.Context, name string, rec engine.Record, gen int64) (int64, error) {
	data, err := json.Marshal(rec)
	if err != nil {
		return 0, err
	}
	conditions := storage.Conditions{GenerationMatch: gen}
	if gen == 0 {
		conditions = storage.Conditions{DoesNotExist: true}
	}
	// Conditional uploads are idempotent under the storage client's default
	// retry policy. A replay after an ambiguous commit conflicts and fails closed.
	obj := s.object(name).If(conditions)
	w := obj.NewWriter(ctx)
	// Retain retries without allocating the default 16 MiB upload buffer for
	// a record whose reader permits at most 4 KiB.
	w.ChunkSize = googleapi.MinUploadChunkSize
	w.ContentType = "application/json"
	w.CacheControl = "no-store"
	if _, err := w.Write(data); err != nil {
		_ = w.Close()
		return 0, writeError(err, gen)
	}
	if err := w.Close(); err != nil {
		return 0, writeError(err, gen)
	}
	return w.Attrs().Generation, nil
}

// A missing object conflicts only when replacing a known generation. On first
// creation, NotFound can mean the bucket is missing and must remain a storage error.
func writeError(err error, gen int64) error {
	if ge, ok := errors.AsType[*googleapi.Error](err); ok && (ge.Code == http.StatusPreconditionFailed || (gen != 0 && ge.Code == http.StatusNotFound)) {
		return engine.ErrConflict
	}
	if code := status.Code(err); code == codes.FailedPrecondition || (gen != 0 && code == codes.NotFound) {
		return engine.ErrConflict
	}
	return err
}

var _ engine.Store = (*store)(nil)
