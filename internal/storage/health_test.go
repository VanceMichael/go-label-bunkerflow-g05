package storage

import (
	"context"
	"testing"
)

func TestHealthListsMigrationTables(t *testing.T) {
	store, err := Open(context.Background(), "file:health?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	health := Health{DB: store.DB}
	if err := health.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
	ok, err := health.HasTable(context.Background(), "transfer_orders")
	if err != nil || !ok {
		t.Fatalf("table=%v err=%v", ok, err)
	}
	tables, err := health.Tables(context.Background())
	if err != nil || len(tables) < 10 {
		t.Fatalf("tables=%d err=%v", len(tables), err)
	}
}
