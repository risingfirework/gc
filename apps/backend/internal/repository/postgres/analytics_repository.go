package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tka/apps/backend/internal/domain"
)

type AnalyticsRepository struct{ db *pgxpool.Pool }

func NewAnalyticsRepository(db *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

func (r *AnalyticsRepository) GetExamResult(ctx context.Context, userID, userExamID string) (*domain.ExamResultResponse, error) {
	const headerQuery = `
		SELECT ue.id, ue.exam_id, e.title, ue.status, e.scoring_method, ue.started_at, ue.finished_at, ue.total_score, e.passing_score
		FROM user_exams ue JOIN exams e ON e.id = ue.exam_id
		WHERE ue.id = $1 AND ue.user_id = $2`
	var result domain.ExamResultResponse
	err := r.db.QueryRow(ctx, headerQuery, userExamID, userID).Scan(&result.UserExamID, &result.ExamID, &result.Title, &result.Status, &result.ScoringMethod, &result.StartedAt, &result.FinishedAt, &result.TotalScore, &result.PassingScore)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserExamNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query exam result: %w", err)
	}
	if result.Status != "submitted" {
		return nil, domain.ErrExamStillRunning
	}
	result.Passed = result.TotalScore >= result.PassingScore

	const detailQuery = `
		SELECT q.id, q.subject_name, q.content_text, q.question_type, q.presentation_type,
		       COALESCE(q.group_code,''), q.stimulus_text, q.question_image_url, q.stimulus_image_url,
		       q.category_labels_json, q.options_json, q.correct_answer, q.score_weight,
		       q.explanation_text, q.explanation_video_url,
		       ua.selected_option, COALESCE(ua.is_correct, false)
		FROM questions q
		LEFT JOIN user_answers ua ON ua.question_id = q.id AND ua.user_exam_id = $1
		WHERE q.exam_id = $2 ORDER BY q.subject_name, q.id`
	rows, err := r.db.Query(ctx, detailQuery, userExamID, result.ExamID)
	if err != nil {
		return nil, fmt.Errorf("query result details: %w", err)
	}
	defer rows.Close()
	type accumulator struct {
		correct, wrong, unanswered, total int
		correctWeight, totalWeight        float64
	}
	bySubject := make(map[string]*accumulator)
	order := make([]string, 0)
	for rows.Next() {
		var review domain.AnswerReview
		var rawOptions, rawCategoryLabels []byte
		if err := rows.Scan(&review.QuestionID, &review.SubjectName, &review.ContentText, &review.QuestionType, &review.PresentationType, &review.GroupCode, &review.StimulusText, &review.QuestionImageURL, &review.StimulusImageURL, &rawCategoryLabels, &rawOptions, &review.CorrectAnswer, &review.ScoreWeight, &review.ExplanationText, &review.ExplanationVideoURL, &review.SelectedOption, &review.IsCorrect); err != nil {
			return nil, fmt.Errorf("scan result detail: %w", err)
		}
		if err := json.Unmarshal(rawCategoryLabels, &review.CategoryLabels); err != nil {
			return nil, fmt.Errorf("decode result category labels: %w", err)
		}
		if err := json.Unmarshal(rawOptions, &review.Options); err != nil {
			return nil, fmt.Errorf("decode result options: %w", err)
		}
		stats := bySubject[review.SubjectName]
		if stats == nil {
			stats = &accumulator{}
			bySubject[review.SubjectName] = stats
			order = append(order, review.SubjectName)
		}
		stats.total++
		stats.totalWeight += review.ScoreWeight
		switch {
		case review.SelectedOption == nil:
			stats.unanswered++
			result.Unanswered++
		case review.IsCorrect:
			stats.correct++
			stats.correctWeight += review.ScoreWeight
			result.CorrectAnswers++
		default:
			stats.wrong++
			result.WrongAnswers++
		}
		result.Review = append(result.Review, review)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate result details: %w", err)
	}
	result.Subjects = make([]domain.SubjectResult, 0, len(order))
	for _, name := range order {
		stats := bySubject[name]
		score := 0.0
		if stats.totalWeight > 0 {
			score = math.Round(stats.correctWeight/stats.totalWeight*10000) / 100
		}
		result.Subjects = append(result.Subjects, domain.SubjectResult{SubjectName: name, CorrectAnswers: stats.correct, WrongAnswers: stats.wrong, Unanswered: stats.unanswered, TotalQuestions: stats.total, Score: score})
	}
	return &result, nil
}

func (r *AnalyticsRepository) ListGlobalRanking(ctx context.Context, currentUserID, level string, limit int) ([]domain.GlobalRankingEntry, error) {
	const query = `
		WITH best_attempt AS (
			SELECT DISTINCT ON (users.id) users.id AS user_id, ue.total_score, ue.started_at, ue.finished_at,
			       EXTRACT(EPOCH FROM (ue.finished_at - ue.started_at))::bigint AS duration_seconds
			FROM user_exams ue
			JOIN users ON users.id = ue.user_id
			WHERE ue.status = 'submitted' AND ue.total_score IS NOT NULL AND ue.finished_at IS NOT NULL
			  AND users.role = 'student' AND ($1 = '' OR users.school_level = $1)
			ORDER BY users.id, ue.total_score DESC, duration_seconds ASC, ue.finished_at ASC
		), ranked AS (
			SELECT user_id, total_score, finished_at, duration_seconds,
			       RANK() OVER (ORDER BY total_score DESC, duration_seconds ASC, finished_at ASC) AS position
			FROM best_attempt
		)
		SELECT ranked.position, COALESCE(NULLIF(users.name,''), split_part(users.email, '@', 1)), users.school_level,
		       ranked.total_score, ranked.duration_seconds, ranked.finished_at, ranked.user_id = $2
		FROM ranked
		JOIN users ON users.id = ranked.user_id
		ORDER BY ranked.position, users.email
		LIMIT $3`
	rows, err := r.db.Query(ctx, query, level, currentUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("query global ranking: %w", err)
	}
	defer rows.Close()

	entries := make([]domain.GlobalRankingEntry, 0, limit)
	for rows.Next() {
		var entry domain.GlobalRankingEntry
		if err := rows.Scan(&entry.Rank, &entry.DisplayName, &entry.SchoolLevel, &entry.Score, &entry.DurationSeconds, &entry.FinishedAt, &entry.IsCurrentUser); err != nil {
			return nil, fmt.Errorf("scan global ranking: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate global ranking: %w", err)
	}
	return entries, nil
}

var _ domain.AnalyticsRepository = (*AnalyticsRepository)(nil)
