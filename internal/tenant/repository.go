package tenant

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

func (r *Repository) Create(ctx context.Context, t *Tenant) error {
	query := `
		INSERT INTO tenants (id, name, type, parent_tenant_id, subdomain, config, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	now := time.Now()
	t.ID = uuid.New()
	t.CreatedAt = now
	t.UpdatedAt = now
	t.IsActive = true

	configJSON, _ := json.Marshal(t.Config)

	_, err := r.db.Pool.Exec(ctx, query,
		t.ID, t.Name, t.Type, t.ParentTenantID, t.Subdomain, configJSON, t.IsActive, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	query := `
		SELECT id, name, type, parent_tenant_id, subdomain, config, is_active, created_at, updated_at
		FROM tenants WHERE id = $1
	`
	t := &Tenant{}
	var configJSON []byte
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Name, &t.Type, &t.ParentTenantID, &t.Subdomain, &configJSON, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if configJSON != nil {
		t.Config = &TenantConfig{}
		json.Unmarshal(configJSON, t.Config)
	}
	return t, nil
}

func (r *Repository) GetBySubdomain(ctx context.Context, subdomain string) (*Tenant, error) {
	query := `
		SELECT id, name, type, parent_tenant_id, subdomain, config, is_active, created_at, updated_at
		FROM tenants WHERE subdomain = $1
	`
	t := &Tenant{}
	var configJSON []byte
	err := r.db.Pool.QueryRow(ctx, query, subdomain).Scan(
		&t.ID, &t.Name, &t.Type, &t.ParentTenantID, &t.Subdomain, &configJSON, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if configJSON != nil {
		t.Config = &TenantConfig{}
		json.Unmarshal(configJSON, t.Config)
	}
	return t, nil
}

func (r *Repository) Update(ctx context.Context, t *Tenant) error {
	configJSON, _ := json.Marshal(t.Config)
	t.UpdatedAt = time.Now()
	query := `UPDATE tenants SET name = $1, config = $2, updated_at = $3 WHERE id = $4`
	_, err := r.db.Pool.Exec(ctx, query, t.Name, configJSON, t.UpdatedAt, t.ID)
	return err
}

func (r *Repository) List(ctx context.Context, parentID *uuid.UUID, page, perPage int) ([]*Tenant, int64, error) {
	var total int64
	countQuery := "SELECT COUNT(*) FROM tenants WHERE 1=1"
	listQuery := `
		SELECT id, name, type, parent_tenant_id, subdomain, config, is_active, created_at, updated_at
		FROM tenants WHERE 1=1
	`
	var args []any
	argIdx := 1

	if parentID != nil {
		countQuery += fmt.Sprintf(" AND parent_tenant_id = $%d", argIdx)
		listQuery += fmt.Sprintf(" AND parent_tenant_id = $%d", argIdx)
		args = append(args, *parentID)
		argIdx++
	}

	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, perPage, (page-1)*perPage)

	rows, err := r.db.Pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tenants []*Tenant
	for rows.Next() {
		t := &Tenant{}
		var configJSON []byte
		if err := rows.Scan(&t.ID, &t.Name, &t.Type, &t.ParentTenantID, &t.Subdomain, &configJSON, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if configJSON != nil {
			t.Config = &TenantConfig{}
			json.Unmarshal(configJSON, t.Config)
		}
		tenants = append(tenants, t)
	}
	return tenants, total, nil
}

func (r *Repository) SubdomainExists(ctx context.Context, subdomain string) (bool, error) {
	var exists bool
	err := r.db.Pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM tenants WHERE subdomain = $1)", subdomain,
	).Scan(&exists)
	return exists, err
}
