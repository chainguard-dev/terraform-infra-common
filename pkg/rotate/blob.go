/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package rotate

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chainguard-dev/clog"

	"gocloud.dev/blob"

	// Add gcsblob support that we need to support gs:// prefixes
	_ "gocloud.dev/blob/gcsblob"
)

type Uploader interface {
	Run(ctx context.Context) error
}

func NewUploader(source, bucket string, flushInterval time.Duration) Uploader {
	return &uploader{
		source:        source,
		bucket:        bucket,
		flushInterval: flushInterval,
	}
}

type uploader struct {
	source        string
	bucket        string
	flushInterval time.Duration
}

func (u *uploader) Run(ctx context.Context) error {
	clog.InfoContextf(ctx, "Uploading combined logs from %s to %s every %g minutes", u.source, u.bucket, u.flushInterval.Minutes())

	done := false

	for {
		// This must be Background since we need to be able to upload even
		// after receiving SIGTERM.
		bgCtx := context.Background()
		bucket, err := blob.OpenBucket(bgCtx, u.bucket)
		if err != nil {
			return err
		}
		defer bucket.Close()

		fileName := strconv.FormatInt(time.Now().UnixNano(), 10)

		fileMap := make(map[string][]string)
		processed := 0

		if err := filepath.WalkDir(u.source, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// Skip non-regular files.
			if !d.Type().IsRegular() {
				return nil
			}
			relPath, err := filepath.Rel(u.source, path)
			if err != nil {
				return err
			}
			dir, base := filepath.Split(relPath)
			if _, ok := fileMap[dir]; !ok {
				fileMap[dir] = []string{base}
			} else {
				fileMap[dir] = append(fileMap[dir], base)
			}

			return nil
		}); err != nil {
			return err
		}
		for k, v := range fileMap {
			clog.InfoContextf(ctx, "Found %d files in dir %s to process", len(v), k)
		}

		for dir, files := range fileMap {
			// Setup the GCS object with the filename to write to
			writer, err := bucket.NewWriter(bgCtx, filepath.Join(dir, fileName), nil)
			if err != nil {
				return err
			}

			for _, f := range files {
				if err := u.BufferWriteToBucket(writer, filepath.Join(u.source, dir, f)); err != nil {
					return fmt.Errorf("failed to upload file to blobstore: %s, %w", filepath.Join(dir, fileName), err)
				}
				processed++
			}

			if err := writer.Close(); err != nil {
				return fmt.Errorf("failed to close blob file: %s %w", fileName, err)
			}

			for _, f := range files {
				path := filepath.Join(u.source, dir, f)
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("failed to delete file: %s %w", path, err)
				}
			}
		}

		if processed > 0 {
			clog.InfoContextf(ctx, "Processed %d files to blobstore", processed)
		}
		if done {
			clog.InfoContextf(ctx, "Exiting flush Run loop")
			return nil
		}
		select {
		case <-time.After(u.flushInterval):
		case <-ctx.Done():
			clog.InfoContext(ctx, "Flushing one more time")
			done = true
		}
	}
}

func Upload(ctx context.Context, fr io.Reader, bucket, fileName string) error {
	b, err := blob.OpenBucket(ctx, bucket)
	if err != nil {
		return err
	}
	defer b.Close()
	// Setup the blob with the filename to write to
	writer, err := b.NewWriter(ctx, fileName, nil)
	if err != nil {
		return err
	}
	n, err := writer.ReadFrom(fr)
	if err != nil {
		return err
	}
	fmt.Printf("Wrote %d bytes\n", n)
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close blob file %w", err)
	}
	return nil
}

func (u *uploader) BufferWriteToBucket(writer *blob.Writer, src string) (err error) {
	f, err := os.Open(src)
	if err != nil {
		return err
	}

	defer func() {
		ferr := f.Close()
		if ferr != nil {
			err = fmt.Errorf("failed to close source file: %s %w", src, err)
		}
	}()

	s := bufio.NewScanner(f)
	// Increase the buffer size. Here we set it to 5MB, this is because the default buffer size is 64KB and some
	// log files that come from broker events can contain very long lines.
	buf := make([]byte, 0, 1024*1024*5) // Initial size of 0, max size of 5MB
	s.Buffer(buf, cap(buf))

	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if len(line) == 0 {
			continue
		}
		if _, err := writer.Write([]byte(line + "\n")); err != nil {
			return err
		}
	}
	if s.Err() != nil {
		// log the error and use alerting to investigates errors
		clog.Errorf("bufio scan error: %v", s.Err())
	}

	return nil
}
