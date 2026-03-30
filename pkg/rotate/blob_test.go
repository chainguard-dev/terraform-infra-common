/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package rotate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"gocloud.dev/blob/memblob"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
)

var (
	wantBeforeBlobs = map[string]string{
		"unit/test/0": "UNIT TEST: 0\n",
		"unit-test-1": "UNIT TEST: 1\n",
		"unit/test/2": "UNIT TEST: 2\n",
		"unit-test-3": "UNIT TEST: 3\n",
		"unit/test/4": "UNIT TEST: 4\n",
	}

	wantBeforeCombineBlobs = []string{
		"UNIT TEST: 1\nUNIT TEST: 3\n",
		"UNIT TEST: 0\nUNIT TEST: 2\nUNIT TEST: 4\n",
	}

	wantAfterCombineBlobs = []string{
		"LAST UT\n",
		"UNIT TEST: 1\nUNIT TEST: 3\n",
		"UNIT TEST: 0\nUNIT TEST: 2\nUNIT TEST: 4\n",
	}
)

// waitForBlobs polls the bucket until the expected number of blobs appear
// or the timeout expires. Returns the blob contents map.
func waitForBlobs(ctx context.Context, t *testing.T, bucket *blob.Bucket, wantCount int, timeout time.Duration) map[string]string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		blobs, err := getFiles(ctx, bucket)
		if err != nil {
			t.Fatalf("Failed to read files from blobstore: %v", err)
		}
		if len(blobs) >= wantCount {
			return blobs
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d blobs, got %d", wantCount, len(blobs))
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func TestBlobUploader(t *testing.T) {
	dir := t.TempDir()
	blobDir := t.TempDir()

	cancelCtx, cancel := context.WithCancel(t.Context())
	bucketName := "file://" + blobDir
	bucket, err := blob.OpenBucket(cancelCtx, bucketName)
	if err != nil {
		t.Fatalf("Failed to create a bucket: %v", err)
	}

	uploader := NewUploader(dir, bucketName, 1*time.Minute)

	// Create a few files there to be uploaded
	for i := range 5 {
		var filename string
		if i%2 == 0 {
			filename = fmt.Sprintf("%s/unit/test/%d", dir, i)
		} else {
			filename = fmt.Sprintf("%s/unit-test-%d", dir, i)
		}
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			t.Errorf("MkdirAll() = %v", err)
		}
		contents := fmt.Sprintf("UNIT TEST: %d", i)
		err := os.WriteFile(filename, []byte(contents), 0600)
		if err != nil {
			t.Errorf("Failed to write file %d: %v\n", i, err)
		}
	}
	for i, b := range []string{"", "\n"} {
		x := i + 5
		var filename string
		if i%2 == 0 {
			filename = fmt.Sprintf("%s/unit/test/%d", dir, x)
		} else {
			filename = fmt.Sprintf("%s/unit-test-%d", dir, x)
		}
		err := os.WriteFile(filename, []byte(b), 0600)
		if err != nil {
			t.Errorf("Failed to write file %d: %v\n", x, err)
		}
	}

	// Start the uploader
	go uploader.Run(cancelCtx)

	// Poll until the expected blobs appear rather than using a fixed sleep.
	blobsBefore := waitForBlobs(cancelCtx, t, bucket, len(wantBeforeCombineBlobs), 10*time.Second)
	less := func(a, b string) bool { return a < b }
	got := slices.Sorted(maps.Values(blobsBefore))
	want := slices.Sorted(slices.Values(wantBeforeCombineBlobs))
	if !slices.Equal(got, want) {
		t.Errorf("blobs mismatch: %s", cmp.Diff(wantBeforeCombineBlobs, slices.Collect(maps.Values(blobsBefore)), cmpopts.SortSlices(less)))
	}

	// Then write one more file and trigger cancel, so this too should be now
	// written there.
	filename := fmt.Sprintf("%s/last", dir)
	err = os.WriteFile(filename, []byte("LAST UT"), 0600)
	if err != nil {
		t.Errorf("Failed to write: %v", err)
	}

	// Verify the file does not get immediately uploaded (still on the flush interval).
	blobsAfter, err := getFiles(cancelCtx, bucket)
	if err != nil {
		t.Errorf("Failed to read files from blobstore: %v", err)
	}
	gotAfter := slices.Sorted(maps.Values(blobsAfter))
	if !slices.Equal(gotAfter, want) {
		t.Errorf("blobs should not have changed yet: %s", cmp.Diff(wantBeforeCombineBlobs, slices.Collect(maps.Values(blobsAfter)), cmpopts.SortSlices(less)))
	}

	// Trigger shutdown — the uploader should flush one more time.
	cancel()
	blobsFinal := waitForBlobs(context.Background(), t, bucket, len(wantAfterCombineBlobs), 10*time.Second)
	gotFinal := slices.Sorted(maps.Values(blobsFinal))
	wantFinal := slices.Sorted(slices.Values(wantAfterCombineBlobs))
	if !slices.Equal(gotFinal, wantFinal) {
		t.Errorf("blobs after cancel mismatch: %s", cmp.Diff(wantAfterCombineBlobs, slices.Collect(maps.Values(blobsFinal)), cmpopts.SortSlices(less)))
	}
}

func TestBlobUploaderNoop(t *testing.T) {
	dir := t.TempDir()
	blobDir := t.TempDir()

	bucketName := "file://" + blobDir

	uploader := NewUploader(dir, bucketName, 1*time.Minute)

	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond*80)
	defer cancel()
	uploader.Run(ctx)
}

func TestBlobUpload(t *testing.T) {
	blobDir := t.TempDir()

	ctx := t.Context()
	bucketName := "file://" + blobDir
	bucket, err := blob.OpenBucket(ctx, bucketName)
	if err != nil {
		t.Fatalf("Failed to create a bucket: %v", err)
	}

	// Upload the files
	for i := range 5 {
		var filename string
		if i%2 == 0 {
			filename = fmt.Sprintf("unit/test/%d", i)
		} else {
			filename = fmt.Sprintf("unit-test-%d", i)
		}
		contents := fmt.Sprintf("UNIT TEST: %d\n", i)
		buf := bytes.NewBuffer([]byte(contents))
		if err := Upload(ctx, buf, bucketName, filename); err != nil {
			t.Errorf("Failed to upload file %d: %v\n", i, err)
		}
	}

	blobsBefore, err := getFiles(ctx, bucket)
	if err != nil {
		t.Errorf("Failed to read files from blobstore: %v", err)
	}
	if !cmp.Equal(blobsBefore, wantBeforeBlobs) {
		t.Errorf("Did not get expected blobs %s", cmp.Diff(wantBeforeBlobs, blobsBefore))
	}
}

func getFiles(ctx context.Context, bucket *blob.Bucket) (map[string]string, error) {
	blobs := map[string]string{}
	iter := bucket.List(nil)
	for {
		obj, err := iter.Next(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return blobs, err
		}
		data, err := bucket.ReadAll(ctx, obj.Key)
		if err != nil {
			return blobs, err
		}
		blobs[obj.Key] = string(data)
	}
	return blobs, nil
}

func TestBufferWriteToBucket(t *testing.T) {
	ctx := t.Context()
	testFilename := filepath.Join("testdata", "long_event_line.json")
	longEventLine, err := os.ReadFile(testFilename)
	if err != nil {
		t.Fatalf("Failed to read long_event_line.json: %v", err)
	}
	u := &uploader{}

	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	dstKey := "testkey-dst"
	writer, err := bucket.NewWriter(ctx, "testkey-dst", nil)

	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}

	err = u.BufferWriteToBucket(writer, testFilename)
	if err != nil {
		t.Fatalf("failed to upload file to blobstore: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := bucket.ReadAll(ctx, dstKey)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(got, longEventLine) {
		t.Errorf("got %v, want %v", got, longEventLine)
	}
}
