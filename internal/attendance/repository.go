package attendance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/edusaas/backend/internal/shared/database"
)

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// ── Sessions ──────────────────────────────────────────────────────────────────

func (r *Repository) CreateSession(ctx context.Context, s *AttendanceSession) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_tenant = '%s'", s.TenantID)); err != nil {
		return err
	}

	query := `
		INSERT INTO attendance_sessions
			(tenant_id, class_id, subject_id, teacher_id, date, qr_token, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at`
	if err := tx.QueryRow(ctx, query,
		s.TenantID, s.ClassID, s.SubjectID, s.TeacherID,
		s.Date, s.QRToken, s.ExpiresAt,
	).Scan(&s.ID, &s.CreatedAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) GetSessionByToken(ctx context.Context, token string) (*AttendanceSession, error) {
	s := &AttendanceSession{}
	query := `
		SELECT s.id, s.tenant_id, s.class_id, s.subject_id, s.teacher_id,
		       s.date, s.qr_token, s.expires_at, s.is_closed, s.created_at,
		       c.name AS class_name,
		       COALESCE(sub.name, '') AS subject_name,
		       u.full_name AS teacher_name
		FROM attendance_sessions s
		JOIN classes c ON c.id = s.class_id
		LEFT JOIN subjects sub ON sub.id = s.subject_id
		JOIN users u ON u.id = s.teacher_id
		WHERE s.qr_token = $1`
	err := r.db.Pool.QueryRow(ctx, query, token).Scan(
		&s.ID, &s.TenantID, &s.ClassID, &s.SubjectID, &s.TeacherID,
		&s.Date, &s.QRToken, &s.ExpiresAt, &s.IsClosed, &s.CreatedAt,
		&s.ClassName, &s.SubjectName, &s.TeacherName,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *Repository) GetSession(ctx context.Context, tenantID, sessionID uuid.UUID) (*AttendanceSession, error) {
	s := &AttendanceSession{}
	query := `
		SELECT s.id, s.tenant_id, s.class_id, s.subject_id, s.teacher_id,
		       s.date, s.qr_token, s.expires_at, s.is_closed, s.created_at,
		       c.name AS class_name,
		       COALESCE(sub.name, '') AS subject_name,
		       u.full_name AS teacher_name
		FROM attendance_sessions s
		JOIN classes c ON c.id = s.class_id
		LEFT JOIN subjects sub ON sub.id = s.subject_id
		JOIN users u ON u.id = s.teacher_id
		WHERE s.id = $1 AND s.tenant_id = $2`
	err := r.db.Pool.QueryRow(ctx, query, sessionID, tenantID).Scan(
		&s.ID, &s.TenantID, &s.ClassID, &s.SubjectID, &s.TeacherID,
		&s.Date, &s.QRToken, &s.ExpiresAt, &s.IsClosed, &s.CreatedAt,
		&s.ClassName, &s.SubjectName, &s.TeacherName,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *Repository) ListSessions(ctx context.Context, tenantID uuid.UUID, classID *uuid.UUID, date *time.Time) ([]*AttendanceSession, error) {
	query := `
		SELECT s.id, s.tenant_id, s.class_id, s.subject_id, s.teacher_id,
		       s.date, s.qr_token, s.expires_at, s.is_closed, s.created_at,
		       c.name AS class_name,
		       COALESCE(sub.name, '') AS subject_name,
		       u.full_name AS teacher_name
		FROM attendance_sessions s
		JOIN classes c ON c.id = s.class_id
		LEFT JOIN subjects sub ON sub.id = s.subject_id
		JOIN users u ON u.id = s.teacher_id
		WHERE s.tenant_id = $1`
	args := []any{tenantID}

	if classID != nil {
		args = append(args, *classID)
		query += fmt.Sprintf(" AND s.class_id = $%d", len(args))
	}
	if date != nil {
		args = append(args, date.Format("2006-01-02"))
		query += fmt.Sprintf(" AND s.date = $%d", len(args))
	}
	query += " ORDER BY s.created_at DESC"

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]*AttendanceSession, 0)
	for rows.Next() {
		s := &AttendanceSession{}
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.ClassID, &s.SubjectID, &s.TeacherID,
			&s.Date, &s.QRToken, &s.ExpiresAt, &s.IsClosed, &s.CreatedAt,
			&s.ClassName, &s.SubjectName, &s.TeacherName,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (r *Repository) CloseSession(ctx context.Context, tenantID, sessionID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE attendance_sessions SET is_closed = true WHERE id = $1 AND tenant_id = $2`,
		sessionID, tenantID,
	)
	return err
}

// ── Records ───────────────────────────────────────────────────────────────────

func (r *Repository) UpsertRecord(ctx context.Context, rec *AttendanceRecord) error {
	query := `
		INSERT INTO attendance_records (session_id, tenant_id, student_id, status, note)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (session_id, student_id)
		DO UPDATE SET status = EXCLUDED.status, note = EXCLUDED.note, scanned_at = NOW()
		RETURNING id, scanned_at`
	return r.db.Pool.QueryRow(ctx, query,
		rec.SessionID, rec.TenantID, rec.StudentID, rec.Status, rec.Note,
	).Scan(&rec.ID, &rec.ScannedAt)
}

func (r *Repository) ListRecords(ctx context.Context, tenantID, sessionID uuid.UUID) ([]*AttendanceRecord, error) {
	query := `
		SELECT ar.id, ar.session_id, ar.tenant_id, ar.student_id,
		       ar.status, ar.scanned_at, COALESCE(ar.note, ''),
		       u.full_name, u.email
		FROM attendance_records ar
		JOIN users u ON u.id = ar.student_id
		WHERE ar.session_id = $1 AND ar.tenant_id = $2
		ORDER BY u.full_name`

	rows, err := r.db.Pool.Query(ctx, query, sessionID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]*AttendanceRecord, 0)
	for rows.Next() {
		rec := &AttendanceRecord{}
		if err := rows.Scan(
			&rec.ID, &rec.SessionID, &rec.TenantID, &rec.StudentID,
			&rec.Status, &rec.ScannedAt, &rec.Note,
			&rec.StudentName, &rec.StudentEmail,
		); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}

func (r *Repository) GetStudentRecord(ctx context.Context, sessionID, studentID uuid.UUID) (*AttendanceRecord, error) {
	rec := &AttendanceRecord{}
	query := `
		SELECT id, session_id, tenant_id, student_id, status, scanned_at, COALESCE(note,'')
		FROM attendance_records
		WHERE session_id = $1 AND student_id = $2`
	err := r.db.Pool.QueryRow(ctx, query, sessionID, studentID).Scan(
		&rec.ID, &rec.SessionID, &rec.TenantID, &rec.StudentID,
		&rec.Status, &rec.ScannedAt, &rec.Note,
	)
	if err != nil {
		return nil, err
	}
	return rec, nil
}
