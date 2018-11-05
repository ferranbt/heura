package repl

import (
	"bufio"
	"fmt"
	"io"

	"github.com/umbracle/heura/heura/evaluator"
	"github.com/umbracle/heura/heura/lexer"
	"github.com/umbracle/heura/heura/object"
	"github.com/umbracle/heura/heura/parser"
)

const PROMPT = ">> "

func Start(in io.Reader, out io.Writer, env *object.Environment) {
	scanner := bufio.NewScanner(in)

	for {
		fmt.Printf(PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		l := lexer.New(line)
		p := parser.New(l)

		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			continue
		}

		evaluated := evaluator.Eval(program, env)

		if evaluated != nil {
			io.WriteString(out, evaluated.Inspect())
			io.WriteString(out, "\n")
		}
	}
}

func printParserErrors(out io.Writer, errors []string) {
	io.WriteString(out, "  parser errors:\n")

	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}
