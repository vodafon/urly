// lib/lexer.go

package lib

import (
	"bytes"
	"fmt"
)

type TokenType int
type StateType int

const (
	// Special tokens
	TokenFullURL TokenType = iota
	TokenPathURL
)

const (
	StateScheme StateType = iota
	StateSchemeSep
	StateHost
	StatePath
	StateParams
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
	sp3 uint16 = 0x23 // #
	sp4 uint16 = 0x24 // %
	sp5 uint16 = 0x25 // &
	sp6 uint16 = 0x26 // '
	sp7 uint16 = 0x27 // =
	sp8 uint16 = 0x28 // !
	sp9 uint16 = 0x28 // @
	sn0 uint16 = 0x30 // _

	eof uint16 = 0x40 // EOF
)

// Character class lookup table
var lookup = [257]uint16{
	/* 00-07  NUL, SOH, STX, ETX, EOT, ENQ, ACK, BEL */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* 08-0F  BS,  HT,  LF,  VT,  FF,  CR,  SO,  SI  */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* 10-17  DLE, DC1, DC2, DC3, DC4, NAK, SYN, ETB */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* 18-1F  CAN, EM,  SUB, ESC, FS,  GS,  RS,  US  */ inv, inv, inv, inv, inv, inv, inv, inv,
	/* 21-27  SP ! " # $ % & '   */ inv, sp1, inv, sp2, sp3, sp4, sp5, sp6,
	/* 28-2F   ( ) * + , - . /   */ inv, inv, inv, inv, inv, inv, vld, slh,
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
	input     []byte
	scheme    []byte
	schemeSep []byte
	host      []byte
	path      []byte
	params    []byte
	pos       int
	width     int
	size      int
	start     int
	stateType StateType
	tokenType TokenType
	tokens    chan Token
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
		fmt.Print(1)
		obj.emitUpdate()
		return
	}

	// pathURL without path?
	if obj.tokenType == TokenPathURL && len(obj.path) == 0 {
		fmt.Print(2)
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

func (l *Lexer) processSchemeValid(r byte) bool {
	if lookup[r] == col {
		l.schemeSep = append(l.schemeSep, r)
		l.stateType = StateSchemeSep
		// if scheme end with http/https we should cut prefix
		if l.size > 6 && isHttpsWord(l.input[l.pos-6:l.pos-1]) {
			l.start = l.pos - 6
			l.size = l.pos - l.start
		}
		if l.size > 5 && isHttpWord(l.input[l.pos-5:l.pos-1]) {
			l.start = l.pos - 5
			l.size = l.pos - l.start
		}
		return true
	}
	return isValidAll(r)
}

func (l *Lexer) processSchemeSepValid(r byte) bool {
	s := lookup[r]
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
			l.tokenType = TokenFullURL
			return true
		}
	}
	return false
}

func (l *Lexer) processHostValid(r byte) bool {
	if lookup[r] == slh {
		l.path = append(l.path, r)
		l.stateType = StatePath
		return true
	}
	if isValidAll(r) {
		l.host = append(l.host, r)
		return true
	}
	return false
}

func (l *Lexer) processPathValid(r byte) bool {
	s := lookup[r]
	if s == vld || s == slh {
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

func (l *Lexer) processParamsValid(r byte) bool {
	return lookup[r] != inv
}

func (l *Lexer) isValid(r byte) bool {
	switch l.stateType {
	case StateScheme:
		return l.processSchemeValid(r)
	case StateSchemeSep:
		return l.processSchemeSepValid(r)
	case StateHost:
		return l.processHostValid(r)
	case StatePath:
		return l.processPathValid(r)
	case StateParams:
		return l.processParamsValid(r)
	}
	return isValidAll(r)
}

type stateFn func(*Lexer) stateFn

func lexText(l *Lexer) stateFn {
	for {
		x := l.next()
		// fmt.Printf("[TEXT2]: %q - %+v\n", r, *l)

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
	httpW := []byte{104, 116, 116, 112}
	return bytes.Equal(word, httpW)
}

func isHttpsWord(word []byte) bool {
	httpW := []byte{104, 116, 116, 112, 115}
	return bytes.Equal(word, httpW)
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
