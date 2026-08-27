package fuel

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/audit"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/repository"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
	"github.com/google/uuid"
)

type Service struct {
	Store *storage.Store
	Audit *audit.Service
}
type ReceiveInput struct {
	LotNumber, Product string
	QuantityKG         float64
	ReceivedAt         time.Time
	Quality            domain.QualityState
}

func New(store *storage.Store, auditSvc *audit.Service) *Service {
	return &Service{Store: store, Audit: auditSvc}
}

func (s *Service) ReceiveLot(ctx context.Context, actor domain.Actor, input ReceiveInput, requestID string) (domain.FuelLot, error) {
	if input.LotNumber == "" || input.Product != "green-methanol" || input.QuantityKG <= 0 || input.Quality != domain.QualityApproved {
		return domain.FuelLot{}, fmt.Errorf("%w: fuel lot", domain.ErrInvalid)
	}
	item := domain.FuelLot{ID: uuid.NewString(), TenantID: actor.TenantID, LotNumber: domain.NormalizeLotNumber(input.LotNumber), Product: input.Product, AvailableKG: input.QuantityKG, Quality: input.Quality, ReceivedAt: input.ReceivedAt}
	err := s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO fuel_lots(id, tenant_id, lot_number, product, available_kg, quality_state, received_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, item.ID, item.TenantID, item.LotNumber, item.Product, item.AvailableKG, item.Quality, storage.StringTime(item.ReceivedAt)); err != nil {
			return err
		}
		return s.Audit.Record(ctx, tx, actor, "fuel_lot.received", item.ID, requestID)
	})
	if err != nil {
		return domain.FuelLot{}, err
	}
	return item, nil
}

func (s *Service) Get(ctx context.Context, actor domain.Actor, id string) (domain.FuelLot, error) {
	var item domain.FuelLot
	var received string
	var quality string
	err := s.Store.DB.QueryRowContext(ctx, `SELECT id, tenant_id, lot_number, product, available_kg, quality_state, received_at FROM fuel_lots WHERE id=? AND tenant_id=?`, id, actor.TenantID).Scan(&item.ID, &item.TenantID, &item.LotNumber, &item.Product, &item.AvailableKG, &quality, &received)
	if err == sql.ErrNoRows {
		return domain.FuelLot{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.FuelLot{}, err
	}
	item.Quality = domain.QualityState(quality)
	item.ReceivedAt, err = storage.ParseTime(received)
	return item, err
}

func (s *Service) ListLots(ctx context.Context, actor domain.Actor, quality string, limit int) ([]domain.FuelLot, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	predicate, args := repository.NewScope(actor).WherePredicate()
	query := `SELECT id, tenant_id, lot_number, product, available_kg, quality_state, received_at FROM fuel_lots WHERE ` + predicate
	if quality != "" {
		query += ` AND quality_state=?`
		args = append(args, quality)
	}
	query += ` ORDER BY received_at, id LIMIT ?`
	args = append(args, limit)
	rows, err := s.Store.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.FuelLot
	for rows.Next() {
		var item domain.FuelLot
		var q, received string
		if err := rows.Scan(&item.ID, &item.TenantID, &item.LotNumber, &item.Product, &item.AvailableKG, &q, &received); err != nil {
			return nil, err
		}
		item.Quality = domain.QualityState(q)
		item.ReceivedAt, err = storage.ParseTime(received)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) Reserve(ctx context.Context, tx *sql.Tx, actor domain.Actor, lotID string, quantity float64) error {
	if quantity <= 0 {
		return domain.ErrInvalid
	}
	result, err := tx.ExecContext(ctx, `UPDATE fuel_lots SET available_kg=available_kg-? WHERE id=? AND tenant_id=? AND quality_state='approved' AND available_kg>=?`, quantity, lotID, actor.TenantID, quantity)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return fmt.Errorf("%w: fuel capacity", domain.ErrConflict)
	}
	return nil
}

func (s *Service) Release(ctx context.Context, tx *sql.Tx, actor domain.Actor, lotID string, quantity float64) error {
	_, err := tx.ExecContext(ctx, `UPDATE fuel_lots SET available_kg=available_kg+? WHERE id=? AND tenant_id=?`, quantity, lotID, actor.TenantID)
	return err
}
