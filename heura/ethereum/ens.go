package ethereum

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// Ens is an Ethereum Name Service resolver
type Ens struct {
	client *ethclient.Client // dummy to create it here every time
	ens    *ens.ENS
}

func NewENS(endpoint string, address common.Address) (*Ens, error) {
	c, err := rpc.Dial(endpoint)
	if err != nil {
		return nil, err
	}

	client := ethclient.NewClient(c)

	i, err := ens.NewENS(&bind.TransactOpts{}, address, client)
	if err != nil {
		return nil, err
	}

	e := &Ens{
		client: client,
		ens:    i,
	}

	return e, nil
}

func (e *Ens) Resolve(name string) (common.Address, error) {
	return e.ens.Addr(name)
}
