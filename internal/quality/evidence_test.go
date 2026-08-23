package quality

import (
	"testing"
	"time"
)

func TestEvidenceDigestCopyAndFreshness(t *testing.T) {
	content := "lab report"
	e := Evidence{ID: "e", SampleID: "s", Digest: DigestEvidence(content), ReceivedAt: time.Now().Add(-time.Hour), Attachments: []string{"report.pdf"}}
	if err := e.Validate(time.Now()); err != nil {
		t.Fatal(err)
	}
	if !e.Matches(content) || !e.Fresh(time.Now()) {
		t.Fatal("evidence invalid")
	}
	copyOf := e.Copy()
	copyOf.Attachments[0] = "tampered"
	if e.Attachments[0] != "report.pdf" {
		t.Fatal("copy shared attachments")
	}
}
