package app

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/audit"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/auth"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/billing"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/bunkering"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/fuel"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/idempotency"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/incident"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/outbox"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/quality"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/schedule"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/terminal"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/vessel"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/worker"
)

type Runtime struct {
	Store      *storage.Store
	Auth       *auth.Service
	Vessel     *vessel.Service
	Terminal   *terminal.Service
	Fuel       *fuel.Service
	Schedule   *schedule.Service
	Bunkering  *bunkering.Service
	Quality    *quality.Service
	Billing    *billing.Service
	Audit      *audit.Service
	Outbox     *outbox.Service
	Incident   *incident.Service
	Operations any
	Worker     *worker.Runner
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
	ready      bool
	closed     bool
}

func New(ctx context.Context, cfg Config, logger *slog.Logger) (*Runtime, error) {
	store, err := storage.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	auditSvc := audit.New(store)
	outboxSvc := outbox.New(store)
	fuelSvc := fuel.New(store, auditSvc)
	qualitySvc := quality.New(store, auditSvc)
	idemp := idempotency.New(store)
	rt := &Runtime{Store: store, Auth: auth.New(store), Audit: auditSvc, Outbox: outboxSvc, Fuel: fuelSvc, Quality: qualitySvc}
	rt.Vessel = vessel.New(store, auditSvc)
	rt.Terminal = terminal.New(store)
	rt.Schedule = schedule.New(store, auditSvc, outboxSvc)
	rt.Bunkering = bunkering.New(store, auditSvc, outboxSvc, fuelSvc, qualitySvc, idemp)
	rt.Billing = billing.New(store, auditSvc, outboxSvc)
	rt.Incident = incident.New(store, auditSvc, outboxSvc)
	rt.Worker = worker.NewRunner(outboxSvc, "runtime-worker", func(ctx context.Context, item domain.OutboxMessage) error { return nil }, logger)
	return rt, nil
}

func (r *Runtime) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	r.cancel = cancel
	r.mu.Lock()
	r.ready = true
	r.mu.Unlock()
	r.Worker.Start(ctx)
}
func (r *Runtime) Ready() bool { r.mu.RLock(); defer r.mu.RUnlock(); return r.ready && !r.closed }
func (r *Runtime) Shutdown(ctx context.Context) error {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return nil
	}
	r.mu.RUnlock()
	if r.cancel != nil {
		r.cancel()
	}
	done := make(chan struct{})
	go func() { r.Worker.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}
	r.mu.Lock()
	r.ready = false
	r.closed = true
	r.mu.Unlock()
	if err := r.Store.Close(); err != nil {
		return fmt.Errorf("close store: %w", err)
	}
	return nil
}

func (r *Runtime) DrainFor(d time.Duration) { time.Sleep(d) }
