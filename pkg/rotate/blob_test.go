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
	"os"
	"path/filepath"
	"testing"
	"time"

	"gocloud.dev/blob/memblob"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	"golang.org/x/exp/maps"
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

func TestBlobUploader(t *testing.T) {
	dir := t.TempDir()
	blobDir := t.TempDir()

	cancelCtx, cancel := context.WithCancel(context.Background())
	bucketName := "file://" + blobDir
	bucket, err := blob.OpenBucket(cancelCtx, bucketName)
	if err != nil {
		t.Fatalf("Failed to create a bucket: %v", err)
	}
	defer os.RemoveAll(dir) // clean up

	uploader := NewUploader(dir, bucketName, 1*time.Minute)

	// Create a few files there to be uploaded
	for i := 0; i < 5; i++ {
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

	// Give a little time for uploads, then make sure we have all the files
	// there.
	time.Sleep(3 * time.Second)
	blobsBefore, err := getFiles(cancelCtx, bucket)
	if err != nil {
		t.Errorf("Failed to read files from blobstore: %v", err)
	}
	less := func(a, b string) bool { return a < b }
	if !cmp.Equal(maps.Values(blobsBefore), wantBeforeCombineBlobs, cmpopts.SortSlices(less)) {
		t.Errorf("Did not get expected blobs '%s'\n%v\n%v", cmp.Diff(wantBeforeCombineBlobs, maps.Values(blobsBefore), cmpopts.SortSlices(less)), wantBeforeCombineBlobs, maps.Values(blobsBefore))
	}

	// Then write one more file and trigger cancel, so this too should be now
	// written there.
	filename := fmt.Sprintf("%s/last", dir)
	err = os.WriteFile(filename, []byte("LAST UT"), 0600)
	if err != nil {
		t.Errorf("Failed to write: %v", err)
	}

	// Now one more check, make sure that the file does not get
	// immediately uploaded. So check that the files have not been
	// uploaded. Then trigger shutdown and ensure the file then gets uploaded
	// as 'not per schedule', but aggressive flush during shutdown.
	blobsAfter, err := getFiles(cancelCtx, bucket)
	if err != nil {
		t.Errorf("Failed to read files from blobstore: %v", err)
	}
	if !cmp.Equal(maps.Values(blobsAfter), wantBeforeCombineBlobs, cmpopts.SortSlices(less)) {
		t.Errorf("Did not get expected blobs '%s'\n%v\n%v", cmp.Diff(wantBeforeCombineBlobs, maps.Values(blobsAfter), cmpopts.SortSlices(less)), wantBeforeCombineBlobs, maps.Values(blobsAfter))
	}

	cancel()
	time.Sleep(3 * time.Second)
	blobsAfter, err = getFiles(cancelCtx, bucket)
	if err != nil {
		t.Errorf("Failed to read files from blobstore: %v", err)
	}
	if !cmp.Equal(maps.Values(blobsAfter), wantAfterCombineBlobs, cmpopts.SortSlices(less)) {
		t.Errorf("Did not get expected blobs '%s'\n%v\n%v", cmp.Diff(wantAfterCombineBlobs, maps.Values(blobsAfter), cmpopts.SortSlices(less)), wantAfterCombineBlobs, maps.Values(blobsAfter))
	}
}

func TestBlobUploaderNoop(t *testing.T) {
	dir := t.TempDir()
	defer os.RemoveAll(dir) // clean up
	blobDir := t.TempDir()

	bucketName := "file://" + blobDir

	uploader := NewUploader(dir, bucketName, 1*time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*80)
	defer cancel()
	uploader.Run(ctx)
}

func TestBlobUpload(t *testing.T) {
	blobDir := t.TempDir()

	ctx := context.Background()
	bucketName := "file://" + blobDir
	bucket, err := blob.OpenBucket(ctx, bucketName)
	if err != nil {
		t.Fatalf("Failed to create a bucket: %v", err)
	}

	// Upload the files
	for i := 0; i < 5; i++ {
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
	ctx := context.Background()
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
