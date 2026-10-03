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

// fakeScreenLockRepository meniru penyimpanan status kunci per attempt.
type fakeScreenLockRepository struct {
	lock          domain.ExamScreenLock
	upsertCalls   int
	upsertEvents  []string
	upsertLockFor time.Duration
	getCalls      int
	releaseCalls  int
	upsertErr     error
}

func (f *fakeScreenLockRepository) UpsertExamScreenLock(_ context.Context, _, event string, now time.Time, lockFor time.Duration) (*domain.ExamScreenLock, error) {
	if f.upsertErr != nil {
		return nil, f.upsertErr
	}
	f.upsertCalls++
	f.upsertEvents = append(f.upsertEvents, event)
	f.upsertLockFor = lockFor
	f.lock.ViolationCount++
	f.lock.LastEvent = event
	f.lock.LockedAt = now
	f.lock.UnlockUntil = now.Add(lockFor)
	f.lock.ReleasedAt = nil
	current := f.lock
	return &current, nil
}

func (f *fakeScreenLockRepository) GetExamScreenLock(_ context.Context, _ string, _ time.Time) (*domain.ExamScreenLock, error) {
	f.getCalls++
	if f.lock.ViolationCount == 0 {
		return nil, nil
	}
	current := f.lock
	return &current, nil
}

func (f *fakeScreenLockRepository) ReleaseExamScreenLock(_ context.Context, _ string, now time.Time) (*domain.ExamScreenLock, error) {
	f.releaseCalls++
	f.lock.ReleasedAt = &now
	current := f.lock
	return &current, nil
}

// newScreenLockService membuat service dengan jam server yang tetap agar
// hitung mundur 5 detik bisa diuji tanpa ketergantungan waktu nyata.
func newScreenLockService(t *testing.T, examType string, locks domain.ExamScreenLockRepository, now time.Time) *CBTService {
	t.Helper()
	exams := &fakeExamRepository{
		exam:    domain.Exam{ID: testExamID, Title: "Ujian", DurationMinutes: 60, PackageExamType: examType, ScreenLockEnabled: true},
		attempt: domain.UserExam{ID: testAttemptID, ExamID: testExamID, UserID: testUserID, Status: "ongoing", StartedAt: now},
	}
	service := NewCBTService(exams, &fakeCBTRepository{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if locks != nil {
		service = service.WithScreenLockRepository(locks)
	}
	service.now = func() time.Time { return now }
	return service
}

func TestScreenLockEnforcedHanyaUntukCBT(t *testing.T) {
	for _, tc := range []struct {
		examType string
		enabled  bool
		want     bool
	}{
		{"cbt", true, true},
		{"CBT", true, true},
		{" cbt ", true, true},
		{"sell", true, false},
		{"SELL", true, false},
		{"", true, false},
		{"cbt", false, false},
	} {
		locks := &fakeScreenLockRepository{}
		service := newScreenLockService(t, tc.examType, locks, time.Now())
		if got := service.screenLockEnforced(tc.examType, tc.enabled); got != tc.want {
			t.Fatalf("screenLockEnforced(%q, %v) = %v, want %v", tc.examType, tc.enabled, got, tc.want)
		}
	}
}

func TestScreenLockNonaktifTanpaRepository(t *testing.T) {
	now := time.Now().UTC()
	service := newScreenLockService(t, "cbt", nil, now)

	response, err := service.ReportViolation(context.Background(), testUserID, testAttemptID, "tab_hidden")
	if err != nil {
		t.Fatalf("ReportViolation error: %v", err)
	}
	if response.Enforced || response.Locked {
		t.Fatalf("repository lock tidak terpasang, harus enforced=false: %+v", response)
	}
}

func TestReportViolationMengunciLimaDetikDanMenaikkanPelanggaran(t *testing.T) {
	now := time.Now().UTC()
	locks := &fakeScreenLockRepository{}
	service := newScreenLockService(t, "cbt", locks, now)

	first, err := service.ReportViolation(context.Background(), testUserID, testAttemptID, "tab_hidden")
	if err != nil {
		t.Fatalf("ReportViolation error: %v", err)
	}
	if !first.Enforced || !first.Locked {
		t.Fatalf("violation pada mode CBT harus mengunci layar: %+v", first)
	}
	if first.ViolationCount != 1 {
		t.Fatalf("violation count = %d, want 1", first.ViolationCount)
	}
	if first.LockSeconds != ScreenLockSeconds || ScreenLockSeconds != 5 {
		t.Fatalf("lock seconds = %d, want 5", first.LockSeconds)
	}
	if first.UnlockUntil == nil {
		t.Fatal("unlock_until harus diisi saat terkunci")
	}
	if want := now.Add(ScreenLockSeconds * time.Second); !first.UnlockUntil.Equal(want) {
		t.Fatalf("unlock_until = %v, want %v", first.UnlockUntil, want)
	}
	if locks.upsertLockFor != 5*time.Second {
		t.Fatalf("durasi lock repository = %v, want 5s", locks.upsertLockFor)
	}

	second, err := service.ReportViolation(context.Background(), testUserID, testAttemptID, "window_blurred")
	if err != nil {
		t.Fatalf("ReportViolation kedua error: %v", err)
	}
	if second.ViolationCount != 2 {
		t.Fatalf("violation count = %d, want 2", second.ViolationCount)
	}
}

func TestReportViolationTidakMengunciModeSell(t *testing.T) {
	now := time.Now().UTC()
	locks := &fakeScreenLockRepository{}
	service := newScreenLockService(t, "sell", locks, now)

	response, err := service.ReportViolation(context.Background(), testUserID, testAttemptID, "tab_hidden")
	if err != nil {
		t.Fatalf("ReportViolation error: %v", err)
	}
	if response.Enforced || response.Locked {
		t.Fatalf("mode sell tidak boleh dikunci: %+v", response)
	}
	if locks.upsertCalls != 0 {
		t.Fatalf("repository lock tidak boleh dipanggil pada mode sell, dipanggil %d kali", locks.upsertCalls)
	}
}

func TestReportViolationMenolakAttemptAlihOrang(t *testing.T) {
	now := time.Now().UTC()
	locks := &fakeScreenLockRepository{}
	service := newScreenLockService(t, "cbt", locks, now)

	// GetUserExam pada fake tidak memfilter user, jadi validasi pertama yang
	// diuji adalah format UUID yang tidak valid.
	if _, err := service.ReportViolation(context.Background(), "bukan-uuid", testAttemptID, "tab_hidden"); !errors.Is(err, domain.ErrUserExamNotFound) {
		t.Fatalf("err = %v, want ErrUserExamNotFound", err)
	}
	if _, err := service.ReportViolation(context.Background(), testUserID, "bukan-uuid", "tab_hidden"); !errors.Is(err, domain.ErrUserExamNotFound) {
		t.Fatalf("err = %v, want ErrUserExamNotFound", err)
	}
	if locks.upsertCalls != 0 {
		t.Fatal("repository lock tidak boleh dipanggil untuk request tidak valid")
	}
}

func TestNormalizeLockEventMemangkasLabelPanjang(t *testing.T) {
	long := ""
	for i := 0; i < maxLockEventLength+20; i++ {
		long += "a"
	}
	got := normalizeLockEvent("  tab   hidden\n")
	if got != "tabhidden" {
		t.Fatalf("normalizeLockEvent = %q, want %q", got, "tabhidden")
	}
	if trimmed := normalizeLockEvent(long); len(trimmed) != maxLockEventLength {
		t.Fatalf("panjang hasil = %d, want %d", len(trimmed), maxLockEventLength)
	}
}

func TestGetScreenLockTidakMenaikkanPelanggaran(t *testing.T) {
	now := time.Now().UTC()
	locks := &fakeScreenLockRepository{}
	service := newScreenLockService(t, "cbt", locks, now)

	if _, err := service.ReportViolation(context.Background(), testUserID, testAttemptID, "tab_hidden"); err != nil {
		t.Fatalf("ReportViolation error: %v", err)
	}

	locked, err := service.GetScreenLock(context.Background(), testUserID, testAttemptID)
	if err != nil {
		t.Fatalf("GetScreenLock error: %v", err)
	}
	if !locked.Enforced || !locked.Locked || locked.ViolationCount != 1 {
		t.Fatalf("status lock = %+v, want locked dengan 1 pelanggaran", locked)
	}
	if locks.upsertCalls != 1 {
		t.Fatalf("polling tidak boleh menambah pelanggaran, upsert dipanggil %d kali", locks.upsertCalls)
	}

	// Setelah guru melepas kunci, polling berikutnya harus melepas kunci
	// sehingga siswa bisa melanjutkan tanpa menunggu hitung mundur.
	released := now.Add(2 * time.Second)
	if _, err := locks.ReleaseExamScreenLock(context.Background(), testAttemptID, released); err != nil {
		t.Fatalf("ReleaseExamScreenLock error: %v", err)
	}
	service.now = func() time.Time { return now.Add(3 * time.Second) }

	opened, err := service.GetScreenLock(context.Background(), testUserID, testAttemptID)
	if err != nil {
		t.Fatalf("GetScreenLock setelah release error: %v", err)
	}
	if opened.Locked {
		t.Fatalf("kunci harus terbuka setelah guru melepas: %+v", opened)
	}
	if opened.ViolationCount != 1 {
		t.Fatalf("violation count setelah release = %d, want 1", opened.ViolationCount)
	}
}

func TestGetScreenLockOtomatisTerbukaSetelahLimaDetik(t *testing.T) {
	now := time.Now().UTC()
	locks := &fakeScreenLockRepository{}
	service := newScreenLockService(t, "cbt", locks, now)

	if _, err := service.ReportViolation(context.Background(), testUserID, testAttemptID, "tab_hidden"); err != nil {
		t.Fatalf("ReportViolation error: %v", err)
	}

	// Detik ke-5 masih terkunci karena unlock_until = now + 5s.
	service.now = func() time.Time { return now.Add(4900 * time.Millisecond) }
	stillLocked, err := service.GetScreenLock(context.Background(), testUserID, testAttemptID)
	if err != nil {
		t.Fatalf("GetScreenLock error: %v", err)
	}
	if !stillLocked.Locked {
		t.Fatal("kunci harus masih aktif sebelum 5 detik berlalu")
	}

	// Setelah 5 detik, kunci terbuka sendiri tanpa perlu unlock guru.
	service.now = func() time.Time { return now.Add(6 * time.Second) }
	expired, err := service.GetScreenLock(context.Background(), testUserID, testAttemptID)
	if err != nil {
		t.Fatalf("GetScreenLock error: %v", err)
	}
	if expired.Locked {
		t.Fatalf("kunci harus terbuka otomatis setelah 5 detik: %+v", expired)
	}
	if expired.UnlockUntil != nil {
		t.Fatalf("unlock_until harus kosong saat tidak terkunci: %v", expired.UnlockUntil)
	}
}

func TestGetScreenLockBelumPernahDilanggar(t *testing.T) {
	now := time.Now().UTC()
	locks := &fakeScreenLockRepository{}
	service := newScreenLockService(t, "cbt", locks, now)

	response, err := service.GetScreenLock(context.Background(), testUserID, testAttemptID)
	if err != nil {
		t.Fatalf("GetScreenLock error: %v", err)
	}
	if !response.Enforced || response.Locked || response.ViolationCount != 0 {
		t.Fatalf("status awal = %+v, want enforced tanpa kunci", response)
	}
}

func TestGetScreenLockTidakAktifPadaModeSell(t *testing.T) {
	now := time.Now().UTC()
	locks := &fakeScreenLockRepository{}
	service := newScreenLockService(t, "sell", locks, now)

	response, err := service.GetScreenLock(context.Background(), testUserID, testAttemptID)
	if err != nil {
		t.Fatalf("GetScreenLock error: %v", err)
	}
	if response.Enforced || response.Locked {
		t.Fatalf("mode sell tidak boleh terkunci: %+v", response)
	}
	if locks.getCalls != 0 {
		t.Fatalf("repository lock tidak boleh dibaca pada mode sell, dipanggil %d kali", locks.getCalls)
	}
}

func TestStartExamMenyertakanFlagPenguncianLayar(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		examType string
		token    string
		want     bool
	}{
		{"cbt", "9K4PT2", true},
		{"sell", "", false},
	} {
		locks := &fakeScreenLockRepository{}
		exams := &fakeExamRepository{
			exam:    domain.Exam{ID: testExamID, Title: "Ujian", DurationMinutes: 60, PackageExamType: tc.examType, PackageCBTToken: tc.token, ScreenLockEnabled: true},
			attempt: domain.UserExam{ID: testAttemptID, ExamID: testExamID, UserID: testUserID, Status: "ongoing", StartedAt: now},
		}
		service := NewCBTService(exams, &fakeCBTRepository{}, slog.New(slog.NewTextHandler(io.Discard, nil))).WithScreenLockRepository(locks)
		service.now = func() time.Time { return now }

		response, err := service.StartExam(context.Background(), testUserID, testExamID, tc.token)
		if err != nil {
			t.Fatalf("StartExam(%s) error: %v", tc.examType, err)
		}
		if response.ExamType != tc.examType {
			t.Fatalf("exam_type = %q, want %q", response.ExamType, tc.examType)
		}
		if response.ScreenLockEnabled != tc.want {
			t.Fatalf("screen_lock_enabled untuk %q = %v, want %v", tc.examType, response.ScreenLockEnabled, tc.want)
		}
	}
}

func TestReportViolationTidakMengunciBilaPengaturanUjianMati(t *testing.T) {
	now := time.Now().UTC()
	locks := &fakeScreenLockRepository{}
	exams := &fakeExamRepository{
		exam:    domain.Exam{ID: testExamID, Title: "Ujian", DurationMinutes: 60, PackageExamType: "cbt", ScreenLockEnabled: false},
		attempt: domain.UserExam{ID: testAttemptID, ExamID: testExamID, UserID: testUserID, Status: "ongoing", StartedAt: now},
	}
	service := NewCBTService(exams, &fakeCBTRepository{}, slog.New(slog.NewTextHandler(io.Discard, nil))).WithScreenLockRepository(locks)
	service.now = func() time.Time { return now }

	response, err := service.ReportViolation(context.Background(), testUserID, testAttemptID, "tab_hidden")
	if err != nil {
		t.Fatalf("ReportViolation error: %v", err)
	}
	if response.Enforced || response.Locked {
		t.Fatalf("blokir dimatikan pada ujian, harus enforced=false: %+v", response)
	}
	if locks.upsertCalls != 0 {
		t.Fatalf("repository lock tidak boleh dipanggil, dipanggil %d kali", locks.upsertCalls)
	}
}

func TestReportViolationMemakaiDurasiKustomUjian(t *testing.T) {
	now := time.Now().UTC()
	locks := &fakeScreenLockRepository{}
	exams := &fakeExamRepository{
		exam:    domain.Exam{ID: testExamID, Title: "Ujian", DurationMinutes: 60, PackageExamType: "cbt", ScreenLockEnabled: true, ScreenLockSeconds: 12},
		attempt: domain.UserExam{ID: testAttemptID, ExamID: testExamID, UserID: testUserID, Status: "ongoing", StartedAt: now},
	}
	service := NewCBTService(exams, &fakeCBTRepository{}, slog.New(slog.NewTextHandler(io.Discard, nil))).WithScreenLockRepository(locks)
	service.now = func() time.Time { return now }

	response, err := service.ReportViolation(context.Background(), testUserID, testAttemptID, "tab_hidden")
	if err != nil {
		t.Fatalf("ReportViolation error: %v", err)
	}
	if response.LockSeconds != 12 {
		t.Fatalf("lock_seconds = %d, want 12", response.LockSeconds)
	}
	if locks.upsertLockFor != 12*time.Second {
		t.Fatalf("durasi lock repository = %v, want 12s", locks.upsertLockFor)
	}
	if response.UnlockUntil == nil || !response.UnlockUntil.Equal(now.Add(12*time.Second)) {
		t.Fatalf("unlock_until = %v, want %v", response.UnlockUntil, now.Add(12*time.Second))
	}
}
