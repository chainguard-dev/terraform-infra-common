/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package engine

import (
	"cmp"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool"
)

// Record is the versioned wire format. An empty Owner denotes a released lease.
type Record struct {
	Version int       `json:"version"`
	Owner   string    `json:"owner"`
	Expires time.Time `json:"expires"`
}

// ErrConflict is an unsuccessful conditional write, including a missing version.
var ErrConflict = errors.New("lease record changed")

// Store atomically reads records with their versions and conditionally writes.
// A zero generation denotes absence; successful writes return a nonzero version.
// Implementations must be strongly consistent and obey context cancellation.
type Store interface {
	Read(context.Context, string) (Record, int64, error)
	Write(context.Context, string, Record, int64) (int64, error)
}

// New constructs the common lease protocol over a store.
func New(store Store, cfg leasepool.Config) (leasepool.Interface, error) {
	cfg.TTL = cmp.Or(cfg.TTL, 30*time.Second)
	cfg.AcquireTimeout = cmp.Or(cfg.AcquireTimeout, 5*time.Second)
	cfg.RenewInterval = cmp.Or(cfg.RenewInterval, cfg.TTL/3)
	cfg.ClockSkew = cmp.Or(cfg.ClockSkew, 2*time.Second)
	if cfg.AcquireTimeout <= 0 || store == nil || cfg.TTL <= 0 || cfg.ClockSkew < 0 || cfg.ClockSkew >= cfg.TTL || cfg.RenewInterval <= 0 || cfg.RenewInterval >= cfg.TTL {
		return nil, errors.New("invalid lease store or timing configuration")
	}
	return &pool{store: store, cfg: cfg}, nil
}

type pool struct {
	store Store
	cfg   leasepool.Config
}

func (p *pool) TryAcquire(ctx context.Context, name string) (leasepool.Lease, error) {
	if name == "" || len(name) > 512 {
		return nil, errors.New("lease name must contain 1 to 512 bytes")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	attempt, stop := context.WithTimeout(ctx, p.cfg.AcquireTimeout)
	defer stop()
	rec, gen, err := p.store.Read(attempt, name)
	if err != nil {
		return nil, fmt.Errorf("read lease: %w", err)
	}
	if gen != 0 {
		if rec.Version != 1 || (rec.Owner != "" && rec.Expires.IsZero()) {
			return nil, errors.New("invalid lease record")
		}
		if rec.Owner != "" && time.Now().Before(rec.Expires.Add(p.cfg.ClockSkew)) {
			return nil, leasepool.ErrBusy
		}
	}
	// Stamp before the RPC: a delayed success must not extend local ownership
	// beyond the expiration that contenders observe.
	until := time.Now().Add(p.cfg.TTL)
	rec = Record{Version: 1, Owner: rand.Text(), Expires: until.UTC()}
	gen, err = p.store.Write(attempt, name, rec, gen)
	if errors.Is(err, ErrConflict) {
		return nil, leasepool.ErrBusy
	}
	if err != nil {
		return nil, fmt.Errorf("acquire lease: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, p.releaseUnclaimed(ctx, name, gen))
	}
	if !time.Now().Before(until) {
		return nil, errors.Join(leasepool.ErrLost, p.releaseUnclaimed(ctx, name, gen))
	}
	owned, cancel := context.WithCancelCause(ctx)
	renewCtx, cancelRenew := context.WithCancel(context.WithoutCancel(owned))
	l := &lease{pool: p, name: name, rec: rec, gen: gen, ctx: owned, cancel: cancel, cancelRenew: cancelRenew, renewed: make(chan time.Time), done: make(chan struct{})}
	go l.maintain(renewCtx, until)
	return l, nil
}

// A confirmed acquisition that cannot be handed to the caller can be released
// safely using its known generation. An ambiguous write never takes this path.
func (p *pool) releaseUnclaimed(ctx context.Context, name string, gen int64) error {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), p.cfg.AcquireTimeout)
	defer cancel()
	_, err := p.store.Write(cleanup, name, Record{Version: 1}, gen)
	if errors.Is(err, ErrConflict) {
		return leasepool.ErrLost
	}
	return err
}

type lease struct {
	pool        *pool
	name        string
	rec         Record
	gen         int64 // Written by maintain; Release reads only after done closes.
	ctx         context.Context
	cancel      context.CancelCauseFunc
	cancelRenew context.CancelFunc
	renewed     chan time.Time
	done        chan struct{}
	once        sync.Once
	releaseErr  error
}

func (l *lease) Context() context.Context { return l.ctx }

func (l *lease) maintain(renewCtx context.Context, until time.Time) {
	defer close(l.done)
	// The watchdog is independent of storage I/O. A stalled renewal cannot
	// keep the ownership context alive beyond its last confirmed deadline.
	// Parent cancellation and Release stop new renewals, but do not interrupt
	// an in-flight CAS: its definitive generation is needed for the tombstone.
	watched := make(chan struct{})
	go func() {
		defer close(watched)
		timer := time.NewTimer(time.Until(until))
		defer timer.Stop()
		for {
			select {
			case <-renewCtx.Done():
				return
			case <-timer.C:
				l.cancel(leasepool.ErrLost)
				l.cancelRenew()
				return
			case next := <-l.renewed:
				if !time.Now().Before(until) {
					l.cancel(leasepool.ErrLost)
					l.cancelRenew()
					return
				}
				until = next
				timer.Reset(time.Until(next))
			}
		}
	}()
	defer func() { l.cancelRenew(); <-watched }()
	ticker := time.NewTicker(l.pool.cfg.RenewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-l.ctx.Done():
			return
		case <-ticker.C:
			if l.ctx.Err() != nil {
				return
			}
			next := time.Now().Add(l.pool.cfg.TTL)
			rec := l.rec
			rec.Expires = next.UTC()
			gen, err := l.pool.store.Write(renewCtx, l.name, rec, l.gen)
			if err != nil {
				l.cancel(fmt.Errorf("renew lease: %w", errors.Join(leasepool.ErrLost, err)))
				return
			}
			l.gen = gen
			select {
			case <-l.ctx.Done():
				return
			case l.renewed <- next:
			}
		}
	}
}

func (l *lease) Release(ctx context.Context) error {
	l.once.Do(func() {
		l.cancel(leasepool.ErrReleased)
		// Let an in-flight CAS return its generation, but do not wait out the
		// whole TTL for a stalled RPC. Cancellation leaves the last confirmed
		// generation intact; a tombstone then conflicts if that RPC committed.
		drain, cancelDrain := context.WithTimeout(ctx, l.pool.cfg.AcquireTimeout)
		defer cancelDrain()
		select {
		case <-l.done:
		case <-drain.Done():
			l.cancelRenew()
			<-l.done // Store implementations must obey cancellation.
		}
		if err := ctx.Err(); err != nil {
			l.releaseErr = err
			return
		}
		// Draining must not consume the tombstone's independent I/O budget.
		cleanup, cancelCleanup := context.WithTimeout(ctx, l.pool.cfg.AcquireTimeout)
		defer cancelCleanup()
		_, err := l.pool.store.Write(cleanup, l.name, Record{Version: 1}, l.gen)
		if errors.Is(err, ErrConflict) {
			err = leasepool.ErrLost
		}
		l.releaseErr = err
	})
	return l.releaseErr
}

var _ leasepool.Interface = (*pool)(nil)
var _ leasepool.Lease = (*lease)(nil)
