package generators

import (
	"strings"
	"testing"
)

func TestGeneratePhrase(t *testing.T) {
	tests := []struct {
		name     string
		numWords int
	}{
		{
			name:     "3-words",
			numWords: 3,
		},
		{
			name:     "5-words",
			numWords: 5,
		},
		{
			name:     "10-words",
			numWords: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phrase := GeneratePhrase(tt.numWords)
			count := countWords(phrase)
			t.Logf("GeneratePhrase(): [%s]", phrase)
			if count != tt.numWords {
				t.Errorf("GeneratePhrase() = %v, want %v", count, tt.numWords)
			}
		})
	}
}

func countWords(phrase string) int {
	return len(strings.Fields(strings.TrimSpace(phrase)))
}
