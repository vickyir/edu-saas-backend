package tenant

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/edusaas/backend/internal/auth"
	"github.com/edusaas/backend/internal/shared/events"
)

var (
	ErrTenantNotFound    = errors.New("tenant not found")
	ErrSubdomainTaken    = errors.New("subdomain already taken")
	ErrInvalidSubdomain  = errors.New("subdomain must be 3-50 lowercase alphanumeric characters or hyphens")
)

var subdomainRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

type Service struct {
	repo        *Repository
	authService *auth.Service
	authRepo    *auth.Repository
	eventBus    *events.Bus
}

func NewService(repo *Repository, authService *auth.Service, authRepo *auth.Repository, eventBus *events.Bus) *Service {
	return &Service{repo: repo, authService: authService, authRepo: authRepo, eventBus: eventBus}
}

// CreateTenant creates a new tenant with its admin user and default roles
func (s *Service) CreateTenant(ctx context.Context, req CreateTenantRequest) (*Tenant, error) {
	req.Subdomain = strings.ToLower(strings.TrimSpace(req.Subdomain))

	if !subdomainRegex.MatchString(req.Subdomain) {
		return nil, ErrInvalidSubdomain
	}

	exists, err := s.repo.SubdomainExists(ctx, req.Subdomain)
	if err != nil {
		return nil, fmt.Errorf("check subdomain: %w", err)
	}
	if exists {
		return nil, ErrSubdomainTaken
	}

	// Create tenant
	tenant := &Tenant{
		Name:           req.Name,
		Type:           req.Type,
		Subdomain:      req.Subdomain,
		ParentTenantID: req.ParentTenantID,
		Config:         req.Config,
	}

	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}

	// Create default roles for this tenant
	if err := s.createDefaultRoles(ctx, tenant.ID); err != nil {
		return nil, fmt.Errorf("create roles: %w", err)
	}

	// Create admin user
	adminUser := &auth.User{
		TenantID: tenant.ID,
		Email:    req.AdminEmail,
		FullName: req.AdminName,
		Status:   auth.StatusActive,
	}
	// Register via auth service (handles password hashing)
	_, err = s.authService.Register(ctx, auth.RegisterRequest{
		Email:    req.AdminEmail,
		Password: req.AdminPassword,
		FullName: req.AdminName,
	}, tenant.ID)
	if err != nil {
		return nil, fmt.Errorf("create admin: %w", err)
	}

	// Assign admin role
	createdUser, _ := s.authRepo.GetUserByEmail(ctx, req.AdminEmail)
	if createdUser != nil {
		roleName := auth.RoleSchoolAdmin
		if req.Type == TenantTypeGovernment {
			roleName = auth.RoleGovAdmin
		}
		_ = s.authService.AssignRoleToUser(ctx, createdUser.ID, tenant.ID, roleName)
	}

	s.eventBus.Publish(events.Event{
		Type:     events.EventTenantCreated,
		TenantID: tenant.ID.String(),
		Payload: map[string]any{
			"name":      tenant.Name,
			"type":      string(tenant.Type),
			"subdomain": tenant.Subdomain,
			"admin":     adminUser.Email,
		},
	})

	return tenant, nil
}

func (s *Service) GetTenant(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTenantNotFound
	}
	return t, nil
}

func (s *Service) GetTenantBySubdomain(ctx context.Context, subdomain string) (*Tenant, error) {
	t, err := s.repo.GetBySubdomain(ctx, subdomain)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTenantNotFound
	}
	return t, nil
}

func (s *Service) UpdateTenant(ctx context.Context, id uuid.UUID, req UpdateTenantRequest) (*Tenant, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil || t == nil {
		return nil, ErrTenantNotFound
	}

	t.Name = req.Name
	if req.Config != nil {
		t.Config = req.Config
	}

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("update tenant: %w", err)
	}
	return t, nil
}

func (s *Service) ListTenants(ctx context.Context, parentID *uuid.UUID, page, perPage int) ([]*Tenant, int64, error) {
	return s.repo.List(ctx, parentID, page, perPage)
}

// createDefaultRoles sets up the system roles for a new tenant
func (s *Service) createDefaultRoles(ctx context.Context, tenantID uuid.UUID) error {
	for roleName, permissions := range auth.SystemPermissions {
		role := &auth.Role{
			TenantID:    tenantID,
			Name:        roleName,
			Permissions: permissions,
			IsSystem:    true,
		}
		if err := s.authRepo.CreateRole(ctx, role); err != nil {
			return fmt.Errorf("create role %s: %w", roleName, err)
		}
	}
	return nil
}
