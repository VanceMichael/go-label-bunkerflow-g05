package storage

import (
	"context"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
	"time"
)

func TestLockingRejectsUnsupportedTable(t *testing.T) {
	if err := AcquireRow(context.Background(), nil, "users", "id", "owner", time.Now(), time.Minute); err != domain.ErrInvalid {
		t.Fatalf("err=%v", err)
	}
	if err := ReleaseRow(context.Background(), nil, "users", "id", "owner"); err != domain.ErrInvalid {
		t.Fatalf("release err=%v", err)
	}
}
