package generators

import (
	"strings"

	fakelish "github.com/nwtgck/go-fakelish"
)

// GeneratePhrase creates a phrase `numWords` long, consisting of non-sense, randomly generated words three (3) to seven (7) letters long
func GeneratePhrase(numWords int) string {
	words := make([]string, 0)

	for i := 0; i < numWords; i++ {
		words = append(words, fakelish.GenerateFakeWord(3, 7))
	}

	return strings.Join(words, " ")
}
