package app

import (
	"context"
	"testing"
)

func TestLifecycleStopsOnceAndReportsReadiness(t *testing.T) {
	var l Lifecycle
	if l.Ready() {
		t.Fatal("new lifecycle ready")
	}
	if err := l.Start(); err != nil {
		t.Fatal(err)
	}
	if !l.Ready() {
		t.Fatal("started lifecycle not ready")
	}
	called := 0
	if err := l.Stop(context.Background(), func(context.Context) error { called++; return nil }); err != nil {
		t.Fatal(err)
	}
	if called != 1 || l.Ready() {
		t.Fatalf("called=%d ready=%v", called, l.Ready())
	}
	if err := l.Stop(context.Background(), func(context.Context) error { called++; return nil }); err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Fatal("stop executed twice")
	}
}
