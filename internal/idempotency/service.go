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

// Querier is the subset of *sql.Tx / *sql.DB that Lookup and Save need so they
// can run inside the caller's transaction and stay on the same atomic step as
// the work they guard.
type Querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (s *Service) Lookup(ctx context.Context, q Querier, actor domain.Actor, key string, request any) (*Replay, error) {
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

// LookupOrder is the typed convenience for create-style handlers: it confirms
// the stored request still matches and, if so, returns the original response
// object. A zero value with a nil error means the key has never been seen.
func LookupOrder[T any](s *Service, ctx context.Context, q Querier, actor domain.Actor, key string, request any) (T, bool, error) {
	var zero T
	replay, err := s.Lookup(ctx, q, actor, key, request)
	if err != nil || replay == nil {
		return zero, false, err
	}
	var item T
	if err := json.Unmarshal([]byte(replay.Body), &item); err != nil {
		return zero, false, err
	}
	return item, true, nil
}

func (s *Service) Save(ctx context.Context, q Execer, actor domain.Actor, key string, request any, code int, body string) error {
	if key == "" {
		return nil
	}
	// A concurrent writer may have persisted the same key between our Lookup
	// and Save. Treat that as a win for the earlier request: keep its response
	// rather than failing the whole transaction, so a replay stays a replay.
	_, err := q.ExecContext(ctx, `INSERT INTO idempotency_keys(id, tenant_id, key_value, request_hash, response_code, response_body, created_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(tenant_id, key_value) DO NOTHING`, uuid.NewString(), actor.TenantID, key, hashRequest(request), code, body, storage.StringTime(time.Now()))
	return err
}
