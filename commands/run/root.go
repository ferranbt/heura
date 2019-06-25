package run

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/umbracle/heura/heura/evaluator"
	"github.com/umbracle/heura/heura/lexer"
	"github.com/umbracle/heura/heura/manager"
	"github.com/umbracle/heura/heura/object"
	"github.com/umbracle/heura/heura/parser"
)

func init() {
	RootCmd.Flags().BoolP("dry", "d", false, "build the script with no execution")
	RootCmd.Flags().StringP("websocket", "w", "wss://mainnet.infura.io/ws", "websocket endpoint to connect")
	RootCmd.Flags().StringP("rpc", "r", "https://mainnet.infura.io", "rpc endpoint to connect")
}

// RootCmd returns the run command
var RootCmd = &cobra.Command{
	Use:   "run",
	Short: "build and run",
	Run:   rootRun,
}

func rootRun(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		fmt.Printf("Only one file expected")
		os.Exit(1)
	}

	file := args[0]

	endpoint, _ := cmd.Flags().GetString("rpc")
	wsEndpoint, _ := cmd.Flags().GetString("websocket")

	env := object.NewEnvironment()
	env.BuildEnvs(os.Environ())
	env.BuildArgs(args)
	env.Set("endpoint", &object.String{Value: endpoint})

	data, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf(err.Error())
		os.Exit(0)
	}

	l := lexer.New(string(data))
	p := parser.New(l)

	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		fmt.Println(p.Errors())
		os.Exit(0)
	}

	if ok, _ := cmd.Flags().GetBool("dry"); ok {
		return
	}

	evaluated := evaluator.Eval(program, env)
	if evaluated != nil {
		fmt.Println(evaluated)
	}

	events := env.GetOnStatements()
	if len(events) == 0 {
		return
	}

	eventManager := manager.NewEventManager(wsEndpoint, env)
	for _, event := range events {
		eventManager.Listen(event)
	}

	handleSignals(eventManager)

}

func handleSignals(s *manager.EventManager) {
	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	select {
	case <-signalCh:
	}

	s.Shutdown()
}
