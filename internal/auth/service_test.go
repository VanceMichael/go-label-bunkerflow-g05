package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

func TestLoginLogoutAndTenantRoles(t *testing.T) {
	store, err := storage.Open(context.Background(), "file:auth-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := New(store)
	token, actor, err := svc.Login(context.Background(), "planner@example.test", "planner-pass")
	if err != nil {
		t.Fatal(err)
	}
	if actor.Role != "planner" || actor.TenantID != "tenant-zj" {
		t.Fatalf("actor = %+v", actor)
	}
	got, err := svc.Authenticate(context.Background(), token)
	if err != nil || got.ID != actor.ID {
		t.Fatalf("authenticate = %+v, %v", got, err)
	}
	if err := svc.Logout(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(context.Background(), token); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("logout error = %v", err)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	store, err := storage.Open(context.Background(), "file:auth-password?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, _, err := New(store).Login(context.Background(), "planner@example.test", "wrong"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error = %v", err)
	}
}
