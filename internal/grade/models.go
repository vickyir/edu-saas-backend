package grade

import (
	"time"

	"github.com/google/uuid"
)

// Grade represents a student's grade record for one subject in an academic year.
type Grade struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	StudentID      uuid.UUID  `json:"student_id"`
	ClassID        uuid.UUID  `json:"class_id"`
	SubjectID      uuid.UUID  `json:"subject_id"`
	AcademicYearID uuid.UUID  `json:"academic_year_id"`
	DailyScore     *float64   `json:"daily_score,omitempty"`
	MidtermScore   *float64   `json:"midterm_score,omitempty"`
	FinalScore     *float64   `json:"final_score,omitempty"`
	GradeLetter    string     `json:"grade_letter,omitempty"`
	Notes          string     `json:"notes,omitempty"`
	RecordedBy     *uuid.UUID `json:"recorded_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	// Enriched
	StudentName  string `json:"student_name,omitempty"`
	SubjectName  string `json:"subject_name,omitempty"`
	SubjectCode  string `json:"subject_code,omitempty"`
	ClassName    string `json:"class_name,omitempty"`
}

// ── DTOs ──────────────────────────────────────────────────────────────────────

type UpsertGradeRequest struct {
	StudentID      string   `json:"student_id"`
	ClassID        string   `json:"class_id"`
	SubjectID      string   `json:"subject_id"`
	AcademicYearID string   `json:"academic_year_id"`
	DailyScore     *float64 `json:"daily_score,omitempty"`
	MidtermScore   *float64 `json:"midterm_score,omitempty"`
	FinalScore     *float64 `json:"final_score,omitempty"`
	GradeLetter    string   `json:"grade_letter,omitempty"`
	Notes          string   `json:"notes,omitempty"`
}

type GradeFilter struct {
	ClassID        *uuid.UUID
	SubjectID      *uuid.UUID
	StudentID      *uuid.UUID
	AcademicYearID *uuid.UUID
}
