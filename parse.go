package main

import (
	"fmt"
	"math/big"
)

type parser struct {
	lex    *lexer
	peeked bool
	peekedT token
}

func newParser(input string) *parser {
	return &parser{lex: newLexer(input)}
}

func (p *parser) next() token {
	if p.peeked {
		p.peeked = false
		return p.peekedT
	}
	return p.lex.next()
}

func (p *parser) peek() token {
	if !p.peeked {
		p.peekedT = p.lex.next()
		p.peeked = true
	}
	return p.peekedT
}

func (p *parser) back(t token) {
	p.peeked = true
	p.peekedT = t
}

// done reports whether there is no more input.
func (p *parser) done() bool {
	return p.peek().typ == tokEOF
}

// skipSpace skips whitespace (the underlying lexer already skips it in next(),
// but this lets main peek at the next rune type without consuming a token).
func (p *parser) skipSpace() {
	p.lex.skipSpace()
}

// parseExpr parses a single S-expression.
func (p *parser) parseExpr() *Expr {
	t := p.next()
	switch t.typ {
	case tokEOF:
		return nil
	case tokAtom:
		return mkSym(t.text)
	case tokNumber:
		n := new(big.Int)
		n.SetString(t.text, 10)
		return mkNum(n)
	case tokQuote:
		inner := p.parseExpr()
		return mkCons(mkSym("QUOTE"), mkCons(inner, nil))
	case tokLpar:
		return p.parseListBody()
	case tokRpar:
		p.back(t)
		return nil
	case tokDot:
		panic("unexpected '.' at top level")
	}
	panic(fmt.Sprintf("parseExpr: unexpected token %v", t))
}

// parseListBody parses the interior of a list after '(' has been consumed.
func (p *parser) parseListBody() *Expr {
	t := p.peek()
	switch t.typ {
	case tokRpar:
		p.next()
		return nil // ()  =  NIL
	case tokEOF:
		panic("unexpected EOF inside list")
	case tokDot:
		panic("unexpected '.' at start of list")
	}

	head := p.parseExpr()

	// Check for dotted-pair notation: (a . b)
	if p.peek().typ == tokDot {
		p.next() // consume '.'
		tail := p.parseExpr()
		rpar := p.next()
		if rpar.typ != tokRpar {
			panic(fmt.Sprintf("expected ')', got %v", rpar))
		}
		return mkCons(head, tail)
	}

	return mkCons(head, p.parseListBody())
}
