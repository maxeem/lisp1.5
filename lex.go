package main

import (
	"fmt"
	"strings"
	"unicode"
)

type tokType int

const (
	tokEOF tokType = iota
	tokAtom
	tokNumber
	tokLpar
	tokRpar
	tokDot
	tokQuote
)

type token struct {
	typ  tokType
	text string
}

func (t token) String() string {
	switch t.typ {
	case tokEOF:
		return "EOF"
	case tokLpar:
		return "("
	case tokRpar:
		return ")"
	case tokDot:
		return "."
	case tokQuote:
		return "'"
	default:
		return t.text
	}
}

type lexer struct {
	input []rune
	pos   int
}

func newLexer(s string) *lexer {
	return &lexer{input: []rune(s)}
}

func (l *lexer) peek() rune {
	if l.pos >= len(l.input) {
		return -1
	}
	return l.input[l.pos]
}

func (l *lexer) advance() rune {
	r := l.peek()
	if r != -1 {
		l.pos++
	}
	return r
}

// skipSpace skips whitespace and ';' comments.
func (l *lexer) skipSpace() {
	for {
		r := l.peek()
		if r == -1 {
			return
		}
		if r == ';' {
			for l.peek() != '\n' && l.peek() != -1 {
				l.advance()
			}
			continue
		}
		if !unicode.IsSpace(r) {
			return
		}
		l.advance()
	}
}

func (l *lexer) next() token {
	l.skipSpace()
	r := l.advance()
	switch {
	case r == -1:
		return token{tokEOF, ""}
	case r == '(':
		return token{tokLpar, "("}
	case r == ')':
		return token{tokRpar, ")"}
	case r == '.':
		return token{tokDot, "."}
	case r == '\'':
		return token{tokQuote, "'"}
	case r == '-':
		if n := l.peek(); n >= '0' && n <= '9' {
			return l.readNumber(r)
		}
		return l.readAtom(r)
	case r >= '0' && r <= '9':
		return l.readNumber(r)
	case unicode.IsLetter(r) || r == '*' || r == '+' || r == '_':
		return l.readAtom(r)
	default:
		panic(fmt.Sprintf("unexpected character: %q", r))
	}
}

func isAtomChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '*' || r == '-' || r == '_'
}

func (l *lexer) readAtom(first rune) token {
	var sb strings.Builder
	sb.WriteRune(unicode.ToUpper(first))
	for isAtomChar(l.peek()) {
		sb.WriteRune(unicode.ToUpper(l.advance()))
	}
	return token{tokAtom, sb.String()}
}

func (l *lexer) readNumber(first rune) token {
	var sb strings.Builder
	sb.WriteRune(first)
	for r := l.peek(); r >= '0' && r <= '9'; r = l.peek() {
		sb.WriteRune(l.advance())
	}
	return token{tokNumber, sb.String()}
}
