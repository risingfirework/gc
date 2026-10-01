package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tka/apps/backend/internal/domain"
)

// ExamScreenLockRepository mengimplementasikan penyimpanan status layar kunci
// satu attempt ujian pada tabel user_exam_screen_locks.
type ExamScreenLockRepository struct {
	db *pgxpool.Pool
}

func NewExamScreenLockRepository(db *pgxpool.Pool) *ExamScreenLockRepository {
	return &ExamScreenLockRepository{db: db}
}

// UpsertExamScreenLock menambah penghitung pelanggaran, memperbarui event
// terakhir, dan menggeser unlock_until. released_at selalu di-reset supaya
// pelanggaran berikutnya mengunci layar lagi.
func (r *ExamScreenLockRepository) UpsertExamScreenLock(ctx context.Context, userExamID, event string, now time.Time, lockFor time.Duration) (*domain.ExamScreenLock, error) {
	const query = `
		INSERT INTO user_exam_screen_locks
		    (user_exam_id, violation_count, locked_at, unlock_until, last_event, updated_at)
		VALUES ($1, 1, $2, $3, $4, $2)
		ON CONFLICT (user_exam_id) DO UPDATE SET
		    violation_count = user_exam_screen_locks.violation_count + 1,
		    locked_at = EXCLUDED.locked_at,
		    unlock_until = EXCLUDED.unlock_until,
		    last_event = EXCLUDED.last_event,
		    released_at = NULL,
		    updated_at = EXCLUDED.updated_at
		RETURNING user_exam_id, violation_count, locked_at, unlock_until, released_at, last_event`
	lock, err := scanExamScreenLock(r.db.QueryRow(ctx, query, userExamID, now, now.Add(lockFor), event))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserExamNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("upsert exam screen lock: %w", err)
	}
	return lock, nil
}

// GetExamScreenLock mengembalikan baris kunci atau nil bila belum pernah ada
// pelanggaran. Locked dihitung di sisi aplikasi memakai now sehingga lock yang
// sudah lewat otomatis teranggap selesai tanpa perlu update.
func (r *ExamScreenLockRepository) GetExamScreenLock(ctx context.Context, userExamID string, now time.Time) (*domain.ExamScreenLock, error) {
	const query = `
		SELECT user_exam_id, violation_count, locked_at, unlock_until, released_at, last_event
		FROM user_exam_screen_locks
		WHERE user_exam_id = $1`
	lock, err := scanExamScreenLock(r.db.QueryRow(ctx, query, userExamID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get exam screen lock: %w", err)
	}
	lock.Locked = lock.IsActiveAt(now)
	return lock, nil
}

// ReleaseExamScreenLock menandai kunci dilepas oleh guru/admin.
func (r *ExamScreenLockRepository) ReleaseExamScreenLock(ctx context.Context, userExamID string, now time.Time) (*domain.ExamScreenLock, error) {
	const query = `
		UPDATE user_exam_screen_locks
		SET released_at = $2, updated_at = $2
		WHERE user_exam_id = $1
		  AND released_at IS NULL
		RETURNING user_exam_id, violation_count, locked_at, unlock_until, released_at, last_event`
	lock, err := scanExamScreenLock(r.db.QueryRow(ctx, query, userExamID, now))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrScreenLockNotLocked
	}
	if err != nil {
		return nil, fmt.Errorf("release exam screen lock: %w", err)
	}
	lock.Locked = lock.IsActiveAt(now)
	return lock, nil
}

// ReleaseParticipantScreenLockAdmin membuka blokir layar seorang siswa. Admin
// berhak membuka blokir pada ujian mana pun, termasuk paket yang bukan miliknya.
func (r *ExamScreenLockRepository) ReleaseParticipantScreenLockAdmin(ctx context.Context, examID, userExamID string, now time.Time) (*domain.ExamScreenLock, error) {
	const query = `
		UPDATE user_exam_screen_locks AS l
		SET released_at = $3, updated_at = $3
		FROM user_exams ue
		WHERE l.user_exam_id = ue.id
		  AND ue.exam_id = $1
		  AND ue.id = $2
		  AND l.released_at IS NULL
		  AND l.unlock_until > $3
		  AND ue.status = 'ongoing'
		RETURNING l.user_exam_id, l.violation_count, l.locked_at, l.unlock_until, l.released_at, l.last_event`
	lock, err := scanExamScreenLock(r.db.QueryRow(ctx, query, examID, userExamID, now))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrScreenLockNotLocked
	}
	if err != nil {
		return nil, fmt.Errorf("release admin screen lock: %w", err)
	}
	lock.Locked = false
	return lock, nil
}

// ReleaseParticipantScreenLockTeacher membuka blokir layar seorang siswa untuk
// guru; dibatasi pada paket miliknya sendiri.
func (r *ExamScreenLockRepository) ReleaseParticipantScreenLockTeacher(ctx context.Context, publisherID, examID, userExamID string, now time.Time) (*domain.ExamScreenLock, error) {
	const query = `
		UPDATE user_exam_screen_locks AS l
		SET released_at = $4, updated_at = $4
		FROM user_exams ue, exams e, packages p
		WHERE l.user_exam_id = ue.id
		  AND ue.exam_id = e.id
		  AND e.package_id = p.id
		  AND p.publisher_id = $1
		  AND e.id = $2
		  AND ue.id = $3
		  AND l.released_at IS NULL
		  AND l.unlock_until > $4
		  AND ue.status = 'ongoing'
		RETURNING l.user_exam_id, l.violation_count, l.locked_at, l.unlock_until, l.released_at, l.last_event`
	lock, err := scanExamScreenLock(r.db.QueryRow(ctx, query, publisherID, examID, userExamID, now))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrScreenLockNotLocked
	}
	if err != nil {
		return nil, fmt.Errorf("release teacher screen lock: %w", err)
	}
	lock.Locked = false
	return lock, nil
}

func scanExamScreenLock(row pgx.Row) (*domain.ExamScreenLock, error) {
	var lock domain.ExamScreenLock
	if err := row.Scan(
		&lock.UserExamID,
		&lock.ViolationCount,
		&lock.LockedAt,
		&lock.UnlockUntil,
		&lock.ReleasedAt,
		&lock.LastEvent,
	); err != nil {
		return nil, err
	}
	lock.Locked = lock.ReleasedAt == nil
	return &lock, nil
}
