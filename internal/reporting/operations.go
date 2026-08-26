package reporting

import (
	"sort"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type OperationRow struct {
	OrderID       string
	VesselID      string
	TerminalID    string
	State         domain.OperationState
	TargetKG      float64
	TransferredKG float64
	WindowStart   time.Time
	WindowEnd     time.Time
}

type Overview struct {
	Planned         int
	Active          int
	AwaitingQuality int
	Completed       int
	Cancelled       int
	TargetKG        float64
	TransferredKG   float64
	NextWindow      *time.Time
}

func Build(rows []OperationRow, now time.Time) Overview {
	overview := Overview{}
	for _, row := range rows {
		overview.TargetKG += row.TargetKG
		overview.TransferredKG += row.TransferredKG
		switch row.State {
		case domain.StatePlanned, domain.StateApproved:
			overview.Planned++
		case domain.StateAlongside, domain.StateTransferring:
			overview.Active++
		case domain.StateSampled:
			overview.AwaitingQuality++
		case domain.StateCompleted:
			overview.Completed++
		case domain.StateCancelled:
			overview.Cancelled++
		}
		if row.WindowStart.After(now) && (overview.NextWindow == nil || row.WindowStart.Before(*overview.NextWindow)) {
			value := row.WindowStart
			overview.NextWindow = &value
		}
	}
	return overview
}

func ActiveRows(rows []OperationRow) []OperationRow {
	result := make([]OperationRow, 0)
	for _, row := range rows {
		if row.State == domain.StateAlongside || row.State == domain.StateTransferring {
			result = append(result, row)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].WindowStart.Equal(result[j].WindowStart) {
			return result[i].OrderID < result[j].OrderID
		}
		return result[i].WindowStart.Before(result[j].WindowStart)
	})
	return result
}

func Utilization(row OperationRow) float64 {
	if row.TargetKG <= 0 {
		return 0
	}
	value := row.TransferredKG / row.TargetKG
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func Overdue(rows []OperationRow, now time.Time) []OperationRow {
	result := make([]OperationRow, 0)
	for _, row := range rows {
		if row.WindowEnd.Before(now) && !domain.IsTerminalState(row.State) {
			result = append(result, row)
		}
	}
	return result
}
