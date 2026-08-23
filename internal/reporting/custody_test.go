package reporting

import (
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

func TestCustodyOverviewAndValidation(t *testing.T) {
	now := time.Now()
	rows := []CustodyRow{
		{SampleID: "s1", ChainRef: "C-1", State: domain.QualityApproved, Receiver: "lab", LastAction: "reviewed", LastActor: "quality", LastAt: now},
		{SampleID: "s2", ChainRef: "C-2", State: domain.QualityRejected, Receiver: "lab", LastAction: "reviewed", LastActor: "quality", LastAt: now.Add(time.Minute)},
		{SampleID: "s3", ChainRef: "C-3", State: domain.QualityReceived, Receiver: "", LastAt: now.Add(2 * time.Minute)},
	}
	for _, row := range rows {
		if err := ValidateCustody(row); err != nil {
			t.Fatalf("row=%+v err=%v", row, err)
		}
	}
	overview := BuildCustody(rows)
	if overview.Approved != 1 || overview.Rejected != 1 || overview.Received != 1 || overview.MissingReceiver != 1 {
		t.Fatalf("overview=%+v", overview)
	}
	if len(RejectedCustody(rows)) != 1 {
		t.Fatal("rejected rows wrong")
	}
	if CustodyLabel(rows[0]) != "C-1 approved by quality" {
		t.Fatalf("label=%s", CustodyLabel(rows[0]))
	}
}
