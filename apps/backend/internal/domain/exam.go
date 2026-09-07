package domain

import "time"

type Exam struct {
	ID              string
	PackageID       string
	Title           string
	DurationMinutes int
	TotalQuestions  int
	PassingScore    float64
	ScoringMethod   string
	CreatedAt       time.Time
}

type QuestionOption struct {
	Key      string `json:"key"`
	Content  string `json:"content"`
	ImageURL string `json:"image_url,omitempty"`
}

const (
	QuestionTypeSingleChoice   = "single_choice"
	QuestionTypeMultipleChoice = "multiple_choice"
	QuestionTypeCategory       = "category"
	PresentationTypeSingle     = "single"
	PresentationTypeGroup      = "group"
)

type Question struct {
	ID                  string
	ExamID              string
	SubjectName         string
	ContentText         string
	QuestionType        string
	PresentationType    string
	GroupCode           string
	StimulusText        string
	QuestionImageURL    string
	StimulusImageURL    string
	CategoryLabels      []string
	Options             []QuestionOption
	CorrectAnswer       string
	ScoreWeight         float64
	Difficulty          float64
	Discrimination      float64
	ExplanationText     string
	ExplanationVideoURL *string
}

// QuestionResponse deliberately has no CorrectAnswer field.
type QuestionResponse struct {
	ID               string           `json:"id"`
	SubjectName      string           `json:"subject_name"`
	ContentText      string           `json:"content_text"`
	QuestionType     string           `json:"question_type"`
	PresentationType string           `json:"presentation_type"`
	GroupCode        string           `json:"group_code,omitempty"`
	StimulusText     string           `json:"stimulus_text,omitempty"`
	QuestionImageURL string           `json:"question_image_url,omitempty"`
	StimulusImageURL string           `json:"stimulus_image_url,omitempty"`
	CategoryLabels   []string         `json:"category_labels,omitempty"`
	Options          []QuestionOption `json:"options"`
}

type UserExam struct {
	ID         string
	UserID     string
	ExamID     string
	Status     string
	StartedAt  time.Time
	FinishedAt *time.Time
	TotalScore *float64
}

type UserAnswer struct {
	ID             string
	UserExamID     string
	QuestionID     string
	SelectedOption string
	IsCorrect      bool
	UpdatedAt      time.Time
}

type ExamStartResponse struct {
	UserExamID       string             `json:"user_exam_id"`
	ExamID           string             `json:"exam_id"`
	Title            string             `json:"title"`
	Status           string             `json:"status"`
	ServerTime       time.Time          `json:"server_time"`
	StartedAt        time.Time          `json:"started_at"`
	EndsAt           time.Time          `json:"ends_at"`
	RemainingSeconds int64              `json:"remaining_seconds"`
	DurationMinutes  int                `json:"duration_minutes"`
	Questions        []QuestionResponse `json:"questions"`
}
