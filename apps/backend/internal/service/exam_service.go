package service

import (
	"context"
	"math"
	"strings"

	"tka/apps/backend/internal/domain"
)

type ExamService struct{ analytics domain.AnalyticsRepository }

func NewExamService(analytics domain.AnalyticsRepository) *ExamService {
	return &ExamService{analytics: analytics}
}

func (s *ExamService) GetResult(ctx context.Context, userID, userExamID string) (*domain.ExamResultResponse, error) {
	if !validUUID(userID) || !validUUID(userExamID) {
		return nil, domain.ErrUserExamNotFound
	}
	return s.analytics.GetExamResult(ctx, userID, userExamID)
}

func (s *ExamService) GetGlobalRanking(ctx context.Context, currentUserID, level string) ([]domain.GlobalRankingEntry, error) {
	if !validUUID(currentUserID) {
		return nil, domain.ErrInvalidInput
	}
	level = strings.TrimSpace(level)
	if level != "" && !validJenjang(level) {
		return nil, domain.ErrInvalidInput
	}
	return s.analytics.ListGlobalRanking(ctx, currentUserID, level, 100)
}

// CalculateExamScore supports deterministic weighted scoring and a bounded 2PL IRT estimate.
// IRT parameters must be calibrated before irt_2pl is enabled for a production exam.
func CalculateExamScore(method string, questions []domain.Question, answers map[string]string) float64 {
	if method == "irt_2pl" {
		return calculateIRT2PL(questions, answers)
	}
	totalWeight, correctWeight := 0.0, 0.0
	for _, question := range questions {
		totalWeight += question.ScoreWeight
		if answers[question.ID] == question.CorrectAnswer {
			correctWeight += question.ScoreWeight
		}
	}
	if totalWeight == 0 {
		return 0
	}
	return math.Round(correctWeight/totalWeight*10000) / 100
}

func calculateIRT2PL(questions []domain.Question, answers map[string]string) float64 {
	if len(questions) == 0 {
		return 0
	}
	theta := 0.0
	for iteration := 0; iteration < 15; iteration++ {
		gradient, information := 0.0, 0.0
		for _, question := range questions {
			a := question.Discrimination
			if a <= 0 {
				a = 1
			}
			probability := 1 / (1 + math.Exp(-a*(theta-question.Difficulty)))
			observed := 0.0
			if answers[question.ID] == question.CorrectAnswer {
				observed = 1
			}
			gradient += a * (observed - probability)
			information += a * a * probability * (1 - probability)
		}
		if information < 1e-9 {
			break
		}
		step := gradient / information
		if step > 1 {
			step = 1
		} else if step < -1 {
			step = -1
		}
		theta = math.Max(-4, math.Min(4, theta+step))
		if math.Abs(step) < 1e-5 {
			break
		}
	}
	return math.Round((theta+4)/8*10000) / 100
}

var _ domain.ExamAnalyticsService = (*ExamService)(nil)
