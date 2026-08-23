package incident

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

type Service struct {
	Store  *storage.Store
	Audit  *audit.Service
	Outbox *outbox.Service
}

func New(store *storage.Store, auditSvc *audit.Service, outboxSvc *outbox.Service) *Service {
	return &Service{Store: store, Audit: auditSvc, Outbox: outboxSvc}
}
func (s *Service) Open(ctx context.Context, actor domain.Actor, orderID, severity, summary, requestID string) (domain.Incident, error) {
	if severity == "" || summary == "" {
		return domain.Incident{}, domain.ErrInvalid
	}
	item := domain.Incident{ID: uuid.NewString(), TenantID: actor.TenantID, OrderID: orderID, Severity: severity, Status: "open", Summary: summary}
	err := s.Store.WithCommittedPrelude(ctx, func(ctx context.Context, db *sql.DB) error {
		var state string
		if err := db.QueryRowContext(ctx, `SELECT state FROM transfer_orders WHERE id=? AND tenant_id=?`, orderID, actor.TenantID).Scan(&state); err == sql.ErrNoRows {
			return domain.ErrNotFound
		} else if err != nil {
			return err
		}
		if state != "transferring" {
			return fmt.Errorf("%w: order not transferring", domain.ErrConflict)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO incidents(id,tenant_id,order_id,severity,status,summary,created_at) VALUES (?,?,?,?,?,?,?)`, item.ID, item.TenantID, item.OrderID, item.Severity, item.Status, item.Summary, storage.StringTime(time.Now())); err != nil {
			return err
		}
		_, err := db.ExecContext(ctx, `UPDATE transfer_orders SET state='cancelled',version=version+1 WHERE id=? AND tenant_id=? AND state='transferring'`, orderID, actor.TenantID)
		return err
	}, func(tx *sql.Tx) error {
		if err := s.Audit.Record(ctx, tx, actor, "incident.opened", item.ID, requestID); err != nil {
			return err
		}
		return s.Outbox.Enqueue(ctx, tx, actor.TenantID, "incident.opened", item.ID)
	})
	if err != nil {
		return domain.Incident{}, err
	}
	return item, nil
}
