package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: lisp1.5 <file>")
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// First line is the title (free-form text).
	title := ""
	if scanner.Scan() {
		title = strings.TrimSpace(scanner.Text())
	}

	// Collect the rest of the file for the tokenizer.
	var sb strings.Builder
	for scanner.Scan() {
		sb.WriteString(scanner.Text())
		sb.WriteByte('\n')
	}

	p := newParser(sb.String())

	fmt.Printf("             %s\n\n\n\n", title)
	fmt.Println(" EVALQUOTE OPERATOR AS OF 1 MARCH 1961.    INPUT LISTS NOW BEING READ.")
	fmt.Println()
	fmt.Println()

	for {
		p.skipSpace()
		if p.done() {
			break
		}

		t := p.next()
		if t.typ == tokEOF {
			break
		}

		// Skip stray closing parens (e.g. the `)))` after STOP).
		if t.typ == tokRpar {
			continue
		}

		if t.typ != tokAtom {
			continue
		}

		fnName := t.text

		if fnName == "STOP" || fnName == "FIN" {
			fmt.Println()
			fmt.Println(" END OF EVALQUOTE OPERATOR")
			fmt.Printf("             FIN      END OF LISP RUN\n")
			return
		}

		// Tokens like END, OF, LISP, RUN appear after STOP — ignore them.
		switch fnName {
		case "END", "OF", "LISP", "RUN":
			continue
		}

		// Read the argument list (an S-expression).
		args := p.parseExpr()

		fmt.Printf("  FUNCTION   EVALQUOTE   HAS BEEN ENTERED, ARGUMENTS..\n")
		fmt.Printf(" %s\n\n", fnName)
		printWrapped(exprStr(args), 72)
		fmt.Println()

		result, evalErr := safeEval(mkSym(fnName), args)

		fmt.Println(" END OF EVALQUOTE, VALUE IS ..")
		if evalErr != "" {
			fmt.Printf(" ERROR: %s\n", evalErr)
		} else {
			printWrapped(exprStr(result), 72)
		}
		fmt.Println()
		fmt.Println()
	}
}

// safeEval calls evalquote, catching panics and returning them as error strings.
func safeEval(fn, args *Expr) (result *Expr, errMsg string) {
	defer func() {
		if r := recover(); r != nil {
			errMsg = fmt.Sprintf("%v", r)
		}
	}()
	result = evalquote(fn, args)
	return
}

// printWrapped prints s with a leading space, wrapping at maxCol characters.
// Lines are wrapped at the last space before maxCol (matching the IBM 7094 LPT).
func printWrapped(s string, maxCol int) {
	prefix := " "
	avail := maxCol - len(prefix)
	for len(s) > avail {
		// Find last space at or before avail.
		cut := avail
		for cut > 0 && s[cut] != ' ' {
			cut--
		}
		if cut == 0 {
			cut = avail // no space found, hard-break
		}
		fmt.Printf("%s%s\n", prefix, s[:cut])
		s = s[cut:]
		if len(s) > 0 && s[0] == ' ' {
			s = s[1:]
		}
	}
	fmt.Printf("%s%s\n", prefix, s)
}
