package quality

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/audit"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
	"github.com/google/uuid"
)

type Service struct {
	Store *storage.Store
	Audit *audit.Service
}
type SampleInput struct{ ChainRef, Receiver string }

func New(store *storage.Store, auditSvc *audit.Service) *Service {
	return &Service{Store: store, Audit: auditSvc}
}

func (s *Service) ReceiveSamples(ctx context.Context, actor domain.Actor, orderID string, inputs []SampleInput, requestID string) ([]domain.Sample, error) {
	if len(inputs) == 0 {
		return nil, domain.ErrInvalid
	}
	items := make([]domain.Sample, 0, len(inputs))
	seen := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		if input.ChainRef == "" || input.Receiver == "" {
			return nil, domain.ErrInvalid
		}
		if _, ok := seen[input.ChainRef]; ok {
			return nil, fmt.Errorf("%w: duplicate chain reference", domain.ErrConflict)
		}
		seen[input.ChainRef] = struct{}{}
		items = append(items, domain.Sample{ID: uuid.NewString(), OrderID: orderID, ChainRef: input.ChainRef, Receiver: input.Receiver, State: domain.QualityReceived})
	}
	if s.Store.Hooks.SamplesPrelude == nil {
		err := s.Store.WithTx(ctx, func(tx *sql.Tx) error {
			var tenantID string
			if err := tx.QueryRowContext(ctx, `SELECT tenant_id FROM transfer_orders WHERE id=? AND tenant_id=?`, orderID, actor.TenantID).Scan(&tenantID); err == sql.ErrNoRows {
				return domain.ErrNotFound
			} else if err != nil {
				return err
			}
			for _, item := range items {
				if _, err := tx.ExecContext(ctx, `INSERT INTO samples(id, order_id, chain_ref, receiver, quality_state, created_at) VALUES (?, ?, ?, ?, ?, ?)`, item.ID, item.OrderID, item.ChainRef, item.Receiver, item.State, storage.StringTime(time.Now())); err != nil {
					return fmt.Errorf("insert sample: %w", err)
				}
				if _, err := tx.ExecContext(ctx, `INSERT INTO custody_events(id, sample_id, actor_id, action, created_at) VALUES (?, ?, ?, 'received', ?)`, uuid.NewString(), item.ID, actor.ID, storage.StringTime(time.Now())); err != nil {
					return err
				}
			}
			if err := s.Audit.Record(ctx, tx, actor, "samples.received", orderID, requestID); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		return items, nil
	}
	err := s.Store.WithCommittedPrelude(ctx, func(ctx context.Context, db *sql.DB) error {
		for _, item := range items {
			if _, err := db.ExecContext(ctx, `INSERT INTO samples(id,order_id,chain_ref,receiver,quality_state,created_at) VALUES (?,?,?,?,?,?)`, item.ID, item.OrderID, item.ChainRef, item.Receiver, item.State, storage.StringTime(time.Now())); err != nil {
				return fmt.Errorf("insert sample: %w", err)
			}
		}
		close(s.Store.Hooks.SamplesPrelude)
		return nil
	}, func(tx *sql.Tx) error {
		var tenantID string
		if err := tx.QueryRowContext(ctx, `SELECT tenant_id FROM transfer_orders WHERE id=? AND tenant_id=?`, orderID, actor.TenantID).Scan(&tenantID); err == sql.ErrNoRows {
			return domain.ErrNotFound
		} else if err != nil {
			return err
		}
		for _, item := range items {
			if _, err := tx.ExecContext(ctx, `INSERT INTO custody_events(id, sample_id, actor_id, action, created_at) VALUES (?, ?, ?, 'received', ?)`, uuid.NewString(), item.ID, actor.ID, storage.StringTime(time.Now())); err != nil {
				return err
			}
		}
		if err := s.Audit.Record(ctx, tx, actor, "samples.received", orderID, requestID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) Review(ctx context.Context, actor domain.Actor, sampleID string, approved bool, requestID string) error {
	return s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		var state, orderID string
		if err := tx.QueryRowContext(ctx, `SELECT quality_state, order_id FROM samples WHERE id=?`, sampleID).Scan(&state, &orderID); err == sql.ErrNoRows {
			return domain.ErrNotFound
		} else if err != nil {
			return err
		}
		if state != string(domain.QualityReceived) {
			return fmt.Errorf("%w: sample state", domain.ErrConflict)
		}
		next := domain.QualityRejected
		if approved {
			next = domain.QualityApproved
		}
		if _, err := tx.ExecContext(ctx, `UPDATE samples SET quality_state=? WHERE id=?`, next, sampleID); err != nil {
			return err
		}
		return s.Audit.Record(ctx, tx, actor, "sample.reviewed", sampleID, requestID)
	})
}

func (s *Service) GetSample(ctx context.Context, actor domain.Actor, sampleID string) (domain.Sample, error) {
	var item domain.Sample
	var state, created string
	err := s.Store.DB.QueryRowContext(ctx, `SELECT s.id, s.order_id, s.chain_ref, s.receiver, s.quality_state, s.created_at FROM samples s JOIN transfer_orders o ON o.id=s.order_id WHERE s.id=? AND o.tenant_id=?`, sampleID, actor.TenantID).Scan(&item.ID, &item.OrderID, &item.ChainRef, &item.Receiver, &state, &created)
	if err == sql.ErrNoRows {
		return item, domain.ErrNotFound
	}
	if err != nil {
		return item, err
	}
	item.State = domain.QualityState(state)
	_ = created
	rows, err := s.Store.DB.QueryContext(ctx, `SELECT id, sample_id, actor_id, action, created_at FROM custody_events WHERE sample_id=? ORDER BY created_at, id`, sampleID)
	if err != nil {
		return item, err
	}
	defer rows.Close()
	for rows.Next() {
		var event domain.CustodyEvent
		var when string
		if err := rows.Scan(&event.ID, &event.SampleID, &event.ActorID, &event.Action, &when); err != nil {
			return item, err
		}
		event.CreatedAt, err = storage.ParseTime(when)
		if err != nil {
			return item, err
		}
		item.History = append(item.History, event)
	}
	return item, rows.Err()
}

func (s *Service) ApprovedForOrder(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, orderID string) (bool, error) {
	var count int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM samples WHERE order_id=? AND quality_state='approved'`, orderID).Scan(&count)
	return count > 0, err
}
