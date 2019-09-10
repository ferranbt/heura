package encoding

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/umbracle/go-web3/testutil"
	"github.com/umbracle/heura/heura/object"
	"github.com/umbracle/minimal/helper/hex"
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
		{
			&object.Array{
				Elements: []object.Object{
					&object.Address{Value: Address},
					&object.Address{Value: Address},
				},
			},
			"address[]",
		},
	}

	for _, cc := range cases {
		t.Run("", func(t *testing.T) {
			tt, err := abi.NewType(cc.Type, nil)
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

			if cc.Input.Type() != object.ARRAY_OBJ {
				if !reflect.DeepEqual(topic, cc.Input) {
					t.Fatalf("bad")
				}
			}
		})
	}
}

type attrTopic struct {
	Type    string
	Value   object.Object
	Indexed bool
}

func (a *attrTopic) TypeStr() string {
	if a.Type == "string" || strings.Contains(a.Type, "[") {
		return a.Type + " memory"
	}
	return a.Type
}

func TestTopicsIntegration(t *testing.T) {

	cases := [][]*attrTopic{
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
		{
			{
				"address[]",
				&object.Array{
					Elements: []object.Object{
						&object.Address{Value: Address},
						&object.Address{Value: Address},
					},
				},
				true,
			},
		},
		{
			{
				"int[]",
				&object.Array{
					Elements: []object.Object{
						&object.Integer{Value: big.NewInt(10)},
						&object.Integer{Value: big.NewInt(20)},
					},
				},
				true,
			},
		},
	}

	// create event contract
	cc := &testutil.Contract{}
	for indx, i := range cases {
		evnt := testutil.NewEvent("Event" + strconv.Itoa(indx))
		for _, attr := range i {
			evnt.Add(attr.Type, attr.Indexed)
		}
		cc.AddEvent(evnt)
	}

	server := testutil.NewTestServer(t, nil)
	defer server.Close()

	solcContract, addr := server.DeployContract(cc)

	etherbase := common.HexToAddress(server.Account(0))
	client := newClientWithEndpoint(server.HTTPAddr())

	abi, err := abi.JSON(bytes.NewReader([]byte(solcContract.Abi)))
	if err != nil {
		t.Fatal(err)
	}

	for indx, cc := range cases {
		t.Run("", func(t *testing.T) {
			// send the tx
			values := []object.Object{}
			for _, i := range cc {
				values = append(values, i.Value)
			}

			method, ok := abi.Methods[fmt.Sprintf("setterEvent%d", indx)]
			if !ok {
				t.Fatalf("method %s not found", fmt.Sprintf("setterEvent%d", indx))
			}

			event, ok := abi.Events[fmt.Sprintf("Event%d", indx)]
			if !ok {
				t.Fatalf("event %s not found", fmt.Sprintf("Event%d", indx))
			}

			data, err := Pack(method.Inputs, values)
			if err != nil {
				t.Fatalf("failed to pack: %v", err)
			}

			addr0 := common.HexToAddress(addr)
			tx := &transaction{
				From: etherbase,
				To:   &addr0,
				Data: hex.EncodeToHex(append(method.Id(), data...)),
			}

			rr, err := client.SendTxAndWait(tx)
			if err != nil {
				t.Fatalf("failed to send tx: %v", err)
			}

			res, err := ParseLog(event.Inputs, rr.Logs[0])
			if err != nil {
				t.Fatalf("failed to parse logs: %v", err)
			}

			if len(res) != len(values) {
				t.Fatal("bad length")
			}
			for indx, i := range values {
				if i.Type() != object.ARRAY_OBJ {
					if !reflect.DeepEqual(i, res[indx]) {
						t.Fatal("bad decoding")
					}
				}
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
