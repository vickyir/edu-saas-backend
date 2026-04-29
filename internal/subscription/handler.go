package subscription

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
	// Public
	mux.HandleFunc("GET /api/v1/plans", h.ListPlans)
	mux.HandleFunc("GET /api/v1/plans/{id}", h.GetPlan)

	// Authenticated (admin manages plans via separate admin routes)
	mux.HandleFunc("POST /api/v1/admin/plans", h.CreatePlan)

	// Tenant subscription management
	mux.HandleFunc("POST /api/v1/subscription", h.Subscribe)
	mux.HandleFunc("GET /api/v1/subscription", h.GetSubscription)
	mux.HandleFunc("PUT /api/v1/subscription/plan", h.ChangePlan)
	mux.HandleFunc("POST /api/v1/subscription/cancel", h.Cancel)
	mux.HandleFunc("GET /api/v1/subscription/invoices", h.ListInvoices)
}

func (h *Handler) ListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.service.ListPlans(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list plans")
		return
	}
	response.JSON(w, http.StatusOK, plans)
}

func (h *Handler) GetPlan(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid plan ID")
		return
	}
	plan, err := h.service.GetPlan(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "plan not found")
		return
	}
	response.JSON(w, http.StatusOK, plan)
}

func (h *Handler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	var req CreatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Slug == "" {
		response.Error(w, http.StatusBadRequest, "name and slug are required")
		return
	}
	plan, err := h.service.CreatePlan(r.Context(), req)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to create plan")
		return
	}
	response.JSON(w, http.StatusCreated, plan)
}

func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sub, err := h.service.Subscribe(r.Context(), claims.TenantID, req)
	if err != nil {
		switch err {
		case ErrAlreadySubscribed:
			response.Error(w, http.StatusConflict, err.Error())
		case ErrPlanNotFound:
			response.Error(w, http.StatusNotFound, err.Error())
		default:
			response.Error(w, http.StatusInternalServerError, "subscription failed")
		}
		return
	}

	response.JSON(w, http.StatusCreated, sub)
}

func (h *Handler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sub, err := h.service.GetSubscription(r.Context(), claims.TenantID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "no active subscription")
		return
	}

	response.JSON(w, http.StatusOK, sub)
}

func (h *Handler) ChangePlan(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ChangePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sub, err := h.service.ChangePlan(r.Context(), claims.TenantID, req)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to change plan")
		return
	}

	response.JSON(w, http.StatusOK, sub)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sub, err := h.service.CancelSubscription(r.Context(), claims.TenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to cancel")
		return
	}

	response.JSON(w, http.StatusOK, sub)
}

func (h *Handler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 50 {
		perPage = 20
	}

	invoices, total, err := h.service.GetInvoices(r.Context(), claims.TenantID, page, perPage)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list invoices")
		return
	}

	response.Paginated(w, invoices, page, perPage, total)
}
