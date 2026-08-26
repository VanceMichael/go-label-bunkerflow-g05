package domain

import "errors"

var (
	ErrNotFound     = errors.New("resource not found")
	ErrConflict     = errors.New("resource conflict")
	ErrForbidden    = errors.New("operation forbidden")
	ErrInvalid      = errors.New("invalid business request")
	ErrCancelled    = errors.New("operation cancelled")
	ErrUnavailable  = errors.New("dependency unavailable")
	ErrNoQuality    = errors.New("quality evidence is not approved")
	ErrLeaseLost    = errors.New("worker lease is no longer owned")
	ErrIdempotency  = errors.New("idempotency key belongs to another request")
	ErrTerminalBusy = errors.New("terminal still has active operations")
)
