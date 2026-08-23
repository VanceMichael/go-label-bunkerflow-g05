package bunkering

import (
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
)

func TestManifestValidatesOrderedStepsAndCopiesSlice(t *testing.T) {
	manifest := Manifest{OrderID: "o", VesselIMO: "9384756", Product: "green-methanol", TargetKG: 100, Steps: []domain.TransferStep{{Position: 1, Name: "connect"}, {Position: 2, Name: "precheck"}, {Position: 3, Name: "transfer"}, {Position: 4, Name: "disconnect"}}}
	if err := manifest.Validate(); err != nil {
		t.Fatal(err)
	}
	copyOf := manifest.Copy()
	copyOf.Steps[0].Name = "tampered"
	if manifest.Steps[0].Name != "connect" {
		t.Fatal("manifest steps shared")
	}
	if _, err := manifest.JSON(); err != nil {
		t.Fatal(err)
	}
}
