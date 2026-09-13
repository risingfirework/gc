package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"tka/apps/backend/internal/domain"
)

const (
	testUserID      = "11111111-1111-4111-8111-111111111111"
	testExamID      = "22222222-2222-4222-8222-222222222222"
	testAttemptID   = "33333333-3333-4333-8333-333333333333"
	testQuestionOne = "44444444-4444-4444-8444-444444444444"
	testQuestionTwo = "55555555-5555-4555-8555-555555555555"
)

type fakeExamRepository struct {
	exam           domain.Exam
	questions      []domain.Question
	attempt        domain.UserExam
	persisted      map[string]string
	fallbackCalls  int
	submitted      []domain.UserAnswer
	submittedScore float64
	expired        []domain.ExpiredUserExam
}

func (f *fakeExamRepository) GetExamWithQuestions(context.Context, string) (*domain.Exam, []domain.Question, error) {
	exam := f.exam
	return &exam, f.questions, nil
}
func (f *fakeExamRepository) LookupCBTByToken(context.Context, string, string) ([]domain.CBTLookupPackage, error) {
	return nil, nil
}
func (f *fakeExamRepository) ListExamsByPackage(context.Context, string, string) ([]domain.ExamSummary, error) {
	return nil, nil
}
func (f *fakeExamRepository) StartOrGetUserExam(context.Context, string, string, time.Time) (*domain.UserExam, error) {
	attempt := f.attempt
	return &attempt, nil
}
func (f *fakeExamRepository) GetUserExam(context.Context, string, string) (*domain.UserExam, *domain.Exam, error) {
	attempt, exam := f.attempt, f.exam
	return &attempt, &exam, nil
}
func (f *fakeExamRepository) UpsertAnswerFallback(context.Context, string, string, string, string) error {
	f.fallbackCalls++
	return nil
}
func (f *fakeExamRepository) GetPersistedAnswers(context.Context, string, string) (map[string]string, error) {
	result := make(map[string]string, len(f.persisted))
	for k, v := range f.persisted {
		result[k] = v
	}
	return result, nil
}
func (f *fakeExamRepository) SubmitUserExam(_ context.Context, _, _ string, answers []domain.UserAnswer, score float64, finishedAt time.Time) (*domain.UserExam, error) {
	f.submitted = answers
	f.submittedScore = score
	f.attempt.Status = "submitted"
	f.attempt.FinishedAt = &finishedAt
	f.attempt.TotalScore = &score
	attempt := f.attempt
	return &attempt, nil
}
func (f *fakeExamRepository) ListExpiredUserExams(context.Context, int) ([]domain.ExpiredUserExam, error) {
	return f.expired, nil
}

type fakeCBTRepository struct {
	timer   domain.ExamTimer
	answers map[string]string
	saveErr error
	getErr  error
	deleted bool
}

func (f *fakeCBTRepository) SetTimer(_ context.Context, timer domain.ExamTimer) error {
	f.timer = timer
	return nil
}
func (f *fakeCBTRepository) GetTimer(context.Context, string) (*domain.ExamTimer, error) {
	timer := f.timer
	return &timer, nil
}
func (f *fakeCBTRepository) SaveAnswer(_ context.Context, _, _, questionID, selected string, _ time.Time) (time.Duration, error) {
	if f.saveErr != nil {
		return 0, f.saveErr
	}
	if f.answers == nil {
		f.answers = map[string]string{}
	}
	f.answers[questionID] = selected
	return 10 * time.Minute, nil
}
func (f *fakeCBTRepository) LockAnswersForSubmit(context.Context, string, string) (map[string]string, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	result := make(map[string]string, len(f.answers))
	for k, v := range f.answers {
		result[k] = v
	}
	return result, nil
}
func (f *fakeCBTRepository) ReleaseSubmitLock(context.Context, string, string) error { return nil }
func (f *fakeCBTRepository) DeleteState(context.Context, string) error               { f.deleted = true; return nil }

func newCBTFixture() (*CBTService, *fakeExamRepository, *fakeCBTRepository) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	exams := &fakeExamRepository{
		exam: domain.Exam{ID: testExamID, Title: "TKA", DurationMinutes: 30, PassingScore: 70},
		questions: []domain.Question{
			{ID: testQuestionOne, ExamID: testExamID, CorrectAnswer: "A", ScoreWeight: 1, Options: []domain.QuestionOption{{Key: "A", Content: "1"}, {Key: "B", Content: "2"}}},
			{ID: testQuestionTwo, ExamID: testExamID, CorrectAnswer: "B", ScoreWeight: 1, Options: []domain.QuestionOption{{Key: "A", Content: "1"}, {Key: "B", Content: "2"}}},
		},
		attempt:   domain.UserExam{ID: testAttemptID, UserID: testUserID, ExamID: testExamID, Status: "ongoing", StartedAt: now},
		persisted: map[string]string{},
	}
	cache := &fakeCBTRepository{answers: map[string]string{}}
	service := NewCBTService(exams, cache, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.now = func() time.Time { return now.Add(5 * time.Minute) }
	return service, exams, cache
}

func TestStartExamDoesNotExposeCorrectAnswers(t *testing.T) {
	service, _, cache := newCBTFixture()
	response, err := service.StartExam(context.Background(), testUserID, testExamID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Questions) != 2 {
		t.Fatalf("questions=%d", len(response.Questions))
	}
	if cache.timer.UserID != testUserID || response.RemainingSeconds != 1500 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestSyncAnswerUsesRedisFastPath(t *testing.T) {
	service, exams, cache := newCBTFixture()
	response, err := service.SyncAnswer(context.Background(), testUserID, domain.SyncAnswerRequest{ExamID: testAttemptID, QuestionID: testQuestionOne, SelectedOption: "A"})
	if err != nil {
		t.Fatal(err)
	}
	if !response.Saved || cache.answers[testQuestionOne] != "A" || exams.fallbackCalls != 0 {
		t.Fatal("answer did not use cache fast path")
	}
}

func TestSyncAnswerFallsBackToPostgres(t *testing.T) {
	service, exams, cache := newCBTFixture()
	cache.saveErr = errors.New("redis disconnected")
	response, err := service.SyncAnswer(context.Background(), testUserID, domain.SyncAnswerRequest{ExamID: testAttemptID, QuestionID: testQuestionOne, SelectedOption: "A"})
	if err != nil {
		t.Fatal(err)
	}
	if !response.Saved || exams.fallbackCalls != 1 {
		t.Fatal("PostgreSQL fallback was not used")
	}
}

func TestSubmitMergesAnswersAndCalculatesWeightedScore(t *testing.T) {
	service, exams, cache := newCBTFixture()
	exams.persisted[testQuestionTwo] = "A"
	cache.answers[testQuestionOne] = "A"
	response, err := service.SubmitExam(context.Background(), testUserID, testAttemptID)
	if err != nil {
		t.Fatal(err)
	}
	if response.TotalScore != 50 || response.Passed {
		t.Fatalf("unexpected result: %+v", response)
	}
	if len(exams.submitted) != 2 || !cache.deleted {
		t.Fatal("answers were not committed or cache was not cleared")
	}
}

func TestSubmitDoesNotCommitWhenRedisIsUnavailable(t *testing.T) {
	service, exams, cache := newCBTFixture()
	cache.getErr = errors.New("redis disconnected")
	_, err := service.SubmitExam(context.Background(), testUserID, testAttemptID)
	if err == nil {
		t.Fatal("expected Redis read error")
	}
	if exams.attempt.Status == "submitted" {
		t.Fatal("exam must not be submitted with potentially missing answers")
	}
}

func TestSubmitStoresEssayAsUngraded(t *testing.T) {
	service, exams, cache := newCBTFixture()
	exams.questions = []domain.Question{
		{ID: testQuestionOne, ExamID: testExamID, QuestionType: domain.QuestionTypeEssay, CorrectAnswer: "Referensi jawaban", ScoreWeight: 1},
	}
	cache.answers[testQuestionOne] = "  Jawaban siswa  "
	response, err := service.SubmitExam(context.Background(), testUserID, testAttemptID)
	if err != nil {
		t.Fatal(err)
	}
	if len(exams.submitted) != 1 {
		t.Fatalf("essay answer was not persisted, submitted=%d", len(exams.submitted))
	}
	answer := exams.submitted[0]
	if answer.SelectedOption != "Jawaban siswa" || answer.IsCorrect {
		t.Fatalf("essay must be stored trimmed and ungraded, got %+v", answer)
	}
	if response.TotalScore != 0 {
		t.Fatalf("essay must not count in auto score, got %v", response.TotalScore)
	}
}
