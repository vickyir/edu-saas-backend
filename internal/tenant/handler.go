package tenant

import (
	"encoding/json"
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

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/tenants", h.Create)
	h.RegisterProtectedRoutes(mux)
}

// RegisterProtectedRoutes registers only the routes that require authentication
func (h *Handler) RegisterProtectedRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/tenants/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/tenants/{id}", h.Update)
	mux.HandleFunc("GET /api/v1/tenants", h.List)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Subdomain == "" || req.AdminEmail == "" || req.AdminPassword == "" {
		response.Error(w, http.StatusBadRequest, "name, subdomain, admin_email, and admin_password are required")
		return
	}

	t, err := h.service.CreateTenant(r.Context(), req)
	if err != nil {
		switch err {
		case ErrSubdomainTaken:
			response.Error(w, http.StatusConflict, err.Error())
		case ErrInvalidSubdomain:
			response.Error(w, http.StatusBadRequest, err.Error())
		default:
			response.Error(w, http.StatusInternalServerError, "failed to create tenant")
		}
		return
	}

	response.JSON(w, http.StatusCreated, t)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}

	// Verify access: user must belong to this tenant or be a gov admin
	claims := auth.ClaimsFromContext(r.Context())
	if claims != nil && claims.TenantID != id {
		hasGovAccess := false
		for _, role := range claims.Roles {
			if role == auth.RoleGovAdmin {
				hasGovAccess = true
				break
			}
		}
		if !hasGovAccess {
			response.Error(w, http.StatusForbidden, "access denied")
			return
		}
	}

	t, err := h.service.GetTenant(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "tenant not found")
		return
	}

	response.JSON(w, http.StatusOK, t)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}

	var req UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	t, err := h.service.UpdateTenant(r.Context(), id, req)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "update failed")
		return
	}

	response.JSON(w, http.StatusOK, t)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	var parentID *uuid.UUID
	if pid := r.URL.Query().Get("parent_id"); pid != "" {
		if parsed, err := uuid.Parse(pid); err == nil {
			parentID = &parsed
		}
	}

	tenants, total, err := h.service.ListTenants(r.Context(), parentID, page, perPage)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list tenants")
		return
	}

	response.Paginated(w, tenants, page, perPage, total)
}
