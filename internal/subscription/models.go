package subscription

import (
	"time"

	"github.com/google/uuid"
)

type BillingCycle string

const (
	CycleMonthly   BillingCycle = "monthly"
	CycleQuarterly BillingCycle = "quarterly"
	CycleYearly    BillingCycle = "yearly"
)

type PlanStatus string

const (
	PlanActive   PlanStatus = "active"
	PlanInactive PlanStatus = "inactive"
)

type SubscriptionStatus string

const (
	SubActive    SubscriptionStatus = "active"
	SubPastDue   SubscriptionStatus = "past_due"
	SubCanceled  SubscriptionStatus = "canceled"
	SubExpired   SubscriptionStatus = "expired"
	SubTrialing  SubscriptionStatus = "trialing"
)

type InvoiceStatus string

const (
	InvoicePending  InvoiceStatus = "pending"
	InvoicePaid     InvoiceStatus = "paid"
	InvoiceOverdue  InvoiceStatus = "overdue"
	InvoiceCanceled InvoiceStatus = "canceled"
)

// ──────────────────────────────────────
// Domain Models
// ──────────────────────────────────────

type Plan struct {
	ID            uuid.UUID         `json:"id"`
	Name          string            `json:"name"`
	Slug          string            `json:"slug"`
	Description   string            `json:"description"`
	BillingCycle  BillingCycle      `json:"billing_cycle"`
	Price         int64             `json:"price"` // in smallest currency unit (IDR)
	Currency      string            `json:"currency"`
	FeatureFlags  map[string]any    `json:"feature_flags"`
	Status        PlanStatus        `json:"status"`
	SortOrder     int               `json:"sort_order"`
	CreatedAt     time.Time         `json:"created_at"`
}

type Subscription struct {
	ID                 uuid.UUID          `json:"id"`
	TenantID           uuid.UUID          `json:"tenant_id"`
	PlanID             uuid.UUID          `json:"plan_id"`
	Status             SubscriptionStatus `json:"status"`
	CurrentPeriodStart time.Time          `json:"current_period_start"`
	CurrentPeriodEnd   time.Time          `json:"current_period_end"`
	AutoRenew          bool               `json:"auto_renew"`
	TrialEndsAt        *time.Time         `json:"trial_ends_at,omitempty"`
	CanceledAt         *time.Time         `json:"canceled_at,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`

	// Joined
	Plan *Plan `json:"plan,omitempty"`
}

type Invoice struct {
	ID             uuid.UUID     `json:"id"`
	SubscriptionID uuid.UUID     `json:"subscription_id"`
	TenantID       uuid.UUID     `json:"tenant_id"`
	Amount         int64         `json:"amount"`
	Currency       string        `json:"currency"`
	Status         InvoiceStatus `json:"status"`
	DueDate        time.Time     `json:"due_date"`
	PaidAt         *time.Time    `json:"paid_at,omitempty"`
	InvoiceNumber  string        `json:"invoice_number"`
	Description    string        `json:"description"`
	CreatedAt      time.Time     `json:"created_at"`
}

// ──────────────────────────────────────
// DTOs
// ──────────────────────────────────────

type CreatePlanRequest struct {
	Name         string         `json:"name"`
	Slug         string         `json:"slug"`
	Description  string         `json:"description"`
	BillingCycle BillingCycle   `json:"billing_cycle"`
	Price        int64          `json:"price"`
	Currency     string         `json:"currency"`
	FeatureFlags map[string]any `json:"feature_flags"`
	SortOrder    int            `json:"sort_order"`
}

type SubscribeRequest struct {
	PlanID    uuid.UUID `json:"plan_id"`
	AutoRenew bool      `json:"auto_renew"`
}

type ChangePlanRequest struct {
	NewPlanID uuid.UUID `json:"new_plan_id"`
}

// Default feature flags per plan tier
var DefaultPlanFeatures = map[string]map[string]any{
	"starter": {
		"max_students":        200,
		"max_teachers":        20,
		"attendance_methods":  []string{"qr"},
		"realtime_monitoring": false,
		"ppdb_enabled":        false,
		"e_report_cards":      true,
		"api_access":          false,
		"custom_branding":     false,
	},
	"professional": {
		"max_students":        1000,
		"max_teachers":        100,
		"attendance_methods":  []string{"qr", "geo"},
		"realtime_monitoring": true,
		"ppdb_enabled":        true,
		"e_report_cards":      true,
		"api_access":          false,
		"custom_branding":     true,
	},
	"enterprise": {
		"max_students":        -1, // unlimited
		"max_teachers":        -1,
		"attendance_methods":  []string{"qr", "geo", "face"},
		"realtime_monitoring": true,
		"ppdb_enabled":        true,
		"e_report_cards":      true,
		"api_access":          true,
		"custom_branding":     true,
	},
}
