/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package filesystem

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool"
	"golang.org/x/sys/unix"
)

// New coordinates independent processes using an existing host directory.
// Simple names retain the legacy <name>.lock filename. The directory must remain
// stable for the lifetime of all participants. New holds no open files of its own.
func New(directory string) (leasepool.Interface, error) {
	if directory == "" {
		return nil, errors.New("lease directory is required")
	}
	directory, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("resolve lease directory: %w", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, fmt.Errorf("open lease directory: %w", err)
	}
	defer root.Close()
	return &pool{directory: directory}, nil
}

type pool struct {
	directory string
}

func (p *pool) TryAcquire(ctx context.Context, name string) (leasepool.Lease, error) {
	if name == "" {
		return nil, errors.New("lease name is required")
	}
	if len(name) > 512 {
		return nil, fmt.Errorf("lease name is %d bytes; maximum is 512", len(name))
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(p.directory)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	filename := name + ".lock"
	// Preserve the locks held by existing callers during a rolling upgrade.
	// Other leasepool names need encoding to fit one filename. A distinct
	// suffix keeps encoded names disjoint from literal <name>.lock files.
	if len(filename) > 255 || name == "." || name == ".." || strings.ContainsAny(name, "/\\") || strings.ContainsRune(name, 0) {
		filename = fmt.Sprintf("%x.sha256-lock", sha256.Sum256([]byte(name)))
	}
	// Create once, then open the existing inode. Separating these operations
	// avoids concurrent O_CREATE opens through os.Root failing on macOS.
	f, err := root.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if errors.Is(err, os.ErrExist) {
		f, err = root.OpenFile(filename, os.O_RDWR, 0o600)
	}
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, unix.EWOULDBLOCK) {
			return nil, leasepool.ErrBusy
		}
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		_ = f.Close()
		return nil, err
	}
	owned, cancel := context.WithCancelCause(ctx)
	return &fileLease{
		ctx:    owned,
		cancel: cancel,
		file:   f,
	}, nil
}

type fileLease struct {
	ctx    context.Context
	cancel context.CancelCauseFunc
	file   *os.File
	once   sync.Once
	err    error
}

func (l *fileLease) Context() context.Context { return l.ctx }

func (l *fileLease) Release(context.Context) error {
	// Closing the descriptor releases the lock even if the caller's context
	// has ended. Do not unlink it: another participant may have it open.
	l.once.Do(func() {
		l.cancel(leasepool.ErrReleased)
		l.err = l.file.Close()
	})
	return l.err
}

var _ leasepool.Interface = (*pool)(nil)
var _ leasepool.Lease = (*fileLease)(nil)
