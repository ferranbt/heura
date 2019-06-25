package main

import (
	"fmt"
	"os"

	"github.com/umbracle/heura/commands"
)

func main() {
	if err := commands.Command().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
