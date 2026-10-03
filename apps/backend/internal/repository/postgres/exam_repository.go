package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tka/apps/backend/internal/domain"
)

type ExamRepository struct {
	db *pgxpool.Pool
}

func NewExamRepository(db *pgxpool.Pool) *ExamRepository {
	return &ExamRepository{db: db}
}

func (r *ExamRepository) GetExamWithQuestions(ctx context.Context, examID string) (*domain.Exam, []domain.Question, error) {
	const examQuery = `
		SELECT e.id, e.package_id, e.title, e.duration_minutes, e.total_questions, e.passing_score, e.scoring_method,
		       e.shuffle_questions, e.shuffle_options, e.created_at,
		       p.exam_type, COALESCE(p.cbt_token,''), e.screen_lock_enabled, e.screen_lock_seconds
		FROM exams e
		JOIN packages p ON p.id = e.package_id
		WHERE e.id = $1`
	var exam domain.Exam
	err := r.db.QueryRow(ctx, examQuery, examID).Scan(
		&exam.ID, &exam.PackageID, &exam.Title, &exam.DurationMinutes,
		&exam.TotalQuestions, &exam.PassingScore, &exam.ScoringMethod,
		&exam.ShuffleQuestions, &exam.ShuffleOptions, &exam.CreatedAt,
		&exam.PackageExamType, &exam.PackageCBTToken,
		&exam.ScreenLockEnabled, &exam.ScreenLockSeconds,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, domain.ErrExamNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("query exam: %w", err)
	}

	const questionsQuery = `
		SELECT id, exam_id, subject_name, content_text, question_type, presentation_type,
		       COALESCE(group_code,''), stimulus_text, question_image_url, stimulus_image_url,
		       category_labels_json, options_json, correct_answer, score_weight,
		       difficulty, discrimination, explanation_text, explanation_video_url
		FROM questions
		WHERE exam_id = $1
		ORDER BY id`
	rows, err := r.db.Query(ctx, questionsQuery, examID)
	if err != nil {
		return nil, nil, fmt.Errorf("query exam questions: %w", err)
	}
	defer rows.Close()

	questions := make([]domain.Question, 0, exam.TotalQuestions)
	for rows.Next() {
		var question domain.Question
		var rawOptions []byte
		var rawCategoryLabels []byte
		if err := rows.Scan(
			&question.ID, &question.ExamID, &question.SubjectName, &question.ContentText,
			&question.QuestionType, &question.PresentationType, &question.GroupCode, &question.StimulusText,
			&question.QuestionImageURL, &question.StimulusImageURL,
			&rawCategoryLabels, &rawOptions, &question.CorrectAnswer, &question.ScoreWeight,
			&question.Difficulty, &question.Discrimination, &question.ExplanationText, &question.ExplanationVideoURL,
		); err != nil {
			return nil, nil, fmt.Errorf("scan question: %w", err)
		}
		if err := json.Unmarshal(rawOptions, &question.Options); err != nil {
			return nil, nil, fmt.Errorf("decode options for question %s: %w", question.ID, err)
		}
		if err := json.Unmarshal(rawCategoryLabels, &question.CategoryLabels); err != nil {
			return nil, nil, fmt.Errorf("decode category labels for question %s: %w", question.ID, err)
		}
		questions = append(questions, question)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate questions: %w", err)
	}
	return &exam, questions, nil
}

func (r *ExamRepository) LookupCBTByToken(ctx context.Context, token, userID string) ([]domain.CBTLookupPackage, error) {
	const query = `
		SELECT p.id, p.title, COALESCE(p.kode,''), p.jenjang, p.price,
		       e.id, e.title, e.duration_minutes, e.total_questions, e.passing_score,
		       e.publish_pembahasan,
		       COALESCE(ue.user_exam_id::text,'')
		FROM packages p
		JOIN exams e ON e.package_id = p.id
		LEFT JOIN LATERAL (
			SELECT ue.id AS user_exam_id
			FROM user_exams ue
			WHERE ue.exam_id = e.id AND ue.user_id = $2 AND ue.status = 'submitted'
			ORDER BY ue.finished_at DESC NULLS LAST
			LIMIT 1
		) ue ON true
		WHERE p.exam_type = 'cbt' AND p.status = 'active' AND e.status = 'active'
		  AND UPPER(TRIM(p.cbt_token)) = UPPER($1)
		ORDER BY p.created_at, p.id, e.created_at, e.id`
	rows, err := r.db.Query(ctx, query, token, userID)
	if err != nil {
		return nil, fmt.Errorf("lookup cbt packages: %w", err)
	}
	defer rows.Close()
	packageIndex := map[string]int{}
	result := make([]domain.CBTLookupPackage, 0)
	for rows.Next() {
		var packageItem domain.CBTLookupPackage
		var exam domain.CBTLookupExam
		var attemptID string
		if err := rows.Scan(
			&packageItem.ID, &packageItem.Title, &packageItem.Kode, &packageItem.Jenjang, &packageItem.Price,
			&exam.ExamID, &exam.Title, &exam.DurationMinutes, &exam.TotalQuestions, &exam.PassingScore,
			&exam.PublishPembahasan,
			&attemptID,
		); err != nil {
			return nil, fmt.Errorf("scan cbt lookup: %w", err)
		}
		exam.Submitted = attemptID != ""
		if attemptID != "" {
			exam.UserExamID = attemptID
		}
		index, exists := packageIndex[packageItem.ID]
		if !exists {
			index = len(result)
			packageIndex[packageItem.ID] = index
			result = append(result, packageItem)
		}
		result[index].Exams = append(result[index].Exams, exam)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cbt lookup: %w", err)
	}
	return result, nil
}

func (r *ExamRepository) ListExamsByPackage(ctx context.Context, userID, packageID string) ([]domain.ExamSummary, error) {
	const query = `
		SELECT e.id, e.package_id, e.title, e.duration_minutes, e.total_questions, e.passing_score, e.scoring_method,
		       e.publish_pembahasan, p.exam_type,
		       COALESCE(ur.user_exam_id::text,''), ur.total_score, ur.finished_at
		FROM exams e
		JOIN packages p ON p.id = e.package_id
		LEFT JOIN LATERAL (
			SELECT ue.id AS user_exam_id, ue.total_score, ue.finished_at
			FROM user_exams ue
			WHERE ue.exam_id = e.id AND ue.user_id = $2 AND ue.status = 'submitted'
			ORDER BY ue.finished_at DESC NULLS LAST
			LIMIT 1
		) ur ON true
		WHERE e.package_id = $1 AND e.status = 'active'
		ORDER BY e.created_at, e.id`
	rows, err := r.db.Query(ctx, query, packageID, userID)
	if err != nil {
		return nil, fmt.Errorf("query package exams: %w", err)
	}
	defer rows.Close()
	exams := make([]domain.ExamSummary, 0)
	for rows.Next() {
		var exam domain.ExamSummary
		var attemptID string
		if err := rows.Scan(
			&exam.ID, &exam.PackageID, &exam.Title, &exam.DurationMinutes,
			&exam.TotalQuestions, &exam.PassingScore, &exam.ScoringMethod,
			&exam.PublishPembahasan, &exam.ExamType,
			&attemptID, &exam.TotalScore, &exam.FinishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan package exam: %w", err)
		}
		if attemptID != "" {
			exam.UserExamID = attemptID
		}
		exams = append(exams, exam)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate package exams: %w", err)
	}
	return exams, nil
}

func (r *ExamRepository) StartOrGetUserExam(ctx context.Context, userID, examID string, startedAt time.Time) (*domain.UserExam, error) {
	const query = `
		INSERT INTO user_exams (user_id, exam_id, status, started_at, shuffle_questions, shuffle_options)
		SELECT $1, e.id, 'ongoing', $3, e.shuffle_questions, e.shuffle_options
		FROM exams e WHERE e.id = $2
		ON CONFLICT (user_id, exam_id) WHERE status = 'ongoing'
		DO UPDATE SET user_id = EXCLUDED.user_id
		RETURNING id, user_id, exam_id, status, started_at, finished_at, total_score`
	var attempt domain.UserExam
	err := r.db.QueryRow(ctx, query, userID, examID, startedAt).Scan(
		&attempt.ID, &attempt.UserID, &attempt.ExamID, &attempt.Status,
		&attempt.StartedAt, &attempt.FinishedAt, &attempt.TotalScore,
	)
	if err != nil {
		return nil, fmt.Errorf("start or get user exam: %w", err)
	}
	return &attempt, nil
}

func (r *ExamRepository) GetUserExam(ctx context.Context, userExamID, userID string) (*domain.UserExam, *domain.Exam, error) {
	const query = `
		SELECT ue.id, ue.user_id, ue.exam_id, ue.status, ue.started_at, ue.finished_at, ue.total_score,
		       e.id, e.package_id, e.title, e.duration_minutes, e.total_questions, e.passing_score, e.scoring_method, e.created_at,
		       p.exam_type, e.screen_lock_enabled, e.screen_lock_seconds
		FROM user_exams ue
		JOIN exams e ON e.id = ue.exam_id
		JOIN packages p ON p.id = e.package_id
		WHERE ue.id = $1 AND ue.user_id = $2`
	var attempt domain.UserExam
	var exam domain.Exam
	err := r.db.QueryRow(ctx, query, userExamID, userID).Scan(
		&attempt.ID, &attempt.UserID, &attempt.ExamID, &attempt.Status,
		&attempt.StartedAt, &attempt.FinishedAt, &attempt.TotalScore,
		&exam.ID, &exam.PackageID, &exam.Title, &exam.DurationMinutes,
		&exam.TotalQuestions, &exam.PassingScore, &exam.ScoringMethod, &exam.CreatedAt,
		&exam.PackageExamType, &exam.ScreenLockEnabled, &exam.ScreenLockSeconds,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, domain.ErrUserExamNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("query user exam: %w", err)
	}
	return &attempt, &exam, nil
}

func (r *ExamRepository) UpsertAnswerFallback(ctx context.Context, userExamID, userID, questionID, selectedOption string) error {
	const query = `
		INSERT INTO user_answers (user_exam_id, question_id, selected_option, is_correct, updated_at)
		SELECT ue.id, q.id, $4, CASE WHEN q.question_type = 'essay' THEN false ELSE q.correct_answer = $4 END, NOW()
		FROM user_exams ue
		JOIN questions q ON q.exam_id = ue.exam_id AND q.id = $3
		WHERE ue.id = $1
		  AND ue.user_id = $2
		  AND ue.status = 'ongoing'
		  AND (
		      q.question_type = 'essay'
		      OR EXISTS (
		          SELECT 1 FROM jsonb_array_elements(q.options_json) option
		          WHERE option->>'key' = $4
		      )
		  )
		ON CONFLICT (user_exam_id, question_id)
		DO UPDATE SET
		    selected_option = EXCLUDED.selected_option,
		    is_correct = EXCLUDED.is_correct,
		    updated_at = NOW()
		RETURNING id`
	var answerID string
	err := r.db.QueryRow(ctx, query, userExamID, userID, questionID, selectedOption).Scan(&answerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrInvalidAnswer
	}
	if err != nil {
		return fmt.Errorf("upsert fallback answer: %w", err)
	}
	return nil
}

func (r *ExamRepository) GetPersistedAnswers(ctx context.Context, userExamID, userID string) (map[string]string, error) {
	const query = `
		SELECT ua.question_id, ua.selected_option
		FROM user_answers ua
		JOIN user_exams ue ON ue.id = ua.user_exam_id
		WHERE ua.user_exam_id = $1 AND ue.user_id = $2`
	rows, err := r.db.Query(ctx, query, userExamID, userID)
	if err != nil {
		return nil, fmt.Errorf("query persisted answers: %w", err)
	}
	defer rows.Close()
	answers := make(map[string]string)
	for rows.Next() {
		var questionID, selectedOption string
		if err := rows.Scan(&questionID, &selectedOption); err != nil {
			return nil, fmt.Errorf("scan persisted answer: %w", err)
		}
		answers[questionID] = selectedOption
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate persisted answers: %w", err)
	}
	return answers, nil
}

func (r *ExamRepository) SubmitUserExam(
	ctx context.Context,
	userExamID, userID string,
	answers []domain.UserAnswer,
	score float64,
	finishedAt time.Time,
) (*domain.UserExam, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin submit transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const lockQuery = `
		SELECT id, user_id, exam_id, status, started_at, finished_at, total_score
		FROM user_exams
		WHERE id = $1 AND user_id = $2
		FOR UPDATE`
	var attempt domain.UserExam
	err = tx.QueryRow(ctx, lockQuery, userExamID, userID).Scan(
		&attempt.ID, &attempt.UserID, &attempt.ExamID, &attempt.Status,
		&attempt.StartedAt, &attempt.FinishedAt, &attempt.TotalScore,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserExamNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock user exam: %w", err)
	}
	if attempt.Status == "submitted" {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit existing submission: %w", err)
		}
		return &attempt, nil
	}

	if len(answers) > 0 {
		userExamUUID, err := uuid.Parse(userExamID)
		if err != nil {
			return nil, domain.ErrUserExamNotFound
		}
		userExamUUIDs := make([]uuid.UUID, len(answers))
		questionUUIDs := make([]uuid.UUID, len(answers))
		selectedOptions := make([]string, len(answers))
		correctness := make([]bool, len(answers))
		for i, answer := range answers {
			userExamUUIDs[i] = userExamUUID
			questionUUIDs[i], err = uuid.Parse(answer.QuestionID)
			if err != nil {
				return nil, domain.ErrQuestionNotFound
			}
			selectedOptions[i] = answer.SelectedOption
			correctness[i] = answer.IsCorrect
		}
		const batchQuery = `
			INSERT INTO user_answers (user_exam_id, question_id, selected_option, is_correct, updated_at)
			SELECT input.user_exam_id, input.question_id, input.selected_option, input.is_correct, $5
			FROM UNNEST($1::uuid[], $2::uuid[], $3::text[], $4::boolean[])
			     AS input(user_exam_id, question_id, selected_option, is_correct)
			ON CONFLICT (user_exam_id, question_id)
			DO UPDATE SET
			    selected_option = EXCLUDED.selected_option,
			    is_correct = EXCLUDED.is_correct,
			    updated_at = EXCLUDED.updated_at`
		if _, err := tx.Exec(ctx, batchQuery, userExamUUIDs, questionUUIDs, selectedOptions, correctness, finishedAt); err != nil {
			return nil, fmt.Errorf("batch upsert answers: %w", err)
		}
	}

	const finishQuery = `
		UPDATE user_exams
		SET status = 'submitted', finished_at = $3, total_score = $4
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, exam_id, status, started_at, finished_at, total_score`
	err = tx.QueryRow(ctx, finishQuery, userExamID, userID, finishedAt, score).Scan(
		&attempt.ID, &attempt.UserID, &attempt.ExamID, &attempt.Status,
		&attempt.StartedAt, &attempt.FinishedAt, &attempt.TotalScore,
	)
	if err != nil {
		return nil, fmt.Errorf("finish user exam: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit submit transaction: %w", err)
	}
	return &attempt, nil
}

func (r *ExamRepository) ListExpiredUserExams(ctx context.Context, limit int) ([]domain.ExpiredUserExam, error) {
	const query = `
		SELECT ue.id, ue.user_id
		FROM user_exams ue
		JOIN exams e ON e.id = ue.exam_id
		WHERE ue.status = 'ongoing'
		  AND ue.started_at + make_interval(mins => e.duration_minutes) <= NOW()
		ORDER BY ue.started_at
		LIMIT $1`
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query expired user exams: %w", err)
	}
	defer rows.Close()
	items := make([]domain.ExpiredUserExam, 0, limit)
	for rows.Next() {
		var item domain.ExpiredUserExam
		if err := rows.Scan(&item.UserExamID, &item.UserID); err != nil {
			return nil, fmt.Errorf("scan expired user exam: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expired user exams: %w", err)
	}
	return items, nil
}

// EnsureUserExamShuffle memastikan permutasi soal/opsi untuk sebuah attempt
// dibuat sekali lalu dipersistenkan, sehingga urutan tetap stabil saat peserta
// kembali atau me-refresh. Tidak ada permutasi yang dibuat (dan kolom dibiarkan
// NULL) bila ujian tidak mengaktifkan fitur acak pada dimensi apa pun.
func (r *ExamRepository) EnsureUserExamShuffle(ctx context.Context, userExamID string) (*domain.UserExamShuffle, error) {
	const head = `
		SELECT e.id, ue.shuffle_questions, ue.shuffle_options,
		       ue.question_order_json, ue.option_order_json
		FROM user_exams ue
		JOIN exams e ON e.id = ue.exam_id
		WHERE ue.id = $1`
	var examID string
	var shuffleQuestions, shuffleOptions bool
	var rawQuestionOrder, rawOptionOrder []byte
	err := r.db.QueryRow(ctx, head, userExamID).Scan(
		&examID, &shuffleQuestions, &shuffleOptions,
		&rawQuestionOrder, &rawOptionOrder,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserExamNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query user exam shuffle: %w", err)
	}
	if !shuffleQuestions && !shuffleOptions {
		return &domain.UserExamShuffle{}, nil
	}
	if rawQuestionOrder != nil || rawOptionOrder != nil {
		return decodeUserExamShuffle(rawQuestionOrder, rawOptionOrder)
	}

	var questionOrder []string
	var optionOrder map[string][]string
	if shuffleQuestions {
		rows, err := r.db.Query(ctx, `SELECT id FROM questions WHERE exam_id = $1 ORDER BY id`, examID)
		if err != nil {
			return nil, fmt.Errorf("query question ids: %w", err)
		}
		ids := make([]string, 0)
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan question id: %w", err)
			}
			ids = append(ids, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate question ids: %w", err)
		}
		questionOrder = append([]string(nil), ids...)
		rand.Shuffle(len(questionOrder), func(i, j int) { questionOrder[i], questionOrder[j] = questionOrder[j], questionOrder[i] })
	}
	if shuffleOptions {
		rows, err := r.db.Query(ctx, `SELECT id, options_json FROM questions WHERE exam_id = $1 AND question_type <> 'essay'`, examID)
		if err != nil {
			return nil, fmt.Errorf("query question options: %w", err)
		}
		optionOrder = make(map[string][]string)
		for rows.Next() {
			var questionID string
			var rawOptions []byte
			if err := rows.Scan(&questionID, &rawOptions); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan question options: %w", err)
			}
			var options []domain.QuestionOption
			if err := json.Unmarshal(rawOptions, &options); err != nil {
				rows.Close()
				return nil, fmt.Errorf("decode options for question %s: %w", questionID, err)
			}
			keys := make([]string, 0, len(options))
			for _, option := range options {
				keys = append(keys, strings.ToUpper(strings.TrimSpace(option.Key)))
			}
			rand.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
			optionOrder[questionID] = keys
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate question options: %w", err)
		}
	}
	var questionJSON, optionJSON []byte
	if len(questionOrder) > 0 {
		if questionJSON, err = json.Marshal(questionOrder); err != nil {
			return nil, fmt.Errorf("encode question order: %w", err)
		}
	}
	if len(optionOrder) > 0 {
		if optionJSON, err = json.Marshal(optionOrder); err != nil {
			return nil, fmt.Errorf("encode option order: %w", err)
		}
	}
	tag, err := r.db.Exec(ctx, `UPDATE user_exams SET question_order_json = $2, option_order_json = $3
		WHERE id = $1 AND question_order_json IS NULL AND option_order_json IS NULL`, userExamID, questionJSON, optionJSON)
	if err != nil {
		return nil, fmt.Errorf("persist user exam shuffle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Request start/resume lain sudah lebih dahulu menyimpan permutasi.
		// Baca pemenangnya agar seluruh response untuk attempt ini identik.
		return r.EnsureUserExamShuffle(ctx, userExamID)
	}
	return &domain.UserExamShuffle{QuestionOrder: questionOrder, OptionOrder: optionOrder}, nil
}

func decodeUserExamShuffle(rawQuestionOrder, rawOptionOrder []byte) (*domain.UserExamShuffle, error) {
	var questionOrder []string
	var optionOrder map[string][]string
	if rawQuestionOrder != nil {
		if err := json.Unmarshal(rawQuestionOrder, &questionOrder); err != nil {
			return nil, fmt.Errorf("decode question order: %w", err)
		}
	}
	if rawOptionOrder != nil {
		if err := json.Unmarshal(rawOptionOrder, &optionOrder); err != nil {
			return nil, fmt.Errorf("decode option order: %w", err)
		}
	}
	return &domain.UserExamShuffle{QuestionOrder: questionOrder, OptionOrder: optionOrder}, nil
}

func (r *ExamRepository) HasActivePackage(ctx context.Context, userID, packageID string, now time.Time) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM user_packages WHERE user_id = $1 AND package_id = $2 AND status = 'active' AND expired_at > $3)`
	var active bool
	if err := r.db.QueryRow(ctx, query, userID, packageID, now).Scan(&active); err != nil {
		return false, fmt.Errorf("check active package: %w", err)
	}
	return active, nil
}

// GrantPackage memberi lisensi paket kepada pengguna tanpa alur pembayaran.
// Dipakai saat attempt CBT selesai/timeout supaya paket otomatis muncul di
// "Paket belajar saya". Paket yang sudah dimiliki cukup diperpanjang bila
// kedaluwarsa lebih awal.
func (r *ExamRepository) GrantPackage(ctx context.Context, userID, packageID string, now time.Time) error {
	const query = `
		INSERT INTO user_packages (user_id, package_id, expired_at, status)
		SELECT $1, p.id, $3::timestamptz + make_interval(days => GREATEST(p.validity_days, 1)), 'active'
		FROM packages p WHERE p.id = $2
		ON CONFLICT (user_id, package_id) DO UPDATE SET
			status = 'active',
			expired_at = GREATEST(user_packages.expired_at, EXCLUDED.expired_at)`
	result, err := r.db.Exec(ctx, query, userID, packageID, now)
	if err != nil {
		return fmt.Errorf("grant package: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrPackageNotFound
	}
	return nil
}

var _ domain.ExamRepository = (*ExamRepository)(nil)
var _ domain.ExamAccessRepository = (*ExamRepository)(nil)
var _ domain.ExamShuffleRepository = (*ExamRepository)(nil)
