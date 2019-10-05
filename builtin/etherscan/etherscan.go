package etherscan

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/umbracle/go-web3"
	"github.com/umbracle/go-web3/abi"

	"github.com/umbracle/heura/heura/object"
)

const (
	etherscanURL = "http://api.etherscan.io/api"
)

type output struct {
	Status  string
	Message string
	Result  string
}

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

func getABI(contract string) (*abi.ABI, error) {
	url := fmt.Sprintf("%s?module=contract&action=getabi&address=%s", etherscanURL, contract)

	req, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer req.Body.Close()

	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	var out output
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}

	return abi.NewABI(out.Result)
}

func getABIBuiltin(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError("expected one param but found %d", len(args))
	}
	if args[0].Type() != object.STRING_OBJ {
		return newError("expected argument to be string, got %s", args[0].Type())
	}
	val, err := getABI(args[0].(*object.String).Value)
	if err != nil {
		return newError(err.Error())
	}
	return &object.Contract{Name: "Artifact", ABI: val}
}

func getContractBuiltin(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError("expected one param but found %d", len(args))
	}
	if args[0].Type() != object.STRING_OBJ {
		return newError("expected argument to be string, got %s", args[0].Type())
	}
	addr := args[0].(*object.String).Value
	val, err := getABI(addr)
	if err != nil {
		return newError(err.Error())
	}
	return &object.Instance{Name: "Artifact", Address: web3.HexToAddress(addr), ABI: val}
}

// Factory is the factory method for the Etherscan backend
func Factory() object.Object {
	h := &object.Hash{}

	h.SetString("ABI", &object.Builtin{
		Fn: getABIBuiltin,
	})
	h.SetString("Contract", &object.Builtin{
		Fn: getContractBuiltin,
	})

	return h
}
