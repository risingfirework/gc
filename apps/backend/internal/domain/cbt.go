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
	ErrInvalidAnswer            = errors.New("invalid selected option")
	ErrTimerNotFound            = errors.New("exam timer not found")
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

// ExamSummary is the public, student-facing exam card for an owned package.
// It includes the latest submitted attempt of the requesting user when present.
type ExamSummary struct {
	ID              string     `json:"id"`
	PackageID       string     `json:"package_id"`
	Title           string     `json:"title"`
	DurationMinutes int        `json:"duration_minutes"`
	TotalQuestions  int        `json:"total_questions"`
	PassingScore    float64    `json:"passing_score"`
	ScoringMethod   string     `json:"scoring_method"`
	UserExamID      string     `json:"user_exam_id,omitempty"`
	TotalScore      *float64   `json:"total_score,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
}

type ExamRepository interface {
	GetExamWithQuestions(ctx context.Context, examID string) (*Exam, []Question, error)
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
	StartExam(ctx context.Context, userID, examID string) (*ExamStartResponse, error)
	ListPackageExams(ctx context.Context, userID, packageID string) ([]ExamSummary, error)
	SyncAnswer(ctx context.Context, userID string, input SyncAnswerRequest) (*SyncAnswerResponse, error)
	SubmitExam(ctx context.Context, userID, userExamID string) (*SubmitExamResponse, error)
	AutoSubmitTask(ctx context.Context) error
}
