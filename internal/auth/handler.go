package auth

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/edusaas/backend/internal/shared/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterPublicRoutes adds auth routes that do NOT require a token.
func (h *Handler) RegisterPublicRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", h.Register)
	mux.HandleFunc("POST /api/v1/auth/login", h.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.Refresh)
	mux.HandleFunc("POST /api/v1/auth/logout", h.Logout)
}

// RegisterProtectedRoutes adds auth routes that REQUIRE a valid token.
// Wire these behind the Authenticate middleware in main.go.
func (h *Handler) RegisterProtectedRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/auth/me", h.GetProfile)
	mux.HandleFunc("PUT /api/v1/auth/me", h.UpdateProfile)
	mux.HandleFunc("PUT /api/v1/auth/password", h.ChangePassword)
}

// RegisterAdminRoutes adds user-management routes (require auth + appropriate role).
func (h *Handler) RegisterAdminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admin/users", h.ListUsers)
	mux.HandleFunc("POST /api/v1/admin/users", h.CreateUser)
	mux.HandleFunc("GET /api/v1/admin/users/{id}", h.GetUser)
	mux.HandleFunc("PUT /api/v1/admin/users/{id}", h.UpdateUser)
	mux.HandleFunc("DELETE /api/v1/admin/users/{id}", h.DeleteUser)
	mux.HandleFunc("PUT /api/v1/admin/users/{id}/role", h.ChangeUserRole)
}

// RegisterRoutes is kept for backwards-compatibility and now only registers public routes.
// Deprecated: prefer RegisterPublicRoutes + RegisterProtectedRoutes.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	h.RegisterPublicRoutes(mux)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" || req.FullName == "" {
		response.Error(w, http.StatusBadRequest, "email, password, and full_name are required")
		return
	}
	if len(req.Password) < 8 {
		response.Error(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	// Tenant ID comes from context (set by middleware or registration flow)
	tenantID := TenantIDFromContext(r.Context())

	result, err := h.service.Register(r.Context(), req, tenantID)
	if err != nil {
		switch err {
		case ErrEmailTaken:
			response.Error(w, http.StatusConflict, err.Error())
		default:
			response.Error(w, http.StatusInternalServerError, "registration failed")
		}
		return
	}

	response.JSON(w, http.StatusCreated, result)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "email and password are required")
		return
	}

	result, err := h.service.Login(r.Context(), req, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		switch err {
		case ErrInvalidCredentials:
			response.Error(w, http.StatusUnauthorized, err.Error())
		case ErrAccountInactive:
			response.Error(w, http.StatusForbidden, err.Error())
		default:
			response.Error(w, http.StatusInternalServerError, "login failed")
		}
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.RefreshTokens(r.Context(), req.RefreshToken)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_ = h.service.Logout(r.Context(), req.RefreshToken)
	response.JSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.service.GetProfile(r.Context(), claims.UserID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "user not found")
		return
	}

	roles, _ := h.service.GetUserRoles(r.Context(), claims.UserID)

	response.JSON(w, http.StatusOK, map[string]any{
		"user":  user,
		"roles": roles,
	})
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.UpdateProfile(r.Context(), claims.UserID, req)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "update failed")
		return
	}

	response.JSON(w, http.StatusOK, user)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.ChangePassword(r.Context(), claims.UserID, req); err != nil {
		switch err {
		case ErrInvalidCredentials:
			response.Error(w, http.StatusBadRequest, "current password is incorrect")
		default:
			response.Error(w, http.StatusInternalServerError, "password change failed")
		}
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "password changed"})
}

// ── Admin User Management ──────────────────────────────────────────────────────

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil {
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
	roleFilter := r.URL.Query().Get("role")

	users, total, err := h.service.ListUsers(r.Context(), claims.TenantID, roleFilter, page, perPage)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	response.Paginated(w, users, page, perPage, total)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateManagedUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.CreateManagedUser(r.Context(), claims.TenantID, claims.Roles, req)
	if err != nil {
		switch err {
		case ErrEmailTaken:
			response.Error(w, http.StatusConflict, "email already registered")
		case ErrPermissionDenied:
			response.Error(w, http.StatusForbidden, "you cannot create a user with that role")
		default:
			response.Error(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	response.JSON(w, http.StatusCreated, user)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	user, err := h.service.GetUser(r.Context(), claims.TenantID, userID)
	if err != nil {
		if err == ErrUserNotFound {
			response.Error(w, http.StatusNotFound, "user not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to get user")
		return
	}
	response.JSON(w, http.StatusOK, user)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req UpdateManagedUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.UpdateManagedUser(r.Context(), claims.TenantID, userID, req)
	if err != nil {
		switch err {
		case ErrUserNotFound:
			response.Error(w, http.StatusNotFound, "user not found")
		case ErrPermissionDenied:
			response.Error(w, http.StatusForbidden, "access denied")
		default:
			response.Error(w, http.StatusInternalServerError, "failed to update user")
		}
		return
	}
	response.JSON(w, http.StatusOK, user)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	if err := h.service.DeleteManagedUser(r.Context(), claims.TenantID, userID); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to delete user")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}

func (h *Handler) ChangeUserRole(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	var body struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Role == "" {
		response.Error(w, http.StatusBadRequest, "role is required")
		return
	}

	if err := h.service.ChangeUserRole(r.Context(), claims.TenantID, userID, body.Role, claims.Roles); err != nil {
		switch err {
		case ErrPermissionDenied:
			response.Error(w, http.StatusForbidden, "you cannot assign that role")
		case ErrUserNotFound:
			response.Error(w, http.StatusNotFound, "user not found")
		default:
			response.Error(w, http.StatusInternalServerError, "failed to change role")
		}
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "role updated"})
}
