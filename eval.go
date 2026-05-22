package main

import (
	"fmt"
	"math/big"
)

// definitions holds globally DEFINE'd functions (simulating the LISP 1.5 oblist).
var definitions = make(map[string]*Expr)

// evalquote is the top-level entry point: evalquote[fn; x] = apply[fn; x; NIL]
func evalquote(fn, x *Expr) *Expr {
	return apply(fn, x, nil)
}

// apply applies function fn to argument list x with association list a.
// Implements apply[] from page 13 of the LISP 1.5 Programmer's Manual.
func apply(fn, x, a *Expr) *Expr {
	if fn == nil {
		panic("apply: NIL is not a function")
	}

	if fn.atom != "" {
		// fn is an atom — check built-ins first, then look it up.
		switch fn.atom {

		// ── Elementary functions (page 13) ───────────────────────────────
		case "CAR":
			return carOf(carOf(x))
		case "CDR":
			return cdrOf(carOf(x))
		case "CONS":
			return mkCons(carOf(x), carOf(cdrOf(x)))
		case "ATOM":
			return boolExpr(isAtom(carOf(x)))
		case "EQ":
			return boolExpr(eqExpr(carOf(x), carOf(cdrOf(x))))
		case "NULL":
			return boolExpr(isNull(carOf(x)))

		// ── CAAR … CDDDDR family ─────────────────────────────────────────
		default:
			if isCxR(fn.atom) {
				return applyCxR(fn.atom, carOf(x))
			}
			fallthrough
		case "DEFINE":
			if fn.atom == "DEFINE" {
				return doDefine(x)
			}
			// Structural / logical
			fallthrough
		case "EQUAL", "NOT", "AND", "OR",
			// Arithmetic
			"PLUS", "TIMES", "DIFFERENCE", "QUOTIENT", "REMAINDER",
			"ADD1", "SUB1",
			// Comparison
			"GREATERP", "LESSP", "NUMBERP", "ZEROP", "MINUSP",
			// List operations
			"LIST", "APPEND", "REVERSE", "LENGTH",
			"MEMBER", "ASSOC", "PAIR", "PAIRLIS",
			"MAPCAR", "APPLY", "PRINT":

			switch fn.atom {
			case "EQUAL":
				return boolExpr(equalExpr(carOf(x), carOf(cdrOf(x))))
			case "NOT":
				return boolExpr(!isTrue(carOf(x)))
			case "AND":
				return applyAnd(x)
			case "OR":
				return applyOr(x)

			case "PLUS":
				return arith2(x, func(a, b *big.Int) *big.Int { return new(big.Int).Add(a, b) })
			case "TIMES":
				return arith2(x, func(a, b *big.Int) *big.Int { return new(big.Int).Mul(a, b) })
			case "DIFFERENCE":
				return arith2(x, func(a, b *big.Int) *big.Int { return new(big.Int).Sub(a, b) })
			case "QUOTIENT":
				return arith2(x, func(a, b *big.Int) *big.Int {
					if b.Sign() == 0 {
						panic("QUOTIENT: division by zero")
					}
					return new(big.Int).Quo(a, b)
				})
			case "REMAINDER":
				return arith2(x, func(a, b *big.Int) *big.Int {
					if b.Sign() == 0 {
						panic("REMAINDER: division by zero")
					}
					return new(big.Int).Rem(a, b)
				})
			case "ADD1":
				return mkNum(new(big.Int).Add(mustNum(carOf(x)), big.NewInt(1)))
			case "SUB1":
				return mkNum(new(big.Int).Sub(mustNum(carOf(x)), big.NewInt(1)))

			case "GREATERP":
				return boolExpr(mustNum(carOf(x)).Cmp(mustNum(carOf(cdrOf(x)))) > 0)
			case "LESSP":
				return boolExpr(mustNum(carOf(x)).Cmp(mustNum(carOf(cdrOf(x)))) < 0)
			case "ZEROP":
				return boolExpr(mustNum(carOf(x)).Sign() == 0)
			case "MINUSP":
				return boolExpr(mustNum(carOf(x)).Sign() < 0)
			case "NUMBERP":
				v := carOf(x)
				return boolExpr(v != nil && v.num != nil)

			case "LIST":
				return x // args are already evaluated into a list
			case "APPEND":
				return appendExpr(carOf(x), carOf(cdrOf(x)))
			case "REVERSE":
				return reverseExpr(carOf(x), nil)
			case "LENGTH":
				return mkInt(int64(lengthOf(carOf(x))))
			case "MEMBER":
				return boolExpr(memberExpr(carOf(x), carOf(cdrOf(x))))
			case "ASSOC":
				p := assocLookup(carOf(x), carOf(cdrOf(x)))
				if p == nil {
					return nil
				}
				return p
			case "PAIR":
				return pairExpr(carOf(x), carOf(cdrOf(x)))
			case "PAIRLIS":
				return pairlisExpr(carOf(x), carOf(cdrOf(x)), carOf(cdrOf(cdrOf(x))))

			case "MAPCAR":
				return mapcarExpr(carOf(x), carOf(cdrOf(x)), a)
			case "APPLY":
				return apply(carOf(x), carOf(cdrOf(x)), a)
			case "PRINT":
				fmt.Printf(" %s\n", exprStr(carOf(x)))
				return carOf(x)
			}
		}

		// Not a built-in: look up in a-list then global definitions.
		pair := assocLookup(fn, a)
		if pair != nil {
			return apply(cdrOf(pair), x, a)
		}
		if def, ok := definitions[fn.atom]; ok {
			return apply(def, x, a)
		}
		panic("undefined function: " + fn.atom)
	}

	// fn is a list: must be a LAMBDA or LABEL expression.
	head := carOf(fn)
	if head != nil && head.atom != "" {
		switch head.atom {
		case "LAMBDA":
			// (LAMBDA (params...) body)
			params := carOf(cdrOf(fn))
			body := carOf(cdrOf(cdrOf(fn)))
			newA := pairlisExpr(params, x, a)
			return eval(body, newA)

		case "LABEL":
			// (LABEL name lambda) — self-referential recursion helper
			name := carOf(cdrOf(fn))
			lambda := carOf(cdrOf(cdrOf(fn)))
			newA := mkCons(mkCons(name, lambda), a)
			return apply(lambda, x, newA)
		}
	}

	panic(fmt.Sprintf("cannot apply: %s", exprStr(fn)))
}

// eval evaluates expression e with association list a.
// Implements eval[] from page 13 of the LISP 1.5 Programmer's Manual.
func eval(e, a *Expr) *Expr {
	if e == nil {
		return nil
	}

	// Atom: look up value.
	if e.atom != "" {
		if e.num != nil {
			return e // numbers are self-evaluating
		}
		switch e.atom {
		case "T":
			return exprT
		case "F":
			return exprF
		case "NIL":
			return nil
		case "*TRUE*":
			return exprTrue
		}
		// Search local a-list.
		pair := assocLookup(e, a)
		if pair != nil {
			return cdrOf(pair)
		}
		// Search global definitions.
		if def, ok := definitions[e.atom]; ok {
			return def
		}
		panic("unbound variable: " + e.atom)
	}

	// List: special form or function application.
	head := carOf(e)

	if head != nil && head.atom != "" {
		switch head.atom {
		case "QUOTE":
			return carOf(cdrOf(e))
		case "COND":
			return evcon(cdrOf(e), a)
		case "LAMBDA", "LABEL":
			return e // these evaluate to themselves
		case "DEFINE":
			// Rare: DEFINE used as an inner expression.
			return doDefine(evlis(cdrOf(e), a))
		default:
			// atom[car[e]] ∧ not a special form → apply[car[e]; evlis[cdr[e]; a]; a]
			return apply(head, evlis(cdrOf(e), a), a)
		}
	}

	// Head is a list (e.g. a lambda expression): eval then apply.
	fn := eval(head, a)
	return apply(fn, evlis(cdrOf(e), a), a)
}

// evcon evaluates a COND clause list.
func evcon(clauses, a *Expr) *Expr {
	if clauses == nil {
		panic("COND: no true clause")
	}
	clause := carOf(clauses)
	if isTrue(eval(carOf(clause), a)) {
		return eval(carOf(cdrOf(clause)), a)
	}
	return evcon(cdrOf(clauses), a)
}

// evlis evaluates each element of list m with a-list a.
func evlis(m, a *Expr) *Expr {
	if m == nil {
		return nil
	}
	return mkCons(eval(carOf(m), a), evlis(cdrOf(m), a))
}

// ── DEFINE ───────────────────────────────────────────────────────────────────

// doDefine processes: x = (((name1 body1) (name2 body2) ...))
func doDefine(x *Expr) *Expr {
	defList := carOf(x)
	for defList != nil {
		def := carOf(defList)
		name := carOf(def)
		body := carOf(cdrOf(def))
		if name == nil || name.atom == "" {
			panic("DEFINE: invalid function name")
		}
		definitions[name.atom] = body
		defList = cdrOf(defList)
	}
	return exprTrue
}

// ── A-list helpers ────────────────────────────────────────────────────────────

// pairlisExpr builds ((p1 . a1) (p2 . a2) ...) prepended to tail.
func pairlisExpr(params, args, tail *Expr) *Expr {
	if params == nil {
		return tail
	}
	return mkCons(
		mkCons(carOf(params), carOf(args)),
		pairlisExpr(cdrOf(params), cdrOf(args), tail),
	)
}

// assocLookup returns the first pair (key . val) in a where key == x, or nil.
func assocLookup(x, a *Expr) *Expr {
	for a != nil {
		pair := carOf(a)
		if eqExpr(carOf(pair), x) {
			return pair
		}
		a = cdrOf(a)
	}
	return nil
}

// ── Equality ─────────────────────────────────────────────────────────────────

// eqExpr implements EQ: true iff both are the same atom (pointer or value).
func eqExpr(a, b *Expr) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.atom == "" || b.atom == "" {
		return false // lists are never EQ
	}
	if a.num != nil && b.num != nil {
		return a.num.Cmp(b.num) == 0
	}
	return a.atom == b.atom
}

// equalExpr implements EQUAL: structural equality.
func equalExpr(a, b *Expr) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if isAtom(a) && isAtom(b) {
		return eqExpr(a, b)
	}
	if isAtom(a) || isAtom(b) {
		return false
	}
	return equalExpr(carOf(a), carOf(b)) && equalExpr(cdrOf(a), cdrOf(b))
}

// ── Arithmetic helpers ────────────────────────────────────────────────────────

func mustNum(e *Expr) *big.Int {
	if e == nil || e.num == nil {
		panic(fmt.Sprintf("expected number, got: %s", exprStr(e)))
	}
	return e.num
}

func arith2(x *Expr, op func(*big.Int, *big.Int) *big.Int) *Expr {
	return mkNum(op(mustNum(carOf(x)), mustNum(carOf(cdrOf(x)))))
}

// ── Boolean helpers ───────────────────────────────────────────────────────────

func applyAnd(x *Expr) *Expr {
	for x != nil {
		if !isTrue(carOf(x)) {
			return nil
		}
		x = cdrOf(x)
	}
	return exprT
}

func applyOr(x *Expr) *Expr {
	for x != nil {
		if isTrue(carOf(x)) {
			return exprT
		}
		x = cdrOf(x)
	}
	return nil
}

// ── List operations ───────────────────────────────────────────────────────────

func appendExpr(a, b *Expr) *Expr {
	if a == nil {
		return b
	}
	return mkCons(carOf(a), appendExpr(cdrOf(a), b))
}

func reverseExpr(lst, acc *Expr) *Expr {
	if lst == nil {
		return acc
	}
	return reverseExpr(cdrOf(lst), mkCons(carOf(lst), acc))
}

func lengthOf(lst *Expr) int {
	n := 0
	for lst != nil && lst.atom == "" {
		n++
		lst = cdrOf(lst)
	}
	return n
}

func memberExpr(x, lst *Expr) bool {
	for lst != nil {
		if equalExpr(x, carOf(lst)) {
			return true
		}
		lst = cdrOf(lst)
	}
	return false
}

func pairExpr(xs, ys *Expr) *Expr {
	if xs == nil {
		return nil
	}
	return mkCons(mkCons(carOf(xs), carOf(ys)), pairExpr(cdrOf(xs), cdrOf(ys)))
}

func mapcarExpr(fn, lst, a *Expr) *Expr {
	if lst == nil {
		return nil
	}
	result := apply(fn, mkCons(carOf(lst), nil), a)
	return mkCons(result, mapcarExpr(fn, cdrOf(lst), a))
}

// ── CxR family (CAAR, CADR, CDAR, CDDR, … CDDDDR) ───────────────────────────

func isCxR(s string) bool {
	if len(s) < 3 || s[0] != 'C' || s[len(s)-1] != 'R' {
		return false
	}
	for _, c := range s[1 : len(s)-1] {
		if c != 'A' && c != 'D' {
			return false
		}
	}
	return true
}

func applyCxR(name string, e *Expr) *Expr {
	// Work right-to-left through the middle letters.
	inner := name[1 : len(name)-1]
	for i := len(inner) - 1; i >= 0; i-- {
		if inner[i] == 'A' {
			e = carOf(e)
		} else {
			e = cdrOf(e)
		}
	}
	return e
}
