package integration

import (
	"context"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

func TestAuditCursorPaginationPreservesBoundaryEvents(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0025?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	stamp := time.Date(2030, 1, 1, 12, 0, 0, 0, time.UTC)
	for index, id := range []string{"a", "b", "c"} {
		_, err := rt.Store.DB.Exec(`INSERT INTO audit_events(id,tenant_id,actor_id,action,object_id,request_id,created_at) VALUES (?,?,?,?,?,?,?)`,
			id, "tenant-zj", "user-planner", "bunkering.completed", id, "req-25", storage.StringTime(stamp.Add(time.Duration(index)*time.Second)))
		if err != nil {
			t.Fatal(err)
		}
	}
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}
	first, err := rt.Audit.Page(ctx, actor, nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 2 || first.Items[0].ID != "a" || first.Items[1].ID != "b" || !first.HasMore || first.Next == nil {
		t.Fatalf("first page=%+v", first)
	}
	pageCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	second, err := rt.Audit.Page(pageCtx, actor, first.Next, 2)
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].ID != "c" || second.HasMore || second.Next != nil {
		t.Fatalf("second page=%+v", second)
	}
}
