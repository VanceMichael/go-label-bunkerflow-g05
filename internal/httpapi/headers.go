package httpapi

import "net/http"

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		next.ServeHTTP(w, r)
	})
}
func MethodAllowed(methods ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, method := range methods {
				if r.Method == method {
					next.ServeHTTP(w, r)
					return
				}
			}
			w.Header().Set("Allow", joinMethods(methods))
			writeError(w, http.StatusMethodNotAllowed, methodError{})
		})
	}
}

type methodError struct{}

func (methodError) Error() string { return "method not allowed" }
func joinMethods(methods []string) string {
	result := ""
	for index, method := range methods {
		if index > 0 {
			result += ", "
		}
		result += method
	}
	return result
}
