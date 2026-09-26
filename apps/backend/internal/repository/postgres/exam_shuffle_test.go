package postgres

import "testing"

func TestDecodeUserExamShuffleSupportsQuestionOnly(t *testing.T) {
	mapping, err := decodeUserExamShuffle([]byte(`["q-2","q-1"]`), nil)
	if err != nil {
		t.Fatalf("decode question-only shuffle: %v", err)
	}
	if len(mapping.QuestionOrder) != 2 || mapping.QuestionOrder[0] != "q-2" || mapping.OptionOrder != nil {
		t.Fatalf("unexpected mapping: %+v", mapping)
	}
}

func TestDecodeUserExamShuffleSupportsOptionOnly(t *testing.T) {
	mapping, err := decodeUserExamShuffle(nil, []byte(`{"q-1":["B","A"]}`))
	if err != nil {
		t.Fatalf("decode option-only shuffle: %v", err)
	}
	if mapping.QuestionOrder != nil || len(mapping.OptionOrder["q-1"]) != 2 || mapping.OptionOrder["q-1"][0] != "B" {
		t.Fatalf("unexpected mapping: %+v", mapping)
	}
}
