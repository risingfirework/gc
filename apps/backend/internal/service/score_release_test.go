package service

import "testing"

func TestValidScoreRelease(t *testing.T) {
	for _, tc := range []struct {
		release string
		want    bool
	}{
		{"after_finish", true},
		{"with_pembahasan", true},
		{"", false},
		{"AFTER_FINISH", false},
		{"after finish", false},
		{"draft", false},
	} {
		if got := validScoreRelease(tc.release); got != tc.want {
			t.Fatalf("validScoreRelease(%q) = %v, want %v", tc.release, got, tc.want)
		}
	}
}
