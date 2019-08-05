package repl

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/umbracle/heura/heura/evaluator"
	"github.com/umbracle/heura/heura/lexer"
	"github.com/umbracle/heura/heura/object"
	"github.com/umbracle/heura/heura/parser"

	prompt "github.com/c-bata/go-prompt"
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

	p := prompt.New(
		executor(env),
		completer,
		prompt.OptionPrefix(">> "),
		prompt.OptionAddKeyBind(quit),
	)
	p.Run()
}

func completer(in prompt.Document) []prompt.Suggest {
	return nil
}

func executor(env *object.Environment) func(s string) {
	return func(s string) {
		p := parser.New(lexer.New(s))

		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			fmt.Printf("  parser errors:\n")
			for _, msg := range p.Errors() {
				fmt.Printf("\t" + msg + "\n")
			}
		} else {
			evaluated := evaluator.Eval(program, env)
			if evaluated != nil {
				fmt.Println(evaluated.Inspect())
			}
		}
	}
}

var quit = prompt.KeyBind{
	Key: prompt.ControlC,
	Fn: func(b *prompt.Buffer) {
		os.Exit(0)
	},
}
