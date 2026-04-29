package auth

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	claimsKey  contextKey = "auth_claims"
	tenantKey  contextKey = "tenant_id"
)

// ContextWithClaims stores auth claims in context
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ClaimsFromContext retrieves auth claims from context
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsKey).(*Claims)
	return claims
}

// ContextWithTenantID stores tenant ID in context
func ContextWithTenantID(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}

// TenantIDFromContext retrieves tenant ID from context
func TenantIDFromContext(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(tenantKey).(uuid.UUID)
	return id
}
