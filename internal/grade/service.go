package grade

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) UpsertGrade(ctx context.Context, tenantID, recorderID uuid.UUID, req UpsertGradeRequest) (*Grade, error) {
	studentID, err := uuid.Parse(req.StudentID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	classID, err := uuid.Parse(req.ClassID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	subjectID, err := uuid.Parse(req.SubjectID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	academicYearID, err := uuid.Parse(req.AcademicYearID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	g := &Grade{
		TenantID:       tenantID,
		StudentID:      studentID,
		ClassID:        classID,
		SubjectID:      subjectID,
		AcademicYearID: academicYearID,
		DailyScore:     req.DailyScore,
		MidtermScore:   req.MidtermScore,
		FinalScore:     req.FinalScore,
		GradeLetter:    req.GradeLetter,
		Notes:          req.Notes,
		RecordedBy:     &recorderID,
	}

	if err := s.repo.UpsertGrade(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *Service) ListGrades(ctx context.Context, tenantID uuid.UUID, f GradeFilter) ([]*Grade, error) {
	return s.repo.ListGrades(ctx, tenantID, f)
}

func (s *Service) GetGrade(ctx context.Context, tenantID, gradeID uuid.UUID) (*Grade, error) {
	g, err := s.repo.GetGrade(ctx, tenantID, gradeID)
	if err != nil {
		return nil, ErrNotFound
	}
	return g, nil
}

func (s *Service) DeleteGrade(ctx context.Context, tenantID, gradeID uuid.UUID) error {
	return s.repo.DeleteGrade(ctx, tenantID, gradeID)
}
