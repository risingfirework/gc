package redis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"tka/apps/backend/internal/domain"
)

func TestCBTRepositoryAtomicAnswerAndTimer(t *testing.T) {
	server := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	repository := NewCBTRepository(client)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)
	timer := domain.ExamTimer{
		UserExamID: "attempt-1", UserID: "user-1", ExamID: "exam-1",
		StartedAt: now, ExpiresAt: now.Add(30 * time.Minute),
	}
	if err := repository.SetTimer(ctx, timer); err != nil {
		t.Fatal(err)
	}
	storedTimer, err := repository.GetTimer(ctx, timer.UserExamID)
	if err != nil {
		t.Fatal(err)
	}
	if storedTimer.UserID != timer.UserID || !storedTimer.ExpiresAt.Equal(timer.ExpiresAt) {
		t.Fatalf("unexpected timer: %+v", storedTimer)
	}

	if _, err := repository.SaveAnswer(ctx, timer.UserExamID, "another-user", "question-1", "A", now); !errors.Is(err, domain.ErrExamForbidden) {
		t.Fatalf("owner check error = %v", err)
	}
	remaining, err := repository.SaveAnswer(ctx, timer.UserExamID, timer.UserID, "question-1", "A", now.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if remaining != 25*time.Minute {
		t.Fatalf("remaining = %s", remaining)
	}
	answers, err := repository.LockAnswersForSubmit(ctx, timer.UserExamID, timer.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if answers["question-1"] != "A" {
		t.Fatalf("answers = %#v", answers)
	}
	if _, err := repository.SaveAnswer(ctx, timer.UserExamID, timer.UserID, "question-1", "B", now.Add(6*time.Minute)); !errors.Is(err, domain.ErrExamSubmissionInProgress) {
		t.Fatalf("save during submit error = %v", err)
	}
	if err := repository.ReleaseSubmitLock(ctx, timer.UserExamID, timer.UserID); err != nil {
		t.Fatal(err)
	}

	if _, err := repository.SaveAnswer(ctx, timer.UserExamID, timer.UserID, "question-1", "B", timer.ExpiresAt); !errors.Is(err, domain.ErrExamExpired) {
		t.Fatalf("expiry error = %v", err)
	}
	if err := repository.DeleteState(ctx, timer.UserExamID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.GetTimer(ctx, timer.UserExamID); !errors.Is(err, domain.ErrTimerNotFound) {
		t.Fatalf("timer deletion error = %v", err)
	}
}

func TestCBTRepositoryConcurrentAutoSave(t *testing.T) {
	server := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	repository := NewCBTRepository(client)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)
	timer := domain.ExamTimer{UserExamID: "attempt-concurrent", UserID: "user-1", ExamID: "exam-1", StartedAt: now, ExpiresAt: now.Add(time.Hour)}
	if err := repository.SetTimer(ctx, timer); err != nil {
		t.Fatal(err)
	}

	const workers = 100
	errorsChannel := make(chan error, workers)
	var group sync.WaitGroup
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			_, err := repository.SaveAnswer(ctx, timer.UserExamID, timer.UserID, fmt.Sprintf("question-%03d", index), "A", now)
			if err != nil {
				errorsChannel <- err
			}
		}(i)
	}
	group.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		t.Errorf("autosave: %v", err)
	}
	answers, err := repository.LockAnswersForSubmit(ctx, timer.UserExamID, timer.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if len(answers) != workers {
		t.Fatalf("saved answers = %d, want %d", len(answers), workers)
	}
}
