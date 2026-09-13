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
		SELECT ue.id, ue.exam_id, e.title, ue.status, e.scoring_method, ue.started_at, ue.finished_at, ue.total_score, e.passing_score,
		       p.exam_type, e.publish_pembahasan
		FROM user_exams ue JOIN exams e ON e.id = ue.exam_id JOIN packages p ON p.id = e.package_id
		WHERE ue.id = $1 AND ue.user_id = $2`
	var result domain.ExamResultResponse
	var examType string
	var publishPembahasan bool
	err := r.db.QueryRow(ctx, headerQuery, userExamID, userID).Scan(&result.UserExamID, &result.ExamID, &result.Title, &result.Status, &result.ScoringMethod, &result.StartedAt, &result.FinishedAt, &result.TotalScore, &result.PassingScore, &examType, &publishPembahasan)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserExamNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query exam result: %w", err)
	}
	if result.Status != "submitted" {
		return nil, domain.ErrExamStillRunning
	}
	if examType == "cbt" && !publishPembahasan {
		return nil, domain.ErrPembahasanNotPublished
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

func (r *AnalyticsRepository) ListGlobalRanking(ctx context.Context, currentUserID, level, mode string, limit, offset int) ([]domain.GlobalRankingEntry, int, error) {
	countQuery := `
		SELECT COUNT(*) FROM (
			SELECT user_id FROM (
				SELECT ue.user_id, ue.exam_id,
				       COUNT(*) OVER (PARTITION BY ue.exam_id) AS cohort_n,
				       ROW_NUMBER() OVER (PARTITION BY ue.user_id, ue.exam_id
				         ORDER BY ue.total_score DESC, EXTRACT(EPOCH FROM (ue.finished_at - ue.started_at)) ASC, ue.finished_at ASC) AS rn
				FROM user_exams ue
				JOIN users ON users.id = ue.user_id
				WHERE ue.status = 'submitted' AND ue.total_score IS NOT NULL AND ue.finished_at IS NOT NULL
				  AND users.role = 'student' AND ($1 = '' OR users.school_level = $1)
			) ranked_rows
			WHERE ranked_rows.rn = 1 AND ranked_rows.cohort_n > 1
			GROUP BY user_id
		) counted`
	var total int
	if err := r.db.QueryRow(ctx, countQuery, level).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count global ranking: %w", err)
	}

	query := `
		WITH best_per_exam AS (
			SELECT DISTINCT ON (ue.user_id, ue.exam_id) ue.user_id, ue.exam_id, ue.total_score AS best_score,
			       EXTRACT(EPOCH FROM (ue.finished_at - ue.started_at))::bigint AS duration_seconds,
			       ue.finished_at
			FROM user_exams ue
			JOIN users ON users.id = ue.user_id
			WHERE ue.status = 'submitted' AND ue.total_score IS NOT NULL AND ue.finished_at IS NOT NULL
			  AND users.role = 'student' AND ($1 = '' OR users.school_level = $1)
			ORDER BY ue.user_id, ue.exam_id, ue.total_score DESC, duration_seconds ASC, ue.finished_at ASC
		), cohort AS (
			SELECT exam_id, user_id, duration_seconds, finished_at,
			       RANK() OVER (PARTITION BY exam_id ORDER BY best_score DESC, duration_seconds ASC, finished_at ASC) AS rank_pos,
			       COUNT(*) OVER (PARTITION BY exam_id)::bigint AS cohort_n
			FROM best_per_exam
		), pct AS (
			SELECT user_id, duration_seconds, finished_at,
			       CASE WHEN cohort_n <= 1 THEN NULL::double precision
			            ELSE (cohort_n - rank_pos)::double precision / (cohort_n - 1) END AS p
			FROM cohort
		), agg AS (
			SELECT user_id,
			       AVG(p) * 100 AS avg_pct,
			       MAX(p) * 100 AS best_pct,
			       COUNT(*)::bigint AS exams_done
			FROM pct
			WHERE p IS NOT NULL
			GROUP BY user_id
		), best_attempt AS (
			SELECT DISTINCT ON (user_id) user_id, duration_seconds, finished_at
			FROM pct
			WHERE p IS NOT NULL
			ORDER BY user_id, p DESC, duration_seconds ASC, finished_at ASC
		), ranked AS (
			SELECT a.user_id, a.avg_pct, a.best_pct, a.exams_done,
			       b.duration_seconds, b.finished_at,
			       CASE WHEN $5 = 'activity'
			         THEN DENSE_RANK() OVER (ORDER BY a.exams_done DESC, a.avg_pct DESC, a.best_pct DESC)
			         ELSE DENSE_RANK() OVER (ORDER BY a.avg_pct DESC, a.exams_done DESC, a.best_pct DESC)
			       END AS position
			FROM agg a
			JOIN best_attempt b ON b.user_id = a.user_id
		)
		SELECT ranked.position, COALESCE(NULLIF(users.name,''), split_part(users.email, '@', 1)), users.school_level,
		       round(ranked.avg_pct::numeric, 2)::double precision, round(ranked.best_pct::numeric, 2)::double precision,
		       ranked.duration_seconds, ranked.finished_at, ranked.user_id = $2, ranked.exams_done
		FROM ranked
		JOIN users ON users.id = ranked.user_id
		ORDER BY ranked.position, users.email
		LIMIT $3 OFFSET $4`
	rows, err := r.db.Query(ctx, query, level, currentUserID, limit, offset, mode)
	if err != nil {
		return nil, 0, fmt.Errorf("query global ranking: %w", err)
	}
	defer rows.Close()

	entries := make([]domain.GlobalRankingEntry, 0, limit)
	for rows.Next() {
		var entry domain.GlobalRankingEntry
		if err := rows.Scan(&entry.Rank, &entry.DisplayName, &entry.SchoolLevel, &entry.Score, &entry.BestPercentile, &entry.DurationSeconds, &entry.FinishedAt, &entry.IsCurrentUser, &entry.ExamsDone); err != nil {
			return nil, 0, fmt.Errorf("scan global ranking: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate global ranking: %w", err)
	}
	return entries, total, nil
}

var _ domain.AnalyticsRepository = (*AnalyticsRepository)(nil)
