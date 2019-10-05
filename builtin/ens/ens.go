package ens

import (
	"fmt"

	"github.com/umbracle/go-web3"
	"github.com/umbracle/go-web3/contract/builtin/ens"
	"github.com/umbracle/go-web3/jsonrpc"
	"github.com/umbracle/heura/heura/object"
)

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

var (
	mainnetAddress = web3.HexToAddress("0x314159265dD8dbb310642f98f50C066173C1259b")
)

func Resolve(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError("expected one param but found %d", len(args))
	}
	if args[0].Type() != object.STRING_OBJ {
		return newError("expected argument to be string, got %s", args[0].Type())
	}
	client, _ := jsonrpc.NewClient("https://mainnet.infura.io")
	resolver := ens.NewENSResolver(mainnetAddress, client)

	addr, err := resolver.Resolve(args[0].(*object.String).Value)
	if err != nil {
		return newError(err.Error())
	}
	return &object.Address{Value: addr.String()}
}

// Factory is the factory method for the ENS backend
func Factory() object.Object {
	h := &object.Hash{}
	h.SetString("Resolve", &object.Builtin{
		Fn: Resolve,
	})
	return h
}
