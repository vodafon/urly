// lib/urly.go

package lib

func ExtractURL(input []byte) ([]byte, error) {
	lex := NewLexer(input)
	res := []byte{}
	for tok := range lex.tokens {
		res = append(res, tok.Value...)
		res = append(res, 10) // \n
	}

	return res, nil
}
