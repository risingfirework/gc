package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrExamNotFound             = errors.New("exam not found")
	ErrUserExamNotFound         = errors.New("user exam not found")
	ErrQuestionNotFound         = errors.New("question not found")
	ErrExamExpired              = errors.New("exam time has expired")
	ErrExamAlreadySubmitted     = errors.New("exam already submitted")
	ErrExamSubmissionInProgress = errors.New("exam submission is in progress")
	ErrExamStillRunning         = errors.New("exam is still running")
	ErrExamForbidden            = errors.New("exam does not belong to user")
	ErrCBTTokenRequired         = errors.New("ujian CBT memerlukan kode rahasia (token)")
	ErrCBTTokenInvalid          = errors.New("kode rahasia (token) CBT salah")
	ErrInvalidAnswer            = errors.New("invalid selected option")
	ErrTimerNotFound            = errors.New("exam timer not found")
	ErrPembahasanNotPublished   = errors.New("pembahasan belum dipublikasikan oleh penyelenggara")
)

type SyncAnswerRequest struct {
	// ExamID is the active attempt ID (user_exams.id), retained as exam_id for the public API contract.
	ExamID         string `json:"exam_id"`
	QuestionID     string `json:"question_id"`
	SelectedOption string `json:"selected_option"`
}

type SyncAnswerResponse struct {
	Saved            bool      `json:"saved"`
	SyncedAt         time.Time `json:"synced_at"`
	RemainingSeconds int64     `json:"remaining_seconds"`
}

type SubmitExamRequest struct {
	// ClientFinishedAt is audit metadata only; server time remains authoritative.
	ClientFinishedAt *time.Time `json:"client_finished_at,omitempty"`
}

type SubmitExamResponse struct {
	UserExamID   string    `json:"user_exam_id"`
	ExamID       string    `json:"exam_id"`
	Status       string    `json:"status"`
	FinishedAt   time.Time `json:"finished_at"`
	TotalScore   float64   `json:"total_score"`
	PassingScore float64   `json:"passing_score"`
	Passed       bool      `json:"passed"`
}

type ExamTimer struct {
	UserExamID string
	UserID     string
	ExamID     string
	StartedAt  time.Time
	ExpiresAt  time.Time
}

type ExpiredUserExam struct {
	UserExamID string
	UserID     string
}

type CBTLookupExam struct {
	ExamID            string  `json:"exam_id"`
	Title             string  `json:"title"`
	DurationMinutes   int     `json:"duration_minutes"`
	TotalQuestions    int     `json:"total_questions"`
	PassingScore      float64 `json:"passing_score"`
	Submitted         bool    `json:"submitted"`
	PublishPembahasan bool    `json:"publish_pembahasan"`
	UserExamID        string  `json:"user_exam_id,omitempty"`
}

// CBTLookupPackage is the public result of a student searching a CBT token.
// It exposes the matching active CBT package and its active exams.
type CBTLookupPackage struct {
	ID      string          `json:"id"`
	Title   string          `json:"title"`
	Kode    string          `json:"kode"`
	Jenjang string          `json:"jenjang"`
	Price   float64         `json:"price"`
	Exams   []CBTLookupExam `json:"exams"`
}

// CBTPublishSetting adalah baris pengaturan publish pembahasan sebuah ujian
// CBT beserta jumlah siswa yang sudah mengerjakan (submitted).
type CBTPublishSetting struct {
	ExamID            string  `json:"exam_id"`
	PackageID         string  `json:"package_id"`
	PackageTitle      string  `json:"package_title"`
	PackageKode       string  `json:"package_kode"`
	Jenjang           string  `json:"jenjang"`
	ExamTitle         string  `json:"exam_title"`
	DurationMinutes   int     `json:"duration_minutes"`
	TotalQuestions    int     `json:"total_questions"`
	PassingScore      float64 `json:"passing_score"`
	PublishPembahasan bool    `json:"publish_pembahasan"`
	Participated      int     `json:"participated"`
	PublisherEmail    string  `json:"publisher_email,omitempty"`
}

// CBTParticipant adalah satu siswa yang mengikuti ujian CBT, baik yang sedang
// mengerjakan (ongoing) maupun yang sudah menyelesaikan (submitted), ditampilkan
// pada daftar peserta ujian di Pengaturan CBT.
type CBTParticipant struct {
	UserExamID      string     `json:"user_exam_id"`
	UserID          string     `json:"user_id"`
	Name            string     `json:"name"`
	Email           string     `json:"email"`
	SchoolLevel     string     `json:"school_level"`
	Status          string     `json:"status"` // "ongoing" | "submitted"
	StartedAt       *time.Time `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at"`
	TotalQuestions  int        `json:"total_questions"`
	CurrentQuestion *int       `json:"current_question"` // 1-based, hanya untuk status "ongoing"
	TotalScore      float64    `json:"total_score"`
	PassingScore    float64    `json:"passing_score"`
	Passed          bool       `json:"passed"`
}

// ExamSummary is the public, student-facing exam card for an owned package.
// It includes the latest submitted attempt of the requesting user when present.
type ExamSummary struct {
	ID                string     `json:"id"`
	PackageID         string     `json:"package_id"`
	Title             string     `json:"title"`
	DurationMinutes   int        `json:"duration_minutes"`
	TotalQuestions    int        `json:"total_questions"`
	PassingScore      float64    `json:"passing_score"`
	ScoringMethod     string     `json:"scoring_method"`
	PublishPembahasan bool       `json:"publish_pembahasan"`
	UserExamID        string     `json:"user_exam_id,omitempty"`
	TotalScore        *float64   `json:"total_score,omitempty"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
}

type ExamRepository interface {
	GetExamWithQuestions(ctx context.Context, examID string) (*Exam, []Question, error)
	LookupCBTByToken(ctx context.Context, token, userID string) ([]CBTLookupPackage, error)
	ListExamsByPackage(ctx context.Context, userID, packageID string) ([]ExamSummary, error)
	StartOrGetUserExam(ctx context.Context, userID, examID string, startedAt time.Time) (*UserExam, error)
	GetUserExam(ctx context.Context, userExamID, userID string) (*UserExam, *Exam, error)
	UpsertAnswerFallback(ctx context.Context, userExamID, userID, questionID, selectedOption string) error
	GetPersistedAnswers(ctx context.Context, userExamID, userID string) (map[string]string, error)
	SubmitUserExam(ctx context.Context, userExamID, userID string, answers []UserAnswer, score float64, finishedAt time.Time) (*UserExam, error)
	ListExpiredUserExams(ctx context.Context, limit int) ([]ExpiredUserExam, error)
}

type ExamAccessRepository interface {
	HasActivePackage(ctx context.Context, userID, packageID string, now time.Time) (bool, error)
}

type CBTRepository interface {
	SetTimer(ctx context.Context, timer ExamTimer) error
	GetTimer(ctx context.Context, userExamID string) (*ExamTimer, error)
	SaveAnswer(ctx context.Context, userExamID, userID, questionID, selectedOption string, now time.Time) (time.Duration, error)
	LockAnswersForSubmit(ctx context.Context, userExamID, userID string) (map[string]string, error)
	ReleaseSubmitLock(ctx context.Context, userExamID, userID string) error
	DeleteState(ctx context.Context, userExamID string) error
}

type CBTService interface {
	LookupCBT(ctx context.Context, userID, token string) ([]CBTLookupPackage, error)
	StartExam(ctx context.Context, userID, examID, token string) (*ExamStartResponse, error)
	ListPackageExams(ctx context.Context, userID, packageID string) ([]ExamSummary, error)
	SyncAnswer(ctx context.Context, userID string, input SyncAnswerRequest) (*SyncAnswerResponse, error)
	SubmitExam(ctx context.Context, userID, userExamID string) (*SubmitExamResponse, error)
	AutoSubmitTask(ctx context.Context) error
}
