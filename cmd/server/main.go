package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edusaas/backend/internal/auth"
	"github.com/edusaas/backend/internal/shared/config"
	"github.com/edusaas/backend/internal/shared/database"
	"github.com/edusaas/backend/internal/shared/events"
	"github.com/edusaas/backend/internal/shared/middleware"
	"github.com/edusaas/backend/internal/shared/response"
	"github.com/edusaas/backend/internal/subscription"
	"github.com/edusaas/backend/internal/tenant"
)

func main() {
	// ── Config ──
	cfg := config.Load()

	// ── Database ──
	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// ── Event Bus ──
	eventBus := events.NewBus()

	// ── Repositories ──
	authRepo := auth.NewRepository(db)
	tenantRepo := tenant.NewRepository(db)
	subRepo := subscription.NewRepository(db)

	// ── Services ──
	authService := auth.NewService(authRepo, cfg.JWT, eventBus)
	tenantService := tenant.NewService(tenantRepo, authService, authRepo, eventBus)
	subService := subscription.NewService(subRepo, eventBus)

	// ── Event Subscribers ──
	eventBus.Subscribe(events.EventTenantCreated, func(e events.Event) {
		log.Printf("[EVENT] Tenant created: %s (%s)", e.Payload["name"], e.TenantID)
	})
	eventBus.Subscribe(events.EventUserRegistered, func(e events.Event) {
		log.Printf("[EVENT] User registered: %s (tenant: %s)", e.Payload["email"], e.TenantID)
	})
	eventBus.Subscribe(events.EventSubscriptionChange, func(e events.Event) {
		log.Printf("[EVENT] Subscription %s for tenant %s", e.Payload["action"], e.TenantID)
	})

	// ── Handlers ──
	authHandler := auth.NewHandler(authService)
	tenantHandler := tenant.NewHandler(tenantService)
	subHandler := subscription.NewHandler(subService)

	// ── Router ──
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"version": "0.1.0",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// Public routes (no auth required)
	authHandler.RegisterRoutes(mux)

	// Tenant creation is public (school onboarding — no user exists yet)
	mux.HandleFunc("POST /api/v1/tenants", tenantHandler.Create)

	// Plans are public (for pricing page)
	mux.HandleFunc("GET /api/v1/plans", subHandler.ListPlans)
	mux.HandleFunc("GET /api/v1/plans/{id}", subHandler.GetPlan)

	// Protected routes — everything else requires authentication
	protectedMux := http.NewServeMux()
	tenantHandler.RegisterProtectedRoutes(protectedMux) // GET/PUT/LIST tenants
	subHandler.RegisterRoutes(protectedMux)              // subscription management

	mux.Handle("/api/v1/tenants/", middleware.Chain(
		protectedMux,
		middleware.Authenticate(authService),
	))
	mux.Handle("/api/v1/subscription", middleware.Chain(
		protectedMux,
		middleware.Authenticate(authService),
	))
	mux.Handle("/api/v1/subscription/", middleware.Chain(
		protectedMux,
		middleware.Authenticate(authService),
	))
	mux.Handle("/api/v1/admin/", middleware.Chain(
		protectedMux,
		middleware.Authenticate(authService),
	))

	// ── Global Middleware ──
	handler := middleware.Chain(
		mux,
		middleware.Logger,
		middleware.CORS(cfg.Server.AllowOrigins),
	)

	// ── Server ──
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 EduSaaS API server starting on :%s (%s)", cfg.Server.Port, cfg.Server.Environment)
		log.Printf("   Health: http://localhost:%s/api/health", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-done
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Server stopped gracefully")
}
