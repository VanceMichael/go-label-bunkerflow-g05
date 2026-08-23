package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersAndMethodGuard(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := SecurityHeaders(MethodAllowed(http.MethodGet)(next))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent || res.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("status=%d headers=%v", res.Code, res.Header())
	}
	post := httptest.NewRequest(http.MethodPost, "/", nil)
	postRes := httptest.NewRecorder()
	handler.ServeHTTP(postRes, post)
	if postRes.Code != http.StatusMethodNotAllowed || postRes.Header().Get("Allow") != "GET" {
		t.Fatalf("post status=%d allow=%s", postRes.Code, postRes.Header().Get("Allow"))
	}
}
