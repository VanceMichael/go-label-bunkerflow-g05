package fuel

import (
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
	"time"
)

func TestAnalysisDecisionAndFreshness(t *testing.T) {
	a := Analysis{MethanolPercent: 99.2, WaterPPM: 20, Density: .79, SulfurPPM: 5, TestedAt: time.Now().Add(-time.Hour), Lab: "lab-zj"}
	if ValidateAnalysis(a) != nil {
		t.Fatal("valid analysis rejected")
	}
	if QualityDecision(a) != domain.QualityApproved {
		t.Fatal("analysis rejected")
	}
	if !IsFresh(a, time.Now()) {
		t.Fatal("fresh analysis marked stale")
	}
	if Explain(a) == "rejected" {
		t.Fatal("explanation rejected")
	}
}
func TestAnalysisRejectsUnsafeValues(t *testing.T) {
	a := Analysis{MethanolPercent: 90, WaterPPM: 20, Density: .79, SulfurPPM: 5, TestedAt: time.Now(), Lab: "lab"}
	if ValidateAnalysis(a) == nil {
		t.Fatal("unsafe analysis accepted")
	}
	if QualityDecision(a) != domain.QualityRejected {
		t.Fatal("decision not rejected")
	}
}
