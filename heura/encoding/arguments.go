package encoding

import (
	"fmt"
	"strconv"

	"github.com/umbracle/go-web3/abi"
	"github.com/umbracle/heura/heura/object"
)

// Pack packs the values
func Pack(arguments abi.Arguments, args []object.Object) ([]byte, error) {
	if len(arguments) != len(args) {
		return nil, fmt.Errorf("not enough arguments to pack. Found %d, Expected %d", len(args), len(arguments))
	}

	elems := make([]interface{}, len(args))
	for indx, arg := range args {
		elem, err := Decode(arg, *arguments[indx].Type)
		if err != nil {
			return nil, err
		}
		elems[indx] = elem
	}
	return abi.Encode(elems, arguments.Type())
}

// Unpack unpacks the ethereum values with the types
func Unpack(arguments abi.Arguments, data []byte) ([]object.Object, error) {
	raw, err := abi.Decode(arguments.Type(), data)
	if err != nil {
		return nil, err
	}

	objs, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("bad unpack, list expected")
	}
	return ArgumentsToObjects(arguments, objs)
}

// ArgumentsToObjects converts arguments in Go format to Heura objects
func ArgumentsToObjects(arguments abi.Arguments, objs map[string]interface{}) ([]object.Object, error) {
	res := []object.Object{}
	for indx, i := range arguments {
		name := i.Name
		if name == "" {
			name = strconv.Itoa(indx)
		}
		elem, err := Encode(objs[name], *arguments[indx].Type)
		if err != nil {
			return nil, err
		}
		res = append(res, elem)
	}
	return res, nil
}
