package httpapi

import (
	"net/url"
	"testing"
	"time"
)

func TestQueryValidationParsesBoundedValues(t *testing.T) {
	values := url.Values{}
	values.Set("limit", "999")
	if got := parseLimit(values.Get("limit"), 25, 100); got != 100 {
		t.Fatalf("limit=%d", got)
	}
	values.Set("from", "2026-08-23T12:00:00Z")
	from, err := parseTimeQuery(values, "from")
	if err != nil || from.IsZero() {
		t.Fatalf("from=%v err=%v", from, err)
	}
	values.Set("enabled", "true")
	enabled, err := parseBoolQuery(values, "enabled")
	if err != nil || !enabled {
		t.Fatalf("enabled=%v err=%v", enabled, err)
	}
}
func TestDateRangeRequiresEndAfterStart(t *testing.T) {
	start := time.Now()
	if ensureDateRange(start, start.Add(time.Hour)) != nil {
		t.Fatal("valid range rejected")
	}
	if ensureDateRange(start, start) == nil {
		t.Fatal("zero range accepted")
	}
}
