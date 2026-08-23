package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/bunkering"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/fuel"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/schedule"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/terminal"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/vessel"
)

type contextKey string

const actorKey contextKey = "actor"
const requestKey contextKey = "request-id"

type Router struct {
	Runtime *app.Runtime
	Logger  *slog.Logger
}

func New(rt *app.Runtime, logger *slog.Logger) *Router {
	if logger == nil {
		logger = slog.Default()
	}
	return &Router{Runtime: rt, Logger: logger}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	requestID := req.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	w.Header().Set("X-Request-ID", requestID)
	ctx := context.WithValue(req.Context(), requestKey, requestID)
	req = req.WithContext(ctx)
	if req.URL.Path == "/healthz" {
		r.health(w, false)
		return
	}
	if req.URL.Path == "/readyz" {
		r.health(w, true)
		return
	}
	if req.URL.Path == "/api/v1/auth/login" && req.Method == http.MethodPost {
		r.login(w, req)
		return
	}
	if req.URL.Path == "/api/v1/auth/logout" && req.Method == http.MethodPost {
		r.withActor(w, req, r.logout)
		return
	}
	r.withActor(w, req, r.routeAuthenticated)
}

func (r *Router) health(w http.ResponseWriter, ready bool) {
	if ready && !r.Runtime.Ready() {
		writeError(w, http.StatusServiceUnavailable, domain.ErrUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": map[bool]string{true: "ready", false: "alive"}[ready]})
}
func (r *Router) withActor(w http.ResponseWriter, req *http.Request, next func(http.ResponseWriter, *http.Request, domain.Actor)) {
	token := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		writeError(w, http.StatusUnauthorized, domain.ErrForbidden)
		return
	}
	actor, err := r.Runtime.Auth.Authenticate(req.Context(), token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	req = req.WithContext(context.WithValue(req.Context(), actorKey, actor))
	next(w, req, actor)
}
func actorFrom(req *http.Request) domain.Actor {
	value := req.Context().Value(actorKey)
	if actor, ok := value.(domain.Actor); ok {
		return actor
	}
	return domain.Actor{}
}
func (r *Router) routeAuthenticated(w http.ResponseWriter, req *http.Request, actor domain.Actor) {
	switch {
	case req.URL.Path == "/api/v1/vessels" && req.Method == http.MethodPost:
		r.createVessel(w, req, actor)
	case req.URL.Path == "/api/v1/vessels" && req.Method == http.MethodGet:
		r.listVessels(w, req, actor)
	case req.URL.Path == "/api/v1/terminals" && req.Method == http.MethodPost:
		r.createTerminal(w, req, actor)
	case req.URL.Path == "/api/v1/terminals" && req.Method == http.MethodGet:
		r.listTerminals(w, req, actor)
	case req.URL.Path == "/api/v1/fuel-lots" && req.Method == http.MethodPost:
		r.receiveLot(w, req, actor)
	case req.URL.Path == "/api/v1/fuel-lots" && req.Method == http.MethodGet:
		r.listLots(w, req, actor)
	case req.URL.Path == "/api/v1/windows" && req.Method == http.MethodPost:
		r.createWindow(w, req, actor)
	case req.URL.Path == "/api/v1/windows" && req.Method == http.MethodGet:
		r.listWindows(w, req, actor)
	case req.URL.Path == "/api/v1/bunkering" && req.Method == http.MethodPost:
		r.createBunkering(w, req, actor)
	case req.URL.Path == "/api/v1/bunkering" && req.Method == http.MethodGet:
		r.listBunkering(w, req, actor)
	case req.URL.Path == "/api/v1/audit/events" && req.Method == http.MethodGet:
		r.listAudit(w, req, actor)
	default:
		writeError(w, http.StatusNotFound, domain.ErrNotFound)
	}
}

type loginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *Router) login(w http.ResponseWriter, req *http.Request) {
	var input loginInput
	if err := decode(req, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	token, actor, err := r.Runtime.Auth.Login(req.Context(), input.Email, input.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "actor": actor})
}
func (r *Router) logout(w http.ResponseWriter, req *http.Request, actor domain.Actor) {
	token := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	if err := r.Runtime.Auth.Logout(req.Context(), token); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (r *Router) createVessel(w http.ResponseWriter, req *http.Request, actor domain.Actor) {
	var input vessel.RegisterInput
	if err := decode(req, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := r.Runtime.Vessel.Register(req.Context(), actor, input, requestID(req))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
func (r *Router) listVessels(w http.ResponseWriter, req *http.Request, actor domain.Actor) {
	items, err := r.Runtime.Vessel.List(req.Context(), actor)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (r *Router) createTerminal(w http.ResponseWriter, req *http.Request, actor domain.Actor) {
	var input struct{ Name, Timezone, OpenFrom, OpenUntil string }
	if err := decode(req, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := r.Runtime.Terminal.Create(req.Context(), actor, terminalInput(input))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
func terminalInput(input struct{ Name, Timezone, OpenFrom, OpenUntil string }) terminal.CreateInput {
	return terminal.CreateInput{Name: input.Name, Timezone: input.Timezone, OpenFrom: input.OpenFrom, OpenUntil: input.OpenUntil}
}
func (r *Router) listTerminals(w http.ResponseWriter, req *http.Request, actor domain.Actor) {
	items, err := r.Runtime.Terminal.List(req.Context(), actor)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (r *Router) receiveLot(w http.ResponseWriter, req *http.Request, actor domain.Actor) {
	var input struct {
		LotNumber, Product string
		QuantityKG         float64   `json:"quantity_kg"`
		ReceivedAt         time.Time `json:"received_at"`
		Quality            domain.QualityState
	}
	if err := decode(req, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := r.Runtime.Fuel.ReceiveLot(req.Context(), actor, fuel.ReceiveInput{LotNumber: input.LotNumber, Product: input.Product, QuantityKG: input.QuantityKG, ReceivedAt: input.ReceivedAt, Quality: input.Quality}, requestID(req))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
func (r *Router) listLots(w http.ResponseWriter, req *http.Request, actor domain.Actor) {
	items, err := r.Runtime.Fuel.ListLots(req.Context(), actor, req.URL.Query().Get("quality"), 50)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (r *Router) createWindow(w http.ResponseWriter, req *http.Request, actor domain.Actor) {
	var input struct {
		TerminalID string    `json:"terminal_id"`
		StartsAt   time.Time `json:"starts_at"`
		EndsAt     time.Time `json:"ends_at"`
	}
	if err := decode(req, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := r.Runtime.Schedule.CreateWindow(req.Context(), actor, schedule.WindowInput{TerminalID: input.TerminalID, StartsAt: input.StartsAt, EndsAt: input.EndsAt}, requestID(req))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
func (r *Router) listWindows(w http.ResponseWriter, req *http.Request, actor domain.Actor) {
	items, err := r.Runtime.Schedule.List(req.Context(), actor, req.URL.Query().Get("status"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (r *Router) createBunkering(w http.ResponseWriter, req *http.Request, actor domain.Actor) {
	var input struct {
		VesselID, WindowID, FuelLotID string
		TargetKG                      float64 `json:"target_kg"`
	}
	if err := decode(req, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := r.Runtime.Bunkering.Create(req.Context(), actor, bunkering.CreateInput{VesselID: input.VesselID, WindowID: input.WindowID, FuelLotID: input.FuelLotID, TargetKG: input.TargetKG, IdempotencyKey: req.Header.Get("Idempotency-Key")}, requestID(req))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
func (r *Router) listBunkering(w http.ResponseWriter, req *http.Request, actor domain.Actor) {
	items, err := r.Runtime.Bunkering.List(req.Context(), actor, req.URL.Query().Get("state"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (r *Router) listAudit(w http.ResponseWriter, req *http.Request, actor domain.Actor) {
	items, err := r.Runtime.Audit.List(req.Context(), actor, 50)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func requestID(req *http.Request) string {
	value, _ := req.Context().Value(requestKey).(string)
	return value
}
func decode(req *http.Request, value any) error {
	defer req.Body.Close()
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(value)
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": "request_error", "message": err.Error()}})
}
func writeDomainError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrIdempotency), errors.Is(err, domain.ErrNoQuality):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, domain.ErrInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrCancelled):
		status = http.StatusRequestTimeout
	}
	writeError(w, status, err)
}
