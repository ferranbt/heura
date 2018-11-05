package encoding

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/umbracle/heura/heura/object"
)

func TestTopicEncoding(t *testing.T) {
	var cases = []struct {
		Input object.Object
		Type  string
	}{
		{
			&object.Integer{Value: big.NewInt(1)},
			"int256",
		},
	}

	for _, cc := range cases {
		t.Run("", func(t *testing.T) {
			tt, err := abi.NewType(cc.Type)
			if err != nil {
				t.Fatalf("format name %s is invalid: %v", cc.Type, err)
			}

			output, err := EncodeTopic(cc.Input, tt)
			if err != nil {
				t.Fatalf("bad topic encoding: %v", err)
			}

			topic, err := ParseTopic(output.Bytes(), tt)
			if err != nil {
				t.Fatalf("bad topic parsing: %v", err)
			}

			if !reflect.DeepEqual(topic, cc.Input) {
				t.Fatalf("bad")
			}
		})
	}
}

var topicTemplate = `pragma solidity ^0.4.22;

contract Sample {
	{{range $indx, $case := .}}
	event Event{{$indx}}({{range $indx, $c := $case}}{{$c.Type}}{{if $c.Indexed}} indexed{{end}} val{{$indx}}, {{end}});
	function set{{$indx}}({{range $indx, $c := $case}}{{$c.Type}} val{{$indx}}, {{end}}) {
		emit Event{{$indx}}({{range $indx, $c := $case}}val{{$indx}}, {{end}});
	}
	{{end}}
}
`

func TestTopicsIntegration(t *testing.T) {
	type Attr struct {
		Type    string
		Value   object.Object
		Indexed bool
	}

	cases := [][]*Attr{
		{
			{"int", &object.Integer{Value: big.NewInt(1)}, true},
			{"int", &object.Integer{Value: big.NewInt(1)}, false},
		},
		{
			{"address", &object.Address{Value: Address}, true},
		},
		{
			{"uint", &object.Integer{Value: big.NewInt(1)}, true},
			{"bool", &object.Boolean{Value: true}, true},
		},
		{
			{"bytes3", &object.Bytes{Value: "0x111111"}, true},
			{"address", &object.Address{Value: Address}, true},
		},
	}

	client := newClient("http://localhost:8545")

	accounts, err := client.listAccounts()
	if err != nil {
		t.Skipf("Client not responding, skip integration test")
	}

	etherbase := accounts[0]

	abi, receipt, err := compileAndDeployTemplate(topicTemplate, cases, etherbase, client)
	if err != nil {
		t.Fatalf("failed to compile and deploy contract: %v", err)
	}

	for indx, cc := range cases {
		t.Run("", func(t *testing.T) {
			// send the tx
			values := []object.Object{}
			for _, i := range cc {
				values = append(values, i.Value)
			}

			method, ok := abi.Methods[fmt.Sprintf("set%d", indx)]
			if !ok {
				t.Fatalf("method %s not found", fmt.Sprintf("set%d", indx))
			}

			event, ok := abi.Events[fmt.Sprintf("Event%d", indx)]
			if !ok {
				t.Fatalf("event %s not found", fmt.Sprintf("Event%d", indx))
			}

			data, err := Pack(method.Inputs, values)
			if err != nil {
				t.Fatalf("failed to pack: %v", err)
			}

			tx := &transaction{
				From: etherbase,
				To:   &receipt.ContractAddress,
				Data: hexutil.Encode(append(method.Id(), data...)),
			}

			rr, err := client.SendTxAndWait(tx)
			if err != nil {
				t.Fatalf("failed to send tx: %v", err)
			}

			res, err := ParseLog(event.Inputs, rr.Logs[0])
			if err != nil {
				t.Fatalf("failed to parse logs: %v", err)
			}

			if !reflect.DeepEqual(res, values) {
				t.Fatal("bad decoding")
			}

			for indx, i := range event.Inputs {
				if i.Indexed {
					val, err := EncodeTopic(values[indx], i.Type)
					if err != nil {
						t.Fatalf("failed to encode topic: %v", err)
					}

					real := rr.Logs[0].Topics[1+indx] // 1 + to avoid the first one which is the signature

					if !reflect.DeepEqual(val, real) {
						t.Fatal("bad encoding")
					}
				}
			}
		})
	}
}
