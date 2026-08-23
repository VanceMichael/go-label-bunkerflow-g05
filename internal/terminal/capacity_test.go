package terminal

import (
	"testing"
	"time"
)

func TestCapacityReserveReleaseAndTrend(t *testing.T) {
	capacity := Capacity{Slots: 2}
	var err error
	capacity, err = capacity.Reserve(1)
	if err != nil || capacity.Available() != 1 {
		t.Fatalf("capacity=%+v err=%v", capacity, err)
	}
	if _, err = capacity.Reserve(2); err == nil {
		t.Fatal("over-capacity reservation accepted")
	}
	capacity, err = capacity.Release(1)
	if err != nil || capacity.Active != 0 {
		t.Fatal("release failed")
	}
	points := Trend([]CapacityPoint{{At: time.Unix(2, 0)}, {At: time.Unix(1, 0)}})
	if !points[0].At.Before(points[1].At) {
		t.Fatal("trend not ordered")
	}
}
