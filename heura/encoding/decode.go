package encoding

import (
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/umbracle/heura/heura/object"
	"github.com/umbracle/minimal/helper/hex"
)

var (
	boolTyp   = reflect.TypeOf(true)
	intTyp    = reflect.TypeOf(int64(0))
	stringTyp = reflect.TypeOf("")
)

// NOTE. Since the objects dont have types, we need to cast some of the values with the independent unpack methods
// If types are added eventually, some object values (int, bytes...) will have the type object
// so we could remove the pack from here.

// Decode converts an object into a go element
func Decode(obj object.Object, t abi.Type) (interface{}, error) {
	switch t.T {
	case abi.SliceTy:
		return decodeSlice(obj, t)

	case abi.IntTy:
		return decodeInt(obj, t)

	case abi.UintTy:
		return decodeUint(obj, t)

	case abi.BoolTy:
		return decodeBool(obj)

	case abi.FixedBytesTy:
		return decodeFixedBytes(obj, t.Type)

	case abi.HashTy:
		return decodeHash(obj, t.Type)

	case abi.AddressTy:
		return decodeAddress(obj, t.Type)

	case abi.StringTy:
		return decodeString(obj)

	case abi.BytesTy:
		return nil, fmt.Errorf("bytes mode not enabled")

	case abi.ArrayTy:
		return nil, fmt.Errorf("array type not covered")

	default:
		return nil, fmt.Errorf("Decode type %s not supported", t.String())
	}
}

func decodeString(obj object.Object) (interface{}, error) {
	if obj.Type() != object.STRING_OBJ {
		return nil, decodeErr(obj, "string")
	}

	return obj.(*object.String).Value, nil
}

func decodeUint(obj object.Object, t abi.Type) (interface{}, error) { // FIX, how to determine uint or not, a funcion in object.Integer, if function fails it is not
	if obj.Type() != object.INTEGER_OBJ {
		return nil, decodeErr(obj, "uint")
	}

	if t.Size == 256 {
		return obj.(*object.Integer).Value, nil
	}

	return reflect.ValueOf(obj.(*object.Integer).Value.Uint64()).Convert(t.Type).Interface(), nil
}

func decodeInt(obj object.Object, t abi.Type) (interface{}, error) {
	if obj.Type() != object.INTEGER_OBJ {
		return nil, decodeErr(obj, "int")
	}

	if t.Size == 256 {
		return obj.(*object.Integer).Value, nil
	}

	return reflect.ValueOf(obj.(*object.Integer).Value.Int64()).Convert(t.Type).Interface(), nil
}

func decodeBool(obj object.Object) (interface{}, error) {
	if obj.Type() != object.BOOLEAN_OBJ {
		return nil, decodeErr(obj, "bool")
	}

	return obj.(*object.Boolean).Value, nil
}

func decodeFixedBytes(obj object.Object, t reflect.Type) (interface{}, error) {
	if obj.Type() != object.BYTES_OBJ {
		return nil, decodeErr(obj, "fixed bytes")
	}

	hex, err := hex.DecodeHex(obj.(*object.Bytes).Value)
	if err != nil {
		return nil, err
	}

	array := reflect.New(t).Elem()
	if len(hex) < t.Len() {
		reflect.Copy(array, reflect.ValueOf(hex[0:len(hex)]))
	} else {
		reflect.Copy(array, reflect.ValueOf(hex[0:t.Len()]))
	}

	return array.Interface(), nil
}

func decodeSlice(obj object.Object, t abi.Type) (interface{}, error) {
	if obj.Type() != object.ARRAY_OBJ {
		return nil, decodeErr(obj, "slice")
	}

	elems := obj.(*object.Array).Elements
	elemType := *t.Elem

	sliceVal := reflect.MakeSlice(t.Type, len(elems), len(elems))
	for i, elt := range elems {
		v, err := Decode(elt, elemType)
		if err != nil {
			return nil, fmt.Errorf("element %d: %s", i, err)
		}

		sliceVal.Index(i).Set(reflect.ValueOf(v))
	}

	return sliceVal.Interface(), nil
}

func decodeHash(obj object.Object, t reflect.Type) (interface{}, error) {
	if obj.Type() != object.BYTES_OBJ {
		return nil, decodeErr(obj, "hash")
	}

	hex, err := hex.DecodeHex(obj.(*object.Bytes).Value)
	if err != nil {
		return nil, err
	}

	array := reflect.New(t).Elem()
	if len(hex) < t.Len() {
		reflect.Copy(array, reflect.ValueOf(hex[0:len(hex)]))
	} else {
		reflect.Copy(array, reflect.ValueOf(hex[0:t.Len()]))
	}

	return array.Interface(), nil
}

func decodeAddress(obj object.Object, t reflect.Type) (interface{}, error) {
	if obj.Type() != object.ADDRESS_OBJ {
		return nil, decodeErr(obj, "address")
	}

	return common.HexToAddress(obj.(*object.Address).Value), nil
}

func decodeErr(obj object.Object, t string) error {
	return fmt.Errorf("failed to decode %s as %s", obj.Type(), t)
}
