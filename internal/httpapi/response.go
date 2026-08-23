package httpapi

import (
	"encoding/json"
	"net/http"
)

type Envelope struct {
	Data      any            `json:"data,omitempty"`
	Error     *ResponseError `json:"error,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
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
