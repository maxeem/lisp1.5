package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ── Tutorial content ─────────────────────────────────────────────────────────

type lesson struct {
	title   string
	content string // pre-formatted, 76 chars wide with 2-space indent
}

var lessons = []lesson{
	{
		title: "INTRODUCTION",
		content: `
  Welcome to the LISP 1.5 Interactive Tutorial.

  LISP 1.5 was designed by John McCarthy at MIT and implemented on the
  IBM 7094 at MIT in 1962.  It is one of the oldest programming languages
  still in active use and introduced fundamental concepts such as
  automatic memory management, recursive functions, S-expressions, and
  higher-order functions.

  This interpreter is compatible with the IBM 7094 LISP 1.5 emulator.
  Programs consist of EVALQUOTE pairs: a function name followed by its
  argument list.  The interpreter evaluates each pair and prints the
  result.

  EVALQUOTE INTERFACE:
    Every top-level expression has the form:
      FUNCTION (ARGS)
    where FUNCTION is either an atom naming a function or a LAMBDA
    expression, and ARGS is a list of already-evaluated arguments.
    Because each argument is already quoted, a list argument must be
    wrapped in an extra pair of parentheses.

  EXAMPLES (from Weissman's LISP 1.5 Primer):

    Input:    CAR ((A B C D))
    Output:   A

    Input:    CDR ((A B C D))
    Output:   (B C D)

    Input:    CONS (A (B C D))
    Output:   (A B C D)

    Input:    PLUS (1 2 3)
    Output:   6

    Input:    (LAMBDA (ONE TWO) (CONS TWO ONE)) (A B)
    Output:   (B . A)

  You can type any LISP expression at the tutorial prompt.  The function
  name is evaluated with the argument list as already-quoted values,
  exactly as in a LISP 1.5 file.

  Sections in this tutorial:
    1  Introduction         10  The PROG Feature
    2  Atoms and Lists      11  Higher-Order Functions
    3  CAR CDR CONS         12  FUNCTION and FUNARG
    4  Equality/Predicates  13  FEXPR and Special Forms
    5  Arithmetic           14  Property Lists
    6  COND                 15  I/O and Symbols
    7  LAMBDA and APPLY     16  Error Handling
    8  DEFINE               17  Destructive Ops/Macros
    9  List Operations      18  Primer Extensions
`,
	},
	{
		title: "ATOMS AND LISTS",
		content: `
  DATA TYPES IN LISP 1.5:

  ATOMS — the indivisible units:
    Symbolic atoms: A  FOO  HELLO  MY-VAR  *TRUE*
    Numeric atoms:  42  -7  0  1000
    Special: NIL (the empty list and false value)

  NIL is both the empty list () and the Boolean false.  It is the only
  false value — every other S-expression is true.

  *TRUE* is the conventional truth value returned by predicates.  The
  symbol T also evaluates to *TRUE*.

  LISTS — sequences of S-expressions:
    (A B C)          — a list of three atoms
    (1 2 3)          — a list of three numbers
    ((A B) C D)      — nested: first element is itself a list
    ()               — empty list, same as NIL

  DOTTED PAIRS — the fundamental cons cell:
    (A . B)          — a cons cell with CAR=A and CDR=B
    (A B C) is shorthand for (A . (B . (C . NIL)))

  EXAMPLES:
    Input:    CAR ((A B) C D)
    Output:   (A B)

    Input:    NULL (NIL)
    Output:   *TRUE*

    Input:    ATOM (FOO)
    Output:   *TRUE*

    Input:    ATOM ((A B))
    Output:   NIL
`,
	},
	{
		title: "ELEMENTARY FUNCTIONS — CAR CDR CONS",
		content: `
  THE FIVE ELEMENTARY FUNCTIONS:

  CAR[x]  — returns the first element (head) of a list
  CDR[x]  — returns the rest (tail) of a list
  CONS[x;y] — constructs a new cons cell: x becomes CAR, y becomes CDR
  ATOM[x] — *TRUE* if x is an atom (including NIL), NIL otherwise
  EQ[x;y] — *TRUE* if x and y are the same atom

  NULL[x] — *TRUE* if x is NIL (the empty list)

  EXAMPLES:
    Input:    CAR (A B C)
    Output:   A

    Input:    CDR (A B C)
    Output:   (B C)

    Input:    CONS (A (B C))
    Output:   (A B C)

    Input:    CONS (A NIL)
    Output:   (A)

    Input:    ATOM (FOO)
    Output:   *TRUE*

    Input:    NULL (NIL)
    Output:   *TRUE*

  CXR COMPOSITIONS — LISP 1.5 provides shortcuts for chained CAR/CDR:
    CADR   = CAR of CDR   (second element)
    CADDR  = CAR of CDR of CDR  (third element)
    CAAR   = CAR of CAR
    CDDR   = CDR of CDR

  EXAMPLES:
    Input:    CADR (A B C)
    Output:   B

    Input:    CADDR (A B C)
    Output:   C

    Input:    CAAR ((A B) C)
    Output:   A
`,
	},
	{
		title: "EQUALITY AND PREDICATES",
		content: `
  EQUALITY PREDICATES:

  EQUAL[x;y] — structural equality; works on lists and atoms
  EQ[x;y]    — atom identity only; do not use on lists

  EXAMPLES:
    Input:    EQUAL ((A B) (A B))
    Output:   *TRUE*

    Input:    EQ (FOO FOO)
    Output:   *TRUE*

    Input:    EQ ((A B) (A B))
    Output:   NIL         ; lists are never EQ

  TYPE PREDICATES:
  ATOM[x]    — *TRUE* if x is an atom (symbol or number)
  NULL[x]    — *TRUE* if x is NIL
  NOT[x]     — *TRUE* if x is NIL (identical to NULL)
  NUMBERP[x] — *TRUE* if x is a number
  FIXP[x]    — *TRUE* if x is a fixed-point (integer) number
  FLOATP[x]  — always NIL (no floating-point in this implementation)
  LISTP[x]   — *TRUE* if x is a non-NIL list (cons cell)

  LOGICAL CONNECTIVES:
  AND[p1;p2;...] — *TRUE* if all arguments are true (short-circuit)
  OR[p1;p2;...]  — *TRUE* if any argument is true (short-circuit)

  EXAMPLES:
    Input:    NUMBERP (42)
    Output:   *TRUE*

    Input:    NUMBERP (FOO)
    Output:   NIL

    Input:    AND (*TRUE* *TRUE*)
    Output:   *TRUE*

    Input:    OR (NIL *TRUE*)
    Output:   *TRUE*
`,
	},
	{
		title: "ARITHMETIC",
		content: `
  BASIC ARITHMETIC:
  PLUS[x;y]       — addition
  DIFFERENCE[x;y] — subtraction
  TIMES[x;y]      — multiplication
  QUOTIENT[x;y]   — integer division (truncates toward zero)
  REMAINDER[x;y]  — remainder after integer division
  MINUS[x]        — negation
  ADD1[x]         — x + 1
  SUB1[x]         — x - 1

  EXAMPLES:
    Input:    PLUS (3 5)
    Output:   8

    Input:    TIMES (6 7)
    Output:   42

    Input:    QUOTIENT (17 5)
    Output:   3

    Input:    REMAINDER (17 5)
    Output:   2

  COMPARISON PREDICATES:
  GREATERP[x;y] — *TRUE* if x > y
  LESSP[x;y]    — *TRUE* if x < y
  ZEROP[x]      — *TRUE* if x = 0
  MINUSP[x]     — *TRUE* if x < 0
  ONEP[x]       — *TRUE* if x = 1
  EVENP[x]      — *TRUE* if x is even

  OTHER ARITHMETIC:
  MAX[x;y]      — larger of x and y
  MIN[x;y]      — smaller of x and y
  ABS[x]        — absolute value
  EXPT[x;n]     — x raised to the power n
  DIVIDE[x;y]   — returns (quotient remainder) as a list

  EXAMPLES:
    Input:    GREATERP (10 3)
    Output:   *TRUE*

    Input:    ABS (-7)
    Output:   7

    Input:    DIVIDE (17 5)
    Output:   (3 2)

  BITWISE OPERATIONS (result printed in IBM 7094 octal Q-notation):
  LOGOR[x;y]      — bitwise OR
  LOGAND[x;y]     — bitwise AND
  LOGXOR[x;y]     — bitwise XOR
  LEFTSHIFT[x;n]  — shift x left by n bits (negative n shifts right)
`,
	},
	{
		title: "CONDITIONAL EXPRESSIONS — COND",
		content: `
  COND is the primary conditional form in LISP 1.5.

  SYNTAX:
    (COND (p1 e1) (p2 e2) ... (pn en))

  COND evaluates each predicate pi in order.  The first pi that is
  not NIL causes the corresponding ei to be returned.  If no predicate
  is true, an error is signaled.

  The idiom (T expr) is used as an else clause, since T always
  evaluates to *TRUE*.

  EXAMPLES:
    (COND ((ZEROP N) 0) (T N)) evaluates to N unless N is zero.

  In EVALQUOTE mode, COND is used inside function bodies:
    Input:    DEFINE (((SIGN (LAMBDA (N)
                (COND ((MINUSP N) (QUOTE NEGATIVE))
                      ((ZEROP N) (QUOTE ZERO))
                      (T (QUOTE POSITIVE)))))))
    Output:   *TRUE*

    Input:    SIGN (5)
    Output:   POSITIVE

    Input:    SIGN (-3)
    Output:   NEGATIVE

    Input:    SIGN (0)
    Output:   ZERO

  NESTING:
  COND clauses can contain any expressions, including nested CONDs.
  The expression (COND (T X)) always returns X.

  *TRUE* VS NIL:
  Only NIL is false.  Numbers, atoms, lists — all are true values.
  (COND (0 (QUOTE YES))) returns YES because 0 is not NIL.
`,
	},
	{
		title: "LAMBDA AND FUNCTION APPLICATION",
		content: `
  LAMBDA EXPRESSIONS define anonymous functions.

  SYNTAX:
    (LAMBDA (param1 param2 ...) body)

  EXAMPLES:
    ((LAMBDA (X) (TIMES X X)) (5))
    applies the squaring function to 5, returning 25.

  In EVALQUOTE mode, the arguments are NOT re-evaluated (they are
  already the values).  So:
    Input:    (LAMBDA (X) (TIMES X X)) (5)
    Output:   25

    Input:    (LAMBDA (X Y) (PLUS X Y)) (3 4)
    Output:   7

  LABEL provides recursion in an anonymous function:
    (LABEL name lambda)
  The name is bound to the lambda inside the lambda body.

  EXAMPLES:
    Input:    (LABEL FAC (LAMBDA (N)
                (COND ((ZEROP N) 1)
                      (T (TIMES N (FAC (SUB1 N))))))) (5)
    Output:   120

  APPLY[fn;args;alist]:
  Applies fn to the list args, with alist as the association list.
    Input:    APPLY (CAR ((A B C)) NIL)
    Output:   A

    Input:    APPLY ((LAMBDA (X) (TIMES X X)) (6) NIL)
    Output:   36

  EVAL[expr;alist]:
  Evaluates expr in the given alist.
    Input:    EVAL ((PLUS 2 3) NIL)
    Output:   5
`,
	},
	{
		title: "DEFINING FUNCTIONS — DEFINE",
		content: `
  DEFINE installs named functions in the global symbol table.

  SYNTAX:
    DEFINE (((name1 (LAMBDA (params) body))
             (name2 (LAMBDA (params) body)) ...))

  DEFINE returns *TRUE* and makes the functions available for all
  subsequent calls.

  EXAMPLES:
    Input:    DEFINE (((FACTORIAL (LAMBDA (N)
                (COND ((ZEROP N) 1)
                      (T (TIMES N (FACTORIAL (SUB1 N)))))))))
    Output:   *TRUE*

    Input:    FACTORIAL (6)
    Output:   720

    Input:    DEFINE (((FIB (LAMBDA (N)
                (COND ((LESSP N 2) N)
                      (T (PLUS (FIB (SUB1 N))
                               (FIB (SUB1 (SUB1 N))))))))))
    Output:   *TRUE*

    Input:    FIB (10)
    Output:   55

  MUTUAL RECURSION:
  Include both functions in the same DEFINE call so each is visible
  to the other:
    DEFINE (((EVEN? (LAMBDA (N)
                      (COND ((ZEROP N) *TRUE*)
                            (T (ODD? (SUB1 N))))))
             (ODD? (LAMBDA (N)
                     (COND ((ZEROP N) NIL)
                           (T (EVEN? (SUB1 N))))))))

  Functions defined with DEFINE persist for the rest of the session.
  You can redefine a function by calling DEFINE again with the same
  name.
`,
	},
	{
		title: "LIST OPERATIONS",
		content: `
  CONSTRUCTION AND COMBINATION:
  LIST[x1;x2;...] — build a list from arguments
  APPEND[x;y]     — concatenate two lists (non-destructive)
  REVERSE[x]      — reverse a list
  LENGTH[x]       — number of top-level elements

  EXAMPLES:
    Input:    LIST (A B C)
    Output:   (A B C)

    Input:    APPEND ((A B) (C D))
    Output:   (A B C D)

    Input:    REVERSE ((A B C))
    Output:   (C B A)

    Input:    LENGTH ((A B C))
    Output:   3

  SEARCHING AND MEMBERSHIP:
  MEMBER[x;lst]   — *TRUE* if x is in lst (uses EQUAL)
  LAST[lst]       — returns the last cons cell (a one-element list)

  SUBSTITUTION:
  SUBST[x;y;z]    — substitute x for every occurrence of y in z
  SUBLIS[a;z]     — substitute using an association list

  ASSOCIATION LISTS:
  ASSOC[x;alist]  — find pair with key x in alist
  SASSOC[x;alist;u] — like ASSOC but calls function u if not found
  PAIR[x;y]       — zip two lists into an association list
  PAIRLIS[x;y;a]  — like PAIR but prepends to existing alist a

  REMOVAL:
  EFFACE[x;lst]   — remove FIRST occurrence of x from lst
  DELETE[x;lst]   — remove ALL occurrences of x from lst

  SET OPERATIONS:
  INTERSECTION[x;y] — elements common to both lists
  UNION[x;y]        — combined elements (no duplicates)

  EXAMPLES:
    Input:    SUBST (A B (B C B))
    Output:   (A C A)

    Input:    ASSOC (B ((A 1) (B 2) (C 3)))
    Output:   (B 2)

    Input:    EFFACE (B (A B C B))
    Output:   (A C B)

    Input:    DELETE (B (A B C B))
    Output:   (A C)
`,
	},
	{
		title: "THE PROG FEATURE",
		content: `
  PROG provides imperative-style programming with local variables,
  mutable assignment, and explicit jumps.

  SYNTAX:
    (PROG (var1 var2 ...) stmt1 stmt2 ... stmtN)

  Variables are initialized to NIL.  Statements are evaluated in
  order.  Atom labels can appear as statement positions for GO.
  PROG returns NIL if execution falls off the end.

  SETQ[var;val]  — assign val to var (PROG-local variable)
  GO[label]      — unconditional jump to label
  RETURN[val]    — exit PROG with value val

  EXAMPLE — iterative factorial:
    DEFINE (((IFAC (LAMBDA (N)
      (PROG (I ACC)
        (SETQ I N)
        (SETQ ACC 1)
      LOOP
        (COND ((ZEROP I) (RETURN ACC)))
        (SETQ ACC (TIMES ACC I))
        (SETQ I (SUB1 I))
        (GO LOOP))))))

    Input:    IFAC (6)
    Output:   720

  EXAMPLE — iterative fibonacci:
    DEFINE (((IFIB (LAMBDA (N)
      (PROG (A B I)
        (SETQ A 0)
        (SETQ B 1)
        (SETQ I 0)
      LOOP
        (COND ((EQUAL I N) (RETURN A)))
        (PROG (TEMP)
          (SETQ TEMP (PLUS A B))
          (SETQ A B)
          (SETQ B TEMP))
        (SETQ I (ADD1 I))
        (GO LOOP)))))

  NOTES:
  - PROG can be nested; inner RETURN/GO affect the innermost PROG.
  - RETURN/GO inside COND inside PROG work correctly.
  - CSETQ sets a global variable (ignores PROG frames).
`,
	},
	{
		title: "HIGHER-ORDER FUNCTIONS",
		content: `
  LISP 1.5 supports functions as first-class values.

  MAPCAR[fn;lst] — apply fn to each element, collect results.
    Note: MAPCAR takes the FUNCTION first, then the LIST.
    In EVALQUOTE mode: MAPCAR (fn (e1 e2 ...))

  MAPLIST[fn;lst] — like MAPCAR but applies fn to each suffix
    (each successive CDR of the list).

  MAP[fn;lst] — like MAPLIST but for side effects; returns NIL.

  MAPC[fn;lst] — like MAPCAR but for side effects; returns NIL.

  EXAMPLES:
    Input:    DEFINE (((DOUBLE (LAMBDA (X) (PLUS X X)))))
    Output:   *TRUE*

    Input:    MAPCAR (DOUBLE (1 2 3 4))
    Output:   (2 4 6 8)

    Input:    MAPCAR (CAR ((A B) (C D) (E F)))
    Output:   (A C E)

  SEARCH[lst;pred;found;notfound]:
  Searches lst for element satisfying pred.
  If found, applies found to the element.
  If not found, calls notfound with no arguments.

  APPLY[fn;args;alist]:
  Applies fn to the list args in the given association list.
    Input:    APPLY (PLUS (3 4) NIL)
    Output:   7

  EVAL[expr;alist]:
  Evaluates expr in the given association list.
    Input:    EVAL ((PLUS 2 3) NIL)
    Output:   5

  Note on argument order: MAPCAR, MAPLIST, MAP, and MAPC in this
  interpreter take the function as the FIRST argument and the list
  as the SECOND argument (matching the IBM 7094 emulator convention).
`,
	},
	{
		title: "FUNCTION AND FUNARG",
		content: `
  THE FUNARG PROBLEM:

  When functions are passed as arguments, variable scoping can behave
  unexpectedly.  Consider:

    (DEFINE ((ADDN (LAMBDA (N) (LAMBDA (X) (PLUS X N))))))

  If you call MAPCAR with (ADDN 5), the inner lambda must "remember"
  that N=5, even when called from a different context.

  FUNCTION captures the current environment:
    (FUNCTION (LAMBDA (X) (PLUS X N)))
  creates a FUNARG triple: (FUNARG lambda saved-alist)

  EXAMPLES:
    DEFINE (((ADDER (LAMBDA (N)
               (FUNCTION (LAMBDA (X) (PLUS X N)))))))

    Input:    APPLY ((ADDER (3)) (10) NIL)
    Output:   13

  Without FUNCTION, the inner lambda would not capture N=3 and would
  fail with an unbound variable error when called later.

  FUNARG FORM:
  The FUNARG triple is produced by FUNCTION and consumed by APPLY:
    (FUNARG lambda alist)
  When applied, lambda is called with the saved alist merged in.

  WHEN TO USE FUNCTION:
  - Passing a lambda to MAPCAR, MAPLIST, etc.
  - Passing a lambda to APPLY
  - Any time the lambda refers to local variables from the enclosing
    function body

  EXAMPLE with MAPCAR:
    DEFINE (((ADD3 (LAMBDA (X) (PLUS X 3)))))
    Input:    MAPCAR (ADD3 (1 2 3))
    Output:   (4 5 6)

    ; Using anonymous lambda safely with FUNCTION:
    ; MAPCAR ((FUNCTION (LAMBDA (X) (PLUS X N))) ...)
    ; where N is bound in the outer scope
`,
	},
	{
		title: "FEXPR AND SPECIAL FORMS",
		content: `
  FEXPR defines functions that receive their arguments UNEVALUATED.
  This is in contrast to LAMBDA, which always evaluates its arguments.

  SYNTAX:
    (FEXPR (args alist) body)

  The first parameter receives the entire unevaluated argument list.
  The second parameter receives the current association list.

  COMPARISON:
    (LAMBDA (X Y) ...)  — X and Y receive evaluated argument values
    (FEXPR (U A) ...)   — U receives the raw unevaluated argument list
                          A receives the current environment (a-list)

  USE CASE:
  FEXPRs are used to define new control structures that need to
  selectively evaluate their arguments.

  EXAMPLE — define a conditional that returns the unevaluated form:
    DEFINE (((SHOW-COND (FEXPR (ARGS ENV)
               (CONS (QUOTE WOULD-EVAL)
                     (CONS (CAR ARGS) NIL))))))

  BUILT-IN SPECIAL FORMS (implemented as special cases in eval):
  QUOTE[x]       — returns x unevaluated
  COND[clauses]  — conditional evaluation
  PROG[...]      — imperative block
  LAMBDA[...]    — evaluates to itself (a function object)
  DEFINE[...]    — installs global definitions
  SETQ[v;e]      — assignment
  GO[label]      — jump within PROG
  RETURN[val]    — exit PROG
  AND[...]       — short-circuit and
  OR[...]        — short-circuit or
  FUNCTION[f]    — creates FUNARG closure
  ERROR[...]     — signal an error
  COMMENT[...]   — documentation (evaluates to NIL)
`,
	},
	{
		title: "PROPERTY LISTS",
		content: `
  In LISP 1.5, every atom has an associated property list — a sequence
  of indicator-value pairs that store metadata about the atom.

  DEFLIST[pairs;indicator]:
  Set a property for multiple atoms at once.
    (DEFLIST ((ATOM1 VAL1) (ATOM2 VAL2) ...) INDICATOR)

  GET[atom;indicator]:
  Retrieve the value associated with an indicator.
  Returns NIL if the property is not set.

  PUTPROP[atom;value;indicator]:
  Set one property on one atom.  (Extension; DEFLIST is standard.)

  REMPROP[atom;indicator]:
  Remove a property from an atom's property list.

  PROP[atom;indicator;notfound]:
  Like GET but calls the notfound function if property is missing.

  ATTRIB[atom;pair]:
  Add a property via a dotted pair (indicator . value).

  SPECIAL INDICATOR — PNAME:
  GET with indicator PNAME returns the atom's print name as a list
  of single-character atoms.
    Input:    GET (FOO PNAME)
    Output:   (F O O)

  EXAMPLES:
    Input:    DEFLIST (((CAT ANIMAL) (DOG ANIMAL) (OAK PLANT)) TYPE)
    Output:   *TRUE*

    Input:    GET (CAT TYPE)
    Output:   ANIMAL

    Input:    GET (DOG TYPE)
    Output:   ANIMAL

    Input:    PUTPROP (CAT 4 LEGS)
    Output:   4

    Input:    GET (CAT LEGS)
    Output:   4

    Input:    REMPROP (CAT LEGS)
    Output:   4

    Input:    GET (CAT LEGS)
    Output:   NIL
`,
	},
	{
		title: "INPUT/OUTPUT AND SYMBOLS",
		content: `
  OUTPUT:
  PRINT[x]  — print x, return x
  TERPRI[]  — print a newline, return NIL

  SYMBOL GENERATION:
  GENSYM[]  — returns a unique symbol: G00001, G00002, ...
  Each call to GENSYM returns a fresh symbol not used elsewhere.

  ATOM <-> CHARACTER LIST CONVERSION:
  EXPLODE[x]        — convert atom to list of single-char atoms
  INTERN[chars]     — convert list of single-char atoms to an atom
  IMPLODE[chars]    — same as INTERN
  COMPRESS[chars]   — same as INTERN

  EXAMPLES:
    Input:    EXPLODE (HELLO)
    Output:   (H E L L O)

    Input:    INTERN ((H E L L O))
    Output:   HELLO

    Input:    EXPLODE (42)
    Output:   (4 2)

    Input:    GENSYM (NIL)
    Output:   G00001

    Input:    GENSYM (NIL)
    Output:   G00002

  OBLIST:
  OBLIST[]  — returns a list of all atoms that have global definitions

  REMOB[atom]:
  Remove an atom from the object list (delete its global definition).

  EXAMPLES:
    Input:    DEFINE (((FOO (LAMBDA (X) X))))
    Output:   *TRUE*

    Input:    REMOB (FOO)
    Output:   FOO

    ; FOO is now undefined
`,
	},
	{
		title: "ERROR HANDLING",
		content: `
  ERROR SIGNALING:
  (ERROR arg1 arg2 ...) — signals an error with the given arguments.
  The arguments are NOT evaluated (it behaves like an FEXPR).
  Execution terminates unless caught by ERRORSET.

  ERRORSET[expr;flag]:
  Evaluates expr in a protected context.
  - If evaluation succeeds: returns (result) — a one-element list
  - If an error occurs: returns NIL

  The flag argument is ignored in this implementation (in the IBM 7094
  emulator it controls whether the error is printed).

  EXAMPLES:
    Input:    ERRORSET ((PLUS 2 3) NIL)
    Output:   (5)

    Input:    ERRORSET ((CAR NIL) NIL)
    Output:   NIL         ; CAR of NIL returns NIL without error

    Input:    ERRORSET ((ERROR OOPS) NIL)
    Output:   NIL         ; error was caught

  Note: to distinguish "result was NIL" from "error occurred",
  check whether ERRORSET returned a list (success) or NIL (error):

    DEFINE (((TRY (LAMBDA (EXPR)
      (PROG (R)
        (SETQ R (ERRORSET EXPR NIL))
        (COND ((NULL R) (PRINT (QUOTE ERROR-CAUGHT)))
              (T (PRINT (CAR R))))))))

  RECLAIM[flag]:
  Triggers garbage collection.  This is a no-op in our Go
  implementation since Go has its own garbage collector.

  TEMPUS-FUGIT:
  Returns NIL.  In the IBM 7094 emulator this printed timing
  information.  Included for compatibility.
`,
	},
	{
		title: "DESTRUCTIVE OPERATIONS AND MACROS",
		content: `
  DESTRUCTIVE OPERATIONS modify existing cons cells in place.
  WARNING: these can corrupt shared structure if used carelessly.

  RPLACA[cell;val] — replace the CAR of cell with val; returns cell
  RPLACD[cell;val] — replace the CDR of cell with val; returns cell
  NCONC[x;y]       — destructively append y to x by modifying the
                     last CDR of x; returns x (or y if x is NIL)

  EXAMPLES:
    Input:    DEFINE (((X (QUOTE (A B C)))))
    Output:   *TRUE*

    Input:    RPLACA (X D)
    Output:   (D B C)        ; X is now (D B C)

  Compare APPEND (non-destructive) with NCONC (destructive):
    APPEND makes a copy of x and attaches y.
    NCONC modifies x directly — cheaper but dangerous with sharing.

  MACROS:
  (MACRO (((name (LAMBDA (form) expansion)))))
  Defines a macro expander.  When the macro name appears as the head
  of a form, the entire form is passed to the macro function, and the
  result is re-evaluated.

  EXAMPLE — a simple SWAP macro:
    DEFINE (((SWAP! (LAMBDA (FORM)
      (PROG (A B)
        (SETQ A (CADR FORM))
        (SETQ B (CADDR FORM))
        (LIST (QUOTE PROG) (LIST (QUOTE TMP))
              (LIST (QUOTE SETQ) (QUOTE TMP) A)
              (LIST (QUOTE SETQ) A B)
              (LIST (QUOTE SETQ) B (QUOTE TMP))))))))

    MACRO (((SWAP! (LAMBDA (FORM) ...))))

  Macros receive the unevaluated form and return an S-expression
  that is then evaluated in the original context.
`,
	},
	{
		title: "PRIMER EXTENSIONS",
		content: `
  This section covers extensions present in our interpreter that are
  described in the LISP 1.5 Primer but NOT in the IBM 7094 emulator.
  These make the language more convenient to use.

  ADDITIONAL PREDICATES:
  EVENP[n]   — *TRUE* if n is even
  ONEP[n]    — *TRUE* if n equals 1
  LISTP[x]   — *TRUE* if x is a non-NIL list (cons cell)
  MINUSP[n]  — *TRUE* if n < 0  (also in standard LISP 1.5)

  EXAMPLES:
    Input:    EVENP (4)
    Output:   *TRUE*

    Input:    ONEP (1)
    Output:   *TRUE*

    Input:    LISTP ((A B))
    Output:   *TRUE*

    Input:    LISTP (FOO)
    Output:   NIL

  ARITHMETIC EXTENSIONS:
  ABSVAL[x]  — absolute value (alias for ABS)
  ENTIER[x]  — floor function; identity for integers
  DIVIDE[x;y] — returns (quotient remainder)  (already in lesson 5)

  SET OPERATIONS:
  INTERSECTION[x;y] — elements in both lists
  UNION[x;y]        — elements in either list (no duplicates)

  EXAMPLES:
    Input:    INTERSECTION ((A B C) (B C D))
    Output:   (B C)

    Input:    UNION ((A B) (B C D))
    Output:   (A B C D)

  SELECT — a case/switch form:
    (SELECT val (v1 e1) (v2 e2) ... default)
  Evaluates val, finds matching vi, returns ei; else evaluates default.

  PROPERTY LIST EXTENSION:
  PUTPROP[atom;value;indicator] — set one property directly.
  (The standard LISP 1.5 uses DEFLIST for this purpose.)

  MACRO SYSTEM:
  See lesson 17.  MACRO is defined as above.

  NOTE ON COMPATIBILITY:
  When writing programs to run on the IBM 7094 emulator, avoid
  EVENP, ONEP, LISTP, ABSVAL, ENTIER, PUTPROP, INTERSECTION, UNION,
  SELECT, and MACRO.  Use DEFLIST instead of PUTPROP.
`,
	},
}

// ── Display helpers ───────────────────────────────────────────────────────────

const tutWidth = 80

func tutHeader(n, total int, title string) {
	bar := strings.Repeat("=", tutWidth)
	fmt.Println(bar)
	fmt.Printf(" LISP 1.5 INTERACTIVE TUTORIAL  [Section %d/%d]\n", n, total)
	fmt.Printf(" %s\n", title)
	fmt.Println(bar)
}

func tutFooter(n, total int) {
	bar := strings.Repeat("=", tutWidth)
	fmt.Println(bar)
	fmt.Printf("  Type a LISP expression to try it, or:\n")
	fmt.Printf("    NEXT (or Enter) -- next section\n")
	fmt.Printf("    PREV           -- previous section\n")
	fmt.Printf("    INDEX          -- table of contents\n")
	fmt.Printf("    N              -- jump to section N (1-%d)\n", total)
	fmt.Printf("    QUIT           -- exit tutorial\n")
	fmt.Println(bar)
	fmt.Printf("TUTORIAL[%d]> ", n)
}

func tutIndex(total int) {
	bar := strings.Repeat("=", tutWidth)
	fmt.Println(bar)
	fmt.Println(" LISP 1.5 INTERACTIVE TUTORIAL -- TABLE OF CONTENTS")
	fmt.Println(bar)
	fmt.Println()
	for i, l := range lessons {
		fmt.Printf("  %2d.  %s\n", i+1, l.title)
	}
	fmt.Println()
	fmt.Printf("  Enter a number (1-%d) to jump to a section.\n", total)
	fmt.Println()
	fmt.Println(bar)
	fmt.Printf("TUTORIAL[INDEX]> ")
}

func showLesson(n int) {
	total := len(lessons)
	l := lessons[n-1]
	tutHeader(n, total, l.title)
	fmt.Print(l.content)
	tutFooter(n, total)
}

// ── RunTutorial ───────────────────────────────────────────────────────────────

// RunTutorial runs the interactive LISP 1.5 tutorial.
// It reads user input from stdin, evaluates LISP expressions using the
// current interpreter state, and navigates between lessons.
// The scanner parameter is accepted for API compatibility but stdin is
// used for interactive input.
func RunTutorial(_ *bufio.Scanner) {
	total := len(lessons)
	cur := 1
	in := bufio.NewScanner(os.Stdin)

	showLesson(cur)

	for {
		if !in.Scan() {
			break
		}
		line := strings.TrimSpace(in.Text())

		// Navigation commands.
		switch strings.ToUpper(line) {
		case "", "NEXT":
			cur++
			if cur > total {
				fmt.Println()
				fmt.Println(" You have completed the LISP 1.5 Tutorial!")
				fmt.Println(" Returning to the interpreter.")
				fmt.Println()
				return
			}
			showLesson(cur)
			continue

		case "PREV":
			cur--
			if cur < 1 {
				cur = 1
			}
			showLesson(cur)
			continue

		case "QUIT":
			fmt.Println()
			fmt.Println(" Exiting tutorial. Returning to the interpreter.")
			fmt.Println()
			return

		case "INDEX":
			tutIndex(total)
			continue
		}

		// Check for numeric jump.
		n := 0
		allDigits := len(line) > 0
		for _, ch := range line {
			if ch < '0' || ch > '9' {
				allDigits = false
				break
			}
			n = n*10 + int(ch-'0')
		}
		if allDigits && n >= 1 && n <= total {
			cur = n
			showLesson(cur)
			continue
		}

		// Treat as a LISP expression: parse and evaluate.
		if line == "" {
			fmt.Printf("TUTORIAL[%d]> ", cur)
			continue
		}

		tutEval(line, cur)
	}
}

// tutEval parses and evaluates a LISP expression typed in tutorial mode.
// In EVALQUOTE mode: the first token is the function, the second is the
// argument list.  We parse both from the line.
func tutEval(line string, cur int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("  ERROR: %v\n", r)
			fmt.Printf("TUTORIAL[%d]> ", cur)
		}
	}()

	p := newParser(line)

	if p.done() {
		fmt.Printf("TUTORIAL[%d]> ", cur)
		return
	}

	// Parse function (first S-expression).
	fn := p.parseExpr()
	if fn == nil {
		fmt.Printf("TUTORIAL[%d]> ", cur)
		return
	}

	// Parse argument list (second S-expression); default to NIL.
	var args *Expr
	if !p.done() {
		args = p.parseExpr()
	}

	result, errMsg := safeEval(fn, args)
	if errMsg != "" {
		fmt.Printf("  ERROR: %s\n", errMsg)
	} else {
		fmt.Printf("  => %s\n", exprStr(result))
	}
	fmt.Printf("TUTORIAL[%d]> ", cur)
}
