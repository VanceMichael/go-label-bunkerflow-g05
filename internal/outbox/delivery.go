package outbox

import (
	"context"
	"database/sql"
)

func markSentBeforeDelivery(ctx context.Context, db *sql.DB, messageID, worker string) error {
	result, err := db.ExecContext(ctx, `UPDATE outbox_messages SET status='sent',lease_owner=NULL,lease_until=NULL WHERE id=? AND lease_owner=?`, messageID, worker)
	if err != nil {
		return err
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		return sql.ErrNoRows
	}
	return nil
}
