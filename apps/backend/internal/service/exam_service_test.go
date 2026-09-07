package service

import (
	"math"
	"testing"

	"tka/apps/backend/internal/domain"
)

func TestCalculateExamScoreWeighted(t *testing.T) {
	questions := []domain.Question{{ID: "1", CorrectAnswer: "A", ScoreWeight: 1}, {ID: "2", CorrectAnswer: "B", ScoreWeight: 3}}
	got := CalculateExamScore("standard", questions, map[string]string{"1": "A", "2": "A"})
	if got != 25 {
		t.Fatalf("expected 25, got %v", got)
	}
}

func TestCalculateExamScoreIRTOrdersAbility(t *testing.T) {
	questions := []domain.Question{
		{ID: "1", CorrectAnswer: "A", Discrimination: 1.2, Difficulty: -0.5},
		{ID: "2", CorrectAnswer: "B", Discrimination: 1.1, Difficulty: 0.5},
		{ID: "3", CorrectAnswer: "C", Discrimination: 0.9, Difficulty: 1},
	}
	low := CalculateExamScore("irt_2pl", questions, map[string]string{})
	high := CalculateExamScore("irt_2pl", questions, map[string]string{"1": "A", "2": "B", "3": "C"})
	if math.IsNaN(low) || math.IsNaN(high) || high <= low {
		t.Fatalf("expected ordered finite scores, low=%v high=%v", low, high)
	}
}
