// lib/lexer.go

package lib

import (
	"bytes"
	"io"
)

type TokenType int
type StateType int

const (
	// Special tokens
	TokenFullURL TokenType = iota
	TokenCustomURL
	TokenPathURL
)

const (
	StateScheme StateType = iota
	StateSchemeSep
	StateHost
	StatePath
	StateParams
)

var (
	httpW  []byte = []byte{104, 116, 116, 112}
	httpsW []byte = []byte{104, 116, 116, 112, 115}
)

// Character class bitmaps
const (
	inv uint16 = 0x00 // invalid for all
	lwr uint16 = 0x01 // a-z
	uwr uint16 = 0x02 // A-Z
	dig uint16 = 0x03 // 0-9
	slh uint16 = 0x04 // slash
	col uint16 = 0x05 // :

	sp1 uint16 = 0x21 // !
	sp2 uint16 = 0x22 // #
	sp3 uint16 = 0x23 // $
	sp4 uint16 = 0x24 // %
	sp5 uint16 = 0x25 // &
	sp6 uint16 = 0x26 // '
	sp7 uint16 = 0x27 // =
	sp8 uint16 = 0x28 // ?
	sp9 uint16 = 0x29 // @
	sn0 uint16 = 0x30 // _
	sn1 uint16 = 0x31 // +
	sn2 uint16 = 0x32 // -
	sn3 uint16 = 0x33 // .

	eof uint16 = 0x40 // EOF
)

// Character class lookup table
var lookup = [257]uint16{
	/* 00-07  NUL, SOH, STX, ETX, EOT, ENQ, ACK, BEL */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* 08-0F  BS,  HT,  LF,  VT,  FF,  CR,  SO,  SI  */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* 10-17  DLE, DC1, DC2, DC3, DC4, NAK, SYN, ETB */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* 18-1F  CAN, EM,  SUB, ESC, FS,  GS,  RS,  US  */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* 21-27  SP ! " # $ % & '   */ inv, sp1, inv, sp2, sp3, sp4, sp5, sp6,
	/* 28-2F   ( ) * + , - . /   */ inv, inv, inv, sn1, inv, sn2, sn3, slh,
	/* 30-37   0 1 2 3 4 5 6 7   */ dig, dig, dig, dig, dig, dig, dig, dig,
	/* 38-3F   8 9 : ; < = > ?   */ dig, dig, col, inv, inv, sp7, inv, sp8,
	/* 40-47   @ A B C D E F G   */ sp9, uwr, uwr, uwr, uwr, uwr, uwr, uwr,
	/* 48-4F   H I J K L M N O   */ uwr, uwr, uwr, uwr, uwr, uwr, uwr, uwr,
	/* 50-57   P Q R S T U V W   */ uwr, uwr, uwr, uwr, uwr, uwr, uwr, uwr,
	/* 58-5F   X Y Z [ \ ] ^ _   */ uwr, uwr, uwr, inv, inv, inv, inv, sn0,
	/* 60-67   ` a b c d e f g   */ inv, lwr, lwr, lwr, lwr, lwr, lwr, lwr,
	/* 68-6F   h i j k l m n o   */ lwr, lwr, lwr, lwr, lwr, lwr, lwr, lwr,
	/* 70-77   p q r s t u v w   */ lwr, lwr, lwr, lwr, lwr, lwr, lwr, lwr,
	/* 78-7F   x y z { | } ~ DEL */ lwr, lwr, lwr, inv, inv, inv, inv, inv,
	/* 80-87 */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* 88-8B */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* 90-97 */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* 98-9F */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* A0-A7 */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* A8-AF */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* B0-B7 */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* B8-BF */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* C0-C7 */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* C8-CF */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* D0-D7 */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* D8-DF */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* E0-E7 */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* E8-EF */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* F0-F7 */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* F8-FF */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* EOF 	 */ eof,
}

type Token struct {
	Type  TokenType
	Value []byte
}

type Lexer struct {
	reader              io.Reader
	input               []byte
	scheme              []byte
	schemeWord          []byte
	schemeSep           []byte
	host                []byte
	path                []byte
	params              []byte
	inputSize           int
	schemeWordsCount    int
	hostWordsCount      int
	pathWordsCount      int
	pathWordsComplexity int
	pos                 int
	width               int
	size                int
	start               int
	pathPrevWordClass   uint16
	stateType           StateType
	tokenType           TokenType
	tokens              chan Token
}

func NewLexer(reader io.Reader) *Lexer {
	l := &Lexer{
		reader:    reader,
		tokens:    make(chan Token),
		tokenType: TokenFullURL,
	}
	go l.run()
	return l
}

func (obj *Lexer) run() {
	for state := lexText; state != nil; {
		state = state(obj)
	}
	close(obj.tokens)
}

func (obj *Lexer) emit(t TokenType) {
	// fullURL without host?
	if obj.tokenType == TokenFullURL && (len(obj.host) == 0 || (obj.hostWordsCount+obj.pathWordsCount) < 4) {
		obj.emitUpdate()
		return
	}

	// pathURL without path?
	if obj.tokenType == TokenPathURL {
		// fmt.Printf("30: '%s': %v\n", calculateComplexity(obj.pathWordsComplexity, obj.pathWordsCount))
		if obj.pathWordsCount < 3 || obj.size < 5 || lookup[obj.byteAt(0)] != slh ||
			(len(obj.params) == 0 && calculateComplexity(obj.pathWordsComplexity, obj.pathWordsCount) > 3.3) {
			obj.emitUpdate()
			return
		}
	}

	// custom urls app:///path can be without host, and without path twitter://
	if obj.tokenType == TokenCustomURL && (obj.schemeWordsCount == 0 || len(obj.schemeSep) != 3) {
		obj.emitUpdate()
		return
	}
	// fmt.Println(3, obj.tokenType, TokenFullURL, obj.tokenType == TokenFullURL, len(obj.host))

	obj.tokens <- Token{
		Type:  t,
		Value: obj.input[obj.start:obj.pos],
	}
	obj.emitUpdate()
}

func (obj *Lexer) byteAt(i int) byte {
	return obj.input[obj.start+i]
}
func (obj *Lexer) emitUpdate() {
	obj.setAt(obj.pos, obj.pos)
	obj.resetTypes()
	obj.resetSlices()
}

func (obj *Lexer) setAt(start, pos int) {
	obj.start = start
	obj.pos = pos
	obj.size = pos - start
}

func (obj *Lexer) resetTypes() {
	obj.stateType = StateScheme
	obj.tokenType = TokenFullURL
}

func (obj *Lexer) resetSlices() {
	obj.scheme = []byte{}
	obj.schemeWord = []byte{}
	obj.schemeSep = []byte{}
	obj.host = []byte{}
	obj.path = []byte{}
	obj.params = []byte{}
	obj.pathWordsCount = 0
	obj.pathWordsComplexity = 0
	obj.pathPrevWordClass = inv
	obj.hostWordsCount = 0
	obj.schemeWordsCount = 0
}

func (l *Lexer) backup() {
	l.pos -= l.width
	l.size -= 1
}

func (l *Lexer) peek() uint16 {
	w := l.width
	r := l.next()
	l.backup()
	l.width = w
	return r
}

func (l *Lexer) next() uint16 {
	if l.pos >= l.inputSize {
		buf := make([]byte, 1000)
		n, err := l.reader.Read(buf)
		if err != nil {
			l.width = 0
			return 256
		}
		l.input = append(l.input, buf...)
		l.inputSize += n
		return l.next()
	}
	r := l.input[l.pos]
	l.pos += 1
	l.width = 1
	l.size += 1
	return uint16(r)
}

func (l *Lexer) processSchemeValid(r byte, s uint16) bool {
	// fmt.Printf("SV1: %v %v %v\n", r, s, s == col)
	if s == col {
		if len(l.scheme) == 0 {
			return false
		}
		l.schemeSep = append(l.schemeSep, r)
		l.schemeWord = []byte{}
		l.stateType = StateSchemeSep
		// if scheme end with http/https we should cut prefix
		if l.size > 5 && isHttpsWord(l.input[l.pos-6:l.pos-1]) {
			l.start = l.pos - 6
			l.size = l.pos - l.start
		} else if l.size > 4 && isHttpWord(l.input[l.pos-5:l.pos-1]) {
			l.start = l.pos - 5
			l.size = l.pos - l.start
		} else {
			l.tokenType = TokenCustomURL
		}
		return true
	}
	// fmt.Printf("SV2: %v %v %v\n", r, s, s == slh)
	if s == slh {
		l.startRelativePath(r)
		return true
	}
	if isAlphaNumeric(s) {
		l.schemeWordsCount += 1
	}
	// valid, _, -, .
	if isAlphaNumeric(s) || s == sn0 || s == sn2 || s == sn3 {
		l.scheme = append(l.scheme, r)
		l.schemeWord = append(l.schemeWord, r)
		return true
	}
	// +
	if s == sn1 {
		l.scheme = append(l.scheme, r)
		l.schemeWord = []byte{}
		return true
	}
	return false
}

func isAlphaNumeric(s uint16) bool {
	return s == lwr || s == uwr || s == dig
}

func (l *Lexer) startRelativePath(r byte) {
	l.path = append(l.path, r)
	l.stateType = StatePath
	if l.tokenType == TokenFullURL {
		l.tokenType = TokenPathURL
		l.start = l.pos - len(l.schemeWord) - 1
		l.scheme = []byte{}
		l.schemeWord = []byte{}
		l.schemeSep = []byte{}
		l.host = []byte{}
		l.params = []byte{}
		l.schemeWordsCount = 0
		l.hostWordsCount = 0
	}
}

func (l *Lexer) processSchemeSepValid(r byte, s uint16) bool {
	switch len(l.schemeSep) {
	case 1:
		if s == slh {
			l.schemeSep = append(l.schemeSep, r)
			return true
		}
	case 2:
		if s == slh {
			l.schemeSep = append(l.schemeSep, r)
			l.stateType = StateHost
			return true
		}
	}
	return false
}

func (l *Lexer) processHostValid(r byte, s uint16) bool {
	if s == slh {
		// https:///x
		if len(l.host) == 0 {
			l.startRelativePath(r)
			return true
		}
		l.path = append(l.path, r)
		l.stateType = StatePath
		return true
	}
	if isAlphaNumeric(s) {
		l.hostWordsCount += 1
	}
	// valid, @, :, _, -, .
	if isAlphaNumeric(s) || s == sp9 || s == col || s == sn0 || s == sn2 || s == sn3 {
		l.host = append(l.host, r)
		return true
	}
	// ?
	if s == sp8 {
		l.params = append(l.params, r)
		l.stateType = StateParams
		return true
	}
	return false
}

func (l *Lexer) processPathValid(r byte, s uint16) bool {
	if s == slh && len(l.path) == 1 && l.tokenType == TokenPathURL {
		l.start = l.pos - 1
		return true
	}
	if isAlphaNumeric(s) {
		l.path = append(l.path, r)
		l.pathWordsCount += 1
		if l.pathPrevWordClass != inv && l.pathPrevWordClass != s {
			l.pathWordsComplexity += 10
		}
		l.pathPrevWordClass = s
		return true
	}
	if s == slh || s == sp1 || s == sp4 || s == sp7 || s == sn0 || s == sn2 || s == sn3 {
		l.path = append(l.path, r)
		return true
	}
	// ?
	if s == sp8 {
		l.params = append(l.params, r)
		l.stateType = StateParams
		return true
	}
	return false
}

func (l *Lexer) processParamsValid(r byte, s uint16) bool {
	if s != inv {
		l.params = append(l.params, r)
		return true
	}
	return false
}

func (l *Lexer) isValid(r byte) bool {
	s := lookup[r]
	// fmt.Printf("10: %v\t'%s'\t%v\t%v\t%v\n", r, string(r), s, l.stateType, l.tokenType)
	switch l.stateType {
	case StateScheme:
		return l.processSchemeValid(r, s)
	case StateSchemeSep:
		return l.processSchemeSepValid(r, s)
	case StateHost:
		return l.processHostValid(r, s)
	case StatePath:
		return l.processPathValid(r, s)
	case StateParams:
		return l.processParamsValid(r, s)
	}
	return isAlphaNumeric(s)
}

type stateFn func(*Lexer) stateFn

func lexText(l *Lexer) stateFn {
	for {
		x := l.next()

		if lookup[x] == eof {
			break
		}

		r := uint8(x)

		if l.isValid(r) {
			continue
		}
		if l.size > 1 {
			l.backup()
		} else {
			l.emitUpdate()
			continue
		}

		l.emit(l.tokenType)
	}

	if l.pos > l.start {
		l.emit(l.tokenType)
	}
	return nil
}

func isHttpWord(word []byte) bool {
	return bytes.Equal(toLowerASCII(word), httpW)
}

func isHttpsWord(word []byte) bool {
	return bytes.Equal(toLowerASCII(word), httpsW)
}

func toLowerASCII(b []byte) []byte {
	// Create a new byte slice to hold the lowercase result
	lower := make([]byte, len(b))

	for i, v := range b {
		// Check if the byte is an uppercase letter
		if v >= 'A' && v <= 'Z' {
			// Convert to lowercase by adding 32
			lower[i] = v + 32
		} else {
			// Keep the byte unchanged if it's not an uppercase letter
			lower[i] = v
		}
	}

	return lower
}

func calculateComplexity(sum int, size int) float64 {
	return float64(sum) / float64(size+5)
}
