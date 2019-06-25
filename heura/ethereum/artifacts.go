package ethereum

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/umbracle/heura/heura/ast"
	builtin_contracts "github.com/umbracle/heura/heura/ethereum/builtin"
)

func isDir(path string) bool {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return true
	}
	return false
}

func isFile(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	}
	return false
}

func readFileArtifact(path string) (*abi.ABI, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	artifact := &abi.ABI{}
	if err := artifact.UnmarshalJSON([]byte(data)); err != nil {
		return nil, err
	}

	return artifact, nil
}

func ReadArtifacts(exprs []ast.Expression) (map[string]*abi.ABI, error) {
	objs := map[string]*abi.ABI{}

	add := func(name string, abi *abi.ABI) {
		if _, ok := objs[name]; !ok {
			objs[name] = abi
		}
	}

	for _, expr := range exprs {
		switch obj := expr.(type) {
		case *ast.Identifier:
			// Load builtin library
			abi, err := ReadBuiltInArtifact(obj.Value)
			if err != nil {
				return nil, err
			}
			add(obj.Value, abi)

		case *ast.StringLiteral:
			// Load from file
			if isDir(obj.Value) {
				files, err := ioutil.ReadDir(obj.Value)
				if err != nil {
					return nil, err
				}

				for _, file := range files {
					abi, err := readFileArtifact(filepath.Join(obj.Value, file.Name()))
					if err != nil {
						return nil, err
					}
					add(filepath.Base(file.Name()), abi)
				}

			} else if isFile(obj.Value) {
				abi, err := readFileArtifact(obj.Value)
				if err != nil {
					return nil, err
				}
				add(filepath.Base(obj.Value), abi)

			} else {
				abi, err := ReadBuiltInArtifact(obj.Value)
				if err != nil {
					return nil, err
				}
				add(obj.Value, abi)
			}
		}
	}

	return objs, nil
}

func ReadArtifact(content string) (*abi.ABI, error) {
	artifact := &abi.ABI{}
	if err := artifact.UnmarshalJSON([]byte(content)); err != nil {
		return nil, err
	}

	return artifact, nil
}

func ReadBuiltInArtifact(name string) (*abi.ABI, error) {
	switch name {
	case "ERC20":
		return ReadArtifact(builtin_contracts.ERC20)
	}

	return nil, fmt.Errorf("Builtin %s not found", name)
}
