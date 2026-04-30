package grade

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/edusaas/backend/internal/shared/database"
)

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpsertGrade(ctx context.Context, g *Grade) error {
	query := `
		INSERT INTO grades
			(tenant_id, student_id, class_id, subject_id, academic_year_id,
			 daily_score, midterm_score, final_score, grade_letter, notes, recorded_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (student_id, subject_id, academic_year_id)
		DO UPDATE SET
			daily_score   = EXCLUDED.daily_score,
			midterm_score = EXCLUDED.midterm_score,
			final_score   = EXCLUDED.final_score,
			grade_letter  = EXCLUDED.grade_letter,
			notes         = EXCLUDED.notes,
			recorded_by   = EXCLUDED.recorded_by,
			updated_at    = NOW()
		RETURNING id, created_at, updated_at`

	return r.db.Pool.QueryRow(ctx, query,
		g.TenantID, g.StudentID, g.ClassID, g.SubjectID, g.AcademicYearID,
		g.DailyScore, g.MidtermScore, g.FinalScore, g.GradeLetter, g.Notes, g.RecordedBy,
	).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
}

func (r *Repository) ListGrades(ctx context.Context, tenantID uuid.UUID, f GradeFilter) ([]*Grade, error) {
	query := `
		SELECT g.id, g.tenant_id, g.student_id, g.class_id, g.subject_id, g.academic_year_id,
		       g.daily_score, g.midterm_score, g.final_score,
		       COALESCE(g.grade_letter,''), COALESCE(g.notes,''),
		       g.recorded_by, g.created_at, g.updated_at,
		       u.full_name, sub.name, sub.code, c.name
		FROM grades g
		JOIN users u ON u.id = g.student_id
		JOIN subjects sub ON sub.id = g.subject_id
		JOIN classes c ON c.id = g.class_id
		WHERE g.tenant_id = $1`

	args := []any{tenantID}
	conditions := []string{}

	if f.ClassID != nil {
		args = append(args, *f.ClassID)
		conditions = append(conditions, fmt.Sprintf("g.class_id = $%d", len(args)))
	}
	if f.SubjectID != nil {
		args = append(args, *f.SubjectID)
		conditions = append(conditions, fmt.Sprintf("g.subject_id = $%d", len(args)))
	}
	if f.StudentID != nil {
		args = append(args, *f.StudentID)
		conditions = append(conditions, fmt.Sprintf("g.student_id = $%d", len(args)))
	}
	if f.AcademicYearID != nil {
		args = append(args, *f.AcademicYearID)
		conditions = append(conditions, fmt.Sprintf("g.academic_year_id = $%d", len(args)))
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY u.full_name, sub.name"

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grades := make([]*Grade, 0)
	for rows.Next() {
		g := &Grade{}
		if err := rows.Scan(
			&g.ID, &g.TenantID, &g.StudentID, &g.ClassID, &g.SubjectID, &g.AcademicYearID,
			&g.DailyScore, &g.MidtermScore, &g.FinalScore,
			&g.GradeLetter, &g.Notes, &g.RecordedBy,
			&g.CreatedAt, &g.UpdatedAt,
			&g.StudentName, &g.SubjectName, &g.SubjectCode, &g.ClassName,
		); err != nil {
			return nil, err
		}
		grades = append(grades, g)
	}
	return grades, nil
}

func (r *Repository) GetGrade(ctx context.Context, tenantID, gradeID uuid.UUID) (*Grade, error) {
	g := &Grade{}
	query := `
		SELECT g.id, g.tenant_id, g.student_id, g.class_id, g.subject_id, g.academic_year_id,
		       g.daily_score, g.midterm_score, g.final_score,
		       COALESCE(g.grade_letter,''), COALESCE(g.notes,''),
		       g.recorded_by, g.created_at, g.updated_at,
		       u.full_name, sub.name, sub.code, c.name
		FROM grades g
		JOIN users u ON u.id = g.student_id
		JOIN subjects sub ON sub.id = g.subject_id
		JOIN classes c ON c.id = g.class_id
		WHERE g.id = $1 AND g.tenant_id = $2`
	err := r.db.Pool.QueryRow(ctx, query, gradeID, tenantID).Scan(
		&g.ID, &g.TenantID, &g.StudentID, &g.ClassID, &g.SubjectID, &g.AcademicYearID,
		&g.DailyScore, &g.MidtermScore, &g.FinalScore,
		&g.GradeLetter, &g.Notes, &g.RecordedBy,
		&g.CreatedAt, &g.UpdatedAt,
		&g.StudentName, &g.SubjectName, &g.SubjectCode, &g.ClassName,
	)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (r *Repository) DeleteGrade(ctx context.Context, tenantID, gradeID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx,
		`DELETE FROM grades WHERE id = $1 AND tenant_id = $2`,
		gradeID, tenantID,
	)
	return err
}
