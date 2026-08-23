package idempotency

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
	"github.com/google/uuid"
)

type Service struct{ Store *storage.Store }
type Replay struct {
	Code int
	Body string
}

func New(store *storage.Store) *Service { return &Service{Store: store} }

func hashRequest(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (s *Service) Lookup(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, actor domain.Actor, key string, request any) (*Replay, error) {
	if key == "" {
		return nil, nil
	}
	var result Replay
	var storedHash string
	err := q.QueryRowContext(ctx, `SELECT response_code, response_body, request_hash FROM idempotency_keys WHERE tenant_id=? AND key_value=?`, actor.TenantID, key).Scan(&result.Code, &result.Body, &storedHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if storedHash != hashRequest(request) {
		return nil, fmt.Errorf("%w: request body changed", domain.ErrIdempotency)
	}
	return &result, nil
}

func (s *Service) Save(ctx context.Context, q interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, actor domain.Actor, key string, request any, code int, body string) error {
	if key == "" {
		return nil
	}
	_, err := q.ExecContext(ctx, `INSERT INTO idempotency_keys(id, tenant_id, key_value, request_hash, response_code, response_body, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, uuid.NewString(), actor.TenantID, key, hashRequest(request), code, body, storage.StringTime(time.Now()))
	return err
}
