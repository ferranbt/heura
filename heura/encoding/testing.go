package encoding

import (
	"context"
	"os"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	ethrpc "github.com/ethereum/go-ethereum/rpc"
)

const (
	defaultGasPrice = "0x70000000"
	defaultGasLimit = "0x500000"
)

type ethClient struct {
	*ethclient.Client
	rpc *ethrpc.Client

	gasPrice string
	gasLimit string
}

func newClientWithEndpoint(endpoint string) *ethClient {
	rpc, err := ethrpc.Dial(endpoint)
	if err != nil {
		panic(err)
	}

	return &ethClient{ethclient.NewClient(rpc), rpc, defaultGasPrice, defaultGasLimit}
}

func (c *ethClient) sendTx(tx *transaction) (*transactionResult, error) {
	tx.GasPrice = defaultGasPrice
	tx.Gas = defaultGasLimit

	var hash common.Hash
	if err := c.rpc.Call(&hash, "eth_sendTransaction", tx); err != nil {
		return nil, err
	}

	result := &transactionResult{
		Hash:   hash,
		client: c,
	}

	return result, nil
}

func (c *ethClient) SendTxAndWait(tx *transaction) (*types.Receipt, error) {
	result, err := c.sendTx(tx)
	if err != nil {
		return nil, err
	}
	return result.Wait()
}

type transaction struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Data     string          `json:"data"`
	Gas      string          `json:"gas"`
	GasPrice string          `json:"gasPrice"`
}

type transactionResult struct {
	Hash   common.Hash
	client *ethClient
}

func (t *transactionResult) Wait() (*types.Receipt, error) {
	for {
		receipt, err := t.client.TransactionReceipt(context.Background(), t.Hash)
		if err != nil {
			if err != ethereum.NotFound {
				return nil, err
			}
		}
		if receipt != nil {
			return receipt, nil
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func random() bool {
	return os.Getenv("RANDOM_TESTS") == "TRUE"
}
