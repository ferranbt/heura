package encoding

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/umbracle/heura/heura/object"
)

// Pack packs the values
func Pack(arguments abi.Arguments, args []object.Object) ([]byte, error) {
	if len(arguments) != len(args) {
		return nil, fmt.Errorf("not enough arguments to pack. Found %d, Expected %d", len(args), len(arguments))
	}

	elems := make([]interface{}, len(args))
	for indx, arg := range args {
		elem, err := Decode(arg, arguments[indx].Type)
		if err != nil {
			return nil, err
		}

		elems[indx] = elem
	}

	return arguments.PackValues(elems)
}

// Unpack unpacks the ethereum values with the types
func Unpack(abi abi.Arguments, data []byte) ([]object.Object, error) {
	objs, err := abi.UnpackValues(data)
	if err != nil {
		return nil, err
	}

	res := make([]object.Object, len(objs))
	for indx, i := range objs {
		elem, err := Encode(i, abi[indx].Type)
		if err != nil {
			return nil, err
		}

		res[indx] = elem
	}

	return res, nil
}
