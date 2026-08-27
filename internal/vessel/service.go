package vessel

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
type RegisterInput struct {
	IMO, Name, Flag, CertificateNumber string
	ExpiresAt                          time.Time
	DeadweightKG                       float64
	Verified                           bool
}

func New(store *storage.Store, auditSvc *audit.Service) *Service {
	return &Service{Store: store, Audit: auditSvc}
}

func (s *Service) Register(ctx context.Context, actor domain.Actor, input RegisterInput, requestID string) (domain.Vessel, error) {
	if !domain.ValidateIMO(input.IMO) || input.Name == "" || input.DeadweightKG <= 0 {
		return domain.Vessel{}, domain.ErrInvalid
	}
	if !input.ExpiresAt.After(time.Now()) || !input.Verified {
		return domain.Vessel{}, fmt.Errorf("%w: certificate", domain.ErrConflict)
	}
	item := domain.Vessel{ID: uuid.NewString(), TenantID: actor.TenantID, IMO: domain.NormalizeIMO(input.IMO), Name: input.Name, Flag: input.Flag, DeadweightT: input.DeadweightKG, Status: "active", Certificate: domain.Certificate{Number: input.CertificateNumber, ExpiresAt: input.ExpiresAt, Verified: input.Verified}}
	err := s.Store.WithCommittedPrelude(ctx, func(ctx context.Context, db *sql.DB) error {
		_, err := db.ExecContext(ctx, `INSERT INTO vessels(id, tenant_id, imo, name, flag, deadweight_kg, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, item.ID, item.TenantID, item.IMO, item.Name, item.Flag, item.DeadweightT, item.Status, storage.StringTime(time.Now()))
		return err
	}, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO vessel_certificates(id, vessel_id, number, expires_at, verified, created_at) VALUES (?, ?, ?, ?, ?, ?)`, uuid.NewString(), item.ID, item.Certificate.Number, storage.StringTime(item.Certificate.ExpiresAt), 1, storage.StringTime(time.Now())); err != nil {
			return err
		}
		if err := s.Audit.Record(ctx, tx, actor, "vessel.registered", item.ID, requestID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return domain.Vessel{}, err
	}
	return item, nil
}

func (s *Service) ReplaceCertificate(ctx context.Context, actor domain.Actor, vesselID, number string, expires time.Time, verified bool, requestID string) error {
	if !verified || !expires.After(time.Now()) {
		return fmt.Errorf("%w: certificate", domain.ErrConflict)
	}
	return s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE vessels SET status='active' WHERE id=? AND tenant_id=?`, vesselID, actor.TenantID)
		if err != nil {
			return err
		}
		if n, _ := result.RowsAffected(); n != 1 {
			return domain.ErrNotFound
		}
		if _, err := tx.ExecContext(ctx, `UPDATE vessel_certificates SET number=?, expires_at=?, verified=? WHERE vessel_id=?`, number, storage.StringTime(expires), 1, vesselID); err != nil {
			return err
		}
		return s.Audit.Record(ctx, tx, actor, "vessel.certificate.replaced", vesselID, requestID)
	})
}

func (s *Service) Get(ctx context.Context, actor domain.Actor, vesselID string) (domain.Vessel, error) {
	var v domain.Vessel
	var expires string
	var verified int
	err := s.Store.DB.QueryRowContext(ctx, `SELECT v.id, v.tenant_id, v.imo, v.name, v.flag, v.deadweight_kg, v.status, c.number, c.expires_at, c.verified FROM vessels v LEFT JOIN vessel_certificates c ON c.vessel_id=v.id WHERE v.id=? AND v.tenant_id=?`, vesselID, actor.TenantID).Scan(&v.ID, &v.TenantID, &v.IMO, &v.Name, &v.Flag, &v.DeadweightT, &v.Status, &v.Certificate.Number, &expires, &verified)
	if err == sql.ErrNoRows {
		return domain.Vessel{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Vessel{}, err
	}
	v.Certificate.ExpiresAt, err = storage.ParseTime(expires)
	v.Certificate.Verified = verified == 1
	return v, err
}

func (s *Service) List(ctx context.Context, actor domain.Actor) ([]domain.Vessel, error) {
	rows, err := s.Store.DB.QueryContext(ctx, `SELECT id, tenant_id, imo, name, flag, deadweight_kg, status FROM vessels WHERE tenant_id=? ORDER BY created_at, id`, actor.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.Vessel
	for rows.Next() {
		var v domain.Vessel
		if err := rows.Scan(&v.ID, &v.TenantID, &v.IMO, &v.Name, &v.Flag, &v.DeadweightT, &v.Status); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}
