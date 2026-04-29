package subscription

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/edusaas/backend/internal/shared/events"
)

var (
	ErrPlanNotFound         = errors.New("plan not found")
	ErrAlreadySubscribed    = errors.New("tenant already has an active subscription")
	ErrNoActiveSubscription = errors.New("no active subscription found")
	ErrFeatureNotAvailable  = errors.New("feature not available in current plan")
)

type Service struct {
	repo     *Repository
	eventBus *events.Bus
}

func NewService(repo *Repository, eventBus *events.Bus) *Service {
	return &Service{repo: repo, eventBus: eventBus}
}

// ──────────────────────────────────────
// Plans
// ──────────────────────────────────────

func (s *Service) CreatePlan(ctx context.Context, req CreatePlanRequest) (*Plan, error) {
	plan := &Plan{
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  req.Description,
		BillingCycle: req.BillingCycle,
		Price:        req.Price,
		Currency:     req.Currency,
		FeatureFlags: req.FeatureFlags,
		SortOrder:    req.SortOrder,
	}

	if plan.FeatureFlags == nil {
		if defaults, ok := DefaultPlanFeatures[req.Slug]; ok {
			plan.FeatureFlags = defaults
		}
	}

	if err := s.repo.CreatePlan(ctx, plan); err != nil {
		return nil, fmt.Errorf("create plan: %w", err)
	}
	return plan, nil
}

func (s *Service) ListPlans(ctx context.Context) ([]*Plan, error) {
	return s.repo.ListPlans(ctx)
}

func (s *Service) GetPlan(ctx context.Context, id uuid.UUID) (*Plan, error) {
	plan, err := s.repo.GetPlanByID(ctx, id)
	if err != nil || plan == nil {
		return nil, ErrPlanNotFound
	}
	return plan, nil
}

// ──────────────────────────────────────
// Subscriptions
// ──────────────────────────────────────

func (s *Service) Subscribe(ctx context.Context, tenantID uuid.UUID, req SubscribeRequest) (*Subscription, error) {
	// Check for existing active subscription
	existing, _ := s.repo.GetActiveSubscription(ctx, tenantID)
	if existing != nil {
		return nil, ErrAlreadySubscribed
	}

	plan, err := s.repo.GetPlanByID(ctx, req.PlanID)
	if err != nil || plan == nil {
		return nil, ErrPlanNotFound
	}

	now := time.Now()
	periodEnd := calculatePeriodEnd(now, plan.BillingCycle)

	sub := &Subscription{
		TenantID:           tenantID,
		PlanID:             plan.ID,
		Status:             SubActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
		AutoRenew:          req.AutoRenew,
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, fmt.Errorf("create subscription: %w", err)
	}

	// Generate invoice
	invoiceNum, _ := s.repo.GenerateInvoiceNumber(ctx)
	invoice := &Invoice{
		SubscriptionID: sub.ID,
		TenantID:       tenantID,
		Amount:         plan.Price,
		Currency:       plan.Currency,
		Status:         InvoicePending,
		DueDate:        now.Add(7 * 24 * time.Hour), // 7 days to pay
		InvoiceNumber:  invoiceNum,
		Description:    fmt.Sprintf("Subscription: %s (%s)", plan.Name, plan.BillingCycle),
	}
	_ = s.repo.CreateInvoice(ctx, invoice)

	sub.Plan = plan

	s.eventBus.Publish(events.Event{
		Type:     events.EventSubscriptionChange,
		TenantID: tenantID.String(),
		Payload: map[string]any{
			"action":  "subscribed",
			"plan":    plan.Slug,
			"plan_id": plan.ID.String(),
		},
	})

	return sub, nil
}

func (s *Service) GetSubscription(ctx context.Context, tenantID uuid.UUID) (*Subscription, error) {
	sub, err := s.repo.GetActiveSubscription(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrNoActiveSubscription
	}
	return sub, nil
}

func (s *Service) ChangePlan(ctx context.Context, tenantID uuid.UUID, req ChangePlanRequest) (*Subscription, error) {
	sub, err := s.repo.GetActiveSubscription(ctx, tenantID)
	if err != nil || sub == nil {
		return nil, ErrNoActiveSubscription
	}

	newPlan, err := s.repo.GetPlanByID(ctx, req.NewPlanID)
	if err != nil || newPlan == nil {
		return nil, ErrPlanNotFound
	}

	// For simplicity, change takes effect at next billing period
	sub.PlanID = newPlan.ID
	sub.Plan = newPlan

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return nil, fmt.Errorf("update subscription: %w", err)
	}

	s.eventBus.Publish(events.Event{
		Type:     events.EventSubscriptionChange,
		TenantID: tenantID.String(),
		Payload: map[string]any{
			"action":  "plan_changed",
			"plan":    newPlan.Slug,
			"plan_id": newPlan.ID.String(),
		},
	})

	return sub, nil
}

func (s *Service) CancelSubscription(ctx context.Context, tenantID uuid.UUID) (*Subscription, error) {
	sub, err := s.repo.GetActiveSubscription(ctx, tenantID)
	if err != nil || sub == nil {
		return nil, ErrNoActiveSubscription
	}

	now := time.Now()
	sub.Status = SubCanceled
	sub.CanceledAt = &now
	sub.AutoRenew = false

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return nil, fmt.Errorf("cancel subscription: %w", err)
	}

	s.eventBus.Publish(events.Event{
		Type:     events.EventSubscriptionChange,
		TenantID: tenantID.String(),
		Payload:  map[string]any{"action": "canceled"},
	})

	return sub, nil
}

// ──────────────────────────────────────
// Feature Gating
// ──────────────────────────────────────

func (s *Service) HasFeature(ctx context.Context, tenantID uuid.UUID, feature string) (bool, error) {
	sub, err := s.repo.GetActiveSubscription(ctx, tenantID)
	if err != nil || sub == nil || sub.Plan == nil {
		return false, nil
	}

	val, ok := sub.Plan.FeatureFlags[feature]
	if !ok {
		return false, nil
	}

	switch v := val.(type) {
	case bool:
		return v, nil
	case float64:
		return v != 0, nil
	case string:
		return v != "", nil
	default:
		return true, nil
	}
}

func (s *Service) GetFeatureLimit(ctx context.Context, tenantID uuid.UUID, feature string) (int, error) {
	sub, err := s.repo.GetActiveSubscription(ctx, tenantID)
	if err != nil || sub == nil || sub.Plan == nil {
		return 0, nil
	}

	val, ok := sub.Plan.FeatureFlags[feature]
	if !ok {
		return 0, nil
	}

	if v, ok := val.(float64); ok {
		return int(v), nil
	}
	return 0, nil
}

// ──────────────────────────────────────
// Invoices
// ──────────────────────────────────────

func (s *Service) GetInvoices(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]*Invoice, int64, error) {
	return s.repo.GetInvoicesByTenant(ctx, tenantID, page, perPage)
}

func (s *Service) MarkInvoicePaid(ctx context.Context, invoiceID uuid.UUID) error {
	now := time.Now()
	return s.repo.UpdateInvoiceStatus(ctx, invoiceID, InvoicePaid, &now)
}

// ──────────────────────────────────────
// Helpers
// ──────────────────────────────────────

func calculatePeriodEnd(start time.Time, cycle BillingCycle) time.Time {
	switch cycle {
	case CycleMonthly:
		return start.AddDate(0, 1, 0)
	case CycleQuarterly:
		return start.AddDate(0, 3, 0)
	case CycleYearly:
		return start.AddDate(1, 0, 0)
	default:
		return start.AddDate(0, 1, 0)
	}
}
