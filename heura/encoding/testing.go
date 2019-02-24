package encoding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/template"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
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

func newClient() *ethClient {
	return newClientWithEndpoint("http://localhost:8545")
}

func newClientWithEndpoint(endpoint string) *ethClient {
	rpc, err := ethrpc.Dial(endpoint)
	if err != nil {
		panic(err)
	}

	return &ethClient{ethclient.NewClient(rpc), rpc, defaultGasPrice, defaultGasLimit}
}

func (c *ethClient) listAccounts() ([]common.Address, error) {
	var accounts []common.Address
	err := c.rpc.Call(&accounts, "eth_accounts")
	return accounts, err
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

func compileAndDeployTemplate(templateStr string, params interface{}, deployer common.Address, client *ethClient) (*abi.ABI, *types.Receipt, error) {
	tmpl, err := template.New("").Parse(templateStr)
	if err != nil {
		return nil, nil, err
	}

	buf := new(bytes.Buffer)
	if err = tmpl.Execute(buf, params); err != nil {
		return nil, nil, err
	}

	source := buf.String()
	source = strings.Replace(source, ", )", ")", -1) // remove trailing commas

	data, err := compiler.CompileSolidityString("solc", source)
	if err != nil {
		return nil, nil, err
	}

	if len(data) != 1 {
		return nil, nil, fmt.Errorf("Expected one contract but found %d", len(data))
	}

	contract, ok := data["<stdin>:Sample"]
	if !ok {
		return nil, nil, fmt.Errorf("Expected the contract to be called Sample")
	}

	abiStr, err := json.Marshal(contract.Info.AbiDefinition)
	if err != nil {
		return nil, nil, err
	}

	abi, err := abi.JSON(bytes.NewReader(abiStr))
	if err != nil {
		return nil, nil, err
	}

	tx := &transaction{
		From: deployer,
		Data: contract.Code,
	}

	rr, err := client.SendTxAndWait(tx)
	if err != nil {
		return nil, nil, err
	}

	return &abi, rr, nil
}

func random() bool {
	return os.Getenv("RANDOM_TESTS") == "TRUE"
}
