package service

import (
	"net/url"
	"sort"
	"strings"

	"tka/apps/backend/internal/domain"
)

func normalizeQuestion(input domain.AdminQuestionRequest) domain.AdminQuestionRequest {
	input.SubjectName = strings.TrimSpace(input.SubjectName)
	input.ContentText = strings.TrimSpace(input.ContentText)
	input.QuestionType = strings.ToLower(strings.TrimSpace(input.QuestionType))
	if input.QuestionType == "" {
		input.QuestionType = domain.QuestionTypeSingleChoice
	}
	input.PresentationType = domain.PresentationTypeSingle
	input.GroupCode = ""
	input.StimulusText = ""
	input.QuestionImageURL = strings.TrimSpace(input.QuestionImageURL)
	input.StimulusImageURL = ""
	for index := range input.Options {
		input.Options[index].Key = strings.ToUpper(strings.TrimSpace(input.Options[index].Key))
		input.Options[index].Content = strings.TrimSpace(input.Options[index].Content)
		input.Options[index].ImageURL = strings.TrimSpace(input.Options[index].ImageURL)
	}
	for index := range input.CategoryLabels {
		input.CategoryLabels[index] = strings.TrimSpace(input.CategoryLabels[index])
	}
	if answer, ok := canonicalAnswer(input.QuestionType, input.Options, input.CategoryLabels, input.CorrectAnswer, false); ok {
		input.CorrectAnswer = answer
	}
	return input
}

func validQuestionImage(value string) bool {
	if value == "" {
		return true
	}
	if len(value) > 2_200_000 {
		return false
	}
	allowedDataPrefixes := []string{"data:image/jpeg;base64,", "data:image/png;base64,", "data:image/webp;base64,", "data:image/gif;base64,"}
	for _, prefix := range allowedDataPrefixes {
		if strings.HasPrefix(value, prefix) && len(value) > len(prefix) {
			return true
		}
	}
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && len(value) <= 2048
}

func canonicalAnswer(questionType string, options []domain.QuestionOption, labels []string, answer string, allowPartial bool) (string, bool) {
	validKeys := make(map[string]bool, len(options))
	for _, option := range options {
		validKeys[strings.ToUpper(strings.TrimSpace(option.Key))] = true
	}
	answer = strings.TrimSpace(answer)
	switch questionType {
	case "", domain.QuestionTypeSingleChoice:
		key := strings.ToUpper(answer)
		return key, validKeys[key]
	case domain.QuestionTypeMultipleChoice:
		seen := map[string]bool{}
		keys := []string{}
		for _, raw := range strings.Split(answer, ",") {
			key := strings.ToUpper(strings.TrimSpace(raw))
			if key == "" || !validKeys[key] || seen[key] {
				return "", false
			}
			seen[key] = true
			keys = append(keys, key)
		}
		if len(keys) == 0 || (!allowPartial && len(keys) < 2) {
			return "", false
		}
		sort.Strings(keys)
		return strings.Join(keys, ","), true
	case domain.QuestionTypeCategory:
		validLabels := map[string]string{}
		for _, label := range labels {
			trimmed := strings.TrimSpace(label)
			if trimmed == "" {
				return "", false
			}
			validLabels[strings.ToLower(trimmed)] = trimmed
		}
		if len(validLabels) < 2 {
			return "", false
		}
		values := map[string]string{}
		for _, raw := range strings.Split(answer, ";") {
			parts := strings.SplitN(raw, "=", 2)
			if len(parts) != 2 {
				return "", false
			}
			key := strings.ToUpper(strings.TrimSpace(parts[0]))
			label, exists := validLabels[strings.ToLower(strings.TrimSpace(parts[1]))]
			if !validKeys[key] || !exists || values[key] != "" {
				return "", false
			}
			values[key] = label
		}
		if len(values) == 0 || (!allowPartial && len(values) != len(validKeys)) {
			return "", false
		}
		keys := make([]string, 0, len(values))
		for key := range values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, key+"="+values[key])
		}
		return strings.Join(parts, ";"), true
	default:
		return "", false
	}
}
