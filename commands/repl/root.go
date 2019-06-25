package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/umbracle/heura/heura/evaluator"
	"github.com/umbracle/heura/heura/lexer"
	"github.com/umbracle/heura/heura/object"
	"github.com/umbracle/heura/heura/parser"
)

// RootCmd returns the repl command
var RootCmd = &cobra.Command{
	Use:   "repl",
	Short: "Start a REPL session",
	Run:   rootRun,
}

func rootRun(cmd *cobra.Command, args []string) {
	env := object.NewEnvironment()
	env.BuildEnvs(os.Environ())
	env.BuildArgs(args)
	env.Set("endpoint", &object.String{Value: "https://mainnet.infura.io"})

	start(os.Stdin, os.Stdout, env)
}

const prompt = ">> "

func start(in io.Reader, out io.Writer, env *object.Environment) {
	scanner := bufio.NewScanner(in)

	for {
		fmt.Printf(prompt)
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
