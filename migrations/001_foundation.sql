-- ============================================
-- EduSaaS Foundation Schema
-- Migration: 001_foundation.sql
-- ============================================

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================
-- TENANTS
-- ============================================
CREATE TABLE tenants (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(255) NOT NULL,
    type            VARCHAR(20) NOT NULL CHECK (type IN ('school', 'government')),
    parent_tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    subdomain       VARCHAR(63) UNIQUE NOT NULL,
    config          JSONB DEFAULT '{}',
    is_active       BOOLEAN DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_subdomain ON tenants(subdomain);
CREATE INDEX idx_tenants_parent ON tenants(parent_tenant_id);
CREATE INDEX idx_tenants_type ON tenants(type);

-- ============================================
-- USERS
-- ============================================
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email           VARCHAR(255) UNIQUE NOT NULL,
    phone           VARCHAR(30),
    full_name       VARCHAR(255) NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'pending')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_status ON users(tenant_id, status) WHERE deleted_at IS NULL;

-- ============================================
-- ROLES & PERMISSIONS
-- ============================================
CREATE TABLE roles (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(50) NOT NULL,
    permissions     JSONB NOT NULL DEFAULT '[]',
    is_system       BOOLEAN DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_roles_tenant ON roles(tenant_id);

CREATE TABLE user_roles (
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_role ON user_roles(role_id);

-- ============================================
-- SESSIONS
-- ============================================
CREATE TABLE sessions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    refresh_token   TEXT NOT NULL,
    user_agent      TEXT,
    ip              VARCHAR(45),
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_token ON sessions(refresh_token);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

-- ============================================
-- PLANS & SUBSCRIPTIONS
-- ============================================
CREATE TABLE plans (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(100) NOT NULL,
    slug            VARCHAR(50) UNIQUE NOT NULL,
    description     TEXT DEFAULT '',
    billing_cycle   VARCHAR(20) NOT NULL CHECK (billing_cycle IN ('monthly', 'quarterly', 'yearly')),
    price           BIGINT NOT NULL DEFAULT 0,
    currency        VARCHAR(3) NOT NULL DEFAULT 'IDR',
    feature_flags   JSONB NOT NULL DEFAULT '{}',
    status          VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    sort_order      INTEGER DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plans_slug ON plans(slug);

CREATE TABLE subscriptions (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id            UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    plan_id              UUID NOT NULL REFERENCES plans(id),
    status               VARCHAR(20) NOT NULL DEFAULT 'active'
                         CHECK (status IN ('active', 'past_due', 'canceled', 'expired', 'trialing')),
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end   TIMESTAMPTZ NOT NULL,
    auto_renew           BOOLEAN DEFAULT true,
    trial_ends_at        TIMESTAMPTZ,
    canceled_at          TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_tenant ON subscriptions(tenant_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_period_end ON subscriptions(current_period_end);

-- Invoice number sequence
CREATE SEQUENCE invoice_number_seq START 1;

CREATE TABLE invoices (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subscription_id   UUID NOT NULL REFERENCES subscriptions(id),
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    amount            BIGINT NOT NULL,
    currency          VARCHAR(3) NOT NULL DEFAULT 'IDR',
    status            VARCHAR(20) NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'paid', 'overdue', 'canceled')),
    due_date          TIMESTAMPTZ NOT NULL,
    paid_at           TIMESTAMPTZ,
    invoice_number    VARCHAR(30) UNIQUE NOT NULL,
    description       TEXT DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invoices_tenant ON invoices(tenant_id);
CREATE INDEX idx_invoices_subscription ON invoices(subscription_id);
CREATE INDEX idx_invoices_status ON invoices(status);

-- ============================================
-- ACADEMIC (Foundation tables for Month 3-4)
-- ============================================
CREATE TABLE academic_years (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    year            VARCHAR(9) NOT NULL,  -- e.g., "2025/2026"
    semester        INTEGER NOT NULL CHECK (semester IN (1, 2)),
    is_active       BOOLEAN DEFAULT false,
    start_date      DATE,
    end_date        DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, year, semester)
);

CREATE TABLE classes (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    academic_year_id UUID NOT NULL REFERENCES academic_years(id),
    name            VARCHAR(50) NOT NULL,
    level           VARCHAR(5),
    homeroom_teacher_id UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_classes_tenant ON classes(tenant_id);

CREATE TABLE subjects (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    code            VARCHAR(20),
    curriculum      VARCHAR(50),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subjects_tenant ON subjects(tenant_id);

-- ============================================
-- ROW LEVEL SECURITY
-- ============================================

-- Enable RLS on tenant-scoped tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE academic_years ENABLE ROW LEVEL SECURITY;
ALTER TABLE classes ENABLE ROW LEVEL SECURITY;
ALTER TABLE subjects ENABLE ROW LEVEL SECURITY;

-- RLS policies: isolate data by tenant_id
-- The app sets: SET app.current_tenant = '<tenant_uuid>'

CREATE POLICY tenant_isolation_users ON users
    USING (tenant_id::text = current_setting('app.current_tenant', true));

CREATE POLICY tenant_isolation_roles ON roles
    USING (tenant_id::text = current_setting('app.current_tenant', true));

CREATE POLICY tenant_isolation_sessions ON sessions
    USING (tenant_id::text = current_setting('app.current_tenant', true));

CREATE POLICY tenant_isolation_subscriptions ON subscriptions
    USING (tenant_id::text = current_setting('app.current_tenant', true));

CREATE POLICY tenant_isolation_invoices ON invoices
    USING (tenant_id::text = current_setting('app.current_tenant', true));

CREATE POLICY tenant_isolation_academic_years ON academic_years
    USING (tenant_id::text = current_setting('app.current_tenant', true));

CREATE POLICY tenant_isolation_classes ON classes
    USING (tenant_id::text = current_setting('app.current_tenant', true));

CREATE POLICY tenant_isolation_subjects ON subjects
    USING (tenant_id::text = current_setting('app.current_tenant', true));

-- ============================================
-- SEED: Default plans
-- ============================================
INSERT INTO plans (id, name, slug, description, billing_cycle, price, currency, feature_flags, sort_order) VALUES
(uuid_generate_v4(), 'Starter', 'starter-monthly', 'For small schools getting started', 'monthly', 299000, 'IDR',
 '{"max_students": 200, "max_teachers": 20, "attendance_methods": ["qr"], "realtime_monitoring": false, "ppdb_enabled": false, "e_report_cards": true, "api_access": false, "custom_branding": false}', 1),

(uuid_generate_v4(), 'Starter', 'starter-yearly', 'For small schools getting started (yearly)', 'yearly', 2990000, 'IDR',
 '{"max_students": 200, "max_teachers": 20, "attendance_methods": ["qr"], "realtime_monitoring": false, "ppdb_enabled": false, "e_report_cards": true, "api_access": false, "custom_branding": false}', 2),

(uuid_generate_v4(), 'Professional', 'professional-monthly', 'For growing schools with advanced features', 'monthly', 799000, 'IDR',
 '{"max_students": 1000, "max_teachers": 100, "attendance_methods": ["qr", "geo"], "realtime_monitoring": true, "ppdb_enabled": true, "e_report_cards": true, "api_access": false, "custom_branding": true}', 3),

(uuid_generate_v4(), 'Professional', 'professional-yearly', 'For growing schools (yearly)', 'yearly', 7990000, 'IDR',
 '{"max_students": 1000, "max_teachers": 100, "attendance_methods": ["qr", "geo"], "realtime_monitoring": true, "ppdb_enabled": true, "e_report_cards": true, "api_access": false, "custom_branding": true}', 4),

(uuid_generate_v4(), 'Enterprise', 'enterprise-monthly', 'For large institutions & government', 'monthly', 1999000, 'IDR',
 '{"max_students": -1, "max_teachers": -1, "attendance_methods": ["qr", "geo", "face"], "realtime_monitoring": true, "ppdb_enabled": true, "e_report_cards": true, "api_access": true, "custom_branding": true}', 5),

(uuid_generate_v4(), 'Enterprise', 'enterprise-yearly', 'For large institutions & government (yearly)', 'yearly', 19990000, 'IDR',
 '{"max_students": -1, "max_teachers": -1, "attendance_methods": ["qr", "geo", "face"], "realtime_monitoring": true, "ppdb_enabled": true, "e_report_cards": true, "api_access": true, "custom_branding": true}', 6);

-- ============================================
-- Helpful views
-- ============================================
CREATE VIEW tenant_user_counts AS
SELECT
    t.id AS tenant_id,
    t.name AS tenant_name,
    COUNT(DISTINCT u.id) AS total_users,
    COUNT(DISTINCT CASE WHEN r.name = 'teacher' THEN u.id END) AS total_teachers,
    COUNT(DISTINCT CASE WHEN r.name = 'student' THEN u.id END) AS total_students
FROM tenants t
LEFT JOIN users u ON u.tenant_id = t.id AND u.deleted_at IS NULL
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN roles r ON r.id = ur.role_id
GROUP BY t.id, t.name;
