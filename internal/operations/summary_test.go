package operations

import (
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
	"time"
)

func TestSummarySeparatesCompletionCancellationAndFailure(t *testing.T) {
	results := []Result{{OrderID: "a", State: domain.StateCompleted, AmountCents: 100, At: time.Unix(2, 0)}, {OrderID: "b", State: domain.StateCancelled, At: time.Unix(1, 0)}, {OrderID: "c", State: domain.StateSampled, Error: "quality", At: time.Unix(3, 0)}}
	summary := Summarize(results)
	if summary.Total != 3 || summary.Completed != 1 || summary.Cancelled != 1 || summary.Failed != 1 || summary.AmountCents != 100 {
		t.Fatalf("summary=%+v", summary)
	}
	sorted := SortResults(results)
	if sorted[0].OrderID != "b" {
		t.Fatal("results not sorted")
	}
}
