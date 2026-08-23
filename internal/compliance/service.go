package compliance

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Finding struct {
	Code     string
	Severity string
	Message  string
	Blocks   bool
}
type VesselInput struct {
	IMO          string
	Certificate  domain.Certificate
	Flag         string
	DeadweightKG float64
}
type OperationInput struct {
	State      domain.OperationState
	Window     domain.BunkerWindow
	Fuel       domain.FuelLot
	Vessel     VesselInput
	Permit     domain.SafetyPermit
	WindKnots  float64
	WaveMeters float64
	Crew       int
	Now        time.Time
}
type Service struct {
	Clock func() time.Time
	Rules domain.SafetyRule
}

func New() *Service {
	return &Service{Clock: time.Now, Rules: domain.SafetyRule{MaxWindKnots: 25, MaxWaveMeters: 2.5, RequiredCrew: 2}}
}
func (s *Service) CheckVessel(v VesselInput) []Finding {
	var findings []Finding
	if !domain.ValidateIMO(v.IMO) {
		findings = append(findings, Finding{"VESSEL_IMO", "error", "IMO identifier is invalid", true})
	}
	if !v.Certificate.Active(s.Clock()) {
		findings = append(findings, Finding{"CERTIFICATE_EXPIRED", "error", "vessel certificate is not active", true})
	}
	if v.DeadweightKG <= 0 {
		findings = append(findings, Finding{"VESSEL_DIMENSIONS", "error", "deadweight must be positive", true})
	}
	if strings.TrimSpace(v.Flag) == "" {
		findings = append(findings, Finding{"FLAG_MISSING", "warning", "flag is not recorded", false})
	}
	return findings
}
func (s *Service) CheckOperation(input OperationInput) []Finding {
	findings := s.CheckVessel(input.Vessel)
	if input.State != domain.StateApproved && input.State != domain.StateAlongside {
		findings = append(findings, Finding{"STATE_NOT_READY", "error", "operation is not approved", true})
	}
	if input.Window.Status != "open" && input.Window.Status != "claimed" {
		findings = append(findings, Finding{"WINDOW_UNAVAILABLE", "error", "bunkering window is not available", true})
	}
	if input.Fuel.Quality != domain.QualityApproved {
		findings = append(findings, Finding{"FUEL_NOT_APPROVED", "error", "fuel lot quality is not approved", true})
	}
	if input.Fuel.AvailableKG <= 0 {
		findings = append(findings, Finding{"FUEL_EMPTY", "error", "fuel lot has no available quantity", true})
	}
	if err := s.Rules.Validate(input.WindKnots, input.WaveMeters, input.Crew); err != nil {
		findings = append(findings, Finding{"WEATHER_OR_CREW", "error", err.Error(), true})
	}
	if input.Permit.Status != "approved" {
		findings = append(findings, Finding{"PERMIT_MISSING", "error", "safety permit is not approved", true})
	}
	return findings
}
func (s *Service) Blocked(findings []Finding) bool {
	for _, finding := range findings {
		if finding.Blocks {
			return true
		}
	}
	return false
}
func (s *Service) Summary(findings []Finding) string {
	if len(findings) == 0 {
		return "ready"
	}
	copyOf := append([]Finding(nil), findings...)
	sort.Slice(copyOf, func(i, j int) bool { return copyOf[i].Code < copyOf[j].Code })
	parts := make([]string, 0, len(copyOf))
	for _, finding := range copyOf {
		parts = append(parts, fmt.Sprintf("%s:%s", finding.Code, finding.Severity))
	}
	return strings.Join(parts, ",")
}
func (s *Service) CheckContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("%w: compliance check stopped", domain.ErrCancelled)
	default:
		return nil
	}
}
func (s *Service) Report(ctx context.Context, input OperationInput) ([]Finding, error) {
	if err := s.CheckContext(ctx); err != nil {
		return nil, err
	}
	findings := s.CheckOperation(input)
	if err := s.CheckContext(ctx); err != nil {
		return nil, err
	}
	return findings, nil
}
