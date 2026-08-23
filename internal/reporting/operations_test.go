package reporting

import (
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

func TestOverviewBuildsOperationalCountsAndNextWindow(t *testing.T) {
	now := time.Now()
	rows := []OperationRow{
		{OrderID: "planned", State: domain.StatePlanned, TargetKG: 100, WindowStart: now.Add(2 * time.Hour), WindowEnd: now.Add(3 * time.Hour)},
		{OrderID: "active", State: domain.StateTransferring, TargetKG: 200, TransferredKG: 50, WindowStart: now.Add(-time.Hour), WindowEnd: now.Add(time.Hour)},
		{OrderID: "sampled", State: domain.StateSampled, TargetKG: 300, TransferredKG: 300, WindowStart: now.Add(4 * time.Hour), WindowEnd: now.Add(5 * time.Hour)},
		{OrderID: "done", State: domain.StateCompleted, TargetKG: 50, TransferredKG: 50, WindowStart: now.Add(-3 * time.Hour), WindowEnd: now.Add(-2 * time.Hour)},
	}
	overview := Build(rows, now)
	if overview.Planned != 1 || overview.Active != 1 || overview.AwaitingQuality != 1 || overview.Completed != 1 {
		t.Fatalf("overview=%+v", overview)
	}
	if overview.TargetKG != 650 || overview.TransferredKG != 400 {
		t.Fatalf("quantities=%+v", overview)
	}
	if overview.NextWindow == nil || !overview.NextWindow.Equal(now.Add(2*time.Hour)) {
		t.Fatalf("next=%v", overview.NextWindow)
	}
}

func TestReportingFiltersActiveAndOverdueRows(t *testing.T) {
	now := time.Now()
	rows := []OperationRow{{OrderID: "late", State: domain.StateApproved, WindowEnd: now.Add(-time.Hour)}, {OrderID: "active", State: domain.StateAlongside, WindowStart: now.Add(-time.Minute), WindowEnd: now.Add(time.Hour)}, {OrderID: "done", State: domain.StateCompleted, WindowEnd: now.Add(-time.Hour)}}
	if len(ActiveRows(rows)) != 1 {
		t.Fatal("active rows wrong")
	}
	overdue := Overdue(rows, now)
	if len(overdue) != 1 || overdue[0].OrderID != "late" {
		t.Fatalf("overdue=%+v", overdue)
	}
}

func TestUtilizationClampsOperationalValues(t *testing.T) {
	if Utilization(OperationRow{TargetKG: 100, TransferredKG: 50}) != .5 {
		t.Fatal("ratio wrong")
	}
	if Utilization(OperationRow{TargetKG: 100, TransferredKG: 150}) != 1 {
		t.Fatal("overrun not clamped")
	}
	if Utilization(OperationRow{}) != 0 {
		t.Fatal("zero target wrong")
	}
}
