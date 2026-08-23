package billing

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/audit"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/outbox"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
	"github.com/google/uuid"
)

const (
	attemptStarted = "started"
	attemptCharged = "charged"
	attemptFailed  = "failed"
)

type Gateway interface {
	Charge(ctx context.Context, key string, amount int64) (string, error)
}
type Service struct {
	Store   *storage.Store
	Audit   *audit.Service
	Outbox  *outbox.Service
	Gateway Gateway
}

type LocalGateway struct {
	Fail    bool
	Charges map[string]int
}

func (g *LocalGateway) Charge(ctx context.Context, key string, amount int64) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if g.Charges == nil {
		g.Charges = map[string]int{}
	}
	g.Charges[key]++
	if g.Fail {
		return "", domain.ErrUnavailable
	}
	_ = amount
	return "gateway-ref-" + key, nil
}

func New(store *storage.Store, auditSvc *audit.Service, outboxSvc *outbox.Service) *Service {
	return &Service{Store: store, Audit: auditSvc, Outbox: outboxSvc, Gateway: &LocalGateway{}}
}

func (s *Service) Generate(ctx context.Context, actor domain.Actor, orderID, requestID string) (domain.Invoice, error) {
	var invoice domain.Invoice
	err := s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		var state string
		var target float64
		if err := tx.QueryRowContext(ctx, `SELECT state,target_kg FROM transfer_orders WHERE id=? AND tenant_id=?`, orderID, actor.TenantID).Scan(&state, &target); err == sql.ErrNoRows {
			return domain.ErrNotFound
		} else if err != nil {
			return err
		}
		if state != string(domain.StateCompleted) {
			return fmt.Errorf("%w: order not completed", domain.ErrConflict)
		}
		invoice = domain.Invoice{ID: uuid.NewString(), OrderID: orderID, Amount: int64(target * 120), Currency: "USD", State: "issued"}
		if _, err := tx.ExecContext(ctx, `INSERT INTO invoices(id,order_id,amount_cents,currency,state,created_at) VALUES (?,?,?,?,?,?)`, invoice.ID, invoice.OrderID, invoice.Amount, invoice.Currency, invoice.State, storage.StringTime(time.Now())); err != nil {
			return err
		}
		if err := s.Audit.Record(ctx, tx, actor, "invoice.issued", invoice.ID, requestID); err != nil {
			return err
		}
		return s.Outbox.Enqueue(ctx, tx, actor.TenantID, "invoice.issued", invoice.ID)
	})
	if err != nil {
		return domain.Invoice{}, err
	}
	return invoice, nil
}

func (s *Service) Pay(ctx context.Context, actor domain.Actor, invoiceID, paymentKey, requestID string) error {
	if paymentKey == "" {
		return domain.ErrInvalid
	}

	// Pre-check the invoice so a missing/conflicting invoice is reported
	// before any gateway interaction, matching the prior behavior.
	var amount int64
	var state string
	if err := s.Store.DB.QueryRowContext(ctx, `SELECT i.amount_cents,i.state FROM invoices i JOIN transfer_orders o ON o.id=i.order_id WHERE i.id=? AND o.tenant_id=?`, invoiceID, actor.TenantID).Scan(&amount, &state); err == sql.ErrNoRows {
		return domain.ErrNotFound
	} else if err != nil {
		return err
	}
	if state == "paid" {
		return nil
	}
	if state != "issued" {
		return domain.ErrConflict
	}

	// Recoverable idempotent flow keyed on paymentKey. The gateway debit is
	// external and cannot be rolled back, so we persist the fact that the
	// gateway accepted a charge for this key *before* mutating the invoice.
	// If local bookkeeping (audit/outbox) then fails and the caller retries
	// with the same key, the recorded "charged" attempt lets us reconcile
	// (mark invoice paid, audit, enqueue) without charging the gateway again.
	if err := s.chargeOrRecover(ctx, actor, invoiceID, paymentKey, amount, requestID); err != nil {
		return err
	}
	return s.reconcile(ctx, actor, invoiceID, paymentKey, requestID)
}

// chargeOrRecover records the gateway debit. On a retry with the same key it
// is a no-op when a charge was already recorded; a prior failed attempt is
// recovered by recharging, since the gateway is idempotent on the payment key
// and a replayed charge is not a double debit.
func (s *Service) chargeOrRecover(ctx context.Context, actor domain.Actor, invoiceID, paymentKey string, amount int64, requestID string) error {
	var chargeErr error
	err := s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		var attemptID, attemptState string
		var attemptAmount int64
		err := tx.QueryRowContext(ctx, `SELECT id,state,amount_cents FROM payment_attempts WHERE tenant_id=? AND payment_key=?`, actor.TenantID, paymentKey).Scan(&attemptID, &attemptState, &attemptAmount)
		switch {
		case err == nil:
			if attemptAmount != amount {
				return fmt.Errorf("%w: payment key reused for different amount", domain.ErrIdempotency)
			}
			if attemptState == attemptCharged {
				return nil
			}
			// Fall through to (re)charge: a "started" or "failed" attempt means
			// the gateway outcome for this key is not confirmed locally.
		case err == sql.ErrNoRows:
			attemptID = fmt.Sprintf("pay-%s", uuid.NewString())
			if _, err := tx.ExecContext(ctx, `INSERT INTO payment_attempts(id,tenant_id,invoice_id,payment_key,amount_cents,state,created_at) VALUES (?,?,?,?,?,?,?)`, attemptID, actor.TenantID, invoiceID, paymentKey, amount, attemptStarted, storage.StringTime(time.Now())); err != nil {
				return err
			}
		default:
			return err
		}

		gatewayRef, gerr := s.Gateway.Charge(ctx, paymentKey, amount)
		if gerr != nil {
			// Persist the failed outcome for diagnostics/retry history, then
			// surface the error. Committing the record (rather than rolling
			// back) lets an operator see why a payment key was retried.
			if _, uerr := tx.ExecContext(ctx, `UPDATE payment_attempts SET state=?,error=? WHERE id=?`, attemptFailed, gerr.Error(), attemptID); uerr != nil {
				return uerr
			}
			chargeErr = gerr
			return nil
		}
		if _, err := tx.ExecContext(ctx, `UPDATE payment_attempts SET state=?,gateway_ref=? WHERE id=?`, attemptCharged, gatewayRef, attemptID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	if chargeErr != nil {
		return fmt.Errorf("charge gateway: %w", chargeErr)
	}
	return nil
}

// reconcile marks the invoice paid and records the local bookkeeping. It is
// idempotent: if it fails and the caller retries, chargeOrRecover is a no-op
// (charge already recorded) and reconcile runs again from the same state.
func (s *Service) reconcile(ctx context.Context, actor domain.Actor, invoiceID, paymentKey, requestID string) error {
	return s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE invoices SET state='paid',payment_key=? WHERE id=? AND state='issued'`, paymentKey, invoiceID)
		if err != nil {
			return err
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			// Invoice is already paid (concurrent success or successful retry).
			return nil
		}
		if err := s.Audit.Record(ctx, tx, actor, "invoice.paid", invoiceID, requestID); err != nil {
			return err
		}
		return s.Outbox.Enqueue(ctx, tx, actor.TenantID, "invoice.paid", invoiceID)
	})
}

func (s *Service) List(ctx context.Context, actor domain.Actor, state string) ([]domain.Invoice, error) {
	query := `SELECT i.id,i.order_id,i.amount_cents,i.currency,i.state,COALESCE(i.payment_key,''),i.created_at FROM invoices i JOIN transfer_orders o ON o.id=i.order_id WHERE o.tenant_id=?`
	args := []any{actor.TenantID}
	if state != "" {
		query += ` AND i.state=?`
		args = append(args, state)
	}
	query += ` ORDER BY i.created_at,i.id`
	rows, err := s.Store.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.Invoice
	for rows.Next() {
		var item domain.Invoice
		var created string
		if err := rows.Scan(&item.ID, &item.OrderID, &item.Amount, &item.Currency, &item.State, &item.PaymentKey, &created); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
