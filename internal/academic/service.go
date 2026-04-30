package academic

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/edusaas/backend/internal/shared/events"
)

// parseDate accepts "YYYY-MM-DD" or full RFC3339; returns nil on empty string.
func parseDate(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	// Try date-only first (what the frontend sends)
	if t, err := time.Parse("2006-01-02", *s); err == nil {
		return &t
	}
	// Fall back to RFC3339
	if t, err := time.Parse(time.RFC3339, *s); err == nil {
		return &t
	}
	return nil
}

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrInvalidInput  = errors.New("invalid input")
)

type Service struct {
	repo     *Repository
	eventBus *events.Bus
}

func NewService(repo *Repository, eventBus *events.Bus) *Service {
	return &Service{repo: repo, eventBus: eventBus}
}

// ── Academic Years ─────────────────────────────────────────────────────────────

func (s *Service) ListAcademicYears(ctx context.Context, tenantID uuid.UUID) ([]*AcademicYear, error) {
	return s.repo.ListAcademicYears(ctx, tenantID)
}

func (s *Service) CreateAcademicYear(ctx context.Context, tenantID uuid.UUID, req CreateAcademicYearRequest) (*AcademicYear, error) {
	if req.Year == "" || req.Semester == 0 {
		return nil, ErrInvalidInput
	}
	y := &AcademicYear{
		TenantID:  tenantID,
		Year:      req.Year,
		Semester:  req.Semester,
		IsActive:  false,
		StartDate: parseDate(req.StartDate),
		EndDate:   parseDate(req.EndDate),
	}
	if err := s.repo.CreateAcademicYear(ctx, y); err != nil {
		return nil, fmt.Errorf("create academic year: %w", err)
	}
	return y, nil
}

func (s *Service) GetAcademicYear(ctx context.Context, tenantID, yearID uuid.UUID) (*AcademicYear, error) {
	y, err := s.repo.GetAcademicYear(ctx, tenantID, yearID)
	if err != nil {
		return nil, err
	}
	if y == nil {
		return nil, ErrNotFound
	}
	return y, nil
}

func (s *Service) UpdateAcademicYear(ctx context.Context, tenantID, yearID uuid.UUID, req CreateAcademicYearRequest) (*AcademicYear, error) {
	y, err := s.repo.GetAcademicYear(ctx, tenantID, yearID)
	if err != nil || y == nil {
		return nil, ErrNotFound
	}
	if req.Year != "" {
		y.Year = req.Year
	}
	if req.Semester != 0 {
		y.Semester = req.Semester
	}
	y.StartDate = parseDate(req.StartDate)
	y.EndDate = parseDate(req.EndDate)
	if err := s.repo.UpdateAcademicYear(ctx, y); err != nil {
		return nil, fmt.Errorf("update academic year: %w", err)
	}
	return y, nil
}

func (s *Service) DeleteAcademicYear(ctx context.Context, tenantID, yearID uuid.UUID) error {
	return s.repo.DeleteAcademicYear(ctx, tenantID, yearID)
}

func (s *Service) ActivateAcademicYear(ctx context.Context, tenantID, yearID uuid.UUID) error {
	return s.repo.ActivateAcademicYear(ctx, tenantID, yearID)
}

// ── Classes ────────────────────────────────────────────────────────────────────

func (s *Service) ListClasses(ctx context.Context, tenantID uuid.UUID, academicYearID *uuid.UUID) ([]*Class, error) {
	return s.repo.ListClasses(ctx, tenantID, academicYearID)
}

func (s *Service) GetClass(ctx context.Context, tenantID, classID uuid.UUID) (*Class, error) {
	c, err := s.repo.GetClass(ctx, tenantID, classID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrNotFound
	}
	return c, nil
}

func (s *Service) CreateClass(ctx context.Context, tenantID uuid.UUID, req CreateClassRequest) (*Class, error) {
	if req.Name == "" {
		return nil, ErrInvalidInput
	}
	yearID, err := uuid.Parse(req.AcademicYearID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	c := &Class{TenantID: tenantID, AcademicYearID: yearID, Name: req.Name, Level: req.Level}
	if req.HomeroomTeacherID != nil {
		tid, err := uuid.Parse(*req.HomeroomTeacherID)
		if err == nil {
			c.HomeroomTeacherID = &tid
		}
	}
	if err := s.repo.CreateClass(ctx, c); err != nil {
		return nil, fmt.Errorf("create class: %w", err)
	}
	return c, nil
}

func (s *Service) UpdateClass(ctx context.Context, tenantID, classID uuid.UUID, req UpdateClassRequest) (*Class, error) {
	c, err := s.repo.GetClass(ctx, tenantID, classID)
	if err != nil || c == nil {
		return nil, ErrNotFound
	}
	c.Name = req.Name
	c.Level = req.Level
	if req.HomeroomTeacherID != nil {
		tid, err := uuid.Parse(*req.HomeroomTeacherID)
		if err == nil {
			c.HomeroomTeacherID = &tid
		}
	} else {
		c.HomeroomTeacherID = nil
	}
	if err := s.repo.UpdateClass(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) DeleteClass(ctx context.Context, tenantID, classID uuid.UUID) error {
	return s.repo.DeleteClass(ctx, tenantID, classID)
}

// ── Enrollments ────────────────────────────────────────────────────────────────

func (s *Service) ListClassStudents(ctx context.Context, tenantID, classID uuid.UUID) ([]*ClassStudent, error) {
	return s.repo.ListClassStudents(ctx, tenantID, classID)
}

func (s *Service) EnrollStudent(ctx context.Context, tenantID, classID uuid.UUID, req EnrollStudentRequest) error {
	studentID, err := uuid.Parse(req.StudentID)
	if err != nil {
		return ErrInvalidInput
	}
	return s.repo.EnrollStudent(ctx, tenantID, classID, studentID)
}

func (s *Service) UnenrollStudent(ctx context.Context, tenantID, classID, studentID uuid.UUID) error {
	return s.repo.UnenrollStudent(ctx, tenantID, classID, studentID)
}

// ── Class Subjects ─────────────────────────────────────────────────────────────

func (s *Service) ListClassSubjects(ctx context.Context, tenantID, classID uuid.UUID) ([]*ClassSubject, error) {
	return s.repo.ListClassSubjects(ctx, tenantID, classID)
}

func (s *Service) AssignSubject(ctx context.Context, tenantID, classID uuid.UUID, req AssignSubjectRequest) error {
	subjectID, err := uuid.Parse(req.SubjectID)
	if err != nil {
		return ErrInvalidInput
	}
	cs := &ClassSubject{ClassID: classID, SubjectID: subjectID, TenantID: tenantID}
	if req.TeacherID != nil {
		tid, err := uuid.Parse(*req.TeacherID)
		if err == nil {
			cs.TeacherID = &tid
		}
	}
	return s.repo.AssignSubjectToClass(ctx, cs)
}

func (s *Service) RemoveSubject(ctx context.Context, tenantID, classID, subjectID uuid.UUID) error {
	return s.repo.RemoveSubjectFromClass(ctx, tenantID, classID, subjectID)
}

// ── Subjects ───────────────────────────────────────────────────────────────────

func (s *Service) ListSubjects(ctx context.Context, tenantID uuid.UUID) ([]*Subject, error) {
	return s.repo.ListSubjects(ctx, tenantID)
}

func (s *Service) CreateSubject(ctx context.Context, tenantID uuid.UUID, req CreateSubjectRequest) (*Subject, error) {
	if req.Name == "" {
		return nil, ErrInvalidInput
	}
	sub := &Subject{TenantID: tenantID, Name: req.Name, Code: req.Code, Curriculum: req.Curriculum}
	if err := s.repo.CreateSubject(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

func (s *Service) UpdateSubject(ctx context.Context, tenantID, subjectID uuid.UUID, req CreateSubjectRequest) (*Subject, error) {
	sub := &Subject{ID: subjectID, TenantID: tenantID, Name: req.Name, Code: req.Code, Curriculum: req.Curriculum}
	if err := s.repo.UpdateSubject(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

func (s *Service) DeleteSubject(ctx context.Context, tenantID, subjectID uuid.UUID) error {
	return s.repo.DeleteSubject(ctx, tenantID, subjectID)
}

// ── Students ───────────────────────────────────────────────────────────────────

func (s *Service) ListStudents(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]*Student, int64, error) {
	return s.repo.ListStudents(ctx, tenantID, page, perPage)
}

func (s *Service) GetStudent(ctx context.Context, tenantID, studentID uuid.UUID) (*Student, error) {
	st, err := s.repo.GetStudent(ctx, tenantID, studentID)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, ErrNotFound
	}
	return st, nil
}

func (s *Service) CreateStudent(ctx context.Context, tenantID uuid.UUID, req CreateStudentRequest) (*Student, error) {
	if req.Email == "" || req.FullName == "" || req.Password == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.CreateStudent(ctx, tenantID, req)
}

func (s *Service) ListTeachers(ctx context.Context, tenantID uuid.UUID) ([]*TeacherRef, error) {
	return s.repo.ListTeachers(ctx, tenantID)
}
