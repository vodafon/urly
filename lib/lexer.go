// lib/lexer.go

package lib

import (
	"bytes"
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
	vld uint16 = 0x01 // valid for all
	slh uint16 = 0x02 // slash
	dig uint16 = 0x03 // 0-9
	col uint16 = 0x04 // :

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

	eof uint16 = 0x40 // EOF
)

// Character class lookup table
var lookup = [257]uint16{
	/* 00-07  NUL, SOH, STX, ETX, EOT, ENQ, ACK, BEL */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* 08-0F  BS,  HT,  LF,  VT,  FF,  CR,  SO,  SI  */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* 10-17  DLE, DC1, DC2, DC3, DC4, NAK, SYN, ETB */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* 18-1F  CAN, EM,  SUB, ESC, FS,  GS,  RS,  US  */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* 21-27  SP ! " # $ % & '   */ inv, sp1, inv, sp2, sp3, sp4, sp5, sp6,
	/* 28-2F   ( ) * + , - . /   */ inv, inv, inv, sn1, inv, vld, vld, slh,
	/* 30-37   0 1 2 3 4 5 6 7   */ vld, vld, vld, vld, vld, vld, vld, vld,
	/* 38-3F   8 9 : ; < = > ?   */ vld, vld, col, inv, inv, sp7, inv, sp8,
	/* 40-47   @ A B C D E F G   */ sp9, vld, vld, vld, vld, vld, vld, vld,
	/* 48-4F   H I J K L M N O   */ vld, vld, vld, vld, vld, vld, vld, vld,
	/* 50-57   P Q R S T U V W   */ vld, vld, vld, vld, vld, vld, vld, vld,
	/* 58-5F   X Y Z [ \ ] ^ _   */ vld, vld, vld, inv, inv, inv, inv, sn0,
	/* 60-67   ` a b c d e f g   */ inv, vld, vld, vld, vld, vld, vld, vld,
	/* 68-6F   h i j k l m n o   */ vld, vld, vld, vld, vld, vld, vld, vld,
	/* 70-77   p q r s t u v w   */ vld, vld, vld, vld, vld, vld, vld, vld,
	/* 78-7F   x y z { | } ~ DEL */ vld, vld, vld, inv, inv, inv, inv, inv,
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
	input      []byte
	scheme     []byte
	schemeWord []byte
	schemeSep  []byte
	host       []byte
	path       []byte
	params     []byte
	pos        int
	width      int
	size       int
	start      int
	stateType  StateType
	tokenType  TokenType
	tokens     chan Token
}

func (obj *Lexer) run() {
	for state := lexText; state != nil; {
		state = state(obj)
	}
	close(obj.tokens)
}

func (obj *Lexer) emit(t TokenType) {
	// fullURL without host?
	if obj.tokenType == TokenFullURL && len(obj.host) == 0 {
		obj.emitUpdate()
		return
	}

	// pathURL without path?
	if obj.tokenType == TokenPathURL && len(obj.path) == 0 {
		obj.emitUpdate()
		return
	}

	// custom urls app:///path can be without host
	if obj.tokenType == TokenCustomURL && (len(obj.path) == 0 && len(obj.host) == 0) {
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

func (obj *Lexer) emitUpdate() {
	obj.setAt(obj.pos+1, obj.pos+1)
}

func (obj *Lexer) setAt(start, pos int) {
	obj.start = start
	obj.pos = pos
	obj.size = 0
	obj.stateType = StateScheme
	obj.tokenType = TokenFullURL
	obj.scheme = []byte{}
	obj.schemeWord = []byte{}
	obj.schemeSep = []byte{}
	obj.host = []byte{}
	obj.path = []byte{}
}

func (obj *Lexer) addType(t StateType) {
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
	if l.pos >= len(l.input) {
		l.width = 0
		return 256
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
	// valid, _
	if s == vld || s == sn0 {
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
	l.pos -= 1
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
	// valid, @, :
	if s == vld || s == sp9 || s == col {
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
	if s == vld || s == slh || s == sp1 || s == sp4 || s == sn0 {
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
	return s != inv
}

func (l *Lexer) isValid(r byte) bool {
	s := lookup[r]
	// fmt.Printf("10: %v %v %v %v\n", r, s, l.stateType, l.tokenType)
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
	return s == vld
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
	return bytes.Equal(word, httpW)
}

func isHttpsWord(word []byte) bool {
	return bytes.Equal(word, httpsW)
}

func isValidAll(ch byte) bool {
	return lookup[ch] == vld
}

func NewLexer(input []byte) *Lexer {
	l := &Lexer{
		input:     input,
		tokens:    make(chan Token),
		tokenType: TokenFullURL,
	}
	go l.run()
	return l
}
