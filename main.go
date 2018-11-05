package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/umbracle/heura/heura/repl"

	"github.com/umbracle/heura/heura/evaluator"
	"github.com/umbracle/heura/heura/lexer"
	"github.com/umbracle/heura/heura/manager"
	"github.com/umbracle/heura/heura/object"
	"github.com/umbracle/heura/heura/parser"
)

const (
	DefaultRPCEndpoint = "https://mainnet.infura.io"
	DefaultWSEndpoint  = "wss://mainnet.infura.io/ws"
)

func main() {
	endpoint := flag.String("endpoint", DefaultRPCEndpoint, "RPC ethereum endpoint")
	wsEndpoint := flag.String("wsendpoint", DefaultWSEndpoint, "Websocket endpoint")

	flag.Parse()

	args := flag.Args()

	env := object.NewEnvironment()
	env.BuildEnvs(os.Environ())
	env.BuildArgs(args)
	env.Set("endpoint", &object.String{Value: *endpoint})

	if len(args) < 1 {
		// run repl
		repl.Start(os.Stdin, os.Stdout, env)
	} else {
		// run the file
		runFile(args[0], env, *wsEndpoint)
	}
}

func runFile(file string, env *object.Environment, wsEndpoint string) {
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
