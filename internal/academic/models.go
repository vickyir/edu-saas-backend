package academic

import (
	"time"

	"github.com/google/uuid"
)

// ── Academic Year ──────────────────────────────────────────────────────────────

type AcademicYear struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	Year      string     `json:"year"`      // "2025/2026"
	Semester  int        `json:"semester"`  // 1 or 2
	IsActive  bool       `json:"is_active"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// ── Class ──────────────────────────────────────────────────────────────────────

type Class struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	AcademicYearID   uuid.UUID  `json:"academic_year_id"`
	Name             string     `json:"name"`  // e.g. "10A", "XI IPA 1"
	Level            string     `json:"level"` // e.g. "10", "11"
	HomeroomTeacherID *uuid.UUID `json:"homeroom_teacher_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	// Enriched fields
	AcademicYear    *AcademicYear `json:"academic_year,omitempty"`
	HomeroomTeacher *TeacherRef   `json:"homeroom_teacher,omitempty"`
	StudentCount    int           `json:"student_count"`
}

type TeacherRef struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
}

// ── Subject ────────────────────────────────────────────────────────────────────

type Subject struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	Name       string    `json:"name"`       // "Matematika"
	Code       string    `json:"code"`       // "MTK"
	Curriculum string    `json:"curriculum"` // "K13", "Merdeka"
	CreatedAt  time.Time `json:"created_at"`
}

// ── Student / Profile ──────────────────────────────────────────────────────────

type Student struct {
	ID          uuid.UUID      `json:"id"`
	TenantID    uuid.UUID      `json:"tenant_id"`
	Email       string         `json:"email"`
	FullName    string         `json:"full_name"`
	Phone       string         `json:"phone,omitempty"`
	Status      string         `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	Profile     *StudentProfile `json:"profile,omitempty"`
}

type StudentProfile struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	NIS          string     `json:"nis,omitempty"`
	BirthDate    *time.Time `json:"birth_date,omitempty"`
	Gender       string     `json:"gender,omitempty"`
	Address      string     `json:"address,omitempty"`
	ParentName   string     `json:"parent_name,omitempty"`
	ParentPhone  string     `json:"parent_phone,omitempty"`
	ParentUserID *uuid.UUID `json:"parent_user_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ClassStudent represents an enrolled student in a class.
type ClassStudent struct {
	ClassID    uuid.UUID `json:"class_id"`
	StudentID  uuid.UUID `json:"student_id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	EnrolledAt time.Time `json:"enrolled_at"`
	Student    *Student  `json:"student,omitempty"`
}

// ClassSubject represents a subject taught in a class (with assigned teacher).
type ClassSubject struct {
	ID        uuid.UUID   `json:"id"`
	ClassID   uuid.UUID   `json:"class_id"`
	SubjectID uuid.UUID   `json:"subject_id"`
	TeacherID *uuid.UUID  `json:"teacher_id,omitempty"`
	TenantID  uuid.UUID   `json:"tenant_id"`
	Subject   *Subject    `json:"subject,omitempty"`
	Teacher   *TeacherRef `json:"teacher,omitempty"`
}

// ── Request / Response DTOs ────────────────────────────────────────────────────

type CreateAcademicYearRequest struct {
	Year      string  `json:"year"`
	Semester  int     `json:"semester"`
	// Accept "YYYY-MM-DD" or RFC3339 from the frontend; parsed in service layer.
	StartDate *string `json:"start_date,omitempty"`
	EndDate   *string `json:"end_date,omitempty"`
}

type CreateClassRequest struct {
	AcademicYearID    string  `json:"academic_year_id"`
	Name              string  `json:"name"`
	Level             string  `json:"level"`
	HomeroomTeacherID *string `json:"homeroom_teacher_id,omitempty"`
}

type UpdateClassRequest struct {
	Name              string  `json:"name"`
	Level             string  `json:"level"`
	HomeroomTeacherID *string `json:"homeroom_teacher_id,omitempty"`
}

type CreateSubjectRequest struct {
	Name       string `json:"name"`
	Code       string `json:"code"`
	Curriculum string `json:"curriculum"`
}

type CreateStudentRequest struct {
	Email       string     `json:"email"`
	FullName    string     `json:"full_name"`
	Phone       string     `json:"phone"`
	Password    string     `json:"password"`
	NIS         string     `json:"nis"`
	BirthDate   *time.Time `json:"birth_date,omitempty"`
	Gender      string     `json:"gender"`
	Address     string     `json:"address"`
	ParentName  string     `json:"parent_name"`
	ParentPhone string     `json:"parent_phone"`
}

type EnrollStudentRequest struct {
	StudentID string `json:"student_id"`
}

type AssignSubjectRequest struct {
	SubjectID string  `json:"subject_id"`
	TeacherID *string `json:"teacher_id,omitempty"`
}
