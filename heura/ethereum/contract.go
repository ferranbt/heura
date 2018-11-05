package ethereum

import (
	"context"
	"fmt"
	"math/big"

	"github.com/umbracle/heura/heura/encoding"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/umbracle/heura/heura/object"
)

// EthClient is the client to send calls
type EthClient interface {
	CallContract(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error)
}

// Contract is a contract instance
type Contract struct {
	Abi     *abi.ABI
	client  EthClient
	Address common.Address
}

// NewContract creates a new contract
func NewContract(abi *abi.ABI, client EthClient, address common.Address) *Contract {
	return &Contract{abi, client, address}
}

func (c *Contract) pack(method string, args []object.Object) ([]byte, error) {
	m, ok := c.Abi.Methods[method]
	if !ok {
		return nil, fmt.Errorf("method %s not found", method)
	}

	data, err := encoding.Pack(m.Inputs, args)
	if err != nil {
		return nil, err
	}

	return append(m.Id(), data...), nil
}

func (c *Contract) unpack(method string, data []byte) ([]object.Object, error) {
	m, ok := c.Abi.Methods[method]
	if !ok {
		return nil, fmt.Errorf("method %s not found", method)
	}

	return encoding.Unpack(m.Outputs, data)
}

// Call a contract method
func (c *Contract) Call(ctx context.Context, method string, args []object.Object) ([]object.Object, error) {
	data, err := c.pack(method, args)
	if err != nil {
		return nil, fmt.Errorf("failed to pack: %v", err)
	}

	callMsg := ethereum.CallMsg{
		To:   &c.Address,
		Data: data,
	}

	output, err := c.client.CallContract(ctx, callMsg, nil)
	if err != nil {
		return nil, fmt.Errorf("call not worked: %v", err)
	}

	result, err := c.unpack(method, output)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack: %v", err)
	}

	return result, nil
}
