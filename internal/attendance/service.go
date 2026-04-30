package attendance

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrSessionClosed = errors.New("session is closed")
	ErrTokenExpired  = errors.New("qr token has expired")
	ErrInvalidInput  = errors.New("invalid input")
	ErrAlreadyScanned = errors.New("attendance already recorded")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// generateToken creates a cryptographically random hex token.
func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ── Sessions ──────────────────────────────────────────────────────────────────

func (s *Service) CreateSession(ctx context.Context, tenantID, teacherID uuid.UUID, req CreateSessionRequest) (*AttendanceSession, error) {
	classID, err := uuid.Parse(req.ClassID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	duration := req.DurationMinutes
	if duration <= 0 {
		duration = 30
	}

	session := &AttendanceSession{
		TenantID:  tenantID,
		ClassID:   classID,
		TeacherID: teacherID,
		Date:      time.Now(),
		QRToken:   token,
		ExpiresAt: time.Now().Add(time.Duration(duration) * time.Minute),
	}

	if req.SubjectID != nil {
		sid, err := uuid.Parse(*req.SubjectID)
		if err == nil {
			session.SubjectID = &sid
		}
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) GetSession(ctx context.Context, tenantID, sessionID uuid.UUID) (*AttendanceSession, error) {
	sess, err := s.repo.GetSession(ctx, tenantID, sessionID)
	if err != nil {
		return nil, ErrNotFound
	}
	return sess, nil
}

func (s *Service) ListSessions(ctx context.Context, tenantID uuid.UUID, classID *uuid.UUID, date *time.Time) ([]*AttendanceSession, error) {
	return s.repo.ListSessions(ctx, tenantID, classID, date)
}

func (s *Service) CloseSession(ctx context.Context, tenantID, sessionID uuid.UUID) error {
	return s.repo.CloseSession(ctx, tenantID, sessionID)
}

func (s *Service) GetSessionSummary(ctx context.Context, tenantID, sessionID uuid.UUID) (*SessionSummary, error) {
	sess, err := s.repo.GetSession(ctx, tenantID, sessionID)
	if err != nil {
		return nil, ErrNotFound
	}

	records, err := s.repo.ListRecords(ctx, tenantID, sessionID)
	if err != nil {
		return nil, err
	}

	stats := AttendanceStats{Total: len(records)}
	for _, rec := range records {
		switch rec.Status {
		case "present":
			stats.Present++
		case "absent":
			stats.Absent++
		case "late":
			stats.Late++
		case "sick":
			stats.Sick++
		case "permission":
			stats.Permission++
		}
	}

	return &SessionSummary{
		Session: sess,
		Records: records,
		Stats:   stats,
	}, nil
}

// ── QR Scan (student self-submit) ─────────────────────────────────────────────

func (s *Service) SubmitAttendance(ctx context.Context, req SubmitAttendanceRequest) (*AttendanceRecord, error) {
	if req.QRToken == "" || req.StudentID == "" {
		return nil, ErrInvalidInput
	}

	studentID, err := uuid.Parse(req.StudentID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	sess, err := s.repo.GetSessionByToken(ctx, req.QRToken)
	if err != nil {
		return nil, ErrNotFound
	}
	if sess.IsClosed {
		return nil, ErrSessionClosed
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	status := req.Status
	if status == "" {
		status = "present"
	}

	rec := &AttendanceRecord{
		SessionID: sess.ID,
		TenantID:  sess.TenantID,
		StudentID: studentID,
		Status:    status,
	}

	if err := s.repo.UpsertRecord(ctx, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// ── Manual / Teacher override ──────────────────────────────────────────────────

func (s *Service) RecordManual(ctx context.Context, tenantID, sessionID uuid.UUID, req ManualAttendanceRequest) (*AttendanceRecord, error) {
	studentID, err := uuid.Parse(req.StudentID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	if req.Status == "" {
		return nil, ErrInvalidInput
	}

	rec := &AttendanceRecord{
		SessionID: sessionID,
		TenantID:  tenantID,
		StudentID: studentID,
		Status:    req.Status,
		Note:      req.Note,
	}

	if err := s.repo.UpsertRecord(ctx, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *Service) ListRecords(ctx context.Context, tenantID, sessionID uuid.UUID) ([]*AttendanceRecord, error) {
	return s.repo.ListRecords(ctx, tenantID, sessionID)
}
