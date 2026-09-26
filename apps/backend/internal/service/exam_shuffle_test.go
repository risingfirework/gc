package service

import (
	"context"
	"testing"

	"tka/apps/backend/internal/domain"
)

func TestApplyDisplayShuffleKeepsCanonicalFrame(t *testing.T) {
	questions := []domain.Question{
		{ID: testQuestionOne, CorrectAnswer: "A", Options: []domain.QuestionOption{{Key: "A", Content: "1"}, {Key: "B", Content: "2"}}},
		{ID: testQuestionTwo, CorrectAnswer: "B", Options: []domain.QuestionOption{{Key: "A", Content: "1"}, {Key: "B", Content: "2"}}},
	}
	response := applyDisplayShuffle(questions, &domain.UserExamShuffle{})
	if len(response) != 2 {
		t.Fatalf("questions=%d", len(response))
	}
	if response[0].ID != testQuestionOne || response[0].Options[0].Key != "A" || response[1].ID != testQuestionTwo {
		t.Fatalf("canonical frame altered: %+v", response)
	}
}

func TestApplyDisplayShuffleReordersAndRelabels(t *testing.T) {
	questions := []domain.Question{
		{ID: testQuestionOne, CorrectAnswer: "A", Options: []domain.QuestionOption{{Key: "A", Content: "satu"}, {Key: "B", Content: "dua"}}},
		{ID: testQuestionTwo, CorrectAnswer: "B", Options: []domain.QuestionOption{{Key: "A", Content: "satu"}, {Key: "B", Content: "dua"}}},
	}
	mapping := domain.UserExamShuffle{
		QuestionOrder: []string{testQuestionTwo, testQuestionOne},
		OptionOrder: map[string][]string{
			testQuestionOne: {"B", "A"},
			testQuestionTwo: {"A", "B"},
		},
	}
	response := applyDisplayShuffle(questions, &mapping)
	if response[0].ID != testQuestionTwo || response[1].ID != testQuestionOne {
		t.Fatalf("question order not applied: %+v", []string{response[0].ID, response[1].ID})
	}
	first := response[1] // testQuestionOne
	if len(first.Options) != 2 || first.Options[0].Key != "A" || first.Options[0].Content != "dua" || first.Options[1].Key != "B" || first.Options[1].Content != "satu" {
		t.Fatalf("option permute not relabeled correctly: %+v", first.Options)
	}
}

func TestApplyDisplayShuffleFallsBackOnPartialQuestionOrder(t *testing.T) {
	questions := []domain.Question{
		{ID: testQuestionOne},
		{ID: testQuestionTwo},
	}
	response := applyDisplayShuffle(questions, &domain.UserExamShuffle{
		QuestionOrder: []string{testQuestionTwo},
	})
	if len(response) != 2 || response[0].ID != testQuestionOne || response[1].ID != testQuestionTwo {
		t.Fatalf("partial question order must use canonical order: %+v", response)
	}
}

func TestDisplayQuestionKeepsAllOptionsOnPartialOrder(t *testing.T) {
	question := domain.Question{
		ID: testQuestionOne, CorrectAnswer: "C",
		Options: []domain.QuestionOption{
			{Key: "A", Content: "satu"}, {Key: "B", Content: "dua"}, {Key: "C", Content: "tiga"},
		},
	}
	mapping := &domain.UserExamShuffle{OptionOrder: map[string][]string{
		testQuestionOne: {"B", "A"}, // permutasi parsial: kunci "C" tidak ter-cover
	}}
	response := displayQuestion(question, mapping)
	if len(response.Options) != 3 {
		t.Fatalf("partial option order dropped options: got %d want 3 (%+v)", len(response.Options), response.Options)
	}
	if response.Options[0].Key != "A" || response.Options[1].Key != "B" || response.Options[2].Key != "C" {
		t.Fatalf("partial order must fall back to canonical frame: %+v", response.Options)
	}
}

func TestTranslatePassthroughOnPartialOrder(t *testing.T) {
	mapping := &domain.UserExamShuffle{OptionOrder: map[string][]string{
		testQuestionOne: {"B", "A"},
	}}
	item := domain.Question{ID: testQuestionOne, QuestionType: domain.QuestionTypeSingleChoice}
	got, ok := translateDisplayFrame(mapping, item, "C")
	if !ok || got != "C" {
		t.Fatalf("partial option order must pass through canonical frame: %q %v", got, ok)
	}
}

func TestTranslateDisplayFrame(t *testing.T) {
	mapping := domain.UserExamShuffle{OptionOrder: map[string][]string{testQuestionOne: {"B", "A"}}}
	options := []domain.QuestionOption{{Key: "A", Content: "satu"}, {Key: "B", Content: "dua"}}

	cases := []struct {
		name     string
		kind     string
		selected string
		want     string
		valid    bool
	}{
		{"single correct", domain.QuestionTypeSingleChoice, "B", "A", true},
		{"single wrong", domain.QuestionTypeSingleChoice, "A", "B", true},
		{"single out of range", domain.QuestionTypeSingleChoice, "C", "", false},
		{"multiple", domain.QuestionTypeMultipleChoice, "B,A", "A,B", true},
		{"category", domain.QuestionTypeCategory, "A=Benar;B=Salah", "B=Benar;A=Salah", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			item := domain.Question{ID: testQuestionOne, QuestionType: tc.kind, Options: options}
			got, ok := translateDisplayFrame(&mapping, item, tc.selected)
			if ok != tc.valid || got != tc.want {
				t.Fatalf("translate(%q)=%q,%v want %q,%v", tc.selected, got, ok, tc.want, tc.valid)
			}
		})
	}

	essay, ok := translateDisplayFrame(&mapping, domain.Question{ID: testQuestionOne, QuestionType: domain.QuestionTypeEssay, Options: options}, "Uraian apa pun")
	if !ok || essay != "Uraian apa pun" {
		t.Fatalf("essay must not be translated: %q %v", essay, ok)
	}
}

func TestStartExamAppliesPersistedShuffle(t *testing.T) {
	service, exams, _ := newCBTFixture()
	exams.exam.ShuffleQuestions = true
	exams.exam.ShuffleOptions = true
	exams.shuffle = domain.UserExamShuffle{
		QuestionOrder: []string{testQuestionTwo, testQuestionOne},
		OptionOrder: map[string][]string{
			testQuestionOne: {"B", "A"},
			testQuestionTwo: {"A", "B"},
		},
	}
	response, err := service.StartExam(context.Background(), testUserID, testExamID, "")
	if err != nil {
		t.Fatal(err)
	}
	if response.Questions[0].ID != testQuestionTwo || response.Questions[1].ID != testQuestionOne {
		t.Fatalf("question order not shuffled: %s, %s", response.Questions[0].ID, response.Questions[1].ID)
	}
	first := response.Questions[1] // testQuestionOne
	if first.Options[0].Key != "A" || first.Options[0].Content != "2" || first.Options[1].Key != "B" || first.Options[1].Content != "1" {
		t.Fatalf("option permute not applied: %+v", first.Options)
	}
}

func TestSubmitTranslatesDisplayFrameAnswers(t *testing.T) {
	service, exams, cache := newCBTFixture()
	exams.exam.ShuffleOptions = true
	exams.shuffle = domain.UserExamShuffle{
		OptionOrder: map[string][]string{
			testQuestionOne: {"B", "A"}, // display A -> kanonik B, display B -> kanonik A
			testQuestionTwo: {"A", "B"},
		},
	}
	// testQuestionOne kanonik A, testQuestionTwo kanonik B.
	cache.answers[testQuestionOne] = "B" // -> kanonik A (benar)
	cache.answers[testQuestionTwo] = "B" // -> kanonik B (benar)
	response, err := service.SubmitExam(context.Background(), testUserID, testAttemptID)
	if err != nil {
		t.Fatal(err)
	}
	if response.TotalScore != 100 || !response.Passed {
		t.Fatalf("unexpected score: %+v", response)
	}
	submitted := map[string]domain.UserAnswer{}
	for _, answer := range exams.submitted {
		submitted[answer.QuestionID] = answer
	}
	if submitted[testQuestionOne].SelectedOption != "A" || !submitted[testQuestionOne].IsCorrect {
		t.Fatalf("q1 not canonical: %+v", submitted[testQuestionOne])
	}
	if submitted[testQuestionTwo].SelectedOption != "B" || !submitted[testQuestionTwo].IsCorrect {
		t.Fatalf("q2 not canonical: %+v", submitted[testQuestionTwo])
	}
}
