-- ============================================
-- EduSaaS Academic Phase 2
-- Migration: 003_academic_phase2.sql
-- Month 3-4: Classes, Students, Attendance, Grades
-- ============================================

-- ============================================
-- STUDENT PROFILES
-- ============================================
CREATE TABLE student_profiles (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    nis             VARCHAR(20),
    birth_date      DATE,
    gender          VARCHAR(10) CHECK (gender IN ('male', 'female')),
    address         TEXT,
    parent_name     VARCHAR(255),
    parent_phone    VARCHAR(30),
    parent_user_id  UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_student_profiles_tenant ON student_profiles(tenant_id);
CREATE INDEX idx_student_profiles_user   ON student_profiles(user_id);

-- ============================================
-- CLASS ENROLLMENTS (many-to-many: class ↔ student)
-- ============================================
CREATE TABLE class_students (
    class_id     UUID NOT NULL REFERENCES classes(id)  ON DELETE CASCADE,
    student_id   UUID NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    tenant_id    UUID NOT NULL REFERENCES tenants(id)  ON DELETE CASCADE,
    enrolled_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (class_id, student_id)
);

CREATE INDEX idx_class_students_class  ON class_students(class_id);
CREATE INDEX idx_class_students_student ON class_students(student_id);
CREATE INDEX idx_class_students_tenant ON class_students(tenant_id);

-- ============================================
-- CLASS-SUBJECT ASSIGNMENTS (with teacher)
-- ============================================
CREATE TABLE class_subjects (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    class_id    UUID NOT NULL REFERENCES classes(id)   ON DELETE CASCADE,
    subject_id  UUID NOT NULL REFERENCES subjects(id)  ON DELETE CASCADE,
    teacher_id  UUID          REFERENCES users(id)     ON DELETE SET NULL,
    tenant_id   UUID NOT NULL REFERENCES tenants(id)   ON DELETE CASCADE,
    UNIQUE (class_id, subject_id)
);

CREATE INDEX idx_class_subjects_class  ON class_subjects(class_id);
CREATE INDEX idx_class_subjects_tenant ON class_subjects(tenant_id);

-- ============================================
-- ATTENDANCE SESSIONS (teacher opens a QR session)
-- ============================================
CREATE TABLE attendance_sessions (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id)  ON DELETE CASCADE,
    class_id    UUID NOT NULL REFERENCES classes(id)  ON DELETE CASCADE,
    subject_id  UUID          REFERENCES subjects(id) ON DELETE SET NULL,
    teacher_id  UUID NOT NULL REFERENCES users(id),
    date        DATE NOT NULL DEFAULT CURRENT_DATE,
    qr_token    VARCHAR(64)  NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ  NOT NULL,
    is_closed   BOOLEAN      NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_att_sessions_tenant ON attendance_sessions(tenant_id);
CREATE INDEX idx_att_sessions_class  ON attendance_sessions(class_id);
CREATE INDEX idx_att_sessions_token  ON attendance_sessions(qr_token);
CREATE INDEX idx_att_sessions_date   ON attendance_sessions(date);

-- ============================================
-- ATTENDANCE RECORDS (one per student per session)
-- ============================================
CREATE TABLE attendance_records (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id  UUID NOT NULL REFERENCES attendance_sessions(id) ON DELETE CASCADE,
    tenant_id   UUID NOT NULL REFERENCES tenants(id)             ON DELETE CASCADE,
    student_id  UUID NOT NULL REFERENCES users(id)               ON DELETE CASCADE,
    status      VARCHAR(12)  NOT NULL DEFAULT 'present'
                CHECK (status IN ('present', 'absent', 'late', 'sick', 'permission')),
    scanned_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    note        TEXT,
    UNIQUE (session_id, student_id)
);

CREATE INDEX idx_att_records_session ON attendance_records(session_id);
CREATE INDEX idx_att_records_student ON attendance_records(student_id);
CREATE INDEX idx_att_records_tenant  ON attendance_records(tenant_id);

-- ============================================
-- GRADES
-- ============================================
CREATE TABLE grades (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id)        ON DELETE CASCADE,
    student_id       UUID NOT NULL REFERENCES users(id)          ON DELETE CASCADE,
    class_id         UUID NOT NULL REFERENCES classes(id)        ON DELETE CASCADE,
    subject_id       UUID NOT NULL REFERENCES subjects(id)       ON DELETE CASCADE,
    academic_year_id UUID NOT NULL REFERENCES academic_years(id),
    daily_score      NUMERIC(5,2),
    midterm_score    NUMERIC(5,2),
    final_score      NUMERIC(5,2),
    grade_letter     VARCHAR(2),
    notes            TEXT,
    recorded_by      UUID REFERENCES users(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, subject_id, academic_year_id)
);

CREATE INDEX idx_grades_tenant  ON grades(tenant_id);
CREATE INDEX idx_grades_student ON grades(student_id);
CREATE INDEX idx_grades_class   ON grades(class_id);
CREATE INDEX idx_grades_subject ON grades(subject_id);
CREATE INDEX idx_grades_year    ON grades(academic_year_id);

-- ============================================
-- ROW LEVEL SECURITY
-- ============================================
ALTER TABLE student_profiles    ENABLE ROW LEVEL SECURITY;
ALTER TABLE class_students      ENABLE ROW LEVEL SECURITY;
ALTER TABLE class_subjects      ENABLE ROW LEVEL SECURITY;
ALTER TABLE attendance_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE attendance_records  ENABLE ROW LEVEL SECURITY;
ALTER TABLE grades              ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_iso_student_profiles ON student_profiles
    USING (tenant_id::text = current_setting('app.current_tenant', true));

CREATE POLICY tenant_iso_class_students ON class_students
    USING (tenant_id::text = current_setting('app.current_tenant', true));

CREATE POLICY tenant_iso_class_subjects ON class_subjects
    USING (tenant_id::text = current_setting('app.current_tenant', true));

CREATE POLICY tenant_iso_att_sessions ON attendance_sessions
    USING (tenant_id::text = current_setting('app.current_tenant', true));

CREATE POLICY tenant_iso_att_records ON attendance_records
    USING (tenant_id::text = current_setting('app.current_tenant', true));

CREATE POLICY tenant_iso_grades ON grades
    USING (tenant_id::text = current_setting('app.current_tenant', true));
