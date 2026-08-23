package compliance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

func TestComplianceBlocksExpiredCertificateAndUnsafeWeather(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	svc := New()
	svc.Clock = func() time.Time { return now }
	input := OperationInput{State: domain.StateApproved, Window: domain.BunkerWindow{Status: "open"}, Fuel: domain.FuelLot{Quality: domain.QualityApproved, AvailableKG: 100}, Vessel: VesselInput{IMO: "bad", Certificate: domain.Certificate{Verified: true, ExpiresAt: now.Add(-time.Hour)}, Flag: "CN", DeadweightKG: 100}, Permit: domain.SafetyPermit{Status: "approved"}, WindKnots: 30, WaveMeters: 3, Crew: 1, Now: now}
	findings, err := svc.Report(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !svc.Blocked(findings) {
		t.Fatal("unsafe operation was not blocked")
	}
	if svc.Summary(findings) == "ready" {
		t.Fatal("empty summary")
	}
}
func TestComplianceAllowsApprovedOperation(t *testing.T) {
	now := time.Now()
	svc := New()
	svc.Clock = func() time.Time { return now }
	input := OperationInput{State: domain.StateApproved, Window: domain.BunkerWindow{Status: "claimed"}, Fuel: domain.FuelLot{Quality: domain.QualityApproved, AvailableKG: 100}, Vessel: VesselInput{IMO: "9384756", Certificate: domain.Certificate{Verified: true, ExpiresAt: now.Add(time.Hour)}, Flag: "CN", DeadweightKG: 100}, Permit: domain.SafetyPermit{Status: "approved"}, WindKnots: 10, WaveMeters: 1, Crew: 3}
	findings, err := svc.Report(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if svc.Blocked(findings) {
		t.Fatalf("findings=%+v", findings)
	}
}
func TestComplianceStopsOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New().Report(ctx, OperationInput{}); !errors.Is(err, domain.ErrCancelled) {
		t.Fatalf("error=%v", err)
	}
}
