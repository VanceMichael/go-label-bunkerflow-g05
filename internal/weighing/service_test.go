package weighing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

func TestSeriesCalculatesDeliveredQuantityWithoutMutatingInput(t *testing.T) {
	now := time.Now()
	series := Series{Readings: []Reading{
		{ID: "2", OrderID: "o", Sequence: 2, GrossKG: 170, TareKG: 20, MeasuredAt: now.Add(time.Minute), DeviceID: "d"},
		{ID: "1", OrderID: "o", Sequence: 1, GrossKG: 100, TareKG: 20, MeasuredAt: now, DeviceID: "d"},
	}}
	delivered, err := series.DeliveredKG()
	if err != nil {
		t.Fatal(err)
	}
	if delivered != 70 {
		t.Fatalf("delivered=%v", delivered)
	}
	if series.Readings[0].ID != "2" {
		t.Fatal("series order was mutated")
	}
	ok, err := series.WithinTolerance(70, 1)
	if err != nil || !ok {
		t.Fatalf("tolerance=%v err=%v", ok, err)
	}
}

func TestSeriesRejectsSequenceGapsAndNegativeDelivery(t *testing.T) {
	now := time.Now()
	gap := Series{Readings: []Reading{{ID: "1", OrderID: "o", Sequence: 2, GrossKG: 100, TareKG: 10, MeasuredAt: now, DeviceID: "d"}}}
	if gap.Validate() == nil {
		t.Fatal("sequence gap accepted")
	}
	negative := Series{Readings: []Reading{
		{ID: "1", OrderID: "o", Sequence: 1, GrossKG: 200, TareKG: 10, MeasuredAt: now, DeviceID: "d"},
		{ID: "2", OrderID: "o", Sequence: 2, GrossKG: 100, TareKG: 10, MeasuredAt: now.Add(time.Minute), DeviceID: "d"},
	}}
	if _, err := negative.DeliveredKG(); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("error=%v", err)
	}
}

type fakeClient struct {
	reading Reading
	err     error
}

func (f fakeClient) Read(context.Context, string) (Reading, error) { return f.reading, f.err }
func TestCapturePropagatesCancellationAndDeviceErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := Service{Client: fakeClient{}}
	if _, err := svc.Capture(ctx, "o"); !errors.Is(err, domain.ErrCancelled) {
		t.Fatalf("cancel error=%v", err)
	}
	svc.Client = fakeClient{err: domain.ErrUnavailable}
	if _, err := svc.Capture(context.Background(), "o"); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("device error=%v", err)
	}
}
