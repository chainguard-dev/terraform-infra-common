/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package filesystem_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool"
	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool/filesystem"
	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool/internal/contract"
	"golang.org/x/sys/unix"
)

func TestContract(t *testing.T) {
	directory := t.TempDir()
	contract.Test(t, func(t *testing.T) leasepool.Interface {
		t.Helper()
		p, err := filesystem.New(directory)
		if err != nil {
			t.Fatal(err)
		}
		return p
	})
}

func TestInvalidDirectory(t *testing.T) {
	for _, directory := range []string{"", filepath.Join(t.TempDir(), "missing")} {
		t.Run(directory, func(t *testing.T) {
			if _, err := filesystem.New(directory); err == nil {
				t.Fatal("New succeeded, want invalid directory error")
			}
		})
	}
}

func TestNames(t *testing.T) {
	directory := t.TempDir()
	p, err := filesystem.New(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", strings.Repeat("x", 513)} {
		if _, err := p.TryAcquire(t.Context(), name); err == nil {
			t.Fatalf("accepted invalid name of length %d", len(name))
		}
	}
	for _, name := range []string{"../../outside", "pool/namespace", strings.Repeat("x", 512)} {
		l, err := p.TryAcquire(t.Context(), name)
		if err != nil {
			t.Fatal(err)
		}
		if err := l.Release(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("lock files: got %d, want 3", len(entries))
	}
}

func TestSymlinkEscape(t *testing.T) {
	directory := t.TempDir()
	name := "escape"
	if err := os.Symlink(filepath.Join(t.TempDir(), "outside"), filepath.Join(directory, name+".lock")); err != nil {
		t.Fatal(err)
	}
	p, err := filesystem.New(directory)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.TryAcquire(t.Context(), name); err == nil {
		t.Fatal("acquired through an escaping symlink")
	}
}

func TestConcurrentRelease(t *testing.T) {
	p, err := filesystem.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	l, err := p.TryAcquire(t.Context(), t.Name())
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if err := l.Release(t.Context()); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if !errors.Is(context.Cause(l.Context()), leasepool.ErrReleased) {
		t.Fatalf("cause: got %v, want released", context.Cause(l.Context()))
	}
}

func TestProcessDeath(t *testing.T) {
	directory := t.TempDir()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProcessHelper$")
	cmd.Env = append(os.Environ(), "LEASEPOOL_HELPER_DIRECTORY="+directory)
	cmd.Stderr = os.Stderr
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	stop := sync.OnceFunc(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	t.Cleanup(stop)
	if ready, err := bufio.NewReader(output).ReadString('\n'); err != nil || ready != "owned\n" {
		t.Fatalf("child readiness: got %q, %v; want owned", ready, err)
	}
	p, err := filesystem.New(directory)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.TryAcquire(t.Context(), "pool"); !errors.Is(err, leasepool.ErrBusy) {
		t.Fatalf("live child: got %v, want busy", err)
	}
	stop()
	l, err := p.TryAcquire(t.Context(), "pool")
	if err != nil {
		t.Fatalf("after child death: %v", err)
	}
	if err := l.Release(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestProcessHelper(t *testing.T) {
	directory := os.Getenv("LEASEPOOL_HELPER_DIRECTORY")
	if directory == "" {
		return
	}
	p, err := filesystem.New(directory)
	if err != nil {
		t.Fatal(err)
	}
	l, err := p.TryAcquire(t.Context(), "pool")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Release(t.Context())
	fmt.Println("owned")
	if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
		t.Fatal(err)
	}
}

// Use the original local backend's filename and OS locking operation directly;
// a second new-backend client would not catch a lock-namespace migration bug.
func TestLegacyLockCompatibility(t *testing.T) {
	for _, name := range []string{"pool", "node-01", "qemu.amd64"} {
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			legacy, err := os.OpenFile(filepath.Join(directory, name+".lock"), os.O_CREATE|os.O_RDWR, 0o600)
			if err != nil {
				t.Fatal(err)
			}
			defer legacy.Close()
			if err := unix.Flock(int(legacy.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
				t.Fatal(err)
			}
			p, err := filesystem.New(directory)
			if err != nil {
				t.Fatal(err)
			}
			if l, err := p.TryAcquire(t.Context(), name); !errors.Is(err, leasepool.ErrBusy) {
				if l != nil {
					_ = l.Release(t.Context())
				}
				t.Fatalf("legacy owner versus new contender: got %v, want busy", err)
			}
			if err := unix.Flock(int(legacy.Fd()), unix.LOCK_UN); err != nil {
				t.Fatal(err)
			}
			l, err := p.TryAcquire(t.Context(), name)
			if err != nil {
				t.Fatal(err)
			}
			defer l.Release(t.Context())
			if err := unix.Flock(int(legacy.Fd()), unix.LOCK_EX|unix.LOCK_NB); !errors.Is(err, unix.EWOULDBLOCK) {
				t.Fatalf("new owner versus legacy contender: got %v, want EWOULDBLOCK", err)
			}
		})
	}
}
