// lib/urly.go

package lib

import "io"

func ExtractURL(inputReader io.Reader) ([]byte, error) {
	lex := NewLexer(inputReader)
	res := []byte{}
	for tok := range lex.tokens {
		res = append(res, tok.Value...)
		res = append(res, 10) // \n
	}

	return res, nil
}
