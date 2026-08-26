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

// MoveOrder transitions a transfer order from one state to another. The
// source state is guarded in the WHERE clause so the update is a no-op (and
// surfaces a conflict) when the row is not currently in the expected source
// state. This prevents transitions such as planned -> alongside from being
// applied to orders that have not yet satisfied their prerequisites.
func MoveOrder(ctx context.Context, tx *sql.Tx, id, tenantID string, from, to domain.OperationState) (sql.Result, error) {
	return tx.ExecContext(ctx, `UPDATE transfer_orders SET state=?, version=version+1 WHERE id=? AND tenant_id=? AND state=?`, to, id, tenantID, from)
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
