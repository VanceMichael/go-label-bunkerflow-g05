package httpapi

import (
	"net/http"
	"runtime/debug"
	"time"
)

func Chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for index := len(middlewares) - 1; index >= 0; index-- {
		handler = middlewares[index](handler)
	}
	return handler
}
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if value := recover(); value != nil {
				_ = debug.Stack()
				writeError(w, http.StatusInternalServerError, domainPanicError{})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type domainPanicError struct{}

func (domainPanicError) Error() string { return "internal server error" }
func Timeout(duration time.Duration, next http.Handler) http.Handler {
	return http.TimeoutHandler(next, duration, "request timed out")
}
func NoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
