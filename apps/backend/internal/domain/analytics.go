package domain

import (
	"context"
	"time"
)

type SubjectResult struct {
	SubjectName    string  `json:"subject_name"`
	CorrectAnswers int     `json:"correct_answers"`
	WrongAnswers   int     `json:"wrong_answers"`
	Unanswered     int     `json:"unanswered"`
	TotalQuestions int     `json:"total_questions"`
	Score          float64 `json:"score"`
}

type AnswerReview struct {
	QuestionID          string           `json:"question_id"`
	SubjectName         string           `json:"subject_name"`
	ContentText         string           `json:"content_text"`
	QuestionType        string           `json:"question_type"`
	PresentationType    string           `json:"presentation_type"`
	GroupCode           string           `json:"group_code,omitempty"`
	StimulusText        string           `json:"stimulus_text,omitempty"`
	QuestionImageURL    string           `json:"question_image_url,omitempty"`
	StimulusImageURL    string           `json:"stimulus_image_url,omitempty"`
	CategoryLabels      []string         `json:"category_labels,omitempty"`
	Options             []QuestionOption `json:"options"`
	SelectedOption      *string          `json:"selected_option,omitempty"`
	CorrectAnswer       string           `json:"correct_answer"`
	IsCorrect           bool             `json:"is_correct"`
	ScoreWeight         float64          `json:"score_weight"`
	ExplanationText     string           `json:"explanation_text"`
	ExplanationVideoURL *string          `json:"explanation_video_url,omitempty"`
}

type ExamResultResponse struct {
	UserExamID     string          `json:"user_exam_id"`
	ExamID         string          `json:"exam_id"`
	Title          string          `json:"title"`
	Status         string          `json:"status"`
	ScoringMethod  string          `json:"scoring_method"`
	StartedAt      time.Time       `json:"started_at"`
	FinishedAt     time.Time       `json:"finished_at"`
	TotalScore     float64         `json:"total_score"`
	PassingScore   float64         `json:"passing_score"`
	Passed         bool            `json:"passed"`
	CorrectAnswers int             `json:"correct_answers"`
	WrongAnswers   int             `json:"wrong_answers"`
	Unanswered     int             `json:"unanswered"`
	Subjects       []SubjectResult `json:"subjects"`
	Review         []AnswerReview  `json:"review"`
}

type GlobalRankingEntry struct {
	Rank            int64     `json:"rank"`
	DisplayName     string    `json:"display_name"`
	SchoolLevel     string    `json:"school_level"`
	Score           float64   `json:"score"`
	DurationSeconds int64     `json:"duration_seconds"`
	FinishedAt      time.Time `json:"finished_at"`
	IsCurrentUser   bool      `json:"is_current_user"`
}

type AnalyticsRepository interface {
	GetExamResult(ctx context.Context, userID, userExamID string) (*ExamResultResponse, error)
	ListGlobalRanking(ctx context.Context, currentUserID, level string, limit int) ([]GlobalRankingEntry, error)
}

type ExamAnalyticsService interface {
	GetResult(ctx context.Context, userID, userExamID string) (*ExamResultResponse, error)
	GetGlobalRanking(ctx context.Context, currentUserID, level string) ([]GlobalRankingEntry, error)
}
