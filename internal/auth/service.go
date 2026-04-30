package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/edusaas/backend/internal/shared/config"
	"github.com/edusaas/backend/internal/shared/events"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrAccountInactive    = errors.New("account is not active")
	ErrPermissionDenied   = errors.New("permission denied")
)

type Service struct {
	repo     *Repository
	cfg      config.JWTConfig
	eventBus *events.Bus
}

func NewService(repo *Repository, cfg config.JWTConfig, eventBus *events.Bus) *Service {
	return &Service{repo: repo, cfg: cfg, eventBus: eventBus}
}

// ──────────────────────────────────────
// Registration
// ──────────────────────────────────────

func (s *Service) Register(ctx context.Context, req RegisterRequest, tenantID uuid.UUID) (*TokenResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	existing, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &User{
		TenantID: tenantID,
		Email:    req.Email,
		Phone:    req.Phone,
		FullName: req.FullName,
		Password: string(hashedPassword),
		Status:   StatusActive,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	s.eventBus.Publish(events.Event{
		Type:     events.EventUserRegistered,
		TenantID: tenantID.String(),
		UserID:   user.ID.String(),
		Payload:  map[string]any{"email": user.Email, "full_name": user.FullName},
	})

	return s.generateTokens(ctx, user, "", "")
}

// ──────────────────────────────────────
// Login
// ──────────────────────────────────────

func (s *Service) Login(ctx context.Context, req LoginRequest, userAgent, ip string) (*TokenResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}
	if user.Status != StatusActive {
		return nil, ErrAccountInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	s.eventBus.Publish(events.Event{
		Type:     events.EventUserLoggedIn,
		TenantID: user.TenantID.String(),
		UserID:   user.ID.String(),
	})

	return s.generateTokens(ctx, user, userAgent, ip)
}

// ──────────────────────────────────────
// Token Management
// ──────────────────────────────────────

func (s *Service) RefreshTokens(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	session, err := s.repo.GetSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if session == nil {
		return nil, ErrInvalidToken
	}

	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}

	// Delete old session
	_ = s.repo.DeleteSession(ctx, session.ID)

	return s.generateTokens(ctx, user, "", "")
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	session, err := s.repo.GetSessionByRefreshToken(ctx, refreshToken)
	if err != nil || session == nil {
		return nil
	}
	return s.repo.DeleteSession(ctx, session.ID)
}

func (s *Service) generateTokens(ctx context.Context, user *User, userAgent, ip string) (*TokenResponse, error) {
	// Get user roles
	roles, err := s.repo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get roles: %w", err)
	}

	roleNames := make([]string, len(roles))
	var allPerms []string
	for i, r := range roles {
		roleNames[i] = r.Name
		allPerms = append(allPerms, r.Permissions...)
	}

	// Access token
	accessClaims := jwt.MapClaims{
		"sub":         user.ID.String(),
		"tenant_id":   user.TenantID.String(),
		"email":       user.Email,
		"full_name":   user.FullName,
		"roles":       roleNames,
		"permissions": allPerms,
		"iat":         time.Now().Unix(),
		"exp":         time.Now().Add(s.cfg.AccessExpiresIn).Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString([]byte(s.cfg.AccessSecret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	// Refresh token
	refreshClaims := jwt.MapClaims{
		"sub":       user.ID.String(),
		"tenant_id": user.TenantID.String(),
		"type":      "refresh",
		"iat":       time.Now().Unix(),
		"exp":       time.Now().Add(s.cfg.RefreshExpiresIn).Unix(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString([]byte(s.cfg.RefreshSecret))
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	// Store session
	session := &Session{
		UserID:       user.ID,
		TenantID:     user.TenantID,
		RefreshToken: refreshStr,
		UserAgent:    userAgent,
		IP:           ip,
		ExpiresAt:    time.Now().Add(s.cfg.RefreshExpiresIn),
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Clear password before returning
	user.Password = ""

	return &TokenResponse{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresIn:    int(s.cfg.AccessExpiresIn.Seconds()),
		User:         user,
	}, nil
}

// ──────────────────────────────────────
// Token Validation (used by middleware)
// ──────────────────────────────────────

type Claims struct {
	UserID      uuid.UUID
	TenantID    uuid.UUID
	Email       string
	FullName    string
	Roles       []string
	Permissions []string
}

func (s *Service) ValidateAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.AccessSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	userID, _ := uuid.Parse(mapClaims["sub"].(string))
	tenantID, _ := uuid.Parse(mapClaims["tenant_id"].(string))

	var roles []string
	if r, ok := mapClaims["roles"].([]any); ok {
		for _, v := range r {
			roles = append(roles, v.(string))
		}
	}

	var perms []string
	if p, ok := mapClaims["permissions"].([]any); ok {
		for _, v := range p {
			perms = append(perms, v.(string))
		}
	}

	return &Claims{
		UserID:      userID,
		TenantID:    tenantID,
		Email:       mapClaims["email"].(string),
		FullName:    mapClaims["full_name"].(string),
		Roles:       roles,
		Permissions: perms,
	}, nil
}

// ──────────────────────────────────────
// Profile Management
// ──────────────────────────────────────

func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	user.Password = ""
	return user, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}

	user.FullName = req.FullName
	user.Phone = req.Phone

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	user.Password = ""
	return user, nil
}

func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		return ErrInvalidCredentials
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	return s.repo.UpdatePassword(ctx, userID, string(hashed))
}

// ──────────────────────────────────────
// Admin User Management
// ──────────────────────────────────────

// allowedRoles returns the set of roles a creator (by their own role) may assign.
var allowedRoles = map[string][]string{
	RoleSuperAdmin:  {RoleSuperAdmin, RoleGovAdmin, RoleSchoolAdmin, RoleTeacher, RoleStudent, RoleParent, "staff"},
	RoleGovAdmin:    {RoleGovAdmin, RoleSchoolAdmin, RoleTeacher, RoleStudent, RoleParent, "staff"},
	RoleSchoolAdmin: {RoleTeacher, RoleStudent, RoleParent, "staff"},
}

func canCreateWithRole(creatorRoles []string, targetRole string) bool {
	for _, cr := range creatorRoles {
		if allowed, ok := allowedRoles[cr]; ok {
			for _, a := range allowed {
				if a == targetRole {
					return true
				}
			}
		}
	}
	return false
}

func (s *Service) ListUsers(ctx context.Context, tenantID uuid.UUID, roleFilter string, page, perPage int) ([]*UserWithRoles, int64, error) {
	return s.repo.ListUsersWithRoles(ctx, tenantID, roleFilter, page, perPage)
}

func (s *Service) GetUser(ctx context.Context, tenantID, userID uuid.UUID) (*UserWithRoles, error) {
	u, err := s.repo.GetUserWithRoles(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (s *Service) CreateManagedUser(ctx context.Context, tenantID uuid.UUID, creatorRoles []string, req CreateManagedUserRequest) (*UserWithRoles, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Email == "" || req.FullName == "" || req.Password == "" || req.Role == "" {
		return nil, fmt.Errorf("email, full_name, password, and role are required")
	}
	if len(req.Password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	if !canCreateWithRole(creatorRoles, req.Role) {
		return nil, ErrPermissionDenied
	}

	existing, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &User{
		TenantID: tenantID,
		Email:    req.Email,
		Phone:    req.Phone,
		FullName: req.FullName,
		Password: string(hashed),
		Status:   StatusActive,
	}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	if err := s.AssignRoleToUser(ctx, user.ID, tenantID, req.Role); err != nil {
		return nil, fmt.Errorf("assign role: %w", err)
	}

	s.eventBus.Publish(events.Event{
		Type:     events.EventUserRegistered,
		TenantID: tenantID.String(),
		UserID:   user.ID.String(),
		Payload:  map[string]any{"email": user.Email, "full_name": user.FullName, "role": req.Role},
	})

	return s.repo.GetUserWithRoles(ctx, tenantID, user.ID)
}

func (s *Service) UpdateManagedUser(ctx context.Context, tenantID, userID uuid.UUID, req UpdateManagedUserRequest) (*UserWithRoles, error) {
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || u == nil {
		return nil, ErrUserNotFound
	}
	if u.TenantID != tenantID {
		return nil, ErrPermissionDenied
	}
	if req.FullName != "" {
		u.FullName = req.FullName
	}
	u.Phone = req.Phone
	if req.Status != "" {
		u.Status = req.Status
	}
	if err := s.repo.UpdateUserAdmin(ctx, u); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return s.repo.GetUserWithRoles(ctx, tenantID, userID)
}

func (s *Service) ChangeUserRole(ctx context.Context, tenantID, userID uuid.UUID, newRole string, creatorRoles []string) error {
	if !canCreateWithRole(creatorRoles, newRole) {
		return ErrPermissionDenied
	}
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || u == nil {
		return ErrUserNotFound
	}
	if u.TenantID != tenantID {
		return ErrPermissionDenied
	}
	if err := s.repo.RemoveAllRoles(ctx, userID); err != nil {
		return fmt.Errorf("remove roles: %w", err)
	}
	return s.AssignRoleToUser(ctx, userID, tenantID, newRole)
}

func (s *Service) DeleteManagedUser(ctx context.Context, tenantID, userID uuid.UUID) error {
	return s.repo.SoftDeleteUser(ctx, tenantID, userID)
}

// ──────────────────────────────────────
// RBAC Helpers
// ──────────────────────────────────────

func (s *Service) AssignRoleToUser(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID, roleName string) error {
	role, err := s.repo.GetRoleByName(ctx, tenantID, roleName)
	if err != nil {
		return fmt.Errorf("get role: %w", err)
	}
	if role == nil {
		return fmt.Errorf("role %s not found", roleName)
	}
	return s.repo.AssignRole(ctx, userID, role.ID)
}

func (s *Service) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*Role, error) {
	return s.repo.GetUserRoles(ctx, userID)
}
