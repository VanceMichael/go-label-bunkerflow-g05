package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Lock struct {
	ID    string
	Owner string
	Until time.Time
}

// RenewOrderLease extends the active lease held by owner until the given time.
// The owner predicate is mandatory: without it a previous holder whose lease was
// taken over could renew after the fact and silently overwrite the new holder's
// lease_until, defeating the ownership control of a takeover.
func RenewOrderLease(ctx context.Context, db *sql.DB, orderID, tenantID, owner string, now, until time.Time) (sql.Result, error) {
	return db.ExecContext(ctx, `UPDATE transfer_orders SET lease_until=?, version=version+1 WHERE id=? AND tenant_id=? AND lease_owner=? AND state='transferring' AND lease_until>?`, StringTime(until), orderID, tenantID, owner, StringTime(now))
}

func AcquireRow(ctx context.Context, tx *sql.Tx, table, id, owner string, now time.Time, ttl time.Duration) error {
	if table != "transfer_orders" && table != "bunker_windows" {
		return domain.ErrInvalid
	}
	if owner == "" || ttl <= 0 {
		return domain.ErrInvalid
	}
	query := fmt.Sprintf(`UPDATE %s SET owner_id=?,version=version+1 WHERE id=? AND (owner_id IS NULL OR owner_id=?)`, table)
	result, err := tx.ExecContext(ctx, query, owner, id, owner)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return domain.ErrConflict
	}
	_ = now
	return nil
}
func ReleaseRow(ctx context.Context, tx *sql.Tx, table, id, owner string) error {
	if table != "transfer_orders" && table != "bunker_windows" {
		return domain.ErrInvalid
	}
	query := fmt.Sprintf(`UPDATE %s SET owner_id=NULL,version=version+1 WHERE id=? AND owner_id=?`, table)
	result, err := tx.ExecContext(ctx, query, id, owner)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return domain.ErrLeaseLost
	}
	return nil
}
