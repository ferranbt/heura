package encoding

import (
	"fmt"
	"math/big"

	"golang.org/x/crypto/sha3"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/ethereum/go-ethereum/common"

	"github.com/umbracle/heura/heura/object"
	"github.com/umbracle/minimal/helper/hex"
)

// ParseLog parses the log both topics and data
func ParseLog(args abi.Arguments, log *types.Log) ([]object.Object, error) {
	indexed, nonIndexed := abi.Arguments{}, abi.Arguments{}
	for _, i := range args {
		if i.Indexed {
			indexed = append(indexed, i)
		} else {
			nonIndexed = append(nonIndexed, i)
		}
	}

	// parse indexed values (topics)
	indexedObjs, err := ParseTopics(indexed, log.Topics[1:])
	if err != nil {
		return nil, fmt.Errorf("failed to parse indexed topics: %v", err)
	}

	// parse non-indexed data
	nonIndexedObjs, err := Unpack(nonIndexed, log.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse non-indexed data: %v", err)
	}

	elems := []object.Object{}
	for _, i := range args {
		var elem object.Object
		if i.Indexed {
			elem, indexedObjs = indexedObjs[0], indexedObjs[1:]
		} else {
			elem, nonIndexedObjs = nonIndexedObjs[0], nonIndexedObjs[1:]
		}

		elems = append(elems, elem)
	}

	return elems, nil
}

// ParseTopics parses the topics
func ParseTopics(args abi.Arguments, topics []common.Hash) ([]object.Object, error) {
	if len(args) != len(topics) {
		return nil, fmt.Errorf("Length should be the same. Arguments %d and topics %d", len(args), len(topics))
	}

	elems := []object.Object{}
	for indx, i := range args {
		elem, err := ParseTopic(topics[indx].Bytes(), i.Type)
		if err != nil {
			return nil, err
		}

		elems = append(elems, elem)
	}

	return elems, nil
}

// EncodeTopics encodes a group of topics
func EncodeTopics(args abi.Arguments, objs []object.Object) ([][]common.Hash, error) {
	if len(args) != len(objs) {
		return nil, fmt.Errorf("Length should be the same. Arguments %d and objs %d", len(args), len(objs))
	}

	topics := [][]common.Hash{}
	for indx, arg := range args {
		t := []common.Hash{}
		if objs[indx] != nil {
			topic, err := EncodeTopic(objs[indx], arg.Type)
			if err != nil {
				return nil, err
			}
			t = append(t, topic)
		}
		topics = append(topics, t)
	}

	return topics, nil
}

// EncodeTopic encodes a topic with a specific type
func EncodeTopic(obj object.Object, t abi.Type) (common.Hash, error) {
	return encodeTopic(obj, t, 0)
}

func encodeTopic(obj object.Object, t abi.Type, arraySize int) (common.Hash, error) {
	switch t.T {
	case abi.BoolTy:
		if obj.Type() != object.BOOLEAN_OBJ {
			return common.Hash{}, encodeTopicErr(obj, t)
		}

		topic := common.Hash{}
		if obj.(*object.Boolean).Value == true {
			topic[common.HashLength-1] = 1
		}
		return topic, nil

	case abi.IntTy, abi.UintTy: // FIX, difference between uint and int
		if obj.Type() != object.INTEGER_OBJ {
			return common.Hash{}, encodeTopicErr(obj, t)
		}
		return common.BigToHash(obj.(*object.Integer).Value), nil

	case abi.HashTy:
		if obj.Type() != object.BYTES_OBJ {
			return common.Hash{}, encodeTopicErr(obj, t)
		}
		return common.HexToHash(obj.(*object.Bytes).Value), nil

	case abi.AddressTy:
		var val string
		switch obj.Type() {
		case object.ADDRESS_OBJ:
			val = obj.(*object.Address).Value

		case object.BYTES_OBJ:
			val = obj.(*object.Bytes).Value

		default:
			return common.Hash{}, encodeTopicErr(obj, t)
		}

		var size int
		if arraySize != 0 {
			size = 64
		} else {
			size = 40
		}
		return common.BytesToHash(common.LeftPadBytes(common.HexToHash(val).Bytes(), size)), nil

	case abi.FixedBytesTy:
		if obj.Type() != object.BYTES_OBJ {
			return common.Hash{}, encodeTopicErr(obj, t)
		}

		topic := common.Hash{}
		bytes, err := hex.DecodeHex(obj.(*object.Bytes).Value)
		if err != nil {
			return common.Hash{}, err
		}

		copy(topic[0:len(bytes)], bytes[:])
		return topic, nil

	case abi.SliceTy:
		if obj.Type() != object.ARRAY_OBJ {
			return common.Hash{}, encodeTopicErr(obj, t)
		}

		arr := obj.(*object.Array)
		size := len(arr.Elements)

		res := []byte{}
		for _, val := range arr.Elements {
			r, err := encodeTopic(val, *t.Elem, size)
			if err != nil {
				return common.Hash{}, err
			}
			res = append(res, r.Bytes()...)
		}
		return common.BytesToHash(hash(res)), nil

	default:
		return common.Hash{}, fmt.Errorf("Topic encoding of type %s not supported", t.String())
	}
}

// ParseTopic decodes a topic
func ParseTopic(data []byte, t abi.Type) (object.Object, error) {
	switch t.T {
	case abi.BoolTy:
		return &object.Boolean{Value: data[common.HashLength-1] == 1}, nil

	case abi.IntTy:
		return &object.Integer{Value: new(big.Int).SetBytes(data)}, nil

	case abi.UintTy:
		return &object.Integer{Value: new(big.Int).SetBytes(data)}, nil

	case abi.HashTy:
		return &object.Bytes{Value: hex.EncodeToHex(data)}, nil

	case abi.AddressTy:
		return &object.Address{Value: hex.EncodeToHex(data[common.HashLength-common.AddressLength:])}, nil

	case abi.FixedBytesTy:
		return &object.Bytes{Value: hex.EncodeToHex(data[0:t.Size])}, nil

	case abi.ArrayTy:
		fallthrough

	case abi.SliceTy:
		// Arrays are converted into sha3 format https://github.com/ethereum/web3.js/issues/344
		return &object.Bytes{Value: hex.EncodeToHex(data)}, nil

	default:
		return nil, fmt.Errorf("Topic parsing of type %s not supported", t.String())
	}
}

func encodeTopicErr(obj object.Object, t abi.Type) error {
	return fmt.Errorf("cannot encode %s as %s", obj.Type(), t.String())
}

func hash(b []byte) []byte {
	f := sha3.NewLegacyKeccak256()
	f.Write(b)
	res := f.Sum(nil)
	return res
}
