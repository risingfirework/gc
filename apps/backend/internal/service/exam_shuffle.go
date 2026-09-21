package service

import (
	"strings"

	"tka/apps/backend/internal/domain"
)

// displayQuestion membangun QuestionResponse dari frame kanonis. Bila mapping
// opsi tersedia untuk soal, opsi diurutkan sesuai permutasi dan hurufnya
// di-relabel ulang secara posisional (A, B, C, ...) sehingga setiap peserta
// memiliki pola kunci jawaban yang berbeda.
func displayQuestion(question domain.Question, mapping *domain.UserExamShuffle) domain.QuestionResponse {
	response := domain.QuestionResponse{
		ID: question.ID, SubjectName: question.SubjectName, ContentText: question.ContentText,
		QuestionType: question.QuestionType, PresentationType: question.PresentationType,
		GroupCode: question.GroupCode, StimulusText: question.StimulusText,
		QuestionImageURL: question.QuestionImageURL, StimulusImageURL: question.StimulusImageURL,
		CategoryLabels: question.CategoryLabels, Options: question.Options,
	}
	order, permutation := completeOptionPermutation(question, mapping.OptionOrder[question.ID])
	if !permutation {
		return response
	}
	byKey := make(map[string]domain.QuestionOption, len(question.Options))
	for _, option := range question.Options {
		byKey[option.Key] = option
	}
	options := make([]domain.QuestionOption, 0, len(order))
	for index, key := range order {
		option, ok := byKey[key]
		if !ok {
			continue
		}
		option.Key = string(rune('A' + index))
		options = append(options, option)
	}
	response.Options = options
	return response
}

// applyDisplayShuffle menyusun seluruh soal sesuai permutasi attempt. Urutan
// kanonis digunakan ketika mapping tidak menyediakan urutan.
func applyDisplayShuffle(questions []domain.Question, mapping *domain.UserExamShuffle) []domain.QuestionResponse {
	responses := make([]domain.QuestionResponse, 0, len(questions))
	for _, question := range questions {
		responses = append(responses, displayQuestion(question, mapping))
	}
	if len(mapping.QuestionOrder) == 0 {
		return responses
	}
	byID := make(map[string]domain.QuestionResponse, len(responses))
	for _, response := range responses {
		byID[response.ID] = response
	}
	ordered := make([]domain.QuestionResponse, 0, len(mapping.QuestionOrder))
	for _, id := range mapping.QuestionOrder {
		if response, ok := byID[id]; ok {
			ordered = append(ordered, response)
		}
	}
	return ordered
}

// translateDisplayFrame menerjemahkan jawaban peserta dari frame tampilan
// (huruf posisional A, B, C, ...) ke frame kanonis (kunci opsi asli) sebelum
// dinormalisasi. Jawaban yang dikirim saat acak opsi nonaktif (tanpa mapping)
// diteruskan apa adanya.
func translateDisplayFrame(mapping *domain.UserExamShuffle, question domain.Question, selected string) (string, bool) {
	order, permutation := completeOptionPermutation(question, mapping.OptionOrder[question.ID])
	if !permutation {
		return selected, true
	}
	switch question.QuestionType {
	case domain.QuestionTypeEssay:
		return selected, true
	case domain.QuestionTypeMultipleChoice:
		parts := strings.Split(selected, ",")
		translated := make([]string, 0, len(parts))
		for _, part := range parts {
			key, ok := displayKey(order, part)
			if !ok {
				return "", false
			}
			translated = append(translated, key)
		}
		return strings.Join(translated, ","), true
	case domain.QuestionTypeCategory:
		parts := strings.Split(selected, ";")
		translated := make([]string, 0, len(parts))
		for _, part := range parts {
			pair := strings.SplitN(part, "=", 2)
			if len(pair) != 2 {
				return "", false
			}
			key, ok := displayKey(order, pair[0])
			if !ok {
				return "", false
			}
			translated = append(translated, key+"="+pair[1])
		}
		return strings.Join(translated, ";"), true
	default:
		return displayKey(order, selected)
	}
}

// completeOptionPermutation memastikan urutan opsi adalah permutasi utuh dari
// seluruh kunci kanonis soal. Bila urutan parsial/stale/duplikat, mapping
// dianggap tidak dapat diterapkan dan frame kanonis dipakai agar tidak ada opsi
// yang hilang dari tampilan atau diterjemahkan secara asimetris.
func completeOptionPermutation(question domain.Question, order []string) ([]string, bool) {
	canonical := make(map[string]bool, len(question.Options))
	for _, option := range question.Options {
		canonical[option.Key] = true
	}
	if len(order) != len(canonical) {
		return nil, false
	}
	seen := make(map[string]bool, len(order))
	for _, key := range order {
		if !canonical[key] || seen[key] {
			return nil, false
		}
		seen[key] = true
	}
	return order, true
}

// displayKey memetakan huruf tampilan ke kunci kanonis berdasarkan urutan opsi
// ber-label posisional milik attempt.
func displayKey(order []string, letter string) (string, bool) {
	letter = strings.ToUpper(strings.TrimSpace(letter))
	if len(letter) != 1 || letter[0] < 'A' || letter[0] > 'Z' {
		return "", false
	}
	index := int(letter[0] - 'A')
	if index >= len(order) {
		return "", false
	}
	return order[index], true
}
