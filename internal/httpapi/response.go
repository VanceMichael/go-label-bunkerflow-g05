package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Envelope struct {
	Data      any            `json:"data,omitempty"`
	Error     *ResponseError `json:"error,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
}

func domainStatus(err error) int {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrIdempotency):
		return http.StatusConflict
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, domain.ErrInvalid):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrCancelled):
		return http.StatusRequestTimeout
	default:
		return http.StatusInternalServerError
	}
}

type ResponseError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func WriteEnvelope(w http.ResponseWriter, status int, data any, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{Data: data, RequestID: requestID})
}
func WriteResponseError(w http.ResponseWriter, status int, code, message string, retryable bool, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{Error: &ResponseError{Code: code, Message: message, Retryable: retryable}, RequestID: requestID})
}
func StatusText(status int) string {
	if status >= 200 && status < 300 {
		return "success"
	}
	if status == http.StatusConflict {
		return "conflict"
	}
	if status == http.StatusUnauthorized {
		return "unauthorized"
	}
	if status >= 500 {
		return "server_error"
	}
	return "request_error"
}
