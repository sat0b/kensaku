package main

import "github.com/ikawaha/kagome/tokenizer"

type OffsetDoc struct {
	Text   string
	Offset int
}

func Tokenize(word string) []OffsetDoc {
	words := make([]OffsetDoc, 0)
	t := tokenizer.New()
	tokens := t.Tokenize(word)
	for i, token := range tokens {
		if token.Class == tokenizer.DUMMY {
			continue
		}
		words = append(words, OffsetDoc{Text: token.Surface, Offset: i})
	}
	return words
}
