package auth

import (
	"time"

	"github.com/google/uuid"
)

// ──────────────────────────────────────
// Domain Models
// ──────────────────────────────────────

type User struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	Email     string     `json:"email"`
	Phone     string     `json:"phone,omitempty"`
	FullName  string     `json:"full_name"`
	Password  string     `json:"-"` // never serialized
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"-"`
}

type Role struct {
	ID          uuid.UUID   `json:"id"`
	TenantID    uuid.UUID   `json:"tenant_id"`
	Name        string      `json:"name"`
	Permissions []string    `json:"permissions"`
	IsSystem    bool        `json:"is_system"`
	CreatedAt   time.Time   `json:"created_at"`
}

type UserRole struct {
	UserID uuid.UUID `json:"user_id"`
	RoleID uuid.UUID `json:"role_id"`
	Role   *Role     `json:"role,omitempty"`
}

type Session struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	RefreshToken string    `json:"-"`
	UserAgent    string    `json:"user_agent"`
	IP           string    `json:"ip"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// ──────────────────────────────────────
// Request / Response DTOs
// ──────────────────────────────────────

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	User         *User  `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type UpdateProfileRequest struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ──────────────────────────────────────
// Permissions Constants
// ──────────────────────────────────────

const (
	StatusActive   = "active"
	StatusInactive = "inactive"
	StatusPending  = "pending"

	RoleSuperAdmin  = "superadmin"
	RoleGovAdmin    = "gov_admin"
	RoleSchoolAdmin = "school_admin"
	RoleTeacher     = "teacher"
	RoleStudent     = "student"
	RoleParent      = "parent"
)

// System permissions
var SystemPermissions = map[string][]string{
	RoleSuperAdmin: {
		"superadmin",
		"tenants:read_all", "tenants:manage", "tenants:create",
		"schools:read_all", "schools:manage",
		"users:read_all", "users:manage",
		"plans:manage",
		"subscription:manage", "subscription:read",
		"reports:read_all", "reports:aggregate",
		"classes:manage", "classes:read",
		"subjects:manage", "subjects:read",
		"attendance:configure", "attendance:read",
		"grades:manage", "grades:read",
	},
	RoleGovAdmin: {
		"tenants:read_all", "tenants:manage",
		"schools:read_all", "schools:manage",
		"reports:read_all", "reports:aggregate",
		"users:read_all", "users:manage",
		"subscription:manage", "subscription:read",
		"classes:read",
		"attendance:read",
	},
	RoleSchoolAdmin: {
		"school:manage", "school:read",
		"users:manage", "users:read_all",
		"teachers:manage", "teachers:read",
		"students:manage", "students:read",
		"classes:manage", "classes:read",
		"subjects:manage", "subjects:read",
		"attendance:configure", "attendance:read",
		"subscription:manage", "subscription:read",
		"reports:read",
	},
	RoleTeacher: {
		"school:read",
		"classes:read",
		"subjects:read",
		"students:read",
		"attendance:record", "attendance:read",
		"grades:manage", "grades:read",
		"schedule:read",
	},
	RoleStudent: {
		"profile:read", "profile:update",
		"classes:read_own",
		"subjects:read_own",
		"grades:read_own",
		"attendance:submit", "attendance:read_own",
		"reports:read_own",
	},
	RoleParent: {
		"profile:read",
		"children:read",
		"attendance:read_children",
		"grades:read_children",
		"monitoring:read",
		"reports:read_children",
	},
}
