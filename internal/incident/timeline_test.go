package incident

import (
	"testing"
	"time"
)

func TestTimelineOrdersAndSummarizesIncidentActions(t *testing.T) {
	now := time.Now()
	timeline := Timeline{}
	if err := timeline.Add(TimelineEvent{ID: "2", IncidentID: "i", Action: "resolved", At: now.Add(time.Hour), Detail: "safe"}); err != nil {
		t.Fatal(err)
	}
	if err := timeline.Add(TimelineEvent{ID: "1", IncidentID: "i", Action: "opened", At: now, Detail: "wind"}); err != nil {
		t.Fatal(err)
	}
	if !timeline.HasAction("opened") || timeline.Ordered()[0].ID != "1" || timeline.Summary() != "opened:wind -> resolved:safe" {
		t.Fatalf("timeline=%s", timeline.Summary())
	}
}
