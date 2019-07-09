package encoding

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/umbracle/heura/heura/object"
	"github.com/umbracle/minimal/helper/hex"
)

// Big batch of reflect types for topic reconstruction.
var (
	BigInt = reflect.TypeOf(new(big.Int))
)

// Encode unpacks a value interface into an object
func Encode(value interface{}, t abi.Type) (object.Object, error) {
	return encode(reflect.ValueOf(value), t)
}

func encode(v reflect.Value, t abi.Type) (object.Object, error) {
	switch t.T {
	case abi.SliceTy:
		return encodeSlice(v, t)

	case abi.IntTy:
		return encodeInt(v)

	case abi.UintTy:
		return encodeUInt(v)

	case abi.BoolTy:
		return encodeBool(v)

	case abi.FixedBytesTy:
		return encodeFixedBytes(v)

	case abi.AddressTy:
		return encodeAddress(v)

	case abi.StringTy:
		return encodeString(v)

	case abi.HashTy:
		return nil, fmt.Errorf("hash type not supported")

	case abi.BytesTy:
		return nil, fmt.Errorf("hash type not supported")

	case abi.ArrayTy:
		return nil, fmt.Errorf("hash type not supported")
	}

	return nil, fmt.Errorf("Encode type %s not supported", t.String())
}

func encodeString(v reflect.Value) (object.Object, error) {
	if v.Kind() != reflect.String {
		return nil, encodeErr(v, "string")
	}

	return &object.String{Value: v.String()}, nil
}

func encodeSlice(v reflect.Value, t abi.Type) (object.Object, error) {
	if v.Kind() != reflect.Slice {
		return nil, encodeErr(v, "slice")
	}

	vs := make([]object.Object, v.Len())
	for i := range vs {
		elem, err := encode(v.Index(i), *t.Elem)
		if err != nil {
			return nil, err
		}

		vs[i] = elem
	}

	return &object.Array{
		Elements: vs,
	}, nil
}

func encodeAddress(v reflect.Value) (object.Object, error) {
	data, err := readBytes(v)
	if err != nil {
		return nil, err
	}

	return &object.Address{Value: hex.EncodeToHex(data)}, nil
}

func readBytes(v reflect.Value) ([]byte, error) {
	switch v.Kind() {
	case reflect.Slice:
		return v.Bytes(), nil

	case reflect.Array:
		elems := []byte{}
		for indx := 0; indx < v.Len(); indx++ {
			elems = append(elems, v.Index(indx).Interface().(byte))
		}
		return elems, nil

	default:
		return []byte{}, encodeErr(v, "bytes")
	}
}

func encodeFixedBytes(v reflect.Value) (object.Object, error) {
	data, err := readBytes(v)
	if err != nil {
		return nil, err
	}

	return &object.Bytes{Value: hex.EncodeToHex(data)}, nil
}

func encodeInt(v reflect.Value) (object.Object, error) {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &object.Integer{Value: big.NewInt(v.Int())}, nil

	case reflect.Ptr:
		if v.Type() == BigInt {
			return &object.Integer{Value: v.Interface().(*big.Int)}, nil
		}
	}

	return nil, encodeErr(v, "int")
}

func encodeUInt(v reflect.Value) (object.Object, error) {
	switch v.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &object.Integer{Value: big.NewInt(int64(v.Uint()))}, nil

	case reflect.Ptr:
		if v.Type() == BigInt {
			return &object.Integer{Value: v.Interface().(*big.Int)}, nil
		}
	}

	return nil, encodeErr(v, "uint")
}

func encodeBool(v reflect.Value) (object.Object, error) {
	if v.Kind() != reflect.Bool {
		return nil, encodeErr(v, "bool")
	}

	return &object.Boolean{Value: v.Bool()}, nil
}

func encodeErr(v reflect.Value, t string) error {
	return fmt.Errorf("failed to encode %s as %s", v.Kind().String(), t)
}
