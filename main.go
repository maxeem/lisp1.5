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
	// Warn if any line exceeds 72 columns — the IBM 7094 reader silently
	// truncates longer lines, which causes parse errors in the emulator.
	var sb strings.Builder
	lineNum := 1
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if len(line) > 72 {
			fmt.Fprintf(os.Stderr, "warning: line %d is %d chars (>72); IBM 7094 emulator will truncate\n",
				lineNum, len(line))
		}
		sb.WriteString(line)
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

		t := p.peek()
		if t.typ == tokEOF {
			break
		}

		// Skip stray closing parens (e.g. the `)))` after STOP).
		if t.typ == tokRpar {
			p.next()
			continue
		}

		// S1 can be either an atom (function name) or a list (lambda expression).
		var fn *Expr
		var fnStr string
		if t.typ == tokAtom {
			p.next()
			fnStr = t.text
			if fnStr == "STOP" || fnStr == "FIN" {
				fmt.Println()
				fmt.Println(" END OF EVALQUOTE OPERATOR")
				fmt.Printf("             FIN      END OF LISP RUN\n")
				return
			}
			// Tokens like END, OF, LISP, RUN appear after STOP — ignore them.
			switch fnStr {
			case "END", "OF", "LISP", "RUN":
				continue
			}
			if fnStr == "TUTORIAL" {
				RunTutorial(scanner)
				continue
			}
			fn = mkSym(fnStr)
		} else {
			// S1 is a list expression (e.g. a LAMBDA).
			fn = p.parseExpr()
			fnStr = exprStr(fn)
		}

		// Read S2: the argument list.
		args := p.parseExpr()

		fmt.Printf("  FUNCTION   EVALQUOTE   HAS BEEN ENTERED, ARGUMENTS..\n")
		printWrapped(fnStr, 72)
		fmt.Println()
		printWrapped(exprStr(args), 72)
		fmt.Println()

		result, evalErr := safeEval(fn, args)

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
