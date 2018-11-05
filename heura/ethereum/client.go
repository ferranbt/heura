package ethereum

import (
	"github.com/ethereum/go-ethereum/ethclient"
	ethrpc "github.com/ethereum/go-ethereum/rpc"
)

func NewClient(endpoint string) *ethclient.Client {
	rpc, err := ethrpc.Dial(endpoint)
	if err != nil {
		panic(err)
	}

	return ethclient.NewClient(rpc)
}
