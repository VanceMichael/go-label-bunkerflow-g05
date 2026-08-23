package outbox

import (
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
	"time"
)

func TestBacklogCopiesAndOrdersMessages(t *testing.T) {
	items := []domain.OutboxMessage{{ID: "late", NextAttempt: time.Unix(2, 0)}, {ID: "early", NextAttempt: time.Unix(1, 0)}}
	ordered := Backlog(items)
	if ordered[0].ID != "early" {
		t.Fatal("backlog order wrong")
	}
	ordered[0].ID = "changed"
	if items[0].ID == "changed" || len(items) != 2 {
		t.Fatal("backlog shared input")
	}
}
