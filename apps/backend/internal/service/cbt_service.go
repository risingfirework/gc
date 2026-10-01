package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"tka/apps/backend/internal/domain"
)

const autoSubmitBatchSize = 500
const maxAnswerLength = 10000

// ScreenLockSeconds adalah durasi layar kunci otomatis setiap siswa terdeteksi
// keluar aplikasi atau berpindah tab di tengah ujian CBT.
const ScreenLockSeconds = 5

// maxLockEventLength membatasi panjang event lifecycle yang disimpan ke database.
const maxLockEventLength = 64

type CBTService struct {
	exams  domain.ExamRepository
	cache  domain.CBTRepository
	logger *slog.Logger
	access domain.ExamAccessRepository
	locks  domain.ExamScreenLockRepository
	now    func() time.Time
}

func NewCBTService(exams domain.ExamRepository, cache domain.CBTRepository, logger *slog.Logger, access ...domain.ExamAccessRepository) *CBTService {
	service := &CBTService{exams: exams, cache: cache, logger: logger, now: time.Now}
	if len(access) > 0 {
		service.access = access[0]
	}
	return service
}

// WithScreenLockRepository mengaktifkan penguncian layar untuk mode CBT.
// Tanpa repository ini ReportViolation selalu melaporkan Enforced false, sehingga
// bank soal mode sell dan deployment lama tetap berperilaku seperti semula.
func (s *CBTService) WithScreenLockRepository(locks domain.ExamScreenLockRepository) *CBTService {
	s.locks = locks
	return s
}

func (s *CBTService) LookupCBT(ctx context.Context, userID, token string) ([]domain.CBTLookupPackage, error) {
	provided := strings.ToUpper(strings.TrimSpace(token))
	if provided == "" {
		return nil, domain.ErrCBTTokenRequired
	}
	return s.exams.LookupCBTByToken(ctx, provided, userID)
}

func (s *CBTService) StartExam(ctx context.Context, userID, examID, token string) (*domain.ExamStartResponse, error) {
	if !validUUID(userID) || !validUUID(examID) {
		return nil, domain.ErrExamNotFound
	}
	exam, questions, err := s.exams.GetExamWithQuestions(ctx, examID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	if s.access != nil {
		allowed, accessErr := s.access.HasActivePackage(ctx, userID, exam.PackageID, now)
		if accessErr != nil {
			return nil, accessErr
		}
		if !allowed {
			return nil, domain.ErrExamForbidden
		}
	}
	if exam.PackageExamType == "cbt" {
		provided := strings.ToUpper(strings.TrimSpace(token))
		if provided == "" {
			return nil, domain.ErrCBTTokenRequired
		}
		if provided != exam.PackageCBTToken {
			return nil, domain.ErrCBTTokenInvalid
		}
	}
	attempt, err := s.exams.StartOrGetUserExam(ctx, userID, examID, now)
	if err != nil {
		return nil, err
	}
	if attempt.Status == "submitted" {
		return nil, domain.ErrExamAlreadySubmitted
	}
	expiresAt := attempt.StartedAt.Add(time.Duration(exam.DurationMinutes) * time.Minute)
	if !now.Before(expiresAt) {
		_, _ = s.SubmitExam(ctx, userID, attempt.ID)
		return nil, domain.ErrExamExpired
	}
	timer := domain.ExamTimer{UserExamID: attempt.ID, UserID: userID, ExamID: examID, StartedAt: attempt.StartedAt, ExpiresAt: expiresAt}
	if err := s.cache.SetTimer(ctx, timer); err != nil {
		// PostgreSQL timestamps remain authoritative; SyncAnswer will use its durable fallback.
		s.logger.WarnContext(ctx, "Redis timer unavailable; using database fallback", "user_exam_id", attempt.ID, "error", err)
	}
	publicQuestions := applyDisplayShuffle(questions, s.userExamShuffle(ctx, attempt.ID))
	return &domain.ExamStartResponse{
		UserExamID: attempt.ID, ExamID: exam.ID, Title: exam.Title, Status: attempt.Status,
		ServerTime: now, StartedAt: attempt.StartedAt, EndsAt: expiresAt,
		RemainingSeconds: remainingSeconds(expiresAt, now), DurationMinutes: exam.DurationMinutes,
		ExamType:          exam.PackageExamType,
		ScreenLockEnabled: s.screenLockEnforced(exam.PackageExamType),
		Questions:         publicQuestions,
	}, nil
}

// screenLockEnforced menentukan apakah penguncian layar berlaku untuk mode bank
// soal. Hanya mode CBT yang dikunci; mode sell dan mode lain tidak tersentuh.
func (s *CBTService) screenLockEnforced(examType string) bool {
	return s.locks != nil && strings.EqualFold(strings.TrimSpace(examType), "cbt")
}

// normalizeLockEvent memangkas dan membersihkan label lifecycle dari klien
// sebelum disimpan, karena kolomnya dibatasi 64 karakter.
func normalizeLockEvent(event string) string {
	event = strings.TrimSpace(event)
	event = strings.Map(func(r rune) rune {
		if r <= 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, event)
	if len(event) > maxLockEventLength {
		return event[:maxLockEventLength]
	}
	return event
}

// ReportViolation mencatat bahwa siswa keluar aplikasi atau berpindah tab di
// tengah ujian, lalu mengembalikan apakah layar harus dikunci. Pembulatan waktu
// memakai jam server supaya hitung mundur klien tidak bisa dimanipulasi.
func (s *CBTService) ReportViolation(ctx context.Context, userID, userExamID, event string) (*domain.ReportViolationResponse, error) {
	if !validUUID(userID) || !validUUID(userExamID) {
		return nil, domain.ErrUserExamNotFound
	}
	examType, err := s.attemptExamType(ctx, userID, userExamID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	if !s.screenLockEnforced(examType) {
		return &domain.ReportViolationResponse{Enforced: false, ServerTime: now}, nil
	}
	lock, err := s.locks.UpsertExamScreenLock(ctx, userExamID, normalizeLockEvent(event), now, ScreenLockSeconds*time.Second)
	if err != nil {
		return nil, err
	}
	return lockResponse(lock, now), nil
}

// GetScreenLock mengembalikan status kunci terkini untuk polling klien, tanpa
// menambah penghitung pelanggaran.
func (s *CBTService) GetScreenLock(ctx context.Context, userID, userExamID string) (*domain.ReportViolationResponse, error) {
	if !validUUID(userID) || !validUUID(userExamID) {
		return nil, domain.ErrUserExamNotFound
	}
	examType, err := s.attemptExamType(ctx, userID, userExamID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	if !s.screenLockEnforced(examType) {
		return &domain.ReportViolationResponse{Enforced: false, ServerTime: now}, nil
	}
	lock, err := s.locks.GetExamScreenLock(ctx, userExamID, now)
	if err != nil {
		return nil, err
	}
	if lock == nil {
		return &domain.ReportViolationResponse{Enforced: true, ServerTime: now}, nil
	}
	return lockResponse(lock, now), nil
}

func lockResponse(lock *domain.ExamScreenLock, now time.Time) *domain.ReportViolationResponse {
	response := &domain.ReportViolationResponse{
		Enforced:       true,
		Locked:         lock.IsActiveAt(now),
		LockSeconds:    ScreenLockSeconds,
		ViolationCount: lock.ViolationCount,
		ServerTime:     now,
	}
	if response.Locked {
		until := lock.UnlockUntil.UTC()
		response.UnlockUntil = &until
	}
	return response
}

// attemptExamType memvalidasi kepemilikan attempt dan mengembalikan mode bank
// soal induknya, sehingga hanya attempt milik siswa sendiri yang bisa diproses.
func (s *CBTService) attemptExamType(ctx context.Context, userID, userExamID string) (string, error) {
	attempt, exam, err := s.exams.GetUserExam(ctx, userExamID, userID)
	if err != nil {
		return "", err
	}
	if attempt == nil || exam == nil {
		return "", domain.ErrUserExamNotFound
	}
	return exam.PackageExamType, nil
}

func (s *CBTService) ListPackageExams(ctx context.Context, userID, packageID string) ([]domain.ExamSummary, error) {
	if !validUUID(userID) || !validUUID(packageID) {
		return nil, domain.ErrExamForbidden
	}
	now := s.now().UTC()
	if s.access != nil {
		allowed, err := s.access.HasActivePackage(ctx, userID, packageID, now)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, domain.ErrExamForbidden
		}
	}
	return s.exams.ListExamsByPackage(ctx, userID, packageID)
}

func (s *CBTService) SyncAnswer(ctx context.Context, userID string, input domain.SyncAnswerRequest) (*domain.SyncAnswerResponse, error) {
	if !validUUID(userID) || !validUUID(input.ExamID) || !validUUID(input.QuestionID) {
		return nil, domain.ErrInvalidAnswer
	}
	input.SelectedOption = strings.TrimSpace(input.SelectedOption)
	if input.SelectedOption == "" || len(input.SelectedOption) > maxAnswerLength {
		return nil, domain.ErrInvalidAnswer
	}
	now := s.now().UTC()
	remaining, err := s.cache.SaveAnswer(ctx, input.ExamID, userID, input.QuestionID, input.SelectedOption, now)
	if err == nil {
		return &domain.SyncAnswerResponse{Saved: true, SyncedAt: now, RemainingSeconds: int64(math.Ceil(remaining.Seconds()))}, nil
	}
	if errors.Is(err, domain.ErrExamForbidden) {
		return nil, err
	}
	if errors.Is(err, domain.ErrExamSubmissionInProgress) {
		return nil, err
	}
	if errors.Is(err, domain.ErrExamExpired) {
		_, _ = s.SubmitExam(ctx, userID, input.ExamID)
		return nil, domain.ErrExamExpired
	}

	// Redis disconnect or a missing timer degrades to a durable PostgreSQL write.
	attempt, exam, dbErr := s.exams.GetUserExam(ctx, input.ExamID, userID)
	if dbErr != nil {
		return nil, dbErr
	}
	if attempt.Status == "submitted" {
		return nil, domain.ErrExamAlreadySubmitted
	}
	expiresAt := attempt.StartedAt.Add(time.Duration(exam.DurationMinutes) * time.Minute)
	if !now.Before(expiresAt) {
		_, _ = s.SubmitExam(ctx, userID, input.ExamID)
		return nil, domain.ErrExamExpired
	}
	if dbErr := s.exams.UpsertAnswerFallback(ctx, input.ExamID, userID, input.QuestionID, input.SelectedOption); dbErr != nil {
		return nil, dbErr
	}
	// Best effort restoration makes following answers fast again after Redis recovers.
	_ = s.cache.SetTimer(ctx, domain.ExamTimer{UserExamID: input.ExamID, UserID: userID, ExamID: exam.ID, StartedAt: attempt.StartedAt, ExpiresAt: expiresAt})
	s.logger.WarnContext(ctx, "answer saved through PostgreSQL fallback", "user_exam_id", input.ExamID, "redis_error", err)
	return &domain.SyncAnswerResponse{Saved: true, SyncedAt: now, RemainingSeconds: remainingSeconds(expiresAt, now)}, nil
}

func (s *CBTService) SubmitExam(ctx context.Context, userID, userExamID string) (*domain.SubmitExamResponse, error) {
	if !validUUID(userID) || !validUUID(userExamID) {
		return nil, domain.ErrUserExamNotFound
	}
	attempt, exam, err := s.exams.GetUserExam(ctx, userExamID, userID)
	if err != nil {
		return nil, err
	}
	if attempt.Status == "submitted" {
		if err := s.cache.DeleteState(ctx, userExamID); err != nil {
			s.logger.WarnContext(ctx, "failed to clean state for an existing submission", "user_exam_id", userExamID, "error", err)
		}
		return submitResponse(attempt, exam), nil
	}
	_, questions, err := s.exams.GetExamWithQuestions(ctx, exam.ID)
	if err != nil {
		return nil, err
	}

	answers, err := s.exams.GetPersistedAnswers(ctx, userExamID, userID)
	if err != nil {
		return nil, err
	}
	cachedAnswers, err := s.cache.LockAnswersForSubmit(ctx, userExamID, userID)
	if err != nil {
		// Do not finalize while potentially newer Redis answers are inaccessible.
		return nil, fmt.Errorf("read Redis answers before submit: %w", err)
	}
	for questionID, selectedOption := range cachedAnswers {
		answers[questionID] = selectedOption
	}
	locked := true
	defer func() {
		if locked {
			s.releaseSubmitLock(ctx, userExamID, userID)
		}
	}()

	questionByID := make(map[string]domain.Question, len(questions))
	for _, question := range questions {
		questionByID[question.ID] = question
	}
	mapping := s.userExamShuffle(ctx, userExamID)
	batch := make([]domain.UserAnswer, 0, len(answers))
	normalizedAnswers := make(map[string]string, len(answers))
	for questionID, selectedOption := range answers {
		question, exists := questionByID[questionID]
		if !exists {
			continue
		}
		display := selectedOption
		if translated, ok := translateDisplayFrame(mapping, question, display); ok {
			display = translated
		}
		canonical, valid := canonicalAnswer(question.QuestionType, question.Options, question.CategoryLabels, display, false)
		if !valid {
			continue
		}
		if question.QuestionType == domain.QuestionTypeEssay {
			// Jawaban esai disimpan untuk dinilai manual dan tidak dihitung dalam skor otomatis.
			batch = append(batch, domain.UserAnswer{UserExamID: userExamID, QuestionID: questionID, SelectedOption: canonical, IsCorrect: false})
			continue
		}
		normalizedAnswers[questionID] = canonical
		correct := canonical == question.CorrectAnswer
		batch = append(batch, domain.UserAnswer{UserExamID: userExamID, QuestionID: questionID, SelectedOption: canonical, IsCorrect: correct})
	}
	score := CalculateExamScore(exam.ScoringMethod, questions, normalizedAnswers)
	finishedAt := s.now().UTC()
	finished, err := s.exams.SubmitUserExam(ctx, userExamID, userID, batch, score, finishedAt)
	if err != nil {
		return nil, err
	}
	locked = false
	if err := s.cache.DeleteState(ctx, userExamID); err != nil {
		// The database commit is authoritative. Redis keys are bounded by TTL and may be retried safely.
		s.logger.WarnContext(ctx, "submitted exam but failed to delete Redis state", "user_exam_id", userExamID, "error", err)
	}
	return submitResponse(finished, exam), nil
}

func (s *CBTService) AutoSubmitTask(ctx context.Context) error {
	items, err := s.exams.ListExpiredUserExams(ctx, autoSubmitBatchSize)
	if err != nil {
		return err
	}
	var failures []error
	for _, item := range items {
		if _, err := s.SubmitExam(ctx, item.UserID, item.UserExamID); err != nil {
			failures = append(failures, fmt.Errorf("auto-submit %s: %w", item.UserExamID, err))
		}
	}
	return errors.Join(failures...)
}

func submitResponse(attempt *domain.UserExam, exam *domain.Exam) *domain.SubmitExamResponse {
	response := &domain.SubmitExamResponse{UserExamID: attempt.ID, ExamID: attempt.ExamID, Status: attempt.Status, PassingScore: exam.PassingScore}
	if attempt.FinishedAt != nil {
		response.FinishedAt = *attempt.FinishedAt
	}
	if attempt.TotalScore != nil {
		response.TotalScore = *attempt.TotalScore
	}
	response.Passed = response.TotalScore >= response.PassingScore
	return response
}

func (s *CBTService) releaseSubmitLock(ctx context.Context, userExamID, userID string) {
	releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	if err := s.cache.ReleaseSubmitLock(releaseCtx, userExamID, userID); err != nil {
		s.logger.WarnContext(releaseCtx, "failed to release Redis submit lock", "user_exam_id", userExamID, "error", err)
	}
}

// userExamShuffle mengambil permutasi attempt yang di-persistenkan untuk
// menyusun tampilan dan menerjemahkan jawaban peserta. Bila repository tidak
// menyediakannya atau gagal, urutan kanonis digunakan (perilaku lama).
func (s *CBTService) userExamShuffle(ctx context.Context, userExamID string) *domain.UserExamShuffle {
	shuffler, ok := s.exams.(domain.ExamShuffleRepository)
	if !ok {
		return &domain.UserExamShuffle{}
	}
	mapping, err := shuffler.EnsureUserExamShuffle(ctx, userExamID)
	if err != nil {
		s.logger.WarnContext(ctx, "shuffle mapping unavailable; using canonical order", "user_exam_id", userExamID, "error", err)
		return &domain.UserExamShuffle{}
	}
	return mapping
}

func validUUID(value string) bool { _, err := uuid.Parse(value); return err == nil }
func remainingSeconds(expiresAt, now time.Time) int64 {
	remaining := time.Until(expiresAt)
	if !now.IsZero() {
		remaining = expiresAt.Sub(now)
	}
	if remaining <= 0 {
		return 0
	}
	return int64(math.Ceil(remaining.Seconds()))
}

var _ domain.CBTService = (*CBTService)(nil)
