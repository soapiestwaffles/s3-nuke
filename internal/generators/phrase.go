package generators

import (
	"strings"

	fakelish "github.com/nwtgck/go-fakelish"
)

func GeneratePhrase(numWords int) string {
	words := make([]string, 0)

	for i := 0; i < numWords; i++ {
		words = append(words, fakelish.GenerateFakeWord(3, 7))
	}

	return strings.Join(words, " ")
}
