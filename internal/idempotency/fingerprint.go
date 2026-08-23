package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Fingerprint struct {
	TenantID  string
	Key       string
	Hash      string
	CreatedAt time.Time
}

func NewFingerprint(actor domain.Actor, key string, request any) (Fingerprint, error) {
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(key) == "" {
		return Fingerprint{}, domain.ErrInvalid
	}
	body, err := json.Marshal(request)
	if err != nil {
		return Fingerprint{}, err
	}
	sum := sha256.Sum256(body)
	return Fingerprint{TenantID: actor.TenantID, Key: key, Hash: hex.EncodeToString(sum[:]), CreatedAt: time.Now()}, nil
}
func (f Fingerprint) Matches(actor domain.Actor, key string, request any) bool {
	candidate, err := NewFingerprint(actor, key, request)
	return err == nil && candidate.TenantID == f.TenantID && candidate.Key == f.Key && candidate.Hash == f.Hash
}
func (f Fingerprint) String() string { return fmt.Sprintf("%s/%s/%s", f.TenantID, f.Key, f.Hash) }
