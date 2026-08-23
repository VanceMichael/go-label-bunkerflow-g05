package httpapi

import "testing"

func TestStatusTextDistinguishesResponseClasses(t *testing.T) {
	cases := map[int]string{200: "success", 409: "conflict", 401: "unauthorized", 422: "request_error", 500: "server_error"}
	for status, want := range cases {
		if got := StatusText(status); got != want {
			t.Fatalf("status %d=%q want %q", status, got, want)
		}
	}
}
