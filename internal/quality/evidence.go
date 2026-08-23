package quality

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Evidence struct {
	ID          string
	SampleID    string
	Lab         string
	Digest      string
	ReceivedAt  time.Time
	Attachments []string
}

func DigestEvidence(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
func (e Evidence) Validate(now time.Time) error {
	if strings.TrimSpace(e.ID) == "" || strings.TrimSpace(e.SampleID) == "" {
		return domain.ErrInvalid
	}
	if len(e.Digest) != 64 {
		return fmt.Errorf("%w: evidence digest", domain.ErrInvalid)
	}
	if e.ReceivedAt.After(now) {
		return fmt.Errorf("%w: evidence timestamp", domain.ErrInvalid)
	}
	if len(e.Attachments) == 0 {
		return fmt.Errorf("%w: evidence attachment", domain.ErrInvalid)
	}
	return nil
}
func (e Evidence) Matches(content string) bool { return e.Digest == DigestEvidence(content) }
func (e Evidence) Copy() Evidence {
	copyOf := e
	copyOf.Attachments = append([]string(nil), e.Attachments...)
	return copyOf
}
func (e Evidence) Fresh(now time.Time) bool {
	return !e.ReceivedAt.After(now) && now.Sub(e.ReceivedAt) < 14*24*time.Hour
}
