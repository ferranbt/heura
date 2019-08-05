package encoding

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/umbracle/heura/heura/object"
)

var argumentsTemplate = `pragma solidity ^0.5.5;

contract Sample {
	{{range $indx, $case := .}}
	function set{{$indx}}({{range $indx, $c := $case}}{{$c.TypeStr}} val{{$indx}}, {{end}}) 
		view public returns ({{range $indx, $c := $case}}{{$c.TypeStr}}, {{end}}) {
		return ({{range $indx, $c := $case}}val{{$indx}}, {{end}});
	}
	{{end}}
}
`

type attr struct {
	Type  string
	Value object.Object
}

func (a *attr) TypeStr() string {
	if a.Type == "string" || strings.Contains(a.Type, "[") {
		return a.Type + " memory"
	}
	return a.Type
}

func TestArgumentsIntegration(t *testing.T) {
	client := newClient()

	accounts, err := client.listAccounts()
	if err != nil {
		t.Skipf("Client not responding, skip integration test")
	}

	etherbase := accounts[0]

	cases := [][]*attr{
		{
			{"int", &object.Integer{Value: big.NewInt(1)}},
			{"int", &object.Integer{Value: big.NewInt(1)}},
		},
		{
			{"address", &object.Address{Value: Address}},
		},
		{
			{"uint", &object.Integer{Value: big.NewInt(1)}},
			{"bool", &object.Boolean{Value: true}},
		},
		{
			{"address[]", &object.Array{Elements: []object.Object{&object.Address{Value: Address}, &object.Address{Value: Address}}}},
		},
		{
			{"bytes3", &object.Bytes{Value: "0x111111"}},
			{"bytes3", &object.Bytes{Value: "0x222222"}},
		},
		{
			{"string", &object.String{Value: "abcdef"}},
		},
	}

	abi, receipt, err := compileAndDeployTemplate(argumentsTemplate, cases, etherbase, client)
	if err != nil {
		t.Fatalf("failed to compile and deploy contract: %v", err)
	}

	for indx, cc := range cases {
		t.Run("", func(t *testing.T) {
			values := []object.Object{}
			for _, i := range cc {
				values = append(values, i.Value)
			}

			method, ok := abi.Methods[fmt.Sprintf("set%d", indx)]
			if !ok {
				t.Fatalf("method %s not found", fmt.Sprintf("set%d", indx))
			}

			data, err := Pack(method.Inputs, values)
			if err != nil {
				t.Fatalf("failed to pack: %v", err)
			}

			msg := ethereum.CallMsg{
				From: etherbase,
				To:   &receipt.ContractAddress,
				Data: append(method.Id(), data...),
			}

			resp, err := client.CallContract(context.Background(), msg, nil)
			if err != nil {
				t.Fatalf("failed to call contract: %v", err)
			}

			// actually inputs and outputs are the same
			result, err := Unpack(method.Outputs, resp)
			if err != nil {
				t.Fatalf("failed to unpack: %v", err)
			}

			if !reflect.DeepEqual(result, values) {
				t.Fatal("bad")
			}
		})
	}
}

func TestArgumentsRandom(t *testing.T) {
	if !random() {
		t.Skip()
	}
}
