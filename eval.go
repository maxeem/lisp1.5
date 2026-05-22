package main

import (
	"fmt"
	"math/big"
	"strings"
)

// definitions holds globally DEFINE'd functions (simulating the LISP 1.5 oblist).
var definitions = make(map[string]*Expr)

var gensymCounter int

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

// setqInProg stores val in the PROG frame where name is already bound,
// or in the innermost frame if no frame has it yet.
// Returns false (no active PROG) if progStack is nil.
func setqInProg(name string, val *Expr) bool {
	if progStack == nil {
		return false
	}
	// Update in the frame that already owns this name.
	for f := progStack; f != nil; f = f.prev {
		if _, ok := f.vars[name]; ok {
			f.vars[name] = val
			return true
		}
	}
	// Not declared in any frame — store in innermost (handles lambda params).
	progStack.vars[name] = val
	return true
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
		case "CONC":
			return appendExpr(carOf(x), carOf(cdrOf(x)))

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
		case "NUMBERP", "FIXP":
			v := carOf(x)
			return boolExpr(v != nil && v.num != nil)
		case "FLOATP":
			return nil // no floating-point in this implementation

		// ── Arithmetic extensions ────────────────────────────────────────
		case "MAX":
			a2, b2 := mustNum(carOf(x)), mustNum(carOf(cdrOf(x)))
			if a2.Cmp(b2) >= 0 {
				return mkNum(new(big.Int).Set(a2))
			}
			return mkNum(new(big.Int).Set(b2))
		case "MIN":
			a2, b2 := mustNum(carOf(x)), mustNum(carOf(cdrOf(x)))
			if a2.Cmp(b2) <= 0 {
				return mkNum(new(big.Int).Set(a2))
			}
			return mkNum(new(big.Int).Set(b2))
		case "ABS":
			return mkNum(new(big.Int).Abs(mustNum(carOf(x))))
		case "EXPT":
			base2 := mustNum(carOf(x))
			exp2 := mustNum(carOf(cdrOf(x)))
			if exp2.Sign() < 0 {
				panic("EXPT: negative exponent")
			}
			return mkNum(new(big.Int).Exp(base2, exp2, nil))

		// ── List operations ─────────────────────────────────────────────
		case "LIST":
			return x
		case "APPEND", "NCONC":
			return appendExpr(carOf(x), carOf(cdrOf(x)))
		case "REVERSE":
			return reverseExpr(carOf(x), nil)
		case "LENGTH":
			return mkInt(int64(lengthOf(carOf(x))))
		case "LAST":
			return lastExpr(carOf(x))
		case "MEMBER":
			return boolExpr(memberExpr(carOf(x), carOf(cdrOf(x))))
		case "EFFACE":
			return effaceExpr(carOf(x), carOf(cdrOf(x)))
		case "ASSOC":
			return assocLookup(carOf(x), carOf(cdrOf(x)))
		case "SASSOC":
			// (SASSOC x y u) — assoc with not-found function u
			result := assocLookup(carOf(x), carOf(cdrOf(x)))
			if result != nil {
				return result
			}
			return apply(carOf(cdrOf(cdrOf(x))), nil, a)
		case "SUBST":
			// (SUBST x y z) — substitute x for every occurrence of y in z
			return substExpr(carOf(x), carOf(cdrOf(x)), carOf(cdrOf(cdrOf(x))))
		case "SUBLIS":
			// (SUBLIS a z) — substitute from a-list a into z
			return sublisExpr(carOf(x), carOf(cdrOf(x)))
		case "COPY":
			return copyExpr(carOf(x))
		case "PAIR":
			return pairExpr(carOf(x), carOf(cdrOf(x)))
		case "PAIRLIS":
			return pairlisExpr(carOf(x), carOf(cdrOf(x)), carOf(cdrOf(cdrOf(x))))
		case "MAPCAR":
			return mapcarExpr(carOf(x), carOf(cdrOf(x)), a)
		case "MAPLIST":
			return maplistExpr(carOf(x), carOf(cdrOf(x)), a)
		case "MAPC":
			return mapcExpr(carOf(x), carOf(cdrOf(x)), a)
		case "SEARCH":
			// (SEARCH x p f u) — search list x for element satisfying p;
			// apply f to it if found, apply u to no args if not found.
			return searchExpr(carOf(x), carOf(cdrOf(x)), carOf(cdrOf(cdrOf(x))), carOf(cdrOf(cdrOf(cdrOf(x)))), a)

		// ── Property lists ───────────────────────────────────────────────
		case "PUTPROP":
			// (PUTPROP atom value indicator)
			return doPutProp(carOf(x), carOf(cdrOf(x)), carOf(cdrOf(cdrOf(x))))
		case "GET":
			// (GET atom indicator)
			return doGet(carOf(x), carOf(cdrOf(x)))
		case "REMPROP":
			// (REMPROP atom indicator)
			return doRemProp(carOf(x), carOf(cdrOf(x)))
		case "DEFLIST":
			// (DEFLIST ((atom val) ...) indicator)
			return doDefList(carOf(x), carOf(cdrOf(x)))
		case "PROP":
			// (PROP atom indicator u) — GET with not-found function
			result := doGet(carOf(x), carOf(cdrOf(x)))
			if result != nil {
				return result
			}
			return apply(carOf(cdrOf(cdrOf(x))), nil, a)

		// ── Meta / I/O ──────────────────────────────────────────────────
		case "APPLY":
			return apply(carOf(x), carOf(cdrOf(x)), a)
		case "EVAL":
			// (EVAL expr a-list)
			return eval(carOf(x), carOf(cdrOf(x)))
		case "PRINT":
			fmt.Printf(" %s\n", exprStr(carOf(x)))
			return carOf(x)
		case "TERPRI":
			fmt.Println()
			return nil
		case "GENSYM":
			gensymCounter++
			return mkSym(fmt.Sprintf("G%04d", gensymCounter))

		// ── Unary arithmetic ─────────────────────────────────────────────
		case "MINUS":
			return mkNum(new(big.Int).Neg(mustNum(carOf(x))))

		// ── Bitwise (operate on integer values) ──────────────────────────
		case "LOGOR":
			return arith2(x, func(a, b *big.Int) *big.Int { return new(big.Int).Or(a, b) })
		case "LOGAND":
			return arith2(x, func(a, b *big.Int) *big.Int { return new(big.Int).And(a, b) })
		case "LOGXOR":
			return arith2(x, func(a, b *big.Int) *big.Int { return new(big.Int).Xor(a, b) })
		case "LEFTSHIFT":
			// (LEFTSHIFT n count) — positive count shifts left, negative right
			n := mustNum(carOf(x))
			shift := mustNum(carOf(cdrOf(x)))
			result := new(big.Int).Set(n)
			if shift.Sign() >= 0 {
				result.Lsh(result, uint(shift.Int64()))
			} else {
				result.Rsh(result, uint(-shift.Int64()))
			}
			return mkNum(result)

		// ── Atom ↔ character-list conversion ────────────────────────────
		case "EXPLODE":
			return explodeExpr(carOf(x))
		case "INTERN":
			return internExpr(carOf(x))

		// ── Oblist ───────────────────────────────────────────────────────
		case "OBLIST":
			return oblistExpr()

		// ── Global definition ───────────────────────────────────────────
		case "DEFINE":
			return doDefine(x)

		// ── QUOTE in EVALQUOTE context: returns its first argument ───────
		case "QUOTE":
			return carOf(x)

		// ── SETQ/CSETQ in EVALQUOTE context: (SETQ varname value) ────────
		case "SETQ", "CSETQ":
			vname := carOf(x)
			if vname == nil || vname.atom == "" {
				panic("SETQ: first arg must be an atom")
			}
			val := eval(carOf(cdrOf(x)), a)
			definitions[vname.atom] = val
			return val

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

	// fn is a list: must be a LAMBDA, LABEL, FUNARG, or FEXPR expression.
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

		case "FUNARG":
			// (FUNARG lambda saved-a) — call lambda with saved a-list
			lambda := carOf(cdrOf(fn))
			savedA := carOf(cdrOf(cdrOf(fn)))
			return apply(lambda, x, savedA)

		case "FEXPR":
			// (FEXPR (argname alistname) body)
			// Called with unevaluated arg list and current a-list.
			params := carOf(cdrOf(fn))
			body := carOf(cdrOf(cdrOf(fn)))
			argParam := carOf(params)
			aParam := carOf(cdrOf(params))
			newA := a
			if argParam != nil && argParam.atom != "" {
				newA = mkCons(mkCons(argParam, x), newA)
			}
			if aParam != nil && aParam.atom != "" {
				newA = mkCons(mkCons(aParam, a), newA)
			}
			return eval(body, newA)
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
		// PROG frames take priority over the a-list so SETQ can shadow lambda params.
		if val, ok := lookupInProg(e.atom); ok {
			return val
		}
		// Search local a-list (lambda params and outer bindings).
		pair := assocLookup(e, a)
		if pair != nil {
			return cdrOf(pair)
		}
		// Self-evaluating truth constants (checked after a-list so they can be shadowed).
		switch e.atom {
		case "T":
			return exprT
		case "*TRUE*":
			return exprTrue
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
		case "LAMBDA", "LABEL", "FEXPR":
			return e // these evaluate to themselves
		case "COMMENT", "IGNORE":
			return nil // documentation forms — evaluate to NIL
		case "PROG2":
			// (PROG2 e1 e2) — eval both, return second
			eval(carOf(cdrOf(e)), a)
			return eval(carOf(cdrOf(cdrOf(e))), a)
		case "AND":
			// Short-circuit AND: evaluate args left-to-right
			for args := cdrOf(e); args != nil; args = cdrOf(args) {
				if !isTrue(eval(carOf(args), a)) {
					return nil
				}
			}
			return exprTrue
		case "OR":
			// Short-circuit OR: evaluate args left-to-right
			for args := cdrOf(e); args != nil; args = cdrOf(args) {
				if v := eval(carOf(args), a); isTrue(v) {
					return exprTrue
				}
			}
			return nil
		case "FUNCTION":
			// (FUNCTION lambda) — capture current a-list for dynamic-to-lexical bridge
			return mkCons(mkSym("FUNARG"), mkCons(carOf(cdrOf(e)), mkCons(a, nil)))
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
		case "SETQ", "CSETQ":
			vname := carOf(cdrOf(e))
			val := eval(carOf(cdrOf(cdrOf(e))), a)
			if head.atom == "CSETQ" || !setqInProg(vname.atom, val) {
				definitions[vname.atom] = val
			}
			return val
		case "SET", "CSET":
			vname := eval(carOf(cdrOf(e)), a)
			val := eval(carOf(cdrOf(cdrOf(e))), a)
			if vname == nil || vname.atom == "" {
				panic("SET: first arg must evaluate to an atom")
			}
			if head.atom == "CSET" || !setqInProg(vname.atom, val) {
				definitions[vname.atom] = val
			}
			return val
		case "ERROR":
			// (ERROR ...) — collect unevaluated args as error message
			panic(fmt.Sprintf("ERROR: %s", exprStr(cdrOf(e))))
		default:
			// Check if the function is an FEXPR — skip evlis if so
			fnDef := lookupFn(head.atom, a)
			if fnDef != nil && carOf(fnDef) != nil && carOf(fnDef).atom == "FEXPR" {
				return apply(fnDef, cdrOf(e), a)
			}
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

// lookupFn returns the function body for name from a-list then definitions, or nil.
func lookupFn(name string, a *Expr) *Expr {
	sym := mkSym(name)
	pair := assocLookup(sym, a)
	if pair != nil {
		return cdrOf(pair)
	}
	if def, ok := definitions[name]; ok {
		return def
	}
	return nil
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
	return exprTrue
}

func applyOr(x *Expr) *Expr {
	for x != nil {
		if isTrue(carOf(x)) {
			return exprTrue
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

// ── List extensions ───────────────────────────────────────────────────────────

func lastExpr(lst *Expr) *Expr {
	if lst == nil {
		return nil
	}
	if cdrOf(lst) == nil {
		return lst
	}
	return lastExpr(cdrOf(lst))
}

func effaceExpr(x, lst *Expr) *Expr {
	if lst == nil {
		return nil
	}
	if equalExpr(x, carOf(lst)) {
		return cdrOf(lst)
	}
	return mkCons(carOf(lst), effaceExpr(x, cdrOf(lst)))
}

func substExpr(x, y, z *Expr) *Expr {
	if equalExpr(y, z) {
		return x
	}
	if isAtom(z) {
		return z
	}
	return mkCons(substExpr(x, y, carOf(z)), substExpr(x, y, cdrOf(z)))
}

func sublisExpr(a, z *Expr) *Expr {
	if isAtom(z) {
		pair := assocLookup(z, a)
		if pair != nil {
			return cdrOf(pair)
		}
		return z
	}
	return mkCons(sublisExpr(a, carOf(z)), sublisExpr(a, cdrOf(z)))
}

func copyExpr(e *Expr) *Expr {
	if e == nil || isAtom(e) {
		return e
	}
	return mkCons(copyExpr(carOf(e)), copyExpr(cdrOf(e)))
}

func maplistExpr(fn, lst, a *Expr) *Expr {
	if lst == nil {
		return nil
	}
	result := apply(fn, mkCons(lst, nil), a)
	return mkCons(result, maplistExpr(fn, cdrOf(lst), a))
}

func mapcExpr(fn, lst, a *Expr) *Expr {
	for lst != nil {
		apply(fn, mkCons(carOf(lst), nil), a)
		lst = cdrOf(lst)
	}
	return nil
}

func searchExpr(lst, pred, found, notFound, a *Expr) *Expr {
	for lst != nil {
		if isTrue(apply(pred, mkCons(carOf(lst), nil), a)) {
			return apply(found, mkCons(carOf(lst), nil), a)
		}
		lst = cdrOf(lst)
	}
	return apply(notFound, nil, a)
}

// ── Atom / character conversion ───────────────────────────────────────────────

func explodeExpr(e *Expr) *Expr {
	var s string
	if e == nil {
		s = "NIL"
	} else if e.num != nil {
		s = e.num.String()
	} else {
		s = e.atom
	}
	var result *Expr
	for i := len(s) - 1; i >= 0; i-- {
		result = mkCons(mkSym(string(s[i])), result)
	}
	return result
}

func internExpr(lst *Expr) *Expr {
	var sb strings.Builder
	for lst != nil {
		ch := carOf(lst)
		if ch == nil || ch.atom == "" {
			panic("INTERN: list must contain single-char atoms")
		}
		sb.WriteString(ch.atom)
		lst = cdrOf(lst)
	}
	return mkSym(sb.String())
}

func oblistExpr() *Expr {
	var result *Expr
	for name := range definitions {
		result = mkCons(mkSym(name), result)
	}
	return result
}

// ── Property lists ────────────────────────────────────────────────────────────

var propLists = make(map[string]map[string]*Expr)

func doPutProp(atom, value, indicator *Expr) *Expr {
	if atom == nil || atom.atom == "" {
		panic("PUTPROP: first arg must be an atom")
	}
	if indicator == nil || indicator.atom == "" {
		panic("PUTPROP: third arg must be an atom indicator")
	}
	if propLists[atom.atom] == nil {
		propLists[atom.atom] = make(map[string]*Expr)
	}
	propLists[atom.atom][indicator.atom] = value
	return value
}

func doGet(atom, indicator *Expr) *Expr {
	if atom == nil || atom.atom == "" {
		return nil
	}
	if indicator == nil || indicator.atom == "" {
		return nil
	}
	if m := propLists[atom.atom]; m != nil {
		return m[indicator.atom]
	}
	return nil
}

func doRemProp(atom, indicator *Expr) *Expr {
	if atom == nil || atom.atom == "" {
		return nil
	}
	if indicator == nil || indicator.atom == "" {
		return nil
	}
	if m := propLists[atom.atom]; m != nil {
		old := m[indicator.atom]
		delete(m, indicator.atom)
		return old
	}
	return nil
}

func doDefList(pairs, indicator *Expr) *Expr {
	if indicator == nil || indicator.atom == "" {
		panic("DEFLIST: second arg must be an atom indicator")
	}
	for pairs != nil {
		pair := carOf(pairs)
		atom := carOf(pair)
		value := carOf(cdrOf(pair))
		doPutProp(atom, value, indicator)
		pairs = cdrOf(pairs)
	}
	return exprTrue
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
