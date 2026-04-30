package grade

import (
	"encoding/json"
	"errors"
	"net/http"

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

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/grades", h.ListGrades)
	mux.HandleFunc("POST /api/v1/grades", h.UpsertGrade)
	mux.HandleFunc("GET /api/v1/grades/{id}", h.GetGrade)
	mux.HandleFunc("DELETE /api/v1/grades/{id}", h.DeleteGrade)
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

// ── Handlers ───────────────────────────────────────────────────────────────────

func (h *Handler) ListGrades(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	f := GradeFilter{}
	if cid := r.URL.Query().Get("class_id"); cid != "" {
		if parsed, err := uuid.Parse(cid); err == nil {
			f.ClassID = &parsed
		}
	}
	if sid := r.URL.Query().Get("subject_id"); sid != "" {
		if parsed, err := uuid.Parse(sid); err == nil {
			f.SubjectID = &parsed
		}
	}
	if stid := r.URL.Query().Get("student_id"); stid != "" {
		if parsed, err := uuid.Parse(stid); err == nil {
			f.StudentID = &parsed
		}
	}
	if ayid := r.URL.Query().Get("academic_year_id"); ayid != "" {
		if parsed, err := uuid.Parse(ayid); err == nil {
			f.AcademicYearID = &parsed
		}
	}

	grades, err := h.service.ListGrades(r.Context(), tenantID, f)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list grades")
		return
	}
	response.JSON(w, http.StatusOK, grades)
}

func (h *Handler) UpsertGrade(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	recorderID, ok := userIDFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UpsertGradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	g, err := h.service.UpsertGrade(r.Context(), tenantID, recorderID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			response.Error(w, http.StatusBadRequest, "student_id, class_id, subject_id, and academic_year_id are required")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to save grade")
		return
	}
	response.JSON(w, http.StatusCreated, g)
}

func (h *Handler) GetGrade(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	gradeID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid grade ID")
		return
	}

	g, err := h.service.GetGrade(r.Context(), tenantID, gradeID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "grade not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to get grade")
		return
	}
	response.JSON(w, http.StatusOK, g)
}

func (h *Handler) DeleteGrade(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	gradeID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid grade ID")
		return
	}

	if err := h.service.DeleteGrade(r.Context(), tenantID, gradeID); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to delete grade")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "grade deleted"})
}
