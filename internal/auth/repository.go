package auth

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
// User Queries
// ──────────────────────────────────────

func (r *Repository) CreateUser(ctx context.Context, u *User) error {
	query := `
		INSERT INTO users (id, tenant_id, email, phone, full_name, password_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	now := time.Now()
	u.ID = uuid.New()
	u.CreatedAt = now
	u.UpdatedAt = now

	_, err := r.db.Pool.Exec(ctx, query,
		u.ID, u.TenantID, u.Email, u.Phone, u.FullName, u.Password, u.Status, u.CreatedAt, u.UpdatedAt,
	)
	return err
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, tenant_id, email, phone, full_name, password_hash, status, created_at, updated_at
		FROM users WHERE email = $1 AND deleted_at IS NULL
	`
	u := &User{}
	err := r.db.Pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.Phone, &u.FullName, &u.Password, &u.Status, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, tenant_id, email, phone, full_name, password_hash, status, created_at, updated_at
		FROM users WHERE id = $1 AND deleted_at IS NULL
	`
	u := &User{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.Phone, &u.FullName, &u.Password, &u.Status, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *Repository) UpdateUser(ctx context.Context, u *User) error {
	query := `
		UPDATE users SET full_name = $1, phone = $2, updated_at = $3
		WHERE id = $4 AND deleted_at IS NULL
	`
	u.UpdatedAt = time.Now()
	_, err := r.db.Pool.Exec(ctx, query, u.FullName, u.Phone, u.UpdatedAt, u.ID)
	return err
}

func (r *Repository) UpdatePassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.Pool.Exec(ctx, query, hashedPassword, time.Now(), userID)
	return err
}

func (r *Repository) GetUsersByTenant(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]*User, int64, error) {
	conn, err := r.db.WithTenant(ctx, tenantID.String())
	if err != nil {
		return nil, 0, err
	}
	defer conn.Release()

	var total int64
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND deleted_at IS NULL", tenantID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, tenant_id, email, phone, full_name, status, created_at, updated_at
		FROM users WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := conn.Query(ctx, query, tenantID, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Email, &u.Phone, &u.FullName, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, nil
}

// ──────────────────────────────────────
// Role Queries
// ──────────────────────────────────────

func (r *Repository) CreateRole(ctx context.Context, role *Role) error {
	query := `
		INSERT INTO roles (id, tenant_id, name, permissions, is_system, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	role.ID = uuid.New()
	role.CreatedAt = time.Now()

	permsJSON, _ := json.Marshal(role.Permissions)

	_, err := r.db.Pool.Exec(ctx, query,
		role.ID, role.TenantID, role.Name, permsJSON, role.IsSystem, role.CreatedAt,
	)
	return err
}

func (r *Repository) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	query := `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := r.db.Pool.Exec(ctx, query, userID, roleID)
	return err
}

func (r *Repository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*Role, error) {
	query := `
		SELECT r.id, r.tenant_id, r.name, r.permissions, r.is_system, r.created_at
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*Role
	for rows.Next() {
		role := &Role{}
		var permsJSON []byte
		if err := rows.Scan(&role.ID, &role.TenantID, &role.Name, &permsJSON, &role.IsSystem, &role.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(permsJSON, &role.Permissions)
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *Repository) GetRoleByName(ctx context.Context, tenantID uuid.UUID, name string) (*Role, error) {
	query := `
		SELECT id, tenant_id, name, permissions, is_system, created_at
		FROM roles WHERE tenant_id = $1 AND name = $2
	`
	role := &Role{}
	var permsJSON []byte
	err := r.db.Pool.QueryRow(ctx, query, tenantID, name).Scan(
		&role.ID, &role.TenantID, &role.Name, &permsJSON, &role.IsSystem, &role.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(permsJSON, &role.Permissions)
	return role, nil
}

// ──────────────────────────────────────
// Session Queries
// ──────────────────────────────────────

func (r *Repository) CreateSession(ctx context.Context, s *Session) error {
	query := `
		INSERT INTO sessions (id, user_id, tenant_id, refresh_token, user_agent, ip, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	s.ID = uuid.New()
	s.CreatedAt = time.Now()
	_, err := r.db.Pool.Exec(ctx, query,
		s.ID, s.UserID, s.TenantID, s.RefreshToken, s.UserAgent, s.IP, s.ExpiresAt, s.CreatedAt,
	)
	return err
}

func (r *Repository) GetSessionByRefreshToken(ctx context.Context, token string) (*Session, error) {
	query := `
		SELECT id, user_id, tenant_id, refresh_token, expires_at
		FROM sessions WHERE refresh_token = $1 AND expires_at > NOW()
	`
	s := &Session{}
	err := r.db.Pool.QueryRow(ctx, query, token).Scan(
		&s.ID, &s.UserID, &s.TenantID, &s.RefreshToken, &s.ExpiresAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return s, err
}

func (r *Repository) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, "DELETE FROM sessions WHERE id = $1", sessionID)
	return err
}

func (r *Repository) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, "DELETE FROM sessions WHERE user_id = $1", userID)
	return err
}

// EmailExists checks if an email is already registered
func (r *Repository) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.db.Pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)", email,
	).Scan(&exists)
	return exists, fmt.Errorf("check email: %w", err)
}
