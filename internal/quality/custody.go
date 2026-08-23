package quality

import (
	"context"
	"database/sql"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

func loadPersistedCustody(ctx context.Context, db *sql.DB, sampleID string) ([]domain.CustodyEvent, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, sample_id, actor_id, action, created_at FROM custody_events WHERE sample_id=? ORDER BY created_at, id`, sampleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var history []domain.CustodyEvent
	for rows.Next() {
		var event domain.CustodyEvent
		var when string
		if err := rows.Scan(&event.ID, &event.SampleID, &event.ActorID, &event.Action, &when); err != nil {
			return nil, err
		}
		event.CreatedAt, err = storage.ParseTime(when)
		if err != nil {
			return nil, err
		}
		history = append(history, event)
	}
	return history, rows.Err()
}
