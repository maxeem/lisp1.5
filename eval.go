package main

import (
	"fmt"
	"math/big"
)

// definitions holds globally DEFINE'd functions (simulating the LISP 1.5 oblist).
var definitions = make(map[string]*Expr)

// ── PROG state ────────────────────────────────────────────────────────────────

// progFrame holds the mutable local variables of one active PROG invocation.
type progFrame struct {
	vars map[string]*Expr
	prev *progFrame
}

// progStack is the stack of active PROG frames (innermost first).
var progStack *progFrame

// progReturn / progGo are used to signal RETURN and GO out of nested eval calls.
type progReturn struct{ val *Expr }
type progGo struct{ label string }

func pushProgFrame(vars *Expr) {
	f := &progFrame{vars: make(map[string]*Expr), prev: progStack}
	for v := vars; v != nil; v = cdrOf(v) {
		if n := carOf(v); n != nil {
			f.vars[n.atom] = nil
		}
	}
	progStack = f
}

func popProgFrame() {
	if progStack != nil {
		progStack = progStack.prev
	}
}

// setqInProg tries to update a PROG-local variable. Returns true if found.
func setqInProg(name string, val *Expr) bool {
	for f := progStack; f != nil; f = f.prev {
		if _, ok := f.vars[name]; ok {
			f.vars[name] = val
			return true
		}
	}
	return false
}

// lookupInProg returns the value of a PROG-local variable, or (nil, false).
func lookupInProg(name string) (*Expr, bool) {
	for f := progStack; f != nil; f = f.prev {
		if val, ok := f.vars[name]; ok {
			return val, true
		}
	}
	return nil, false
}

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
		// CxR family: CAAR, CADR, CDAR, CDDR, … CDDDDR
		if isCxR(fn.atom) {
			return applyCxR(fn.atom, carOf(x))
		}

		switch fn.atom {
		// ── Elementary functions (page 13) ─────────────────────────────
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

		// ── Equality / logic ────────────────────────────────────────────
		case "EQUAL":
			return boolExpr(equalExpr(carOf(x), carOf(cdrOf(x))))
		case "NOT":
			return boolExpr(!isTrue(carOf(x)))
		case "AND":
			return applyAnd(x)
		case "OR":
			return applyOr(x)

		// ── Arithmetic ──────────────────────────────────────────────────
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

		// ── Numeric predicates ──────────────────────────────────────────
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

		// ── List operations ─────────────────────────────────────────────
		case "LIST":
			return x
		case "APPEND":
			return appendExpr(carOf(x), carOf(cdrOf(x)))
		case "REVERSE":
			return reverseExpr(carOf(x), nil)
		case "LENGTH":
			return mkInt(int64(lengthOf(carOf(x))))
		case "MEMBER":
			return boolExpr(memberExpr(carOf(x), carOf(cdrOf(x))))
		case "ASSOC":
			return assocLookup(carOf(x), carOf(cdrOf(x)))
		case "PAIR":
			return pairExpr(carOf(x), carOf(cdrOf(x)))
		case "PAIRLIS":
			return pairlisExpr(carOf(x), carOf(cdrOf(x)), carOf(cdrOf(cdrOf(x))))
		case "MAPCAR":
			return mapcarExpr(carOf(x), carOf(cdrOf(x)), a)

		// ── Meta / I/O ──────────────────────────────────────────────────
		case "APPLY":
			return apply(carOf(x), carOf(cdrOf(x)), a)
		case "PRINT":
			fmt.Printf(" %s\n", exprStr(carOf(x)))
			return carOf(x)

		// ── Global definition ───────────────────────────────────────────
		case "DEFINE":
			return doDefine(x)

		// ── QUOTE in EVALQUOTE context: returns its first argument ───────
		case "QUOTE":
			return carOf(x)

		// ── LABEL in EVALQUOTE context: ((name fn) args) ─────────────────
		// e.g. LABEL ((FAC (LAMBDA (N) ...)) (6))
		case "LABEL":
			nameFn := carOf(x)
			name := carOf(nameFn)
			fn := carOf(cdrOf(nameFn))
			actualArgs := carOf(cdrOf(x))
			newA := mkCons(mkCons(name, fn), a)
			return apply(fn, actualArgs, newA)

		default:
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
	}

	// fn is a list: must be a LAMBDA or LABEL expression.
	head := carOf(fn)
	if head != nil && head.atom != "" {
		switch head.atom {
		case "LAMBDA":
			params := carOf(cdrOf(fn))
			body := carOf(cdrOf(cdrOf(fn)))
			return eval(body, pairlisExpr(params, x, a))

		case "LABEL":
			// (LABEL name lambda)
			name := carOf(cdrOf(fn))
			lambda := carOf(cdrOf(cdrOf(fn)))
			return apply(lambda, x, mkCons(mkCons(name, lambda), a))
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
			return exprT // T is the logical-true constant
		case "F":
			return nil // F = NIL in this implementation
		case "*TRUE*":
			return exprTrue
		}
		// Search local a-list.
		pair := assocLookup(e, a)
		if pair != nil {
			return cdrOf(pair)
		}
		// Search active PROG frames (for mutable SETQ variables).
		if val, ok := lookupInProg(e.atom); ok {
			return val
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
			return doDefine(evlis(cdrOf(e), a))
		case "PROG":
			return evalProg(cdrOf(e), a)
		case "RETURN":
			panic(progReturn{eval(carOf(cdrOf(e)), a)})
		case "GO":
			lbl := carOf(cdrOf(e))
			if lbl == nil || lbl.atom == "" {
				panic("GO: expected label atom")
			}
			panic(progGo{lbl.atom})
		case "SETQ":
			vname := carOf(cdrOf(e))
			val := eval(carOf(cdrOf(cdrOf(e))), a)
			if !setqInProg(vname.atom, val) {
				definitions[vname.atom] = val
			}
			return val
		case "SET":
			vname := eval(carOf(cdrOf(e)), a)
			val := eval(carOf(cdrOf(cdrOf(e))), a)
			if vname == nil || vname.atom == "" {
				panic("SET: first arg must evaluate to an atom")
			}
			if !setqInProg(vname.atom, val) {
				definitions[vname.atom] = val
			}
			return val
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

// ── PROG ─────────────────────────────────────────────────────────────────────

// evalProg executes a PROG form: args = ((vars...) stmt1 stmt2 ...)
func evalProg(args, a *Expr) *Expr {
	vars := carOf(args)
	body := cdrOf(args)

	pushProgFrame(vars)
	defer popProgFrame()

	cur := body
	for cur != nil {
		stmt := carOf(cur)
		cur = cdrOf(cur)

		if isAtom(stmt) && stmt != nil {
			continue // atom label — skip, used as GO target
		}

		head := carOf(stmt)
		if head == nil {
			continue
		}

		switch head.atom {
		case "RETURN":
			return eval(carOf(cdrOf(stmt)), a)
		case "GO":
			lbl := carOf(cdrOf(stmt))
			if lbl == nil || lbl.atom == "" {
				panic("GO: expected label atom")
			}
			next := findProgLabel(lbl.atom, body)
			if next == nil {
				panic("GO: label not found: " + lbl.atom)
			}
			cur = next
		case "SETQ":
			vname := carOf(cdrOf(stmt))
			val := eval(carOf(cdrOf(cdrOf(stmt))), a)
			if !setqInProg(vname.atom, val) {
				definitions[vname.atom] = val
			}
		case "SET":
			vname := eval(carOf(cdrOf(stmt)), a)
			val := eval(carOf(cdrOf(cdrOf(stmt))), a)
			if vname == nil || vname.atom == "" {
				panic("SET: first arg must be an atom")
			}
			if !setqInProg(vname.atom, val) {
				definitions[vname.atom] = val
			}
		default:
			// General expression — evaluate for side effects.
			// Catch RETURN/GO panics from nested eval (e.g. inside COND).
			var goLabel string
			var returnVal *Expr
			returned := false

			func() {
				defer func() {
					if r := recover(); r != nil {
						switch v := r.(type) {
						case progReturn:
							returnVal = v.val
							returned = true
						case progGo:
							goLabel = v.label
						default:
							panic(r)
						}
					}
				}()
				eval(stmt, a)
			}()

			if returned {
				return returnVal
			}
			if goLabel != "" {
				next := findProgLabel(goLabel, body)
				if next == nil {
					panic("GO: label not found: " + goLabel)
				}
				cur = next
			}
		}
	}
	return nil // PROG falls off end → NIL
}

// findProgLabel returns the statements starting AFTER the given label atom.
func findProgLabel(label string, stmts *Expr) *Expr {
	for stmts != nil {
		s := carOf(stmts)
		if isAtom(s) && s != nil && s.atom == label {
			return cdrOf(stmts)
		}
		stmts = cdrOf(stmts)
	}
	return nil
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
