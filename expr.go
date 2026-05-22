package main

import (
	"fmt"
	"math/big"
	"strings"
)

// Expr is a LISP 1.5 S-expression.
//
// Representation:
//   - nil          → NIL (empty list / logical false)
//   - atom!=""     → symbol atom (e.g. "FACTORIAL", "T", "F")
//   - atom!="", num!=nil → number (atom holds the decimal string for printing)
//   - atom==""     → cons cell, car/cdr set
type Expr struct {
	atom string
	num  *big.Int
	car  *Expr
	cdr  *Expr
}

var (
	exprT    = &Expr{atom: "T"}
	exprF    = &Expr{atom: "F"}
	exprTrue = &Expr{atom: "*TRUE*"}
)

func mkSym(s string) *Expr {
	u := strings.ToUpper(s)
	if u == "NIL" {
		return nil // NIL is always the empty list / Go nil
	}
	return &Expr{atom: u}
}

func mkNum(n *big.Int) *Expr {
	return &Expr{atom: n.String(), num: new(big.Int).Set(n)}
}

func mkInt(n int64) *Expr {
	return mkNum(big.NewInt(n))
}

func mkCons(a, d *Expr) *Expr {
	return &Expr{car: a, cdr: d}
}

// isAtom reports whether e is an atom (symbol or number), including nil (NIL).
func isAtom(e *Expr) bool {
	return e == nil || e.atom != ""
}

// isNull reports whether e is NIL.
func isNull(e *Expr) bool {
	return e == nil
}

// isTrue reports whether e is a true value (not NIL, not F).
func isTrue(e *Expr) bool {
	if e == nil {
		return false
	}
	if e.atom == "F" {
		return false
	}
	return true
}

// boolExpr converts a Go bool to *TRUE* or NIL, matching the IBM 7094 emulator.
func boolExpr(b bool) *Expr {
	if b {
		return exprTrue
	}
	return nil
}

// carOf returns the CAR of e, or nil if e is an atom.
func carOf(e *Expr) *Expr {
	if e == nil || e.atom != "" {
		return nil
	}
	return e.car
}

// cdrOf returns the CDR of e, or nil if e is an atom.
func cdrOf(e *Expr) *Expr {
	if e == nil || e.atom != "" {
		return nil
	}
	return e.cdr
}

// String returns the printed representation of e.
func (e *Expr) String() string {
	return exprStr(e)
}

func exprStr(e *Expr) string {
	if e == nil {
		return "NIL"
	}
	if e.atom != "" {
		return e.atom
	}
	var sb strings.Builder
	sb.WriteByte('(')
	cur := e
	for {
		fmt.Fprint(&sb, exprStr(cur.car))
		tail := cur.cdr
		if tail == nil {
			break
		}
		if tail.atom != "" {
			// Dotted pair: (a . b)
			fmt.Fprintf(&sb, " . %s", tail.atom)
			break
		}
		sb.WriteByte(' ')
		cur = tail
	}
	sb.WriteByte(')')
	return sb.String()
}
