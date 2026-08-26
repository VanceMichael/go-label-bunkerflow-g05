package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
	"github.com/google/uuid"
)

type Service struct {
	Store      *storage.Store
	SessionTTL time.Duration
}

func New(store *storage.Store) *Service { return &Service{Store: store, SessionTTL: 8 * time.Hour} }

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Service) Login(ctx context.Context, email, password string) (string, domain.Actor, error) {
	var actor domain.Actor
	var stored string
	var status string
	err := s.Store.DB.QueryRowContext(ctx, `SELECT id, tenant_id, email, role, password_hash, status FROM users WHERE email = ?`, email).Scan(&actor.ID, &actor.TenantID, &actor.Email, &actor.Role, &stored, &status)
	if err == sql.ErrNoRows || stored != password || status != "active" {
		return "", domain.Actor{}, fmt.Errorf("%w: invalid credentials", domain.ErrForbidden)
	}
	if err != nil {
		return "", domain.Actor{}, fmt.Errorf("lookup user: %w", err)
	}
	token := uuid.NewString()
	expires := time.Now().UTC().Add(s.SessionTTL)
	_, err = s.Store.DB.ExecContext(ctx, `INSERT INTO sessions(id, user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`, uuid.NewString(), actor.ID, tokenHash(token), storage.StringTime(expires), storage.StringTime(time.Now()))
	if err != nil {
		return "", domain.Actor{}, fmt.Errorf("create session: %w", err)
	}
	return token, actor, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (domain.Actor, error) {
	var actor domain.Actor
	var expires string
	var revoked sql.NullString
	err := s.Store.DB.QueryRowContext(ctx, `SELECT u.id, u.tenant_id, u.email, u.role, s.expires_at, s.revoked_at FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=?`, tokenHash(token)).Scan(&actor.ID, &actor.TenantID, &actor.Email, &actor.Role, &expires, &revoked)
	if err == sql.ErrNoRows {
		return domain.Actor{}, fmt.Errorf("%w: session", domain.ErrForbidden)
	}
	if err != nil {
		return domain.Actor{}, fmt.Errorf("authenticate: %w", err)
	}
	expiry, err := storage.ParseTime(expires)
	if err != nil {
		return domain.Actor{}, err
	}
	if revoked.Valid || !expiry.After(time.Now().UTC()) {
		return domain.Actor{}, fmt.Errorf("%w: expired session", domain.ErrForbidden)
	}
	return actor, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	_, err := s.Store.DB.ExecContext(ctx, `UPDATE sessions SET revoked_at=? WHERE token_hash=? AND revoked_at IS NULL`, storage.StringTime(time.Now()), tokenHash(token))
	if err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	return nil
}

func (s *Service) RevokeAll(ctx context.Context, actorID string) error {
	_, err := s.Store.DB.ExecContext(ctx, `UPDATE sessions SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL`, storage.StringTime(time.Now()), actorID)
	return err
}
