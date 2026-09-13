package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
		SELECT e.id, e.package_id, e.title, e.duration_minutes, e.total_questions, e.passing_score, e.scoring_method, e.created_at,
		       p.exam_type, COALESCE(p.cbt_token,'')
		FROM exams e
		JOIN packages p ON p.id = e.package_id
		WHERE e.id = $1`
	var exam domain.Exam
	err := r.db.QueryRow(ctx, examQuery, examID).Scan(
		&exam.ID, &exam.PackageID, &exam.Title, &exam.DurationMinutes,
		&exam.TotalQuestions, &exam.PassingScore, &exam.ScoringMethod, &exam.CreatedAt,
		&exam.PackageExamType, &exam.PackageCBTToken,
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
		       e.publish_pembahasan,
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
		  AND (
		      p.exam_type <> 'cbt'
		      OR e.publish_pembahasan
		      OR ur.user_exam_id IS NULL
		  )
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
			&exam.PublishPembahasan,
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
		INSERT INTO user_exams (user_id, exam_id, status, started_at)
		VALUES ($1, $2, 'ongoing', $3)
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
		       e.id, e.package_id, e.title, e.duration_minutes, e.total_questions, e.passing_score, e.scoring_method, e.created_at
		FROM user_exams ue
		JOIN exams e ON e.id = ue.exam_id
		WHERE ue.id = $1 AND ue.user_id = $2`
	var attempt domain.UserExam
	var exam domain.Exam
	err := r.db.QueryRow(ctx, query, userExamID, userID).Scan(
		&attempt.ID, &attempt.UserID, &attempt.ExamID, &attempt.Status,
		&attempt.StartedAt, &attempt.FinishedAt, &attempt.TotalScore,
		&exam.ID, &exam.PackageID, &exam.Title, &exam.DurationMinutes,
		&exam.TotalQuestions, &exam.PassingScore, &exam.ScoringMethod, &exam.CreatedAt,
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

func (r *ExamRepository) HasActivePackage(ctx context.Context, userID, packageID string, now time.Time) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM user_packages WHERE user_id = $1 AND package_id = $2 AND status = 'active' AND expired_at > $3)`
	var active bool
	if err := r.db.QueryRow(ctx, query, userID, packageID, now).Scan(&active); err != nil {
		return false, fmt.Errorf("check active package: %w", err)
	}
	return active, nil
}

var _ domain.ExamRepository = (*ExamRepository)(nil)
var _ domain.ExamAccessRepository = (*ExamRepository)(nil)
