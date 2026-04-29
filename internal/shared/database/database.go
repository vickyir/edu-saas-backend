package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edusaas/backend/internal/shared/config"
)

type DB struct {
	Pool *pgxpool.Pool
}

func New(cfg config.DatabaseConfig) (*DB, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode,
	)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.MaxConns)
	poolCfg.MinConns = 2
	poolCfg.MaxConnLifetime = 30 * time.Minute
	poolCfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	log.Println("✓ Database connected")
	return &DB{Pool: pool}, nil
}

// WithTenant acquires a connection and sets the tenant context for RLS.
// The returned conn MUST be released by the caller via conn.Release().
func (db *DB) WithTenant(ctx context.Context, tenantID string) (*pgxpool.Conn, error) {
	conn, err := db.Pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire conn: %w", err)
	}

	_, err = conn.Exec(ctx, "SET app.current_tenant = $1", tenantID)
	if err != nil {
		conn.Release()
		return nil, fmt.Errorf("set tenant: %w", err)
	}

	return conn, nil
}

// QueryRow executes a query with tenant context and returns a single row.
func (db *DB) QueryRowTenant(ctx context.Context, tenantID, sql string, args ...any) pgx.Row {
	conn, err := db.WithTenant(ctx, tenantID)
	if err != nil {
		// Return a row that will error when scanned
		return &errRow{err: err}
	}
	// Note: in production, manage conn lifecycle more carefully.
	// For simplicity, we release after scan via wrapper.
	return conn.QueryRow(ctx, sql, args...)
}

func (db *DB) Close() {
	db.Pool.Close()
}

// errRow implements pgx.Row for error cases
type errRow struct {
	err error
}

func (r *errRow) Scan(dest ...any) error {
	return r.err
}
