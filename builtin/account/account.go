package account

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/umbracle/heura/heura/ethereum"
	"github.com/umbracle/heura/heura/object"
)

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

// Factory is the factory method for the Etherscan backend
func Factory(env *object.Environment) object.Object {
	rpcEndpoint, err := env.GetRPCEndpoint()
	if err != nil {
		return newError(err.Error())
	}

	client := ethereum.NewClient(rpcEndpoint)

	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("expected one parameter but found %d", len(args))
			}

			addr, err := getAddr(args[0])
			if err != nil {
				return newError(err.Error())
			}

			h := &object.Hash{}
			h.SetString("nonce", &object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 0 {
						return newError("expected zero params but found %d", len(args))
					}
					nonce, err := client.NonceAt(context.Background(), addr, nil)
					if err != nil {
						return newError(err.Error())
					}
					return &object.Integer{Value: new(big.Int).SetUint64(nonce)}
				},
			})
			h.SetString("balance", &object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 0 {
						return newError("expected zero params but found %d", len(args))
					}
					balance, err := client.BalanceAt(context.Background(), addr, nil)
					if err != nil {
						return newError(err.Error())
					}
					return &object.Integer{Value: balance}
				},
			})
			return h
		},
	}
}

func getAddr(obj object.Object) (common.Address, error) {
	var addr common.Address

	switch obj.Type() {
	case object.BYTES_OBJ:
		aux, err := obj.(*object.Bytes).ToAddress()
		if err != nil {
			return addr, err
		}
		addr = aux.ToAddress()

	case object.ADDRESS_OBJ:
		addr = obj.(*object.Address).ToAddress()

	default:
		return addr, fmt.Errorf("expected address type but found %s", obj.Type())
	}
	return addr, nil
}
