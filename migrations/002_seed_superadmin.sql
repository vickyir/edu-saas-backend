-- ============================================
-- Superadmin Seed
-- Run AFTER 001_foundation.sql
--
-- Usage:
--   psql -h localhost -U edusaas -d edusaas -f backend/migrations/002_seed_superadmin.sql
--
-- Default credentials:
--   Email:    superadmin@edusaas.id
--   Password: SuperAdmin123!
-- ============================================

DO $$
DECLARE
    v_tenant_id  UUID;
    v_user_id    UUID;
    v_role_id    UUID;
    -- bcrypt hash of "SuperAdmin123!" (cost=10)
    v_password   TEXT := '$2a$10$YQ8HrMJbpKoLPMf5VY9kGuUQbMqCw0dN6kVxJQxGHdZGz8jVVJ/Oi';
BEGIN
    -- 1. Create platform tenant
    INSERT INTO tenants (id, name, type, subdomain, config, is_active)
    VALUES (
        uuid_generate_v4(),
        'EduSaaS Platform',
        'government',
        'platform',
        '{"timezone": "Asia/Jakarta"}'::jsonb,
        true
    )
    RETURNING id INTO v_tenant_id;

    RAISE NOTICE 'Created platform tenant: %', v_tenant_id;

    -- 2. Create superadmin role
    INSERT INTO roles (id, tenant_id, name, permissions, is_system)
    VALUES (
        uuid_generate_v4(),
        v_tenant_id,
        'superadmin',
        '["superadmin","tenants:read_all","tenants:manage","tenants:create","schools:read_all","schools:manage","users:read_all","users:manage","plans:manage","subscription:manage","subscription:read","reports:read_all","reports:aggregate","classes:manage","classes:read","subjects:manage","subjects:read","attendance:configure","attendance:read","grades:manage","grades:read"]'::jsonb,
        true
    )
    RETURNING id INTO v_role_id;

    RAISE NOTICE 'Created superadmin role: %', v_role_id;

    -- 3. Also create all other default roles for this tenant
    INSERT INTO roles (id, tenant_id, name, permissions, is_system) VALUES
    (uuid_generate_v4(), v_tenant_id, 'gov_admin',
     '["tenants:read_all","tenants:manage","schools:read_all","schools:manage","reports:read_all","reports:aggregate","users:read_all","users:manage","subscription:manage","subscription:read","classes:read","attendance:read"]'::jsonb, true),
    (uuid_generate_v4(), v_tenant_id, 'school_admin',
     '["school:manage","school:read","users:manage","users:read_all","teachers:manage","teachers:read","students:manage","students:read","classes:manage","classes:read","subjects:manage","subjects:read","attendance:configure","attendance:read","subscription:manage","subscription:read","reports:read"]'::jsonb, true),
    (uuid_generate_v4(), v_tenant_id, 'teacher',
     '["school:read","classes:read","subjects:read","students:read","attendance:record","attendance:read","grades:manage","grades:read","schedule:read"]'::jsonb, true),
    (uuid_generate_v4(), v_tenant_id, 'student',
     '["profile:read","profile:update","classes:read_own","subjects:read_own","grades:read_own","attendance:submit","attendance:read_own","reports:read_own"]'::jsonb, true),
    (uuid_generate_v4(), v_tenant_id, 'parent',
     '["profile:read","children:read","attendance:read_children","grades:read_children","monitoring:read","reports:read_children"]'::jsonb, true);

    -- 4. Create superadmin user
    INSERT INTO users (id, tenant_id, email, phone, full_name, password_hash, status)
    VALUES (
        uuid_generate_v4(),
        v_tenant_id,
        'superadmin@edusaas.id',
        '',
        'Super Admin',
        v_password,
        'active'
    )
    RETURNING id INTO v_user_id;

    RAISE NOTICE 'Created superadmin user: %', v_user_id;

    -- 5. Assign superadmin role
    INSERT INTO user_roles (user_id, role_id)
    VALUES (v_user_id, v_role_id);

    RAISE NOTICE '✓ Superadmin setup complete!';
    RAISE NOTICE '  Email:    superadmin@edusaas.id';
    RAISE NOTICE '  Password: SuperAdmin123!';
    RAISE NOTICE '  CHANGE THIS PASSWORD IMMEDIATELY after first login.';
END $$;
