package ens

import (
	"fmt"

	"github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/umbracle/heura/heura/ethereum"
	"github.com/umbracle/heura/heura/object"
)

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

func resolve(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError("expected one param but found %d", len(args))
	}
	if args[0].Type() != object.STRING_OBJ {
		return newError("expected argument to be string, got %s", args[0].Type())
	}
	// TODO, fetch the endpoint from somewhere else
	ens, err := ethereum.NewENS("https://mainnet.infura.io", ens.MainNetAddress)
	if err != nil {
		panic(err)
	}
	addr, err := ens.Resolve(args[0].(*object.String).Value)
	if err != nil {
		panic(err)
	}
	return &object.Address{Value: addr.String()}
}

// Factory is the factory method for the ENS backend
func Factory(env *object.Environment) object.Object {
	h := &object.Hash{}
	h.SetString("Resolve", &object.Builtin{
		Fn: resolve,
	})
	return h
}
