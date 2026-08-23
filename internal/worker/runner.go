package worker

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/outbox"
)

type Runner struct {
	Outbox   *outbox.Service
	WorkerID string
	Broker   func(context.Context, domain.OutboxMessage) error
	Interval time.Duration
	logger   *slog.Logger
	wg       sync.WaitGroup
}

func NewRunner(outboxSvc *outbox.Service, id string, broker func(context.Context, domain.OutboxMessage) error, logger *slog.Logger) *Runner {
	if logger == nil {
		logger = slog.Default()
	}
	return &Runner{Outbox: outboxSvc, WorkerID: id, Broker: broker, Interval: 100 * time.Millisecond, logger: logger}
}
func (r *Runner) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-ticker.C:
			item, err := r.Outbox.Claim(ctx, r.WorkerID, now)
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}
			if err != nil {
				r.logger.Error("claim outbox", "error", err)
				continue
			}
			if err := r.publish(detachedPublishContext(ctx), item, now); err != nil {
				r.logger.Warn("publish outbox", "error", err, "message_id", item.ID)
			}
		}
	}
}
func (r *Runner) publish(ctx context.Context, item domain.OutboxMessage, now time.Time) error {
	if r.Broker != nil {
		if err := r.Broker(ctx, item); err != nil {
			return r.Outbox.Fail(ctx, item, r.WorkerID, now, err)
		}
	}
	return r.Outbox.Publish(ctx, item, r.WorkerID, now)
}
func (r *Runner) Start(ctx context.Context) {
	r.wg.Add(1)
	go func() { defer r.wg.Done(); _ = r.Run(ctx) }()
}
func (r *Runner) Wait() { r.wg.Wait() }
