package service

import (
	"testing"

	"tka/apps/backend/internal/domain"
)

func TestCanonicalTKAAnswers(t *testing.T) {
	options := []domain.QuestionOption{{Key: "A", Content: "Satu"}, {Key: "B", Content: "Dua"}, {Key: "C", Content: "Tiga"}}
	tests := []struct {
		name, questionType, input, want string
		labels                         []string
	}{
		{"single", domain.QuestionTypeSingleChoice, " a ", "A", nil},
		{"mcma sorted", domain.QuestionTypeMultipleChoice, "C, A", "A,C", nil},
		{"category sorted", domain.QuestionTypeCategory, "C=Benar;A=Benar;B=Salah", "A=Benar;B=Salah;C=Benar", []string{"Benar", "Salah"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, valid := canonicalAnswer(test.questionType, options, test.labels, test.input, false)
			if !valid || got != test.want {
				t.Fatalf("canonicalAnswer() = %q, %v; want %q, true", got, valid, test.want)
			}
		})
	}
}

func TestCanonicalTKAAnswersRejectsIncompleteKeys(t *testing.T) {
	options := []domain.QuestionOption{{Key: "A", Content: "Satu"}, {Key: "B", Content: "Dua"}}
	if _, valid := canonicalAnswer(domain.QuestionTypeMultipleChoice, options, nil, "A", false); valid {
		t.Fatal("MCMA with one correct answer must be rejected")
	}
	if _, valid := canonicalAnswer(domain.QuestionTypeCategory, options, []string{"Benar", "Salah"}, "A=Benar", false); valid {
		t.Fatal("incomplete category answer must be rejected")
	}
}
