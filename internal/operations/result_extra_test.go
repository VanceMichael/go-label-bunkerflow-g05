package operations

import (
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
	"time"
)

func TestFailedFiltersOnlyOperationalFailures(t *testing.T) {
	results := []Result{{OrderID: "a", State: domain.StateCompleted, At: time.Now()}, {OrderID: "b", State: domain.StateSampled, Error: "timeout", At: time.Now()}}
	failed := Failed(results)
	if len(failed) != 1 || failed[0].OrderID != "b" {
		t.Fatalf("failed=%+v", failed)
	}
}
