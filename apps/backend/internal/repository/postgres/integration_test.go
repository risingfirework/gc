//go:build integration

package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tka/apps/backend/internal/domain"
)

// Integration test terhadap Postgres sungguhan.
// Cara menjalankan (butuh container tka-postgres, lihat docker-compose.yml):
//
//	go test -tags integration ./internal/repository/postgres/ -v
//
// Test membuat database throwaway `tka_integration_test`, menerapkan seluruh
// migrasi naik, lalu menjalankan skenario repository. Database di-drop ulang
// pada tiap run sehingga aman terhadap data development.

const (
	integrationDBName     = "tka_integration_test"
	defaultAdminURLPrefix = "postgres://tka:tka_local_password@127.0.0.1:55432/"
	defaultTestURLPrefix  = "postgres://tka:tka_local_password@127.0.0.1:55432/" + integrationDBName + "?sslmode=disable"
)

func waitForTestDatabase(t *testing.T) (adminConn *pgx.Conn, testURL string) {
	t.Helper()
	adminURL := os.Getenv("TKA_TEST_ADMIN_URL")
	if adminURL == "" {
		adminURL = defaultAdminURLPrefix + "postgres?sslmode=disable"
	}
	if os.Getenv("TKA_TEST_DATABASE_URL") != "" {
		testURL = os.Getenv("TKA_TEST_DATABASE_URL")
	} else {
		testURL = defaultTestURLPrefix
	}

	ctx := context.Background()
	config, err := pgx.ParseConfig(adminURL)
	if err != nil {
		t.Fatalf("parse admin url: %v", err)
	}
	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatalf("connect admin postgres: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })

	for _, stmt := range []string{
		"DROP DATABASE IF EXISTS " + integrationDBName + " WITH (FORCE)",
		"CREATE DATABASE " + integrationDBName,
	} {
		if _, err := conn.Exec(ctx, stmt); err != nil {
			t.Fatalf("reset integration database (%q): %v", stmt, err)
		}
	}
	return conn, testURL
}

func applyMigrations(t *testing.T, testURL string) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	config, err := pgx.ParseConfig(testURL)
	if err != nil {
		t.Fatalf("parse test url: %v", err)
	}
	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatalf("connect integration database: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })

	dir := os.Getenv("TKA_MIGRATIONS_DIR")
	if dir == "" {
		dir = "../../../db/migrations"
	}
	upPaths, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	sort.Strings(upPaths)
	for _, path := range upPaths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", path, err)
		}
		if _, err := conn.Exec(ctx, string(raw)); err != nil {
			t.Fatalf("apply migration %s: %v", filepath.Base(path), err)
		}
	}

	pool, err := pgxpool.New(ctx, testURL)
	if err != nil {
		t.Fatalf("open test pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestIntegrationMigrateUpDownRoundTrip(t *testing.T) {
	_, testURL := waitForTestDatabase(t)
	pool := applyMigrations(t, testURL)
	ctx := context.Background()

	// Terapkan down lalu up lagi pada 000048 (index housekeeping) untuk
	// memastikan index benar-benar bisa dibalik.
	down, err := os.ReadFile(filepath.Join("../../../db/migrations/000048_index_housekeeping.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("apply 000048 down: %v", err)
	}
	assertIndexExists(t, ctx, pool, "package_views_package_idx", true)

	up, err := os.ReadFile(filepath.Join("../../../db/migrations/000048_index_housekeeping.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("re-apply 000048 up: %v", err)
	}
	assertIndexExists(t, ctx, pool, "package_views_package_idx", false)
	assertIndexExists(t, ctx, pool, "user_exams_submitted_rank_idx", true)
	assertIndexExists(t, ctx, pool, "users_school_level_idx", true)
	assertIndexExists(t, ctx, pool, "packages_exam_type_status_idx", true)
	assertIndexExists(t, ctx, pool, "notifications_unread_idx", true)
}

func assertIndexExists(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name string, want bool) {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = $1)`, name).Scan(&exists); err != nil {
		t.Fatalf("query index %s: %v", name, err)
	}
	if exists != want {
		t.Fatalf("index %s existence = %v, want %v", name, exists, want)
	}
}

func TestIntegrationUserRepositorySmoke(t *testing.T) {
	_, testURL := waitForTestDatabase(t)
	pool := applyMigrations(t, testURL)
	ctx := context.Background()
	repos := &ExamRepository{db: pool}
	userRepo := &UserRepository{db: pool}

	user := &domain.User{Email: "Ngara.TKA@Example.COM", PasswordHash: "hash", Role: "student", SchoolLevel: "SMA"}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.ID == "" {
		t.Fatal("create user did not return id")
	}

	// Login lookup case-insensitive memakai index LOWER(email).
	found, err := userRepo.FindByEmail(ctx, "ngara.tka@example.com")
	if err != nil {
		t.Fatalf("find by email: %v", err)
	}
	if found.ID != user.ID {
		t.Fatalf("find by email returned different user")
	}

	// Email duplikat di-tolak via index unik LOWER(email).
	dup := &domain.User{Email: strings.ToUpper(user.Email), PasswordHash: "hash", Role: "student", SchoolLevel: "SMA"}
	if err := userRepo.Create(ctx, dup); !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Fatalf("duplicate email error = %v, want ErrEmailAlreadyExists", err)
	}

	if repos == nil {
		t.Fatal("unreachable")
	}
}

func seedExamPackage(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (packageID, examID string, questionIDs []string) {
	t.Helper()
	if err := pool.QueryRow(ctx, `INSERT INTO packages (title, description, price, validity_days, status, exam_type, jenjang) VALUES ('Paket TKA', '', 50000, 7, 'active', 'cbt', 'SMA') RETURNING id`).Scan(&packageID); err != nil {
		t.Fatalf("insert package: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO exams (package_id, title, duration_minutes, total_questions, passing_score, status) VALUES ($1, 'Tryout UTBK', 60, 2, 50, 'active') RETURNING id`, packageID).Scan(&examID); err != nil {
		t.Fatalf("insert exam: %v", err)
	}
	for i, answer := range []string{"A", "B"} {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO questions (exam_id, subject_name, content_text, question_type, presentation_type, options_json, correct_answer, score_weight, status) VALUES ($1, 'Matematika', $2, 'single_choice', 'single', $3, $4, 1, 'active') RETURNING id`,
			examID, fmt.Sprintf("Soal %d", i+1), `[{"key":"A","content":"satu"},{"key":"B","content":"dua"}]`, answer).Scan(&id); err != nil {
			t.Fatalf("insert question %d: %v", i, err)
		}
		questionIDs = append(questionIDs, id)
	}
	return packageID, examID, questionIDs
}

func TestIntegrationExamLifecycleAndShuffle(t *testing.T) {
	_, testURL := waitForTestDatabase(t)
	pool := applyMigrations(t, testURL)
	ctx := context.Background()

	repos := &ExamRepository{db: pool}
	userRepo := &UserRepository{db: pool}
	user := &domain.User{Email: "siswa@example.com", PasswordHash: "hash", Role: "student", SchoolLevel: "SMA"}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	packageID, examID, questionIDs := seedExamPackage(t, ctx, pool)

	now := time.Now().UTC()

	// StartOrGetUserExam: attempt ongoing, idempotent pada okupansi yang sama.
	attempt, err := repos.StartOrGetUserExam(ctx, user.ID, examID, now)
	if err != nil {
		t.Fatalf("start exam: %v", err)
	}
	attemptAgain, err := repos.StartOrGetUserExam(ctx, user.ID, examID, now.Add(time.Second))
	if err != nil {
		t.Fatalf("restart same exam: %v", err)
	}
	if attempt.ID != attemptAgain.ID || attempt.Status != "ongoing" {
		t.Fatalf("same ongoing attempt expected, got %s vs %s", attempt.ID, attemptAgain.ID)
	}

	// Shuffle nonaktif -> tidak ada permutasi, dan kolom JSON tetap NULL.
	shuffle, err := repos.EnsureUserExamShuffle(ctx, attempt.ID)
	if err != nil {
		t.Fatalf("ensure shuffle off: %v", err)
	}
	if len(shuffle.QuestionOrder) != 0 || len(shuffle.OptionOrder) != 0 {
		t.Fatalf("shuffle must be empty when flags off: %+v", shuffle)
	}
	var storedJSON []byte
	if err := pool.QueryRow(ctx, `SELECT question_order_json FROM user_exams WHERE id = $1`, attempt.ID).Scan(&storedJSON); err != nil {
		t.Fatalf("read question_order_json: %v", err)
	}
	if storedJSON != nil {
		t.Fatalf("question_order_json must be NULL when shuffle is off")
	}

	// Aktifkan shuffle. Attempt yang sudah berjalan mempertahankan snapshot
	// nonaktif; attempt baru menghasilkan permutasi yang stabil.
	if _, err := pool.Exec(ctx, `UPDATE exams SET shuffle_questions = TRUE, shuffle_options = TRUE WHERE id = $1`, examID); err != nil {
		t.Fatalf("enable shuffle: %v", err)
	}
	unchanged, err := repos.EnsureUserExamShuffle(ctx, attempt.ID)
	if err != nil {
		t.Fatalf("ensure existing attempt remains unshuffled: %v", err)
	}
	if len(unchanged.QuestionOrder) != 0 || len(unchanged.OptionOrder) != 0 {
		t.Fatalf("existing attempt must retain its shuffle snapshot: %+v", unchanged)
	}
	secondUser := &domain.User{Email: "siswa-shuffle@example.com", PasswordHash: "hash", Role: "student", SchoolLevel: "SMA"}
	if err := userRepo.Create(ctx, secondUser); err != nil {
		t.Fatalf("create shuffled user: %v", err)
	}
	shuffledAttempt, err := repos.StartOrGetUserExam(ctx, secondUser.ID, examID, now)
	if err != nil {
		t.Fatalf("start shuffled exam: %v", err)
	}
	first, err := repos.EnsureUserExamShuffle(ctx, shuffledAttempt.ID)
	if err != nil {
		t.Fatalf("ensure shuffle on: %v", err)
	}
	second, err := repos.EnsureUserExamShuffle(ctx, shuffledAttempt.ID)
	if err != nil {
		t.Fatalf("ensure shuffle repeated: %v", err)
	}
	if !sameShuffle(first, second) {
		t.Fatalf("persisted shuffle changed between calls: %+v vs %+v", first, second)
	}
	if len(first.QuestionOrder) != len(questionIDs) {
		t.Fatalf("question order must cover all questions: %+v", first.QuestionOrder)
	}
	for _, id := range questionIDs {
		keys := first.OptionOrder[id]
		if len(keys) != 2 {
			t.Fatalf("option order for %s must cover all keys: %v", id, keys)
		}
		seen := map[string]bool{}
		for _, key := range keys {
			if key != "A" && key != "B" {
				t.Fatalf("unexpected option key %q", key)
			}
			seen[key] = true
		}
		if !seen["A"] || !seen["B"] {
			t.Fatalf("option order for %s missing key: %v", id, keys)
		}
	}
	if err := pool.QueryRow(ctx, `SELECT question_order_json FROM user_exams WHERE id = $1`, shuffledAttempt.ID).Scan(&storedJSON); err != nil {
		t.Fatalf("re-read question_order_json: %v", err)
	}
	if storedJSON == nil {
		t.Fatal("question_order_json must be persisted when shuffle is on")
	}

	// GetUserExam melengkapi data exam; GetExamWithQuestions mengembalikan soal.
	gotAttempt, exam, err := repos.GetUserExam(ctx, attempt.ID, user.ID)
	if err != nil {
		t.Fatalf("get user exam: %v", err)
	}
	if gotAttempt.ID != attempt.ID || exam.ID != examID {
		t.Fatalf("get user exam mismatch: %+v %+v", gotAttempt, exam)
	}
	if exam.PackageExamType != "cbt" {
		t.Fatalf("GetUserExam must load package exam_type, got %q", exam.PackageExamType)
	}
	_, questions, err := repos.GetExamWithQuestions(ctx, examID)
	if err != nil {
		t.Fatalf("get exam with questions: %v", err)
	}
	if len(questions) != 2 {
		t.Fatalf("questions = %d, want 2", len(questions))
	}

	// Submit: satu attempt submitted, idempotent, lalu attempt baru bisa dibuat.
	answers := []domain.UserAnswer{
		{UserExamID: attempt.ID, QuestionID: questionIDs[0], SelectedOption: "A", IsCorrect: true},
		{UserExamID: attempt.ID, QuestionID: questionIDs[1], SelectedOption: "B", IsCorrect: true},
	}
	submitted, err := repos.SubmitUserExam(ctx, attempt.ID, user.ID, answers, 100, now.Add(30*time.Minute))
	if err != nil {
		t.Fatalf("submit exam: %v", err)
	}
	if submitted.Status != "submitted" || submitted.TotalScore == nil || *submitted.TotalScore != 100 {
		t.Fatalf("submitted attempt mismatch: %+v", submitted)
	}
	if _, err := repos.SubmitUserExam(ctx, attempt.ID, user.ID, answers, 100, now.Add(31*time.Minute)); err != nil {
		t.Fatalf("resubmit must be idempotent: %v", err)
	}
	nextAttempt, err := repos.StartOrGetUserExam(ctx, user.ID, examID, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("start new attempt after submit: %v", err)
	}
	if nextAttempt.ID == attempt.ID {
		t.Fatal("new attempt must be distinct from the submitted one")
	}

	// Access: lisensi aktif -> HasActivePackage true, expired -> false.
	if err := pool.QueryRow(ctx, `INSERT INTO user_packages (user_id, package_id, expired_at, status) VALUES ($1, $2, $3, 'active') ON CONFLICT (user_id, package_id) DO UPDATE SET expired_at = EXCLUDED.expired_at, status = 'active'`,
		user.ID, packageID, now.Add(7*24*time.Hour)).Scan(); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("grant package: %v", err)
	}
	active, err := repos.HasActivePackage(ctx, user.ID, packageID, now)
	if err != nil {
		t.Fatalf("has active package: %v", err)
	}
	if !active {
		t.Fatal("HasActivePackage must be true before expiry")
	}
}

func TestIntegrationExamScreenLockSettings(t *testing.T) {
	_, testURL := waitForTestDatabase(t)
	pool := applyMigrations(t, testURL)
	ctx := context.Background()

	exams := &ExamRepository{db: pool}
	users := &UserRepository{db: pool}
	admin := &AdminRepository{db: pool}

	_, examID, _ := seedExamPackage(t, ctx, pool)

	// Default dari migrasi 000053: aktif dengan durasi 5 detik.
	defaults, err := admin.getCBTPublishSetting(ctx, examID)
	if err != nil {
		t.Fatalf("read default cbt setting: %v", err)
	}
	if !defaults.ScreenLockEnabled || defaults.ScreenLockSeconds != 5 {
		t.Fatalf("default screen lock = %v/%d, want true/5", defaults.ScreenLockEnabled, defaults.ScreenLockSeconds)
	}

	updated, err := admin.SetExamScreenLock(ctx, examID, false, 12)
	if err != nil {
		t.Fatalf("set exam screen lock: %v", err)
	}
	if updated.ScreenLockEnabled || updated.ScreenLockSeconds != 12 {
		t.Fatalf("updated screen lock = %v/%d, want false/12", updated.ScreenLockEnabled, updated.ScreenLockSeconds)
	}

	// GetUserExam harus ikut memuat pengaturan blokir agar service CBT
	// mengunci sesuai konfigurasi ujian.
	user := &domain.User{Email: "lock@example.com", PasswordHash: "hash", Role: "student", SchoolLevel: "SMA"}
	if err := users.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	now := time.Now().UTC()
	attempt, err := exams.StartOrGetUserExam(ctx, user.ID, examID, now)
	if err != nil {
		t.Fatalf("start exam: %v", err)
	}
	_, loaded, err := exams.GetUserExam(ctx, attempt.ID, user.ID)
	if err != nil {
		t.Fatalf("get user exam: %v", err)
	}
	if loaded.ScreenLockEnabled || loaded.ScreenLockSeconds != 12 {
		t.Fatalf("GetUserExam screen lock = %v/%d, want false/12", loaded.ScreenLockEnabled, loaded.ScreenLockSeconds)
	}

	// Reset menghapus catatan pelanggaran sehingga status kembali normal.
	if _, err := pool.Exec(ctx, `INSERT INTO user_exam_screen_locks (user_exam_id, violation_count, locked_at, unlock_until) VALUES ($1, 3, $2, $3)`, attempt.ID, now, now.Add(30*time.Second)); err != nil {
		t.Fatalf("seed screen lock: %v", err)
	}
	if err := admin.ResetParticipantScreenLock(ctx, examID, attempt.ID); err != nil {
		t.Fatalf("reset participant screen lock: %v", err)
	}
	var remaining int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_exam_screen_locks WHERE user_exam_id = $1`, attempt.ID).Scan(&remaining); err != nil {
		t.Fatalf("count screen locks: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("screen lock rows after reset = %d, want 0", remaining)
	}

	// Operasi idempoten saat attempt tanpa catatan pelanggaran.
	if err := admin.ResetParticipantScreenLock(ctx, examID, attempt.ID); err != nil {
		t.Fatalf("reset is expected to be idempotent: %v", err)
	}
}

func sameShuffle(a, b *domain.UserExamShuffle) bool {
	if len(a.QuestionOrder) != len(b.QuestionOrder) || len(a.OptionOrder) != len(b.OptionOrder) {
		return false
	}
	for i := range a.QuestionOrder {
		if a.QuestionOrder[i] != b.QuestionOrder[i] {
			return false
		}
	}
	for id, keys := range a.OptionOrder {
		if !equalStrings(b.OptionOrder[id], keys) {
			return false
		}
	}
	return true
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
