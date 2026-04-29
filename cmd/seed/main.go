package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/edusaas/backend/internal/shared/config"
	"github.com/edusaas/backend/internal/shared/database"
)

func main() {
	cfg := config.Load()

	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Config
	email := "superadmin@edusaas.id"
	password := "SuperAdmin123!"
	fullName := "Super Admin"

	// Allow override via args
	if len(os.Args) > 1 {
		email = os.Args[1]
	}
	if len(os.Args) > 2 {
		password = os.Args[2]
	}

	// Check if superadmin already exists
	var exists bool
	err = db.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists)
	if err != nil {
		log.Fatalf("DB query failed: %v", err)
	}
	if exists {
		fmt.Printf("User %s already exists. Skipping.\n", email)
		os.Exit(0)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Create platform tenant
	tenantID := uuid.New()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO tenants (id, name, type, subdomain, config, is_active)
		VALUES ($1, 'EduSaaS Platform', 'government', 'platform', '{"timezone":"Asia/Jakarta"}', true)
	`, tenantID)
	if err != nil {
		// Tenant might already exist
		err2 := db.Pool.QueryRow(ctx, "SELECT id FROM tenants WHERE subdomain = 'platform'").Scan(&tenantID)
		if err2 != nil {
			log.Fatalf("Failed to create or find platform tenant: %v / %v", err, err2)
		}
		fmt.Println("Platform tenant already exists, using existing one.")
	} else {
		fmt.Printf("✓ Created platform tenant: %s\n", tenantID)
	}

	// Create superadmin role
	roleID := uuid.New()
	permissions := `["superadmin","tenants:read_all","tenants:manage","tenants:create","schools:read_all","schools:manage","users:read_all","users:manage","plans:manage","subscription:manage","subscription:read","reports:read_all","reports:aggregate","classes:manage","classes:read","subjects:manage","subjects:read","attendance:configure","attendance:read","grades:manage","grades:read"]`

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO roles (id, tenant_id, name, permissions, is_system)
		VALUES ($1, $2, 'superadmin', $3::jsonb, true)
		ON CONFLICT (tenant_id, name) DO UPDATE SET permissions = $3::jsonb
	`, roleID, tenantID, permissions)
	if err != nil {
		log.Fatalf("Failed to create superadmin role: %v", err)
	}

	// Get the actual role ID (might be existing if ON CONFLICT fired)
	err = db.Pool.QueryRow(ctx, "SELECT id FROM roles WHERE tenant_id = $1 AND name = 'superadmin'", tenantID).Scan(&roleID)
	if err != nil {
		log.Fatalf("Failed to get role ID: %v", err)
	}
	fmt.Printf("✓ Superadmin role ready: %s\n", roleID)

	// Create all other default roles for this tenant
	otherRoles := map[string]string{
		"gov_admin":    `["tenants:read_all","tenants:manage","schools:read_all","schools:manage","reports:read_all","reports:aggregate","users:read_all","users:manage","subscription:manage","subscription:read","classes:read","attendance:read"]`,
		"school_admin": `["school:manage","school:read","users:manage","users:read_all","teachers:manage","teachers:read","students:manage","students:read","classes:manage","classes:read","subjects:manage","subjects:read","attendance:configure","attendance:read","subscription:manage","subscription:read","reports:read"]`,
		"teacher":      `["school:read","classes:read","subjects:read","students:read","attendance:record","attendance:read","grades:manage","grades:read","schedule:read"]`,
		"student":      `["profile:read","profile:update","classes:read_own","subjects:read_own","grades:read_own","attendance:submit","attendance:read_own","reports:read_own"]`,
		"parent":       `["profile:read","children:read","attendance:read_children","grades:read_children","monitoring:read","reports:read_children"]`,
	}
	for name, perms := range otherRoles {
		_, _ = db.Pool.Exec(ctx, `
			INSERT INTO roles (id, tenant_id, name, permissions, is_system)
			VALUES ($1, $2, $3, $4::jsonb, true)
			ON CONFLICT (tenant_id, name) DO NOTHING
		`, uuid.New(), tenantID, name, perms)
	}

	// Create superadmin user
	userID := uuid.New()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO users (id, tenant_id, email, phone, full_name, password_hash, status)
		VALUES ($1, $2, $3, '', $4, $5, 'active')
	`, userID, tenantID, email, fullName, string(hashedPassword))
	if err != nil {
		log.Fatalf("Failed to create superadmin user: %v", err)
	}
	fmt.Printf("✓ Created superadmin user: %s\n", userID)

	// Assign role
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, userID, roleID)
	if err != nil {
		log.Fatalf("Failed to assign role: %v", err)
	}

	fmt.Println("")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("  ✓ Superadmin setup complete!")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("  Email:    %s\n", email)
	fmt.Printf("  Password: %s\n", password)
	fmt.Println("  ⚠ CHANGE THIS PASSWORD after first login!")
	fmt.Println("═══════════════════════════════════════")
}
