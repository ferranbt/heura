package encoding

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/umbracle/heura/heura/object"
)

var (
	Address = "0x1111111111111111111111111111111111111111"
)

func mustDecode(s string) common.Address {
	return common.HexToAddress(s)
}

func TestEncoding(t *testing.T) {
	var cases = []struct {
		Input  object.Object
		Type   string
		Output interface{}
	}{
		{
			&object.Integer{Value: big.NewInt(1)},
			"int256",
			big.NewInt(1),
		},
		{
			&object.Bytes{Value: "0x1000"},
			"bytes2",
			[2]byte{16, 0},
		},
		{
			&object.Array{Elements: []object.Object{&object.Integer{Value: big.NewInt(1)}}},
			"int256[]",
			[]*big.Int{big.NewInt(1)},
		},
		{
			&object.Array{
				Elements: []object.Object{
					&object.Array{
						Elements: []object.Object{
							&object.Integer{Value: big.NewInt(1)},
						},
					},
				},
			},
			"int256[][]",
			[][]*big.Int{[]*big.Int{big.NewInt(1)}},
		},
		{
			&object.Integer{Value: big.NewInt(1)},
			"int8",
			int8(1),
		},
		{
			&object.Address{Value: Address},
			"address",
			mustDecode(Address),
		},
		{
			&object.Array{
				Elements: []object.Object{
					&object.Address{Value: Address},
					&object.Address{Value: Address},
				},
			},
			"address[]",
			[]common.Address{mustDecode(Address), mustDecode(Address)},
		},
		{
			&object.Array{
				Elements: []object.Object{
					&object.Bytes{Value: "0x1000"},
					&object.Bytes{Value: "0x1000"},
				},
			},
			"bytes2[]",
			[][2]byte{[2]byte{16, 0}, [2]byte{16, 0}},
		},
		{
			&object.Integer{Value: big.NewInt(1)},
			"uint8",
			uint8(1),
		},
	}

	for _, cc := range cases {
		t.Run("", func(t *testing.T) {
			ttt, err := abi.NewType(cc.Type)
			if err != nil {
				t.Fatal(err.Error())
			}

			obj, err := Decode(cc.Input, ttt)
			if err != nil {
				t.Fatal(err.Error())
			}

			if !reflect.DeepEqual(obj, cc.Output) {
				t.Fatal("bad decoding")
			}

			obj2, err := Encode(obj, ttt)
			if err != nil {
				t.Fatal(err.Error())
			}

			if !reflect.DeepEqual(cc.Input, obj2) {
				t.Fatal("bad encoding")
			}
		})
	}
}
