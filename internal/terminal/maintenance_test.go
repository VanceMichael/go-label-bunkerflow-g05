package terminal

import (
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
	"time"
)

func TestMaintenanceBlocksOverlappingWindow(t *testing.T) {
	now := time.Now()
	maintenance := Maintenance{ID: "m", TerminalID: "t", StartsAt: now, EndsAt: now.Add(time.Hour), Reason: "hose inspection", Status: "active"}
	if err := maintenance.Valid(); err != nil {
		t.Fatal(err)
	}
	window := domain.BunkerWindow{TerminalID: "t", StartsAt: now.Add(30 * time.Minute), EndsAt: now.Add(90 * time.Minute), Status: "open"}
	if !maintenance.Conflicts(window) {
		t.Fatal("maintenance conflict not detected")
	}
	if len(Due([]Maintenance{maintenance}, now)) != 0 {
		t.Fatal("active maintenance marked due")
	}
}
