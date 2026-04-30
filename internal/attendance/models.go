package attendance

import (
	"time"

	"github.com/google/uuid"
)

// AttendanceSession represents a teacher-opened QR attendance session.
type AttendanceSession struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	ClassID   uuid.UUID  `json:"class_id"`
	SubjectID *uuid.UUID `json:"subject_id,omitempty"`
	TeacherID uuid.UUID  `json:"teacher_id"`
	Date      time.Time  `json:"date"`
	QRToken   string     `json:"qr_token"`
	ExpiresAt time.Time  `json:"expires_at"`
	IsClosed  bool       `json:"is_closed"`
	CreatedAt time.Time  `json:"created_at"`
	// Enriched
	ClassName   string `json:"class_name,omitempty"`
	SubjectName string `json:"subject_name,omitempty"`
	TeacherName string `json:"teacher_name,omitempty"`
}

// AttendanceRecord represents one student's attendance for a session.
type AttendanceRecord struct {
	ID        uuid.UUID `json:"id"`
	SessionID uuid.UUID `json:"session_id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	StudentID uuid.UUID `json:"student_id"`
	Status    string    `json:"status"` // present, absent, late, sick, permission
	ScannedAt time.Time `json:"scanned_at"`
	Note      string    `json:"note,omitempty"`
	// Enriched
	StudentName  string `json:"student_name,omitempty"`
	StudentEmail string `json:"student_email,omitempty"`
}

// ── DTOs ──────────────────────────────────────────────────────────────────────

type CreateSessionRequest struct {
	ClassID   string  `json:"class_id"`
	SubjectID *string `json:"subject_id,omitempty"`
	// Duration in minutes the QR code remains valid (default 30)
	DurationMinutes int `json:"duration_minutes,omitempty"`
}

type SubmitAttendanceRequest struct {
	// QRToken is the token embedded in the QR code
	QRToken   string `json:"qr_token"`
	StudentID string `json:"student_id"`
	Status    string `json:"status,omitempty"` // defaults to "present"
}

type ManualAttendanceRequest struct {
	StudentID string `json:"student_id"`
	Status    string `json:"status"` // present, absent, late, sick, permission
	Note      string `json:"note,omitempty"`
}

type SessionSummary struct {
	Session *AttendanceSession  `json:"session"`
	Records []*AttendanceRecord `json:"records"`
	Stats   AttendanceStats     `json:"stats"`
}

type AttendanceStats struct {
	Total      int `json:"total"`
	Present    int `json:"present"`
	Absent     int `json:"absent"`
	Late       int `json:"late"`
	Sick       int `json:"sick"`
	Permission int `json:"permission"`
}
