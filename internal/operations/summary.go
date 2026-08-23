package operations

import (
	"sort"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Result struct {
	OrderID     string
	State       domain.OperationState
	AmountCents int64
	At          time.Time
	Error       string
}
type Summary struct {
	Total       int
	Completed   int
	Cancelled   int
	Failed      int
	AmountCents int64
}

func Summarize(results []Result) Summary {
	summary := Summary{Total: len(results)}
	for _, result := range results {
		summary.AmountCents += result.AmountCents
		switch result.State {
		case domain.StateCompleted:
			summary.Completed++
		case domain.StateCancelled:
			summary.Cancelled++
		default:
			if result.Error != "" {
				summary.Failed++
			}
		}
	}
	return summary
}
func SortResults(results []Result) []Result {
	copyOf := append([]Result(nil), results...)
	sort.SliceStable(copyOf, func(i, j int) bool {
		if copyOf[i].At.Equal(copyOf[j].At) {
			return copyOf[i].OrderID < copyOf[j].OrderID
		}
		return copyOf[i].At.Before(copyOf[j].At)
	})
	return copyOf
}
func Failed(results []Result) []Result {
	filtered := make([]Result, 0)
	for _, result := range results {
		if result.Error != "" {
			filtered = append(filtered, result)
		}
	}
	return filtered
}
