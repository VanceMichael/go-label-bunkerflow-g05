package outbox

import (
	"context"
	"database/sql"
	"sort"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

type HealthSnapshot struct {
	Pending       int
	Leased        int
	Dead          int
	OldestPending *time.Time
}

func (s *Service) Health(ctx context.Context) (HealthSnapshot, error) {
	snapshot := HealthSnapshot{}
	if err := s.Store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_messages WHERE status='pending'`).Scan(&snapshot.Pending); err != nil {
		return snapshot, err
	}
	if err := s.Store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_messages WHERE lease_until>?`, storage.StringTime(time.Now())).Scan(&snapshot.Leased); err != nil {
		return snapshot, err
	}
	if err := s.Store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_messages WHERE status='dead'`).Scan(&snapshot.Dead); err != nil {
		return snapshot, err
	}
	var created sql.NullString
	if err := s.Store.DB.QueryRowContext(ctx, `SELECT created_at FROM outbox_messages WHERE status='pending' ORDER BY created_at LIMIT 1`).Scan(&created); err == nil && created.Valid {
		value, parseErr := storage.ParseTime(created.String)
		if parseErr != nil {
			return snapshot, parseErr
		}
		snapshot.OldestPending = &value
	} else if err != sql.ErrNoRows {
		return snapshot, err
	}
	return snapshot, nil
}
func (s HealthSnapshot) Healthy() bool { return s.Dead == 0 && s.Pending < 1000 }
func Backlog(messages []domain.OutboxMessage) []domain.OutboxMessage {
	copyOf := append([]domain.OutboxMessage(nil), messages...)
	sort.SliceStable(copyOf, func(i, j int) bool { return copyOf[i].NextAttempt.Before(copyOf[j].NextAttempt) })
	return copyOf
}
