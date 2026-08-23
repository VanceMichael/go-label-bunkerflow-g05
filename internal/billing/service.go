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

type Gateway interface {
	Charge(context.Context, string, int64) error
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

func (g *LocalGateway) Charge(ctx context.Context, key string, amount int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if g.Charges == nil {
		g.Charges = map[string]int{}
	}
	g.Charges[key]++
	if g.Fail {
		return domain.ErrUnavailable
	}
	_ = amount
	return nil
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
	var amount int64
	var state string
	if err := s.Store.DB.QueryRowContext(ctx, `SELECT amount_cents,state FROM invoices i JOIN transfer_orders o ON o.id=i.order_id WHERE i.id=? AND o.tenant_id=?`, invoiceID, actor.TenantID).Scan(&amount, &state); err == sql.ErrNoRows {
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
	if err := s.Gateway.Charge(ctx, paymentKey, amount); err != nil {
		return fmt.Errorf("charge gateway: %w", err)
	}
	return s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE invoices SET state='paid',payment_key=? WHERE id=? AND state='issued'`, paymentKey, invoiceID)
		if err != nil {
			return err
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			return domain.ErrConflict
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
