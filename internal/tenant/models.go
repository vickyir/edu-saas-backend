package tenant

import (
	"time"

	"github.com/google/uuid"
)

type TenantType string

const (
	TenantTypeSchool     TenantType = "school"
	TenantTypeGovernment TenantType = "government"
)

type SchoolLevel string

const (
	LevelSD  SchoolLevel = "sd"
	LevelSMP SchoolLevel = "smp"
	LevelSMA SchoolLevel = "sma"
	LevelSMK SchoolLevel = "smk"
)

type Tenant struct {
	ID             uuid.UUID   `json:"id"`
	Name           string      `json:"name"`
	Type           TenantType  `json:"type"`
	ParentTenantID *uuid.UUID  `json:"parent_tenant_id,omitempty"`
	Subdomain      string      `json:"subdomain"`
	Config         *TenantConfig `json:"config,omitempty"`
	IsActive       bool        `json:"is_active"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type TenantConfig struct {
	SchoolLevel    SchoolLevel `json:"school_level,omitempty"`
	NPSN           string      `json:"npsn,omitempty"` // National School ID
	Address        string      `json:"address,omitempty"`
	City           string      `json:"city,omitempty"`
	Province       string      `json:"province,omitempty"`
	Phone          string      `json:"phone,omitempty"`
	Email          string      `json:"email,omitempty"`
	Logo           string      `json:"logo,omitempty"`
	Latitude       float64     `json:"latitude,omitempty"`
	Longitude      float64     `json:"longitude,omitempty"`
	GeofenceRadius int         `json:"geofence_radius_m,omitempty"` // meters
	Timezone       string      `json:"timezone,omitempty"`
}

// ──────────────────────────────────────
// DTOs
// ──────────────────────────────────────

type CreateTenantRequest struct {
	Name           string      `json:"name"`
	Type           TenantType  `json:"type"`
	Subdomain      string      `json:"subdomain"`
	ParentTenantID *uuid.UUID  `json:"parent_tenant_id,omitempty"`
	Config         *TenantConfig `json:"config,omitempty"`

	// Admin user created with the tenant
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
	AdminName     string `json:"admin_name"`
}

type UpdateTenantRequest struct {
	Name   string        `json:"name"`
	Config *TenantConfig `json:"config,omitempty"`
}

type TenantStats struct {
	TotalUsers    int `json:"total_users"`
	TotalTeachers int `json:"total_teachers"`
	TotalStudents int `json:"total_students"`
	TotalClasses  int `json:"total_classes"`
}
