package bunkering

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/audit"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/fuel"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/idempotency"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/outbox"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/quality"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
	"github.com/google/uuid"
)

type Service struct {
	Store       *storage.Store
	Audit       *audit.Service
	Outbox      *outbox.Service
	Fuel        *fuel.Service
	Quality     *quality.Service
	Idempotency *idempotency.Service
}
type CreateInput struct {
	VesselID, WindowID, FuelLotID string
	TargetKG                      float64
	IdempotencyKey                string
}

func New(store *storage.Store, auditSvc *audit.Service, outboxSvc *outbox.Service, fuelSvc *fuel.Service, qualitySvc *quality.Service, idempotencySvc *idempotency.Service) *Service {
	return &Service{Store: store, Audit: auditSvc, Outbox: outboxSvc, Fuel: fuelSvc, Quality: qualitySvc, Idempotency: idempotencySvc}
}

func (s *Service) Create(ctx context.Context, actor domain.Actor, input CreateInput, requestID string) (domain.TransferOrder, error) {
	if input.TargetKG <= 0 || input.VesselID == "" || input.WindowID == "" || input.FuelLotID == "" {
		return domain.TransferOrder{}, domain.ErrInvalid
	}
	if replay, err := s.Idempotency.Lookup(ctx, s.Store.DB, actor, input.IdempotencyKey, input); err != nil {
		return domain.TransferOrder{}, err
	} else if replay != nil {
		return s.getByID(ctx, actor, extractID(replay.Body))
	}
	item := domain.TransferOrder{ID: uuid.NewString(), TenantID: actor.TenantID, VesselID: input.VesselID, WindowID: input.WindowID, FuelLotID: input.FuelLotID, TargetKG: input.TargetKG, State: domain.StatePlanned, Version: 1}
	err := s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		var vesselStatus, windowStatus, qualityState string
		if err := tx.QueryRowContext(ctx, `SELECT status FROM vessels WHERE id=? AND tenant_id=?`, item.VesselID, actor.TenantID).Scan(&vesselStatus); err == sql.ErrNoRows {
			return domain.ErrNotFound
		} else if err != nil {
			return err
		}
		if vesselStatus != "active" {
			return fmt.Errorf("%w: vessel unavailable", domain.ErrConflict)
		}
		if err := tx.QueryRowContext(ctx, `SELECT status FROM bunker_windows WHERE id=? AND tenant_id=?`, item.WindowID, actor.TenantID).Scan(&windowStatus); err == sql.ErrNoRows {
			return domain.ErrNotFound
		} else if err != nil {
			return err
		}
		if windowStatus != "open" && windowStatus != "claimed" {
			return fmt.Errorf("%w: window unavailable", domain.ErrConflict)
		}
		if err := tx.QueryRowContext(ctx, `SELECT quality_state FROM fuel_lots WHERE id=? AND tenant_id=?`, item.FuelLotID, actor.TenantID).Scan(&qualityState); err == sql.ErrNoRows {
			return domain.ErrNotFound
		} else if err != nil {
			return err
		}
		if qualityState != string(domain.QualityApproved) {
			return fmt.Errorf("%w: fuel quality", domain.ErrNoQuality)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO transfer_orders(id, tenant_id, vessel_id, window_id, fuel_lot_id, target_kg, transferred_kg, state, version, idempotency_key, created_at) VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?)`, item.ID, item.TenantID, item.VesselID, item.WindowID, item.FuelLotID, item.TargetKG, item.State, item.Version, nullable(input.IdempotencyKey), storage.StringTime(time.Now())); err != nil {
			return err
		}
		for position, name := range []string{"connect", "precheck", "transfer", "disconnect"} {
			if _, err := tx.ExecContext(ctx, `INSERT INTO transfer_steps(id, order_id, position, name, status) VALUES (?, ?, ?, ?, 'pending')`, uuid.NewString(), item.ID, position+1, name); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO safety_permits(id, order_id, status, created_at) VALUES (?, ?, 'pending', ?)`, uuid.NewString(), item.ID, storage.StringTime(time.Now())); err != nil {
			return err
		}
		if err := s.Audit.Record(ctx, tx, actor, "bunkering.created", item.ID, requestID); err != nil {
			return err
		}
		if err := s.Outbox.Enqueue(ctx, tx, actor.TenantID, "bunkering.created", item.ID); err != nil {
			return err
		}
		return s.Idempotency.Save(ctx, tx, actor, input.IdempotencyKey, input, 201, item.ID)
	})
	if err != nil {
		return domain.TransferOrder{}, err
	}
	return item, nil
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func extractID(value string) string { return value }

func (s *Service) Approve(ctx context.Context, actor domain.Actor, orderID, requestID string) error {
	return s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		var state string
		if err := tx.QueryRowContext(ctx, `SELECT state FROM transfer_orders WHERE id=? AND tenant_id=?`, orderID, actor.TenantID).Scan(&state); err == sql.ErrNoRows {
			return domain.ErrNotFound
		} else if err != nil {
			return err
		}
		if state != string(domain.StatePlanned) {
			return fmt.Errorf("%w: state %s", domain.ErrConflict, state)
		}
		var permit string
		if err := tx.QueryRowContext(ctx, `SELECT status FROM safety_permits WHERE order_id=?`, orderID).Scan(&permit); err != nil {
			return err
		}
		if permit != "pending" {
			return fmt.Errorf("%w: permit", domain.ErrConflict)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE safety_permits SET status='approved', issued_by=? WHERE order_id=?`, actor.ID, orderID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE transfer_orders SET state='approved', version=version+1 WHERE id=? AND tenant_id=? AND state='planned'`, orderID, actor.TenantID); err != nil {
			return err
		}
		if err := s.Audit.Record(ctx, tx, actor, "bunkering.approved", orderID, requestID); err != nil {
			return err
		}
		return s.Outbox.Enqueue(ctx, tx, actor.TenantID, "bunkering.approved", orderID)
	})
}

func (s *Service) MarkAlongside(ctx context.Context, actor domain.Actor, orderID string, requestID string) error {
	return s.transition(ctx, actor, orderID, domain.StateApproved, domain.StateAlongside, "bunkering.alongside", requestID)
}

func (s *Service) StartTransfer(ctx context.Context, actor domain.Actor, orderID, owner string, requestID string) error {
	return s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		var current domain.OperationState
		var lot string
		var target, transferred float64
		var state string
		if err := tx.QueryRowContext(ctx, `SELECT state, fuel_lot_id, target_kg, transferred_kg FROM transfer_orders WHERE id=? AND tenant_id=?`, orderID, actor.TenantID).Scan(&state, &lot, &target, &transferred); err == sql.ErrNoRows {
			return domain.ErrNotFound
		} else if err != nil {
			return err
		}
		current = domain.OperationState(state)
		if current != domain.StateAlongside {
			return fmt.Errorf("%w: start from %s", domain.ErrConflict, current)
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("%w: %v", domain.ErrCancelled, err)
		}
		if err := s.Fuel.Reserve(ctx, tx, actor, lot, target-transferred); err != nil {
			return err
		}
		lease := time.Now().UTC().Add(5 * time.Minute)
		if _, err := tx.ExecContext(ctx, `UPDATE transfer_orders SET state='transferring', lease_owner=?, lease_until=?, version=version+1 WHERE id=? AND tenant_id=? AND state='alongside'`, owner, storage.StringTime(lease), orderID, actor.TenantID); err != nil {
			return err
		}
		if err := s.Audit.Record(ctx, tx, actor, "bunkering.started", orderID, requestID); err != nil {
			return err
		}
		return s.Outbox.Enqueue(ctx, tx, actor.TenantID, "bunkering.started", orderID)
	})
}

func (s *Service) transition(ctx context.Context, actor domain.Actor, id string, from, to domain.OperationState, action, requestID string) error {
	if err := domain.Transition(from, to); err != nil {
		return err
	}
	return s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE transfer_orders SET state=?, version=version+1 WHERE id=? AND tenant_id=? AND state=?`, to, id, actor.TenantID, from)
		if err != nil {
			return err
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			return fmt.Errorf("%w: transition", domain.ErrConflict)
		}
		if err := s.Audit.Record(ctx, tx, actor, action, id, requestID); err != nil {
			return err
		}
		return s.Outbox.Enqueue(ctx, tx, actor.TenantID, action, id)
	})
}

func (s *Service) RenewLease(ctx context.Context, actor domain.Actor, orderID, owner string, now time.Time) error {
	until := now.Add(5 * time.Minute)
	result, err := s.Store.DB.ExecContext(ctx, `UPDATE transfer_orders SET lease_until=?, version=version+1 WHERE id=? AND tenant_id=? AND state='transferring' AND lease_owner=? AND lease_until>?`, storage.StringTime(until), orderID, actor.TenantID, owner, storage.StringTime(now))
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return domain.ErrLeaseLost
	}
	return nil
}

func (s *Service) CompleteStep(ctx context.Context, actor domain.Actor, orderID string, position int, requestID string) error {
	return s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		var status string
		if err := tx.QueryRowContext(ctx, `SELECT state FROM transfer_orders WHERE id=? AND tenant_id=?`, orderID, actor.TenantID).Scan(&status); err != nil {
			return err
		}
		if status != string(domain.StateTransferring) {
			return domain.ErrConflict
		}
		var stepStatus string
		if err := tx.QueryRowContext(ctx, `SELECT status FROM transfer_steps WHERE order_id=? AND position=?`, orderID, position).Scan(&stepStatus); err != nil {
			return err
		}
		if stepStatus != "pending" {
			return domain.ErrConflict
		}
		if position > 1 {
			var previous string
			if err := tx.QueryRowContext(ctx, `SELECT status FROM transfer_steps WHERE order_id=? AND position=?`, orderID, position-1).Scan(&previous); err != nil {
				return err
			}
			if previous != "completed" {
				return fmt.Errorf("%w: prior step", domain.ErrConflict)
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE transfer_steps SET status='completed', confirmed_at=? WHERE order_id=? AND position=?`, storage.StringTime(time.Now()), orderID, position); err != nil {
			return err
		}
		if position == 4 {
			if _, err := tx.ExecContext(ctx, `UPDATE transfer_orders SET state='sampled', version=version+1 WHERE id=? AND tenant_id=? AND state='transferring'`, orderID, actor.TenantID); err != nil {
				return err
			}
		}
		if err := s.Audit.Record(ctx, tx, actor, "transfer.step.completed", orderID, requestID); err != nil {
			return err
		}
		return nil
	})
}

func (s *Service) Complete(ctx context.Context, actor domain.Actor, orderID string, requestID string) (domain.Invoice, error) {
	var invoice domain.Invoice
	err := s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		var state string
		var lot string
		var target float64
		if err := tx.QueryRowContext(ctx, `SELECT state, fuel_lot_id, target_kg FROM transfer_orders WHERE id=? AND tenant_id=?`, orderID, actor.TenantID).Scan(&state, &lot, &target); err == sql.ErrNoRows {
			return domain.ErrNotFound
		} else if err != nil {
			return err
		}
		if state != string(domain.StateSampled) {
			return fmt.Errorf("%w: order not sampled", domain.ErrConflict)
		}
		approved, err := s.Quality.ApprovedForOrder(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if !approved {
			return domain.ErrNoQuality
		}
		if _, err := tx.ExecContext(ctx, `UPDATE transfer_orders SET state='completed', lease_owner=NULL, lease_until=NULL, transferred_kg=? WHERE id=? AND tenant_id=? AND state='sampled'`, target, orderID, actor.TenantID); err != nil {
			return err
		}
		invoice = domain.Invoice{ID: uuid.NewString(), OrderID: orderID, Amount: int64(target * 120), Currency: "USD", State: "issued"}
		if _, err := tx.ExecContext(ctx, `INSERT INTO invoices(id, order_id, amount_cents, currency, state, created_at) VALUES (?, ?, ?, ?, ?, ?)`, invoice.ID, invoice.OrderID, invoice.Amount, invoice.Currency, invoice.State, storage.StringTime(time.Now())); err != nil {
			return err
		}
		if err := s.Audit.Record(ctx, tx, actor, "bunkering.completed", orderID, requestID); err != nil {
			return err
		}
		return s.Outbox.Enqueue(ctx, tx, actor.TenantID, "bunkering.completed", orderID)
	})
	if err != nil {
		return domain.Invoice{}, err
	}
	return invoice, nil
}

func (s *Service) Abort(ctx context.Context, actor domain.Actor, orderID, requestID string) error {
	return s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		var state, lot string
		var target float64
		if err := tx.QueryRowContext(ctx, `SELECT state, fuel_lot_id, target_kg FROM transfer_orders WHERE id=? AND tenant_id=?`, orderID, actor.TenantID).Scan(&state, &lot, &target); err != nil {
			return err
		}
		current := domain.OperationState(state)
		if current == domain.StateCompleted || current == domain.StateCancelled {
			return domain.ErrConflict
		}
		if err := s.Fuel.Release(ctx, tx, actor, lot, target); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE transfer_orders SET state='cancelled', lease_owner=NULL, lease_until=NULL, version=version+1 WHERE id=? AND tenant_id=?`, orderID, actor.TenantID); err != nil {
			return err
		}
		if err := s.Audit.Record(ctx, tx, actor, "bunkering.cancelled", orderID, requestID); err != nil {
			return err
		}
		return s.Outbox.Enqueue(ctx, tx, actor.TenantID, "bunkering.cancelled", orderID)
	})
}

func (s *Service) getByID(ctx context.Context, actor domain.Actor, id string) (domain.TransferOrder, error) {
	var item domain.TransferOrder
	var state string
	var owner, until sql.NullString
	err := s.Store.DB.QueryRowContext(ctx, `SELECT id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,lease_owner,lease_until,version FROM transfer_orders WHERE id=? AND tenant_id=?`, id, actor.TenantID).Scan(&item.ID, &item.TenantID, &item.VesselID, &item.WindowID, &item.FuelLotID, &item.TargetKG, &item.TransferredKG, &state, &owner, &until, &item.Version)
	if err == sql.ErrNoRows {
		return item, domain.ErrNotFound
	}
	if err != nil {
		return item, err
	}
	item.State = domain.OperationState(state)
	item.LeaseOwner = owner.String
	if until.Valid {
		item.LeaseUntil, err = storage.ParseTime(until.String)
	}
	return item, err
}

func (s *Service) Get(ctx context.Context, actor domain.Actor, id string) (domain.TransferOrder, error) {
	return s.getByID(ctx, actor, id)
}

func (s *Service) List(ctx context.Context, actor domain.Actor, state string) ([]domain.TransferOrder, error) {
	query := `SELECT id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,lease_owner,lease_until,version FROM transfer_orders WHERE tenant_id=?`
	args := []any{actor.TenantID}
	if state != "" {
		query += ` AND state=?`
		args = append(args, state)
	}
	query += ` ORDER BY created_at,id`
	rows, err := s.Store.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.TransferOrder
	for rows.Next() {
		var item domain.TransferOrder
		var st string
		var owner, until sql.NullString
		if err := rows.Scan(&item.ID, &item.TenantID, &item.VesselID, &item.WindowID, &item.FuelLotID, &item.TargetKG, &item.TransferredKG, &st, &owner, &until, &item.Version); err != nil {
			return nil, err
		}
		item.State = domain.OperationState(st)
		item.LeaseOwner = owner.String
		if until.Valid {
			item.LeaseUntil, err = storage.ParseTime(until.String)
			if err != nil {
				return nil, err
			}
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
