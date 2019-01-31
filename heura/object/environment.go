package object

import (
	"fmt"
	"strings"
)

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

func NewEnvironment() *Environment {
	s := make(map[string]Object)
	return &Environment{
		store:    s,
		outer:    nil,
		Builtins: map[string]*Builtin{},
	}
}

type Environment struct {
	store    map[string]Object
	outer    *Environment
	Builtins map[string]*Builtin
}

func (e *Environment) AddBuiltins(builtins map[string]*Builtin) {
	for name, b := range builtins {
		e.Builtins[name] = b
	}
}

func (e *Environment) AddBuiltin(name string, b *Builtin) {
	e.Builtins[name] = b
}

func (e *Environment) GetContract(name string) *Contract {
	contract, ok := e.GetContracts()[name]
	if !ok {
		return nil
	}
	return contract
}

func (e *Environment) GetContracts() map[string]*Contract {
	contracts := map[string]*Contract{}

	for _, i := range e.store {
		if j, ok := i.(*Contract); ok {
			contracts[j.Name] = j
		}
	}

	return contracts
}

func (e *Environment) GetOnStatements() []*Event {
	events := []*Event{}

	for _, envt := range e.store {
		if e, ok := envt.(*Event); ok {
			events = append(events, e)
		}
	}

	return events
}

func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]

	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}

	return obj, ok
}

func (e *Environment) Set(name string, val Object) Object {
	e.store[name] = val
	return val
}

func (e *Environment) GetRPCEndpoint() (string, error) {
	obj, ok := e.Get("endpoint")
	if !ok {
		fmt.Println(e.store)
		return "", fmt.Errorf("not found endpoint")
	}

	address, ok := obj.(*String)
	if !ok {
		return "", fmt.Errorf("endpoint is not an string")
	}

	return address.Value, nil
}

func (e *Environment) BuildArgs(envs []string) {
	elems := []Object{}
	for _, i := range envs {
		elems = append(elems, &String{Value: i})
	}

	e.Set("args", &Array{
		Elements: elems,
	})
}

func (e *Environment) BuildEnvs(envs []string) {
	pairs := make(map[HashKey]HashPair)

	for _, env := range envs {
		pair := strings.Split(env, "=")

		key := &String{Value: pair[0]}
		val := &String{Value: pair[1]}

		hashed := key.HashKey()
		pairs[hashed] = HashPair{Key: key, Value: val}
	}

	e.Set("env", &Hash{Pairs: pairs})
}
