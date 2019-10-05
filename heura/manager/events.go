package manager

import (
	"fmt"
	"time"

	"github.com/umbracle/go-web3"
	"github.com/umbracle/go-web3/abi"
	"github.com/umbracle/go-web3/jsonrpc"

	"github.com/umbracle/heura/heura/encoding"
	"github.com/umbracle/heura/heura/evaluator"
	"github.com/umbracle/heura/heura/object"
)

// EventManager is a wrapper to handle event logs
type EventManager struct {
	client  *jsonrpc.Client
	env     *object.Environment
	closeCh chan struct{}
}

// NewEventManager creates a new event manager
func NewEventManager(wsEndpoint string, env *object.Environment) *EventManager {
	client, _ := jsonrpc.NewClient(wsEndpoint)
	return &EventManager{
		client:  client,
		closeCh: make(chan struct{}),
		env:     env,
	}
}

// Listen listens for events
func (e *EventManager) Listen(event *object.Event) error {
	contract := e.env.GetContract(event.Contract)
	if contract == nil {
		return fmt.Errorf("Contract %s not found", event.Contract)
	}

	eventAbi, ok := contract.ABI.Events[event.Method]
	if !ok {
		return fmt.Errorf("Event abi not found on contract")
	}

	go e.listen(event, contract, eventAbi)
	return nil
}

func (e *EventManager) listen(event *object.Event, contract *object.Contract, eventAbi *abi.Event) {
	sig := eventAbi.ID()

	filter := &web3.LogFilter{
		Topics: []*web3.Hash{
			&sig,
		},
	}

	var lastBlock *web3.Block
	for {
		select {
		case <-e.closeCh:
			return

		case <-time.After(3 * time.Second):
			block, err := e.client.Eth().GetBlockByNumber(web3.Latest, false)
			if err != nil {
				fmt.Println(err)
				continue
			}

			if lastBlock != nil && lastBlock.Hash == block.Hash {
				continue
			}

			filter.BlockHash = &block.Hash
			logs, err := e.client.Eth().GetLogs(filter)
			if err != nil {
				fmt.Println(err)
				continue
			}

			for _, log := range logs {
				res, err := abi.ParseLog(eventAbi.Inputs, log)
				if err != nil {
					fmt.Println(err)
					continue
				}
				objs, err := encoding.ArgumentsToObjects(eventAbi.Inputs, res)
				if err != nil {
					fmt.Println(err)
					continue
				}

				evaluator.ApplyEvent(*event, objs, log)
			}
			lastBlock = block
		}
	}
}

// Shutdown closes the manager and all the event listeners
func (e *EventManager) Shutdown() {
	close(e.closeCh)
}
