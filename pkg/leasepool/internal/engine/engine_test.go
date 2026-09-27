/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package engine

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool"
)

// faultStore models atomic CAS, including writes that commit before their
// response is lost. It is the protocol's storage boundary, not a fake lease.
type faultStore struct {
	mu          sync.Mutex
	rec         Record
	gen         int64
	writeErr    error
	commitError bool
	block       bool
	readErr     error
	blockRead   bool
}

func (s *faultStore) Read(ctx context.Context, _ string) (Record, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blockRead {
		<-ctx.Done()
	}
	return s.rec, s.gen, errors.Join(ctx.Err(), s.readErr)
}
func (s *faultStore) Write(ctx context.Context, _ string, r Record, gen int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.block {
		<-ctx.Done()
		return 0, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if gen != s.gen {
		return 0, ErrConflict
	}
	if s.writeErr != nil && !s.commitError {
		return 0, s.writeErr
	}
	s.gen++
	s.rec = r
	return s.gen, s.writeErr
}

func TestExpiryAndStaleRelease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := &faultStore{}
		p, err := New(s, leasepool.Config{})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(t.Context())
		old, err := p.TryAcquire(ctx, "pool")
		if err != nil {
			t.Fatal(err)
		}
		cancel()
		synctest.Wait()
		time.Sleep(31 * time.Second)
		if _, err := p.TryAcquire(t.Context(), "pool"); !errors.Is(err, leasepool.ErrBusy) {
			t.Fatalf("clock-skew margin: got %v, want busy", err)
		}
		time.Sleep(2 * time.Second)
		next, err := p.TryAcquire(t.Context(), "pool")
		if err != nil {
			t.Fatal(err)
		}
		defer next.Release(t.Context())
		if err := old.Release(t.Context()); !errors.Is(err, leasepool.ErrLost) {
			t.Fatalf("stale release: got %v, want lost", err)
		}
		if _, err := p.TryAcquire(t.Context(), "pool"); !errors.Is(err, leasepool.ErrBusy) {
			t.Fatalf("successor lost: %v", err)
		}
	})
}

func TestRenewalFailures(t *testing.T) {
	for _, tc := range []struct {
		name          string
		block, commit bool
	}{
		{name: "rejected"}, {name: "response_lost", commit: true}, {name: "stalled", block: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				s := &faultStore{}
				p, err := New(s, leasepool.Config{})
				if err != nil {
					t.Fatal(err)
				}
				l, err := p.TryAcquire(t.Context(), "pool")
				if err != nil {
					t.Fatal(err)
				}
				synctest.Wait()
				s.mu.Lock()
				s.writeErr = errors.New("storage unavailable")
				s.commitError = tc.commit
				s.block = tc.block
				s.mu.Unlock()
				time.Sleep(31 * time.Second)
				synctest.Wait()
				if !errors.Is(context.Cause(l.Context()), leasepool.ErrLost) {
					t.Fatalf("cause: got %v, want lost", context.Cause(l.Context()))
				}
				// Release must never adopt the unknown generation of an ambiguous write.
				s.mu.Lock()
				s.writeErr = nil
				s.block = false
				s.mu.Unlock()
				err = l.Release(t.Context())
				if tc.commit && !errors.Is(err, leasepool.ErrLost) {
					t.Fatalf("ambiguous renewal release: got %v, want lost", err)
				}
			})
		})
	}
}

func TestAmbiguousAcquire(t *testing.T) {
	s := &faultStore{writeErr: errors.New("reply lost"), commitError: true}
	p, err := New(s, leasepool.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.TryAcquire(t.Context(), "pool"); err == nil || errors.Is(err, leasepool.ErrBusy) {
		t.Fatalf("acquire: got %v, want storage error", err)
	}
	if _, err := p.TryAcquire(t.Context(), "pool"); !errors.Is(err, leasepool.ErrBusy) {
		t.Fatalf("committed lease: got %v, want busy", err)
	}
}

func TestReadFailure(t *testing.T) {
	want := errors.New("storage unavailable")
	p, err := New(&faultStore{readErr: want}, leasepool.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.TryAcquire(t.Context(), "pool"); !errors.Is(err, want) {
		t.Fatalf("got %v, want storage error", err)
	}
}

func TestInvalidConfig(t *testing.T) {
	for _, cfg := range []leasepool.Config{{TTL: -1}, {RenewInterval: -1}, {TTL: time.Second}, {ClockSkew: -1}, {RenewInterval: time.Minute}} {
		if _, err := New(&faultStore{}, cfg); err == nil {
			t.Errorf("accepted config: %+v", cfg)
		}
	}
}

func TestAcquireTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p, err := New(&faultStore{blockRead: true}, leasepool.Config{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.TryAcquire(t.Context(), "pool"); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("got %v, want acquisition deadline", err)
		}
	})
}

// delayedRenewalStore commits a renewal even when cancellation loses its reply.
// Only the storage boundary is replaced; acquisition, renewal and release are real.
type delayedRenewalStore struct{ faultStore }

func (s *delayedRenewalStore) Write(ctx context.Context, name string, rec Record, gen int64) (int64, error) {
	if gen != 0 && rec.Owner != "" {
		timer := time.NewTimer(time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			if _, err := s.faultStore.Write(context.WithoutCancel(ctx), name, rec, gen); err != nil {
				return 0, err
			}
			return 0, ctx.Err()
		case <-timer.C:
		}
	}
	return s.faultStore.Write(ctx, name, rec, gen)
}

func TestReleaseDuringRenewal(t *testing.T) {
	for _, tc := range []struct {
		name              string
		cancelParent, run bool
	}{
		{name: "release"}, {name: "parent cancellation", cancelParent: true}, {name: "successful Run", run: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				p, err := New(&delayedRenewalStore{}, leasepool.Config{})
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				if tc.run {
					err = leasepool.Run(ctx, p, "pool", func(context.Context) error {
						time.Sleep(10 * time.Second)
						synctest.Wait()
						return nil
					})
				} else {
					var l leasepool.Lease
					l, err = p.TryAcquire(ctx, "pool")
					if err != nil {
						t.Fatal(err)
					}
					time.Sleep(10 * time.Second)
					synctest.Wait()
					if tc.cancelParent {
						cancel()
					}
					err = l.Release(t.Context())
				}
				if err != nil {
					t.Errorf("graceful release = %v, want nil", err)
				}
				next, err := p.TryAcquire(t.Context(), "pool")
				if err != nil {
					t.Fatalf("successor after graceful release: %v", err)
				}
				if err := next.Release(t.Context()); err != nil {
					t.Fatal(err)
				}
			})
		})
	}
}

type delayedAcquireStore struct {
	faultStore
	delay  time.Duration
	cancel context.CancelFunc
}

func (s *delayedAcquireStore) Write(ctx context.Context, name string, rec Record, gen int64) (int64, error) {
	next, err := s.faultStore.Write(ctx, name, rec, gen)
	if err == nil && gen == 0 {
		time.Sleep(s.delay)
		if s.cancel != nil {
			s.cancel()
		}
	}
	return next, err
}

func TestConfirmedAcquisitionCleanup(t *testing.T) {
	for _, tc := range []struct {
		name     string
		delay    time.Duration
		canceled bool
		want     error
	}{
		{name: "canceled", canceled: true, want: context.Canceled},
		{name: "expired", delay: 31 * time.Second, want: leasepool.ErrLost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				s := &delayedAcquireStore{delay: tc.delay}
				if tc.canceled {
					s.cancel = cancel
				}
				p, err := New(s, leasepool.Config{AcquireTimeout: time.Minute})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := p.TryAcquire(ctx, "pool"); !errors.Is(err, tc.want) {
					t.Fatalf("acquire: got %v, want %v", err, tc.want)
				}
				next, err := p.TryAcquire(t.Context(), "pool")
				if err != nil {
					t.Fatalf("confirmed acquisition stranded: %v", err)
				}
				if err := next.Release(t.Context()); err != nil {
					t.Fatal(err)
				}
			})
		})
	}
}

func TestRunReportsLossOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := &faultStore{}
		p, err := New(s, leasepool.Config{})
		if err != nil {
			t.Fatal(err)
		}
		err = leasepool.Run(t.Context(), p, "pool", func(ctx context.Context) error {
			s.mu.Lock()
			s.writeErr, s.commitError = errors.New("response lost"), true
			s.mu.Unlock()
			<-ctx.Done()
			s.mu.Lock()
			s.writeErr = nil
			s.mu.Unlock()
			return nil
		})
		if !errors.Is(err, leasepool.ErrLost) || strings.Count(err.Error(), leasepool.ErrLost.Error()) != 1 {
			t.Fatalf("Run = %v, want one ownership-loss diagnostic", err)
		}
	})
}

// Only the renewal RPC is delayed; acquisition and tombstoning still exercise
// the same atomic CAS store. A canceled renewal may or may not have committed.
type stalledRenewalStore struct {
	faultStore
	delay  time.Duration
	commit bool
}

func (s *stalledRenewalStore) Write(ctx context.Context, name string, rec Record, gen int64) (int64, error) {
	s.mu.Lock()
	renewing := gen != 0 && rec.Owner != "" && rec.Owner == s.rec.Owner
	s.mu.Unlock()
	if renewing {
		if s.commit {
			if _, err := s.faultStore.Write(ctx, name, rec, gen); err != nil {
				return 0, err
			}
		}
		if s.delay == 0 {
			<-ctx.Done()
			return 0, ctx.Err()
		}
		timer := time.NewTimer(s.delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-timer.C:
		}
	}
	return s.faultStore.Write(ctx, name, rec, gen)
}

func TestReleaseRenewalBudgets(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		run, commit                bool
		delay, budget, wantElapsed time.Duration
		want                       error
	}{
		{name: "caller deadline", wantElapsed: 5 * time.Second, want: context.DeadlineExceeded},
		{name: "Run stalled before commit", run: true, wantElapsed: 5 * time.Second},
		{name: "Run ambiguous commit", run: true, commit: true, wantElapsed: 5 * time.Second, want: leasepool.ErrLost},
		{name: "Run confirmed slow renewal", run: true, delay: 6 * time.Second, budget: 10 * time.Second, wantElapsed: 6 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				p, err := New(&stalledRenewalStore{delay: tc.delay, commit: tc.commit}, leasepool.Config{AcquireTimeout: tc.budget})
				if err != nil {
					t.Fatal(err)
				}
				var start time.Time
				if tc.run {
					err = leasepool.Run(t.Context(), p, "pool", func(context.Context) error {
						time.Sleep(10 * time.Second)
						synctest.Wait()
						start = time.Now()
						return nil
					})
				} else {
					var l leasepool.Lease
					l, err = p.TryAcquire(t.Context(), "pool")
					if err != nil {
						t.Fatal(err)
					}
					time.Sleep(10 * time.Second)
					synctest.Wait()
					start = time.Now()
					ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
					defer cancel()
					err = l.Release(ctx)
				}
				if !errors.Is(err, tc.want) {
					t.Errorf("release = %v, want %v", err, tc.want)
				}
				if got := time.Since(start); got != tc.wantElapsed {
					t.Errorf("release duration = %s, want %s", got, tc.wantElapsed)
				}
				next, err := p.TryAcquire(t.Context(), "pool")
				if tc.want != nil {
					if !errors.Is(err, leasepool.ErrBusy) {
						t.Fatalf("successor = %v, want busy", err)
					}
				} else {
					if err != nil {
						t.Fatal(err)
					}
					if err := next.Release(t.Context()); err != nil {
						t.Fatal(err)
					}
				}
			})
		})
	}
}
