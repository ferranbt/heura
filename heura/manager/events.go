package manager

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"

	eth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/umbracle/heura/heura/encoding"
	"github.com/umbracle/heura/heura/ethereum"
	"github.com/umbracle/heura/heura/evaluator"
	"github.com/umbracle/heura/heura/object"
)

// EventManager is a wrapper to handle event logs
type EventManager struct {
	client  *ethclient.Client
	env     *object.Environment
	closeCh chan struct{}
}

// NewEventManager creates a new event manager
func NewEventManager(wsEndpoint string, env *object.Environment) *EventManager {
	return &EventManager{
		client:  ethereum.NewClient(wsEndpoint),
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

	go e.listen(event, contract, &eventAbi)
	return nil
}

func (e *EventManager) listen(event *object.Event, contract *object.Contract, eventAbi *abi.Event) {
	query := eth.FilterQuery{
		Topics: [][]common.Hash{
			[]common.Hash{
				eventAbi.Id(),
			},
		},
	}

	if event.Address != nil {
		query.Addresses = []common.Address{*event.Address}
	}

	logs := make(chan types.Log)
	sub, err := e.client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		fmt.Printf("Failed to subscribe to logs: %v", err)
		return
	}

	for {
		select {
		case log := <-logs:

			values, err := encoding.ParseLog(eventAbi.Inputs, &log)
			if err != nil {
				fmt.Printf("Failed to parse topics: %v\n", err)
				continue
			}
			evaluator.ApplyEvent(*event, values, &log)

		case err := <-sub.Err():
			fmt.Printf("Subscription closed: %vn", err)
			return

		case <-e.closeCh:
			return
		}
	}
}

// Shutdown closes the manager and all the event listeners
func (e *EventManager) Shutdown() {
	close(e.closeCh)
}
