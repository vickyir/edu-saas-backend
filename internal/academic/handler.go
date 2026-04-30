package academic

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

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

// RegisterRoutes registers all academic routes (all require authentication).
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Academic Years
	mux.HandleFunc("GET /api/v1/academic-years", h.ListAcademicYears)
	mux.HandleFunc("POST /api/v1/academic-years", h.CreateAcademicYear)
	mux.HandleFunc("GET /api/v1/academic-years/{id}", h.GetAcademicYear)
	mux.HandleFunc("PUT /api/v1/academic-years/{id}", h.UpdateAcademicYear)
	mux.HandleFunc("DELETE /api/v1/academic-years/{id}", h.DeleteAcademicYear)
	mux.HandleFunc("PUT /api/v1/academic-years/{id}/activate", h.ActivateAcademicYear)

	// Classes
	mux.HandleFunc("GET /api/v1/classes", h.ListClasses)
	mux.HandleFunc("POST /api/v1/classes", h.CreateClass)
	mux.HandleFunc("GET /api/v1/classes/{id}", h.GetClass)
	mux.HandleFunc("PUT /api/v1/classes/{id}", h.UpdateClass)
	mux.HandleFunc("DELETE /api/v1/classes/{id}", h.DeleteClass)

	// Class Enrollments
	mux.HandleFunc("GET /api/v1/classes/{id}/students", h.ListClassStudents)
	mux.HandleFunc("POST /api/v1/classes/{id}/students", h.EnrollStudent)
	mux.HandleFunc("DELETE /api/v1/classes/{id}/students/{studentId}", h.UnenrollStudent)

	// Class Subjects
	mux.HandleFunc("GET /api/v1/classes/{id}/subjects", h.ListClassSubjects)
	mux.HandleFunc("POST /api/v1/classes/{id}/subjects", h.AssignSubject)
	mux.HandleFunc("DELETE /api/v1/classes/{id}/subjects/{subjectId}", h.RemoveSubject)

	// Subjects
	mux.HandleFunc("GET /api/v1/subjects", h.ListSubjects)
	mux.HandleFunc("POST /api/v1/subjects", h.CreateSubject)
	mux.HandleFunc("PUT /api/v1/subjects/{id}", h.UpdateSubject)
	mux.HandleFunc("DELETE /api/v1/subjects/{id}", h.DeleteSubject)

	// Students
	mux.HandleFunc("GET /api/v1/students", h.ListStudents)
	mux.HandleFunc("POST /api/v1/students", h.CreateStudent)
	mux.HandleFunc("GET /api/v1/students/{id}", h.GetStudent)

	// Teachers reference list
	mux.HandleFunc("GET /api/v1/teachers", h.ListTeachers)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func tenantFromCtx(r *http.Request) (uuid.UUID, bool) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		return uuid.Nil, false
	}
	return claims.TenantID, true
}

// ── Academic Years ─────────────────────────────────────────────────────────────

func (h *Handler) ListAcademicYears(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	years, err := h.service.ListAcademicYears(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list academic years")
		return
	}
	response.JSON(w, http.StatusOK, years)
}

func (h *Handler) CreateAcademicYear(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateAcademicYearRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	year, err := h.service.CreateAcademicYear(r.Context(), tenantID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			response.Error(w, http.StatusBadRequest, "year and semester are required")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to create academic year")
		return
	}
	response.JSON(w, http.StatusCreated, year)
}

func (h *Handler) GetAcademicYear(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	yearID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid academic year ID")
		return
	}
	year, err := h.service.GetAcademicYear(r.Context(), tenantID, yearID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "academic year not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to get academic year")
		return
	}
	response.JSON(w, http.StatusOK, year)
}

func (h *Handler) UpdateAcademicYear(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	yearID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid academic year ID")
		return
	}
	var req CreateAcademicYearRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	year, err := h.service.UpdateAcademicYear(r.Context(), tenantID, yearID, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "academic year not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to update academic year")
		return
	}
	response.JSON(w, http.StatusOK, year)
}

func (h *Handler) DeleteAcademicYear(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	yearID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid academic year ID")
		return
	}
	if err := h.service.DeleteAcademicYear(r.Context(), tenantID, yearID); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to delete academic year")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "academic year deleted"})
}

func (h *Handler) ActivateAcademicYear(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	yearID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid academic year ID")
		return
	}

	if err := h.service.ActivateAcademicYear(r.Context(), tenantID, yearID); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to activate academic year")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "academic year activated"})
}

// ── Classes ────────────────────────────────────────────────────────────────────

func (h *Handler) ListClasses(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var academicYearID *uuid.UUID
	if ayid := r.URL.Query().Get("academic_year_id"); ayid != "" {
		if parsed, err := uuid.Parse(ayid); err == nil {
			academicYearID = &parsed
		}
	}

	classes, err := h.service.ListClasses(r.Context(), tenantID, academicYearID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list classes")
		return
	}
	response.JSON(w, http.StatusOK, classes)
}

func (h *Handler) GetClass(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	classID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid class ID")
		return
	}

	class, err := h.service.GetClass(r.Context(), tenantID, classID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "class not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to get class")
		return
	}
	response.JSON(w, http.StatusOK, class)
}

func (h *Handler) CreateClass(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateClassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	class, err := h.service.CreateClass(r.Context(), tenantID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			response.Error(w, http.StatusBadRequest, "name and academic_year_id are required")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to create class")
		return
	}
	response.JSON(w, http.StatusCreated, class)
}

func (h *Handler) UpdateClass(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	classID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid class ID")
		return
	}

	var req UpdateClassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	class, err := h.service.UpdateClass(r.Context(), tenantID, classID, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "class not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to update class")
		return
	}
	response.JSON(w, http.StatusOK, class)
}

func (h *Handler) DeleteClass(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	classID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid class ID")
		return
	}

	if err := h.service.DeleteClass(r.Context(), tenantID, classID); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to delete class")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "class deleted"})
}

// ── Class Enrollments ──────────────────────────────────────────────────────────

func (h *Handler) ListClassStudents(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	classID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid class ID")
		return
	}

	students, err := h.service.ListClassStudents(r.Context(), tenantID, classID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list class students")
		return
	}
	response.JSON(w, http.StatusOK, students)
}

func (h *Handler) EnrollStudent(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	classID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid class ID")
		return
	}

	var req EnrollStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.EnrollStudent(r.Context(), tenantID, classID, req); err != nil {
		if errors.Is(err, ErrInvalidInput) {
			response.Error(w, http.StatusBadRequest, "valid student_id is required")
			return
		}
		if errors.Is(err, ErrAlreadyExists) {
			response.Error(w, http.StatusConflict, "student already enrolled")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to enroll student")
		return
	}
	response.JSON(w, http.StatusCreated, map[string]string{"message": "student enrolled"})
}

func (h *Handler) UnenrollStudent(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	classID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid class ID")
		return
	}

	studentID, err := uuid.Parse(r.PathValue("studentId"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid student ID")
		return
	}

	if err := h.service.UnenrollStudent(r.Context(), tenantID, classID, studentID); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to unenroll student")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "student unenrolled"})
}

// ── Class Subjects ─────────────────────────────────────────────────────────────

func (h *Handler) ListClassSubjects(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	classID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid class ID")
		return
	}

	subjects, err := h.service.ListClassSubjects(r.Context(), tenantID, classID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list class subjects")
		return
	}
	response.JSON(w, http.StatusOK, subjects)
}

func (h *Handler) AssignSubject(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	classID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid class ID")
		return
	}

	var req AssignSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.AssignSubject(r.Context(), tenantID, classID, req); err != nil {
		if errors.Is(err, ErrInvalidInput) {
			response.Error(w, http.StatusBadRequest, "valid subject_id is required")
			return
		}
		if errors.Is(err, ErrAlreadyExists) {
			response.Error(w, http.StatusConflict, "subject already assigned to class")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to assign subject")
		return
	}
	response.JSON(w, http.StatusCreated, map[string]string{"message": "subject assigned"})
}

func (h *Handler) RemoveSubject(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	classID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid class ID")
		return
	}

	subjectID, err := uuid.Parse(r.PathValue("subjectId"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid subject ID")
		return
	}

	if err := h.service.RemoveSubject(r.Context(), tenantID, classID, subjectID); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to remove subject")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "subject removed"})
}

// ── Subjects ───────────────────────────────────────────────────────────────────

func (h *Handler) ListSubjects(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	subjects, err := h.service.ListSubjects(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list subjects")
		return
	}
	response.JSON(w, http.StatusOK, subjects)
}

func (h *Handler) CreateSubject(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sub, err := h.service.CreateSubject(r.Context(), tenantID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			response.Error(w, http.StatusBadRequest, "subject name is required")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to create subject")
		return
	}
	response.JSON(w, http.StatusCreated, sub)
}

func (h *Handler) UpdateSubject(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	subjectID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid subject ID")
		return
	}

	var req CreateSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sub, err := h.service.UpdateSubject(r.Context(), tenantID, subjectID, req)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to update subject")
		return
	}
	response.JSON(w, http.StatusOK, sub)
}

func (h *Handler) DeleteSubject(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	subjectID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid subject ID")
		return
	}

	if err := h.service.DeleteSubject(r.Context(), tenantID, subjectID); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to delete subject")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "subject deleted"})
}

// ── Students ───────────────────────────────────────────────────────────────────

func (h *Handler) ListStudents(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	students, total, err := h.service.ListStudents(r.Context(), tenantID, page, perPage)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list students")
		return
	}
	response.Paginated(w, students, page, perPage, total)
}

func (h *Handler) GetStudent(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	studentID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid student ID")
		return
	}

	student, err := h.service.GetStudent(r.Context(), tenantID, studentID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "student not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to get student")
		return
	}
	response.JSON(w, http.StatusOK, student)
}

func (h *Handler) CreateStudent(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	student, err := h.service.CreateStudent(r.Context(), tenantID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			response.Error(w, http.StatusBadRequest, "email, full_name, and password are required")
			return
		}
		if errors.Is(err, ErrAlreadyExists) {
			response.Error(w, http.StatusConflict, "email already exists")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to create student")
		return
	}
	response.JSON(w, http.StatusCreated, student)
}

// ── Teachers ───────────────────────────────────────────────────────────────────

func (h *Handler) ListTeachers(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromCtx(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	teachers, err := h.service.ListTeachers(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list teachers")
		return
	}
	response.JSON(w, http.StatusOK, teachers)
}
