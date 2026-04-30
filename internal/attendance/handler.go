package attendance

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/edusaas/backend/internal/auth"
	"github.com/edusaas/backend/internal/shared/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes wires all attendance routes.
// All routes except SubmitQR require authentication.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Teacher / admin routes (behind auth middleware in main.go)
	mux.HandleFunc("GET /api/v1/attendance/sessions", h.ListSessions)
	mux.HandleFunc("POST /api/v1/attendance/sessions", h.CreateSession)
	mux.HandleFunc("GET /api/v1/attendance/sessions/{id}", h.GetSessionSummary)
	mux.HandleFunc("PUT /api/v1/attendance/sessions/{id}/close", h.CloseSession)
	mux.HandleFunc("POST /api/v1/attendance/sessions/{id}/records", h.RecordManual)
	mux.HandleFunc("GET /api/v1/attendance/sessions/{id}/records", h.ListRecords)
}

// RegisterPublicRoutes wires routes that don't require authentication (QR scan).
func (h *Handler) RegisterPublicRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/attendance/scan", h.SubmitQR)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func tenantFromCtx(r *http.Request) (uuid.UUID, bool) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		return uuid.Nil, false
	}
	return claims.TenantID, true
}

func userIDFromCtx(r *http.Request) (uuid.UUID, bool) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		return uuid.Nil, false
	}
	return claims.UserID, true
}

// ── Session handlers ───────────────────────────────────────────────────────────

func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var classID *uuid.UUID
	if cid := r.URL.Query().Get("class_id"); cid != "" {
		if parsed, err := uuid.Parse(cid); err == nil {
			classID = &parsed
		}
	}

	var date *time.Time
	if d := r.URL.Query().Get("date"); d != "" {
		if parsed, err := time.Parse("2006-01-02", d); err == nil {
			date = &parsed
		}
	}

	sessions, err := h.service.ListSessions(r.Context(), tenantID, classID, date)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}
	response.JSON(w, http.StatusOK, sessions)
}

func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	teacherID, ok := userIDFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	session, err := h.service.CreateSession(r.Context(), tenantID, teacherID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			response.Error(w, http.StatusBadRequest, "valid class_id is required")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	response.JSON(w, http.StatusCreated, session)
}

func (h *Handler) GetSessionSummary(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	summary, err := h.service.GetSessionSummary(r.Context(), tenantID, sessionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "session not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to get session")
		return
	}
	response.JSON(w, http.StatusOK, summary)
}

func (h *Handler) CloseSession(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	if err := h.service.CloseSession(r.Context(), tenantID, sessionID); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to close session")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "session closed"})
}

func (h *Handler) RecordManual(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	var req ManualAttendanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rec, err := h.service.RecordManual(r.Context(), tenantID, sessionID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			response.Error(w, http.StatusBadRequest, "valid student_id and status are required")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to record attendance")
		return
	}
	response.JSON(w, http.StatusCreated, rec)
}

func (h *Handler) ListRecords(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	records, err := h.service.ListRecords(r.Context(), tenantID, sessionID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list records")
		return
	}
	response.JSON(w, http.StatusOK, records)
}

// ── QR scan (public — no auth) ─────────────────────────────────────────────────

func (h *Handler) SubmitQR(w http.ResponseWriter, r *http.Request) {
	var req SubmitAttendanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rec, err := h.service.SubmitAttendance(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "qr_token and student_id are required")
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "invalid QR code")
		case errors.Is(err, ErrSessionClosed):
			response.Error(w, http.StatusGone, "attendance session is closed")
		case errors.Is(err, ErrTokenExpired):
			response.Error(w, http.StatusGone, "QR code has expired")
		default:
			response.Error(w, http.StatusInternalServerError, "failed to submit attendance")
		}
		return
	}
	response.JSON(w, http.StatusCreated, rec)
}
