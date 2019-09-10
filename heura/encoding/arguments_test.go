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
	"github.com/umbracle/go-web3"
	"github.com/umbracle/go-web3/testutil"
	"github.com/umbracle/heura/heura/object"
	"github.com/umbracle/minimal/helper/hex"
)

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

	cc := &testutil.Contract{}
	for indx, i := range cases {
		args := []string{}
		for _, j := range i {
			args = append(args, j.TypeStr())
		}
		cc.AddDualCaller("set"+strconv.Itoa(indx), args...)
	}

	server := testutil.NewTestServer(t, nil)
	defer server.Close()

	solcContract, addr := server.DeployContract(cc)

	abi, err := abi.JSON(bytes.NewReader([]byte(solcContract.Abi)))
	if err != nil {
		t.Fatal(err)
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

			msg := &web3.CallMsg{
				To:   addr,
				Data: hex.EncodeToHex(append(method.Id(), data...)),
			}
			raw, err := server.Call(msg)
			if err != nil {
				t.Fatal(err)
			}

			// actually inputs and outputs are the same
			result, err := Unpack(method.Outputs, hex.MustDecodeHex(raw))
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
