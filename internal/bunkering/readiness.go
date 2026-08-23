package bunkering

import (
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/compliance"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Readiness struct{ Compliance *compliance.Service }

func NewReadiness(service *compliance.Service) *Readiness { return &Readiness{Compliance: service} }
func (r *Readiness) Check(input compliance.OperationInput) (string, []compliance.Finding, error) {
	findings, err := r.Compliance.Report(nilContext{}, input)
	if err != nil {
		return "unknown", nil, err
	}
	if r.Compliance.Blocked(findings) {
		return "blocked", findings, nil
	}
	return "ready", findings, nil
}

type nilContext struct{}

func (nilContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (nilContext) Done() <-chan struct{}       { return nil }
func (nilContext) Err() error                  { return nil }
func (nilContext) Value(any) any               { return nil }
func (r *Readiness) Require(input compliance.OperationInput) error {
	status, findings, err := r.Check(input)
	if err != nil {
		return err
	}
	if status != "ready" {
		return fmt.Errorf("%w: %s", domain.ErrConflict, r.Compliance.Summary(findings))
	}
	return nil
}
