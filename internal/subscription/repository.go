package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edusaas/backend/internal/shared/database"
)

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// ──────────────────────────────────────
// Plan Queries
// ──────────────────────────────────────

func (r *Repository) CreatePlan(ctx context.Context, p *Plan) error {
	query := `
		INSERT INTO plans (id, name, slug, description, billing_cycle, price, currency, feature_flags, status, sort_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	if p.Currency == "" {
		p.Currency = "IDR"
	}
	p.Status = PlanActive

	flagsJSON, _ := json.Marshal(p.FeatureFlags)

	_, err := r.db.Pool.Exec(ctx, query,
		p.ID, p.Name, p.Slug, p.Description, p.BillingCycle, p.Price, p.Currency,
		flagsJSON, p.Status, p.SortOrder, p.CreatedAt,
	)
	return err
}

func (r *Repository) GetPlanByID(ctx context.Context, id uuid.UUID) (*Plan, error) {
	query := `
		SELECT id, name, slug, description, billing_cycle, price, currency, feature_flags, status, sort_order, created_at
		FROM plans WHERE id = $1
	`
	return r.scanPlan(r.db.Pool.QueryRow(ctx, query, id))
}

func (r *Repository) GetPlanBySlug(ctx context.Context, slug string) (*Plan, error) {
	query := `
		SELECT id, name, slug, description, billing_cycle, price, currency, feature_flags, status, sort_order, created_at
		FROM plans WHERE slug = $1 AND status = 'active'
	`
	return r.scanPlan(r.db.Pool.QueryRow(ctx, query, slug))
}

func (r *Repository) ListPlans(ctx context.Context) ([]*Plan, error) {
	query := `
		SELECT id, name, slug, description, billing_cycle, price, currency, feature_flags, status, sort_order, created_at
		FROM plans WHERE status = 'active'
		ORDER BY sort_order ASC, price ASC
	`
	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []*Plan
	for rows.Next() {
		p, err := r.scanPlanRow(rows)
		if err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	return plans, nil
}

func (r *Repository) scanPlan(row pgx.Row) (*Plan, error) {
	p := &Plan{}
	var flagsJSON []byte
	err := row.Scan(
		&p.ID, &p.Name, &p.Slug, &p.Description, &p.BillingCycle, &p.Price,
		&p.Currency, &flagsJSON, &p.Status, &p.SortOrder, &p.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if flagsJSON != nil {
		json.Unmarshal(flagsJSON, &p.FeatureFlags)
	}
	return p, nil
}

func (r *Repository) scanPlanRow(rows pgx.Rows) (*Plan, error) {
	p := &Plan{}
	var flagsJSON []byte
	err := rows.Scan(
		&p.ID, &p.Name, &p.Slug, &p.Description, &p.BillingCycle, &p.Price,
		&p.Currency, &flagsJSON, &p.Status, &p.SortOrder, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if flagsJSON != nil {
		json.Unmarshal(flagsJSON, &p.FeatureFlags)
	}
	return p, nil
}

// ──────────────────────────────────────
// Subscription Queries
// ──────────────────────────────────────

func (r *Repository) CreateSubscription(ctx context.Context, s *Subscription) error {
	query := `
		INSERT INTO subscriptions (id, tenant_id, plan_id, status, current_period_start, current_period_end, auto_renew, trial_ends_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	now := time.Now()
	s.ID = uuid.New()
	s.CreatedAt = now
	s.UpdatedAt = now

	_, err := r.db.Pool.Exec(ctx, query,
		s.ID, s.TenantID, s.PlanID, s.Status,
		s.CurrentPeriodStart, s.CurrentPeriodEnd,
		s.AutoRenew, s.TrialEndsAt, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *Repository) GetActiveSubscription(ctx context.Context, tenantID uuid.UUID) (*Subscription, error) {
	query := `
		SELECT s.id, s.tenant_id, s.plan_id, s.status, s.current_period_start, s.current_period_end,
		       s.auto_renew, s.trial_ends_at, s.canceled_at, s.created_at, s.updated_at,
		       p.id, p.name, p.slug, p.description, p.billing_cycle, p.price, p.currency, p.feature_flags, p.status, p.sort_order, p.created_at
		FROM subscriptions s
		JOIN plans p ON p.id = s.plan_id
		WHERE s.tenant_id = $1 AND s.status IN ('active', 'trialing')
		ORDER BY s.created_at DESC LIMIT 1
	`
	s := &Subscription{}
	p := &Plan{}
	var flagsJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, tenantID).Scan(
		&s.ID, &s.TenantID, &s.PlanID, &s.Status,
		&s.CurrentPeriodStart, &s.CurrentPeriodEnd,
		&s.AutoRenew, &s.TrialEndsAt, &s.CanceledAt, &s.CreatedAt, &s.UpdatedAt,
		&p.ID, &p.Name, &p.Slug, &p.Description, &p.BillingCycle, &p.Price,
		&p.Currency, &flagsJSON, &p.Status, &p.SortOrder, &p.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if flagsJSON != nil {
		json.Unmarshal(flagsJSON, &p.FeatureFlags)
	}
	s.Plan = p
	return s, nil
}

func (r *Repository) UpdateSubscription(ctx context.Context, s *Subscription) error {
	s.UpdatedAt = time.Now()
	query := `
		UPDATE subscriptions 
		SET plan_id = $1, status = $2, current_period_start = $3, current_period_end = $4,
		    auto_renew = $5, canceled_at = $6, updated_at = $7
		WHERE id = $8
	`
	_, err := r.db.Pool.Exec(ctx, query,
		s.PlanID, s.Status, s.CurrentPeriodStart, s.CurrentPeriodEnd,
		s.AutoRenew, s.CanceledAt, s.UpdatedAt, s.ID,
	)
	return err
}

func (r *Repository) GetExpiredSubscriptions(ctx context.Context) ([]*Subscription, error) {
	query := `
		SELECT id, tenant_id, plan_id, status, current_period_start, current_period_end,
		       auto_renew, trial_ends_at, canceled_at, created_at, updated_at
		FROM subscriptions
		WHERE status = 'active' AND current_period_end < NOW()
	`
	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*Subscription
	for rows.Next() {
		s := &Subscription{}
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.PlanID, &s.Status,
			&s.CurrentPeriodStart, &s.CurrentPeriodEnd,
			&s.AutoRenew, &s.TrialEndsAt, &s.CanceledAt, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

// ──────────────────────────────────────
// Invoice Queries
// ──────────────────────────────────────

func (r *Repository) CreateInvoice(ctx context.Context, inv *Invoice) error {
	query := `
		INSERT INTO invoices (id, subscription_id, tenant_id, amount, currency, status, due_date, invoice_number, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	inv.ID = uuid.New()
	inv.CreatedAt = time.Now()
	if inv.Currency == "" {
		inv.Currency = "IDR"
	}

	_, err := r.db.Pool.Exec(ctx, query,
		inv.ID, inv.SubscriptionID, inv.TenantID, inv.Amount, inv.Currency,
		inv.Status, inv.DueDate, inv.InvoiceNumber, inv.Description, inv.CreatedAt,
	)
	return err
}

func (r *Repository) GetInvoicesByTenant(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]*Invoice, int64, error) {
	var total int64
	err := r.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM invoices WHERE tenant_id = $1", tenantID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, subscription_id, tenant_id, amount, currency, status, due_date, paid_at, invoice_number, description, created_at
		FROM invoices WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Pool.Query(ctx, query, tenantID, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var invoices []*Invoice
	for rows.Next() {
		inv := &Invoice{}
		if err := rows.Scan(
			&inv.ID, &inv.SubscriptionID, &inv.TenantID, &inv.Amount, &inv.Currency,
			&inv.Status, &inv.DueDate, &inv.PaidAt, &inv.InvoiceNumber, &inv.Description, &inv.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		invoices = append(invoices, inv)
	}
	return invoices, total, nil
}

func (r *Repository) UpdateInvoiceStatus(ctx context.Context, invoiceID uuid.UUID, status InvoiceStatus, paidAt *time.Time) error {
	query := `UPDATE invoices SET status = $1, paid_at = $2 WHERE id = $3`
	_, err := r.db.Pool.Exec(ctx, query, status, paidAt, invoiceID)
	return err
}

func (r *Repository) GenerateInvoiceNumber(ctx context.Context) (string, error) {
	var seq int64
	err := r.db.Pool.QueryRow(ctx, "SELECT nextval('invoice_number_seq')").Scan(&seq)
	if err != nil {
		return "", err
	}
	now := time.Now()
	return fmt.Sprintf("INV-%d%02d-%06d", now.Year(), now.Month(), seq), nil
}
