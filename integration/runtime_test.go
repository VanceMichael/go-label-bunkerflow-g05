package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/httpapi"
)

func TestHTTPAuthenticationAndHealthLifecycle(t *testing.T) {
	rt, err := app.New(context.Background(), app.Config{DatabaseURL: "file:integration-health?mode=memory&cache=shared", HTTPAddr: ":0"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	rt.Start(context.Background())
	handler := httpapi.New(rt, nil)
	for _, path := range []string{"/healthz", "/readyz"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, res.Code, res.Body.String())
		}
	}
	loginBody := `{"email":"planner@example.test","password":"planner-pass"}`
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	login.Header.Set("Content-Type", "application/json")
	loginRes := httptest.NewRecorder()
	handler.ServeHTTP(loginRes, login)
	if loginRes.Code != http.StatusOK {
		t.Fatalf("login status=%d", loginRes.Code)
	}
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(loginRes.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Token == "" {
		t.Fatal("empty token")
	}
	unauth := httptest.NewRequest(http.MethodGet, "/api/v1/vessels", nil)
	unauthRes := httptest.NewRecorder()
	handler.ServeHTTP(unauthRes, unauth)
	if unauthRes.Code != http.StatusUnauthorized {
		t.Fatalf("unauth status=%d", unauthRes.Code)
	}
	logout := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logout.Header.Set("Authorization", "Bearer "+payload.Token)
	logoutRes := httptest.NewRecorder()
	handler.ServeHTTP(logoutRes, logout)
	if logoutRes.Code != http.StatusOK {
		t.Fatalf("logout status=%d body=%s", logoutRes.Code, logoutRes.Body.String())
	}
}

func TestRuntimeShutdownTurnsReadinessOff(t *testing.T) {
	rt, err := app.New(context.Background(), app.Config{DatabaseURL: "file:integration-shutdown?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rt.Start(context.Background())
	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		done <- rt.Shutdown(ctx)
	}()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if rt.Ready() {
		t.Fatal("runtime remained ready")
	}
}
