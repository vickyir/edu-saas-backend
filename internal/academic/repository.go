package academic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/edusaas/backend/internal/shared/database"
)

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// ── Academic Years ─────────────────────────────────────────────────────────────

func (r *Repository) ListAcademicYears(ctx context.Context, tenantID uuid.UUID) ([]*AcademicYear, error) {
	conn, err := r.db.WithTenant(ctx, tenantID.String())
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	rows, err := conn.Query(ctx, `
		SELECT id, tenant_id, year, semester, is_active, start_date, end_date, created_at
		FROM academic_years WHERE tenant_id = $1 ORDER BY year DESC, semester ASC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*AcademicYear, 0)
	for rows.Next() {
		y := &AcademicYear{}
		if err := rows.Scan(&y.ID, &y.TenantID, &y.Year, &y.Semester, &y.IsActive, &y.StartDate, &y.EndDate, &y.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, y)
	}
	return list, nil
}

func (r *Repository) CreateAcademicYear(ctx context.Context, y *AcademicYear) error {
	y.ID = uuid.New()
	y.CreatedAt = time.Now()
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO academic_years (id, tenant_id, year, semester, is_active, start_date, end_date, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, y.ID, y.TenantID, y.Year, y.Semester, y.IsActive, y.StartDate, y.EndDate, y.CreatedAt)
	return err
}

func (r *Repository) ActivateAcademicYear(ctx context.Context, tenantID, yearID uuid.UUID) error {
	// Deactivate all, then activate the selected one
	_, err := r.db.Pool.Exec(ctx, `UPDATE academic_years SET is_active = false WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return err
	}
	_, err = r.db.Pool.Exec(ctx, `UPDATE academic_years SET is_active = true WHERE id = $1 AND tenant_id = $2`, yearID, tenantID)
	return err
}

func (r *Repository) GetAcademicYear(ctx context.Context, tenantID, yearID uuid.UUID) (*AcademicYear, error) {
	y := &AcademicYear{}
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, tenant_id, year, semester, is_active, start_date, end_date, created_at
		FROM academic_years WHERE id = $1 AND tenant_id = $2
	`, yearID, tenantID).Scan(&y.ID, &y.TenantID, &y.Year, &y.Semester, &y.IsActive, &y.StartDate, &y.EndDate, &y.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return y, err
}

func (r *Repository) UpdateAcademicYear(ctx context.Context, y *AcademicYear) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE academic_years SET year = $1, semester = $2, start_date = $3, end_date = $4
		WHERE id = $5 AND tenant_id = $6
	`, y.Year, y.Semester, y.StartDate, y.EndDate, y.ID, y.TenantID)
	return err
}

func (r *Repository) DeleteAcademicYear(ctx context.Context, tenantID, yearID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx,
		`DELETE FROM academic_years WHERE id = $1 AND tenant_id = $2`, yearID, tenantID)
	return err
}

func (r *Repository) GetActiveAcademicYear(ctx context.Context, tenantID uuid.UUID) (*AcademicYear, error) {
	y := &AcademicYear{}
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, tenant_id, year, semester, is_active, start_date, end_date, created_at
		FROM academic_years WHERE tenant_id = $1 AND is_active = true LIMIT 1
	`, tenantID).Scan(&y.ID, &y.TenantID, &y.Year, &y.Semester, &y.IsActive, &y.StartDate, &y.EndDate, &y.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return y, err
}

// ── Classes ────────────────────────────────────────────────────────────────────

func (r *Repository) ListClasses(ctx context.Context, tenantID uuid.UUID, academicYearID *uuid.UUID) ([]*Class, error) {
	conn, err := r.db.WithTenant(ctx, tenantID.String())
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	query := `
		SELECT c.id, c.tenant_id, c.academic_year_id, c.name, c.level, c.homeroom_teacher_id, c.created_at,
		       ay.year, ay.semester,
		       u.full_name, u.email,
		       COUNT(cs.student_id) AS student_count
		FROM classes c
		JOIN academic_years ay ON ay.id = c.academic_year_id
		LEFT JOIN users u ON u.id = c.homeroom_teacher_id
		LEFT JOIN class_students cs ON cs.class_id = c.id
		WHERE c.tenant_id = $1
	`
	args := []any{tenantID}
	if academicYearID != nil {
		query += ` AND c.academic_year_id = $2`
		args = append(args, *academicYearID)
	}
	query += ` GROUP BY c.id, ay.year, ay.semester, u.full_name, u.email ORDER BY c.name`

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*Class, 0)
	for rows.Next() {
		c := &Class{AcademicYear: &AcademicYear{}}
		var teacherName, teacherEmail *string
		var teacherID *uuid.UUID
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.AcademicYearID, &c.Name, &c.Level, &teacherID, &c.CreatedAt,
			&c.AcademicYear.Year, &c.AcademicYear.Semester,
			&teacherName, &teacherEmail, &c.StudentCount,
		); err != nil {
			return nil, err
		}
		if teacherID != nil && teacherName != nil {
			c.HomeroomTeacherID = teacherID
			c.HomeroomTeacher = &TeacherRef{ID: *teacherID, FullName: *teacherName, Email: *teacherEmail}
		}
		list = append(list, c)
	}
	return list, nil
}

func (r *Repository) GetClass(ctx context.Context, tenantID, classID uuid.UUID) (*Class, error) {
	c := &Class{AcademicYear: &AcademicYear{}}
	var teacherName, teacherEmail *string
	var teacherID *uuid.UUID
	err := r.db.Pool.QueryRow(ctx, `
		SELECT c.id, c.tenant_id, c.academic_year_id, c.name, c.level, c.homeroom_teacher_id, c.created_at,
		       ay.year, ay.semester,
		       u.full_name, u.email,
		       COUNT(cs.student_id) AS student_count
		FROM classes c
		JOIN academic_years ay ON ay.id = c.academic_year_id
		LEFT JOIN users u ON u.id = c.homeroom_teacher_id
		LEFT JOIN class_students cs ON cs.class_id = c.id
		WHERE c.id = $1 AND c.tenant_id = $2
		GROUP BY c.id, ay.year, ay.semester, u.full_name, u.email
	`, classID, tenantID).Scan(
		&c.ID, &c.TenantID, &c.AcademicYearID, &c.Name, &c.Level, &teacherID, &c.CreatedAt,
		&c.AcademicYear.Year, &c.AcademicYear.Semester,
		&teacherName, &teacherEmail, &c.StudentCount,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if teacherID != nil && teacherName != nil {
		c.HomeroomTeacherID = teacherID
		c.HomeroomTeacher = &TeacherRef{ID: *teacherID, FullName: *teacherName, Email: *teacherEmail}
	}
	return c, nil
}

func (r *Repository) CreateClass(ctx context.Context, c *Class) error {
	c.ID = uuid.New()
	c.CreatedAt = time.Now()
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO classes (id, tenant_id, academic_year_id, name, level, homeroom_teacher_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, c.ID, c.TenantID, c.AcademicYearID, c.Name, c.Level, c.HomeroomTeacherID, c.CreatedAt)
	return err
}

func (r *Repository) UpdateClass(ctx context.Context, c *Class) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE classes SET name=$1, level=$2, homeroom_teacher_id=$3
		WHERE id=$4 AND tenant_id=$5
	`, c.Name, c.Level, c.HomeroomTeacherID, c.ID, c.TenantID)
	return err
}

func (r *Repository) DeleteClass(ctx context.Context, tenantID, classID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM classes WHERE id=$1 AND tenant_id=$2`, classID, tenantID)
	return err
}

// ── Class Enrollments ──────────────────────────────────────────────────────────

func (r *Repository) ListClassStudents(ctx context.Context, tenantID, classID uuid.UUID) ([]*ClassStudent, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT cs.class_id, cs.student_id, cs.tenant_id, cs.enrolled_at,
		       u.full_name, u.email, u.phone, u.status,
		       sp.nis, sp.gender, sp.parent_name, sp.parent_phone
		FROM class_students cs
		JOIN users u ON u.id = cs.student_id
		LEFT JOIN student_profiles sp ON sp.user_id = cs.student_id
		WHERE cs.class_id = $1 AND cs.tenant_id = $2
		ORDER BY u.full_name
	`, classID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*ClassStudent, 0)
	for rows.Next() {
		cs := &ClassStudent{Student: &Student{Profile: &StudentProfile{}}}
		if err := rows.Scan(
			&cs.ClassID, &cs.StudentID, &cs.TenantID, &cs.EnrolledAt,
			&cs.Student.FullName, &cs.Student.Email, &cs.Student.Phone, &cs.Student.Status,
			&cs.Student.Profile.NIS, &cs.Student.Profile.Gender,
			&cs.Student.Profile.ParentName, &cs.Student.Profile.ParentPhone,
		); err != nil {
			return nil, err
		}
		cs.Student.ID = cs.StudentID
		cs.Student.TenantID = cs.TenantID
		list = append(list, cs)
	}
	return list, nil
}

func (r *Repository) EnrollStudent(ctx context.Context, tenantID, classID, studentID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO class_students (class_id, student_id, tenant_id)
		VALUES ($1, $2, $3) ON CONFLICT DO NOTHING
	`, classID, studentID, tenantID)
	return err
}

func (r *Repository) UnenrollStudent(ctx context.Context, tenantID, classID, studentID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `
		DELETE FROM class_students WHERE class_id=$1 AND student_id=$2 AND tenant_id=$3
	`, classID, studentID, tenantID)
	return err
}

// ── Class Subjects ─────────────────────────────────────────────────────────────

func (r *Repository) ListClassSubjects(ctx context.Context, tenantID, classID uuid.UUID) ([]*ClassSubject, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT cs.id, cs.class_id, cs.subject_id, cs.teacher_id, cs.tenant_id,
		       s.name, s.code, s.curriculum,
		       u.full_name, u.email
		FROM class_subjects cs
		JOIN subjects s ON s.id = cs.subject_id
		LEFT JOIN users u ON u.id = cs.teacher_id
		WHERE cs.class_id = $1 AND cs.tenant_id = $2
		ORDER BY s.name
	`, classID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*ClassSubject, 0)
	for rows.Next() {
		cs := &ClassSubject{Subject: &Subject{}}
		var teacherName, teacherEmail *string
		var teacherID *uuid.UUID
		if err := rows.Scan(
			&cs.ID, &cs.ClassID, &cs.SubjectID, &teacherID, &cs.TenantID,
			&cs.Subject.Name, &cs.Subject.Code, &cs.Subject.Curriculum,
			&teacherName, &teacherEmail,
		); err != nil {
			return nil, err
		}
		cs.Subject.ID = cs.SubjectID
		if teacherID != nil && teacherName != nil {
			cs.TeacherID = teacherID
			cs.Teacher = &TeacherRef{ID: *teacherID, FullName: *teacherName, Email: *teacherEmail}
		}
		list = append(list, cs)
	}
	return list, nil
}

func (r *Repository) AssignSubjectToClass(ctx context.Context, cs *ClassSubject) error {
	cs.ID = uuid.New()
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO class_subjects (id, class_id, subject_id, teacher_id, tenant_id)
		VALUES ($1,$2,$3,$4,$5) ON CONFLICT (class_id, subject_id) DO UPDATE SET teacher_id=$4
	`, cs.ID, cs.ClassID, cs.SubjectID, cs.TeacherID, cs.TenantID)
	return err
}

func (r *Repository) RemoveSubjectFromClass(ctx context.Context, tenantID, classID, subjectID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `
		DELETE FROM class_subjects WHERE class_id=$1 AND subject_id=$2 AND tenant_id=$3
	`, classID, subjectID, tenantID)
	return err
}

// ── Subjects ───────────────────────────────────────────────────────────────────

func (r *Repository) ListSubjects(ctx context.Context, tenantID uuid.UUID) ([]*Subject, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, tenant_id, name, code, curriculum, created_at
		FROM subjects WHERE tenant_id = $1 ORDER BY name
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*Subject, 0)
	for rows.Next() {
		s := &Subject{}
		if err := rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.Code, &s.Curriculum, &s.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, nil
}

func (r *Repository) CreateSubject(ctx context.Context, s *Subject) error {
	s.ID = uuid.New()
	s.CreatedAt = time.Now()
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO subjects (id, tenant_id, name, code, curriculum, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, s.ID, s.TenantID, s.Name, s.Code, s.Curriculum, s.CreatedAt)
	return err
}

func (r *Repository) UpdateSubject(ctx context.Context, s *Subject) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE subjects SET name=$1, code=$2, curriculum=$3 WHERE id=$4 AND tenant_id=$5
	`, s.Name, s.Code, s.Curriculum, s.ID, s.TenantID)
	return err
}

func (r *Repository) DeleteSubject(ctx context.Context, tenantID, subjectID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM subjects WHERE id=$1 AND tenant_id=$2`, subjectID, tenantID)
	return err
}

// ── Students ───────────────────────────────────────────────────────────────────

func (r *Repository) ListStudents(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]*Student, int64, error) {
	var total int64
	err := r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		JOIN roles ro ON ro.id = ur.role_id AND ro.name = 'student'
		WHERE u.tenant_id = $1 AND u.deleted_at IS NULL
	`, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Pool.Query(ctx, `
		SELECT u.id, u.tenant_id, u.email, u.full_name, u.phone, u.status, u.created_at,
		       sp.id, sp.nis, sp.birth_date, sp.gender, sp.address, sp.parent_name, sp.parent_phone
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		JOIN roles ro ON ro.id = ur.role_id AND ro.name = 'student'
		LEFT JOIN student_profiles sp ON sp.user_id = u.id
		WHERE u.tenant_id = $1 AND u.deleted_at IS NULL
		ORDER BY u.full_name
		LIMIT $2 OFFSET $3
	`, tenantID, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]*Student, 0)
	for rows.Next() {
		s := &Student{Profile: &StudentProfile{}}
		var profileID *uuid.UUID
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.Email, &s.FullName, &s.Phone, &s.Status, &s.CreatedAt,
			&profileID, &s.Profile.NIS, &s.Profile.BirthDate, &s.Profile.Gender,
			&s.Profile.Address, &s.Profile.ParentName, &s.Profile.ParentPhone,
		); err != nil {
			return nil, 0, err
		}
		if profileID != nil {
			s.Profile.ID = *profileID
			s.Profile.UserID = s.ID
			s.Profile.TenantID = s.TenantID
		} else {
			s.Profile = nil
		}
		list = append(list, s)
	}
	return list, total, nil
}

func (r *Repository) GetStudent(ctx context.Context, tenantID, userID uuid.UUID) (*Student, error) {
	s := &Student{Profile: &StudentProfile{}}
	var profileID *uuid.UUID
	err := r.db.Pool.QueryRow(ctx, `
		SELECT u.id, u.tenant_id, u.email, u.full_name, u.phone, u.status, u.created_at,
		       sp.id, sp.nis, sp.birth_date, sp.gender, sp.address, sp.parent_name, sp.parent_phone
		FROM users u
		LEFT JOIN student_profiles sp ON sp.user_id = u.id
		WHERE u.id = $1 AND u.tenant_id = $2 AND u.deleted_at IS NULL
	`, userID, tenantID).Scan(
		&s.ID, &s.TenantID, &s.Email, &s.FullName, &s.Phone, &s.Status, &s.CreatedAt,
		&profileID, &s.Profile.NIS, &s.Profile.BirthDate, &s.Profile.Gender,
		&s.Profile.Address, &s.Profile.ParentName, &s.Profile.ParentPhone,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if profileID != nil {
		s.Profile.ID = *profileID
		s.Profile.UserID = s.ID
		s.Profile.TenantID = s.TenantID
	} else {
		s.Profile = nil
	}
	return s, nil
}

// CreateStudent creates a user + student_profile + assigns the student role.
func (r *Repository) CreateStudent(ctx context.Context, tenantID uuid.UUID, req CreateStudentRequest) (*Student, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	userID := uuid.New()
	now := time.Now()
	_, err = r.db.Pool.Exec(ctx, `
		INSERT INTO users (id, tenant_id, email, phone, full_name, password_hash, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,'active',$7,$7)
	`, userID, tenantID, req.Email, req.Phone, req.FullName, string(hashed), now)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// Assign student role
	var roleID uuid.UUID
	err = r.db.Pool.QueryRow(ctx, `SELECT id FROM roles WHERE tenant_id=$1 AND name='student'`, tenantID).Scan(&roleID)
	if err == nil {
		_, _ = r.db.Pool.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, userID, roleID)
	}

	// Create profile
	profileID := uuid.New()
	_, _ = r.db.Pool.Exec(ctx, `
		INSERT INTO student_profiles (id, user_id, tenant_id, nis, birth_date, gender, address, parent_name, parent_phone, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10)
	`, profileID, userID, tenantID, req.NIS, req.BirthDate, req.Gender, req.Address, req.ParentName, req.ParentPhone, now)

	return &Student{
		ID: userID, TenantID: tenantID, Email: req.Email,
		FullName: req.FullName, Phone: req.Phone, Status: "active", CreatedAt: now,
		Profile: &StudentProfile{
			ID: profileID, UserID: userID, TenantID: tenantID,
			NIS: req.NIS, Gender: req.Gender, Address: req.Address,
			ParentName: req.ParentName, ParentPhone: req.ParentPhone,
		},
	}, nil
}

// GetRoleByName retrieves a role from the DB, used to validate role names.
func (r *Repository) GetRoleByName(ctx context.Context, tenantID uuid.UUID, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.Pool.QueryRow(ctx, `SELECT id FROM roles WHERE tenant_id=$1 AND name=$2`, tenantID, name).Scan(&id)
	return id, err
}

// ListTeachers returns all users with the teacher role for a tenant.
func (r *Repository) ListTeachers(ctx context.Context, tenantID uuid.UUID) ([]*TeacherRef, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT u.id, u.full_name, u.email
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		JOIN roles ro ON ro.id = ur.role_id AND ro.name = 'teacher'
		WHERE u.tenant_id = $1 AND u.deleted_at IS NULL
		ORDER BY u.full_name
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*TeacherRef, 0)
	for rows.Next() {
		t := &TeacherRef{}
		if err := rows.Scan(&t.ID, &t.FullName, &t.Email); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, nil
}

// unused helper kept for compile: suppress unused import warning
var _ = json.Marshal
