package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/edusaas/backend/internal/auth"
	"github.com/edusaas/backend/internal/shared/response"
)

// ──────────────────────────────────────
// Logging
// ──────────────────────────────────────

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(wrapped, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, wrapped.status, time.Since(start))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// ──────────────────────────────────────
// CORS
// ──────────────────────────────────────

func CORS(allowOrigins string) func(http.Handler) http.Handler {
	// Support comma-separated list of origins, e.g. "http://localhost:3000,http://localhost:3001"
	allowed := make(map[string]bool)
	for _, o := range strings.Split(allowOrigins, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			allowed[o] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Reflect the origin back if it is in the allowlist; otherwise use the first entry.
			if allowed[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if len(allowed) > 0 {
				for o := range allowed {
					w.Header().Set("Access-Control-Allow-Origin", o)
					break
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.Header().Set("Vary", "Origin")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ──────────────────────────────────────
// Authentication
// ──────────────────────────────────────

func Authenticate(authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				response.Error(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				response.Error(w, http.StatusUnauthorized, "invalid authorization format")
				return
			}

			claims, err := authService.ValidateAccessToken(parts[1])
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := auth.ContextWithClaims(r.Context(), claims)
			ctx = auth.ContextWithTenantID(ctx, claims.TenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ──────────────────────────────────────
// RBAC - Permission Check
// ──────────────────────────────────────

func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := auth.ClaimsFromContext(r.Context())
			if claims == nil {
				response.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			for _, p := range claims.Permissions {
				if p == permission {
					next.ServeHTTP(w, r)
					return
				}
			}

			response.Error(w, http.StatusForbidden, "insufficient permissions")
		})
	}
}

// RequireRole checks if the user has one of the specified roles
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := auth.ClaimsFromContext(r.Context())
			if claims == nil {
				response.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			for _, userRole := range claims.Roles {
				for _, required := range roles {
					if userRole == required {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			response.Error(w, http.StatusForbidden, "insufficient role")
		})
	}
}

// ──────────────────────────────────────
// Chain applies middlewares in order
// ──────────────────────────────────────

func Chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
