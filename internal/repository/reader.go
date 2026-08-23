package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

type Reader struct{ Store *storage.Store }

func New(store *storage.Store) *Reader { return &Reader{Store: store} }
func (r *Reader) OrdersByState(ctx context.Context, actor domain.Actor, state string, limit int) ([]domain.TransferOrder, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.Store.DB.QueryContext(ctx, `SELECT id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,lease_owner,lease_until,version FROM transfer_orders WHERE tenant_id=? AND state=? ORDER BY created_at,id LIMIT ?`, actor.TenantID, state, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrders(rows)
}
func scanOrders(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]domain.TransferOrder, error) {
	var result []domain.TransferOrder
	for rows.Next() {
		var item domain.TransferOrder
		var state string
		var lease sql.NullString
		if err := rows.Scan(&item.ID, &item.TenantID, &item.VesselID, &item.WindowID, &item.FuelLotID, &item.TargetKG, &item.TransferredKG, &state, &item.LeaseOwner, &lease, &item.Version); err != nil {
			return nil, err
		}
		item.State = domain.OperationState(state)
		if lease.Valid {
			item.LeaseUntil, _ = storage.ParseTime(lease.String)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
func (r *Reader) CountForTenant(ctx context.Context, actor domain.Actor, table string) (int, error) {
	allowed := map[string]bool{"vessels": true, "fuel_lots": true, "bunker_windows": true, "transfer_orders": true, "audit_events": true}
	if !allowed[table] {
		return 0, domain.ErrInvalid
	}
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE tenant_id=?", table)
	err := r.Store.DB.QueryRowContext(ctx, query, actor.TenantID).Scan(&count)
	return count, err
}
func (r *Reader) SearchVessels(ctx context.Context, actor domain.Actor, term string) ([]domain.Vessel, error) {
	pattern := "%" + strings.ToUpper(strings.TrimSpace(term)) + "%"
	rows, err := r.Store.DB.QueryContext(ctx, `SELECT id,tenant_id,imo,name,flag,deadweight_kg,status FROM vessels WHERE tenant_id=? AND (UPPER(imo) LIKE ? OR UPPER(name) LIKE ?) ORDER BY name,id`, actor.TenantID, pattern, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.Vessel
	for rows.Next() {
		var item domain.Vessel
		if err := rows.Scan(&item.ID, &item.TenantID, &item.IMO, &item.Name, &item.Flag, &item.DeadweightT, &item.Status); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
func (r *Reader) RecentOrders(ctx context.Context, actor domain.Actor, after time.Time) ([]domain.TransferOrder, error) {
	rows, err := r.Store.DB.QueryContext(ctx, `SELECT id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,lease_owner,lease_until,version FROM transfer_orders WHERE tenant_id=? AND created_at>=? ORDER BY created_at,id`, actor.TenantID, storage.StringTime(after))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrders(rows)
}
