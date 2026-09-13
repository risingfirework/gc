// Package redis mengimplementasikan repository domain di atas Redis.
package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"tka/apps/backend/internal/domain"
)

const examStateRetention = 24 * time.Hour

var saveAnswerScript = goredis.NewScript(`
local expires_at = redis.call('HGET', KEYS[1], 'expires_at_unix_ms')
if not expires_at then
    return {-1, 0}
end
local owner_id = redis.call('HGET', KEYS[1], 'user_id')
if owner_id ~= ARGV[1] then
    return {-2, 0}
end
if redis.call('HGET', KEYS[1], 'status') == 'submitting' then
    return {-3, 0}
end
local remaining = tonumber(expires_at) - tonumber(ARGV[2])
if remaining <= 0 then
    return {0, 0}
end
redis.call('HSET', KEYS[2], ARGV[3], ARGV[4])
redis.call('PEXPIREAT', KEYS[2], tonumber(expires_at) + tonumber(ARGV[5]))
return {1, remaining}
`)

var lockAnswersScript = goredis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then
    return {-1}
end
if redis.call('HGET', KEYS[1], 'user_id') ~= ARGV[1] then
    return {-2}
end
if redis.call('HGET', KEYS[1], 'status') == 'submitting' then
    return {-3}
end
redis.call('HSET', KEYS[1], 'status', 'submitting')
local answers = redis.call('HGETALL', KEYS[2])
local result = {1}
for _, value in ipairs(answers) do table.insert(result, value) end
return result
`)

var releaseSubmitLockScript = goredis.NewScript(`
if redis.call('HGET', KEYS[1], 'user_id') == ARGV[1]
   and redis.call('HGET', KEYS[1], 'status') == 'submitting' then
    redis.call('HSET', KEYS[1], 'status', 'ongoing')
    return 1
end
return 0
`)

type CBTRepository struct {
	client goredis.UniversalClient
}

func NewCBTRepository(client goredis.UniversalClient) *CBTRepository {
	return &CBTRepository{client: client}
}

func (r *CBTRepository) SetTimer(ctx context.Context, timer domain.ExamTimer) error {
	retentionDeadline := timer.ExpiresAt.Add(examStateRetention)
	_, err := r.client.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
		pipe.HSet(ctx, timerKey(timer.UserExamID), map[string]any{
			"user_id":            timer.UserID,
			"exam_id":            timer.ExamID,
			"started_at_unix_ms": timer.StartedAt.UnixMilli(),
			"expires_at_unix_ms": timer.ExpiresAt.UnixMilli(),
		})
		pipe.HSetNX(ctx, timerKey(timer.UserExamID), "status", "ongoing")
		pipe.ExpireAt(ctx, timerKey(timer.UserExamID), retentionDeadline)
		return nil
	})
	if err != nil {
		return fmt.Errorf("set exam timer: %w", err)
	}
	return nil
}

func (r *CBTRepository) GetTimer(ctx context.Context, userExamID string) (*domain.ExamTimer, error) {
	values, err := r.client.HGetAll(ctx, timerKey(userExamID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get exam timer: %w", err)
	}
	if len(values) == 0 {
		return nil, domain.ErrTimerNotFound
	}
	startedAtMS, err := strconv.ParseInt(values["started_at_unix_ms"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("decode timer start: %w", err)
	}
	expiresAtMS, err := strconv.ParseInt(values["expires_at_unix_ms"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("decode timer expiry: %w", err)
	}
	return &domain.ExamTimer{
		UserExamID: userExamID, UserID: values["user_id"], ExamID: values["exam_id"],
		StartedAt: time.UnixMilli(startedAtMS).UTC(), ExpiresAt: time.UnixMilli(expiresAtMS).UTC(),
	}, nil
}

func (r *CBTRepository) SaveAnswer(
	ctx context.Context,
	userExamID, userID, questionID, selectedOption string,
	now time.Time,
) (time.Duration, error) {
	result, err := saveAnswerScript.Run(ctx, r.client,
		[]string{timerKey(userExamID), answersKey(userExamID)},
		userID, now.UnixMilli(), questionID, selectedOption, examStateRetention.Milliseconds(),
	).Slice()
	if err != nil {
		return 0, fmt.Errorf("save answer atomically: %w", err)
	}
	if len(result) != 2 {
		return 0, errors.New("unexpected Redis script response")
	}
	status, ok := result[0].(int64)
	if !ok {
		return 0, errors.New("invalid Redis script status")
	}
	remainingMS, ok := result[1].(int64)
	if !ok {
		return 0, errors.New("invalid Redis timer response")
	}
	switch status {
	case -3:
		return 0, domain.ErrExamSubmissionInProgress
	case -2:
		return 0, domain.ErrExamForbidden
	case -1:
		return 0, domain.ErrTimerNotFound
	case 0:
		return 0, domain.ErrExamExpired
	case 1:
		return time.Duration(remainingMS) * time.Millisecond, nil
	default:
		return 0, errors.New("unknown Redis script status")
	}
}

func (r *CBTRepository) LockAnswersForSubmit(ctx context.Context, userExamID, userID string) (map[string]string, error) {
	result, err := lockAnswersScript.Run(ctx, r.client, []string{timerKey(userExamID), answersKey(userExamID)}, userID).Slice()
	if err != nil {
		return nil, fmt.Errorf("lock answers for submit: %w", err)
	}
	if len(result) == 0 {
		return nil, errors.New("empty Redis submit lock response")
	}
	status, ok := result[0].(int64)
	if !ok {
		return nil, errors.New("invalid Redis submit lock status")
	}
	switch status {
	case -3:
		return nil, domain.ErrExamSubmissionInProgress
	case -2:
		return nil, domain.ErrExamForbidden
	case -1:
		return nil, domain.ErrTimerNotFound
	case 1:
	default:
		return nil, errors.New("unknown Redis submit lock status")
	}
	if (len(result)-1)%2 != 0 {
		return nil, errors.New("invalid Redis answers hash response")
	}
	answers := make(map[string]string, (len(result)-1)/2)
	for i := 1; i < len(result); i += 2 {
		questionID, questionOK := result[i].(string)
		selectedOption, optionOK := result[i+1].(string)
		if !questionOK || !optionOK {
			return nil, errors.New("invalid Redis answer value")
		}
		answers[questionID] = selectedOption
	}
	return answers, nil
}

func (r *CBTRepository) ReleaseSubmitLock(ctx context.Context, userExamID, userID string) error {
	if err := releaseSubmitLockScript.Run(ctx, r.client, []string{timerKey(userExamID)}, userID).Err(); err != nil {
		return fmt.Errorf("release submit lock: %w", err)
	}
	return nil
}

func (r *CBTRepository) DeleteState(ctx context.Context, userExamID string) error {
	if err := r.client.Del(ctx, timerKey(userExamID), answersKey(userExamID)).Err(); err != nil {
		return fmt.Errorf("delete exam state: %w", err)
	}
	return nil
}

// A shared Redis Cluster hash-tag keeps both keys in the same slot so Lua
// scripts and multi-key deletes stay atomic after moving from a single node.
func timerKey(userExamID string) string   { return "EXAM_TIMER:{" + userExamID + "}" }
func answersKey(userExamID string) string { return "EXAM_ANSWERS:{" + userExamID + "}" }

var _ domain.CBTRepository = (*CBTRepository)(nil)
