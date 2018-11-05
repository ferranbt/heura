package evaluator

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/umbracle/heura/heura/token"

	"github.com/ethereum/go-ethereum/contracts/ens"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/umbracle/heura/heura/ast"
	"github.com/umbracle/heura/heura/ethereum"
	"github.com/umbracle/heura/heura/object"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {

	case *ast.Program:
		return evalProgram(node, env)

	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

	case *ast.ArtifactStatement:

		abis, err := ethereum.ReadArtifacts(node.Folders)
		if err != nil {
			return newError(err.Error())
		}

		for name, abi := range abis {
			env.Set(name, &object.Contract{Name: name, ABI: abi})
		}

		return nil

	case *ast.OnStatement:
		params := node.Parameters
		body := node.Body

		contract := node.Contract.String()
		method := node.Method.String()

		// check if the objects are valid
		c, ok := env.Get(contract)
		if !ok {
			return newError("contract not found")
		}

		cc, ok := c.(*object.Contract)
		if !ok {
			return newError("object found but its no contract")
		}

		m, ok := cc.ABI.Events[method]
		if !ok {
			return newError("Method not found for event")
		}

		if len(m.Inputs) != len(params) {
			return newError("Event len different %d and %d", len(m.Inputs), len(params))
		}

		event := &object.Event{
			Contract:   contract,
			Method:     method,
			ABI:        cc.ABI,
			Parameters: params,
			Body:       body,
			Env:        env,
		}

		// Check if we listen for a specific address
		if node.Address != nil {
			obj := Eval(node.Address, env)
			addr, err := evalAddress(env, obj)
			if err != nil {
				return newError(err.Error())
			}

			i := addr.ToAddress()
			event.Address = &i
		}

		/*
			// TODO. Check for specific topics
			topicObjs := []object.Object{}
			topicArgs := abi.Arguments{}

			for indx, i := range m.Inputs {
				if i.Indexed {
					topicObjs = append(topicObjs, Eval(params[indx].Value, env))
					topicArgs = append(topicArgs, i)
				}
			}

			topics, err := encoding.EncodeTopics(topicArgs, topicObjs)
			if err != nil {
				return newError(err.Error())
			}
		*/

		env.Set(fmt.Sprintf("%s_%s", contract, method), event)
		return nil

	case *ast.IntegerLiteral:
		return &object.Integer{Value: big.NewInt(node.Value)}

	case *ast.BytesLiteral:
		return &object.Bytes{Value: node.Value}

	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)

	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)

	case *ast.InfixExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}

		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}

		return evalInfixExpression(node.Operator, left, right)

	case *ast.BlockStatement:
		return evalBlockStatement(node, env)

	case *ast.IfExpression:
		return evalIfExpression(node, env)

	case *ast.ReturnStatement:
		val := Eval(node.ReturnValue, env)
		if isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}

	case *ast.LetStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}

		// decode the number of values
		values := []object.Object{}
		if val.Type() == object.MULTIPLE_OBJ {
			elem := val.(*object.Multiple)
			values = elem.Values
		} else {
			values = append(values, val)
		}

		if len(node.Name) != len(values) {
			return newError("Length of let and values is different: %d, %d", len(node.Name), len(values))
		}

		for indx, val := range values {
			env.Set(node.Name[indx].Value, val)
		}

	case *ast.Identifier:
		return evalIdentifier(node, env)

	case *ast.FunctionLiteral:
		params := node.Parameters
		body := node.Body

		val := &object.Function{
			Parameters: params,
			Body:       body,
			Env:        env,
		}

		env.Set(node.Name.Value, val)
		return val

	case *ast.CallExpression:
		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}

		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}

		return ApplyFunction(env, function, args)

	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

	case *ast.ArrayLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}

		return &object.Array{Elements: elements}

	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}

		if node.TokenLiteral() == token.DOT {
			return evalDotIndexExpression(env, left, node.Index)
		}

		index := Eval(node.Index, env)
		if isError(index) {
			return index
		}
		return evalIndexExpression(left, index)

	case *ast.HashLiteral:
		return evalHashLiteral(node, env)

	case *ast.MultipleExpression:
		values := []object.Object{}
		for _, val := range node.Expressions {
			val := Eval(val, env)
			if isError(val) {
				return val
			}
			values = append(values, val)
		}

		return &object.Multiple{Values: values}
	}

	return nil
}

func evalIndexExpression(left, index object.Object) object.Object {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return evalArrayIndexExpression(left, index)
	case left.Type() == object.HASH_OBJ:
		return evalHashIndexExpression(left, index)
	default:
		return newError("index operator not supported: %s", left.Type())
	}
}

func evalDotIndexExpression(env *object.Environment, left object.Object, index ast.Expression) object.Object {
	switch {
	case left.Type() == object.HASH_OBJ:
		switch obj := index.(type) {
		case *ast.Identifier:
			return evalHashIndexExpression(left, &object.String{Value: obj.Value})
		case *ast.IndexExpression:
			left = evalHashIndexExpression(left, &object.String{Value: obj.Left.(*ast.Identifier).Value})
			return evalDotIndexExpression(env, left, obj.Index)
		}
		return newError("Dot access to hash object requires an identifier")

	case left.Type() == object.INSTANCE_OBJ:
		return evalInstanceCall(left.(*object.Instance), index, env)
	}

	return newError("dot index operator not supported: %s", left.Type())
}

func evalAddress(env *object.Environment, obj object.Object) (*object.Address, error) {
	var address *object.Address

	switch arg := obj.(type) {
	case *object.Bytes:
		addr, err := arg.ToAddress()
		if err != nil {
			return nil, fmt.Errorf("failed to convert bytes to address: %v", err)
		}
		address = addr
	case *object.Address:
		address = arg
	case *object.String:
		// Ens resolve

		endpoint, err := env.GetRPCEndpoint()
		if err != nil {
			return nil, err
		}

		ens, err := ethereum.NewENS(endpoint, ens.MainNetAddress)
		if err != nil {
			return nil, err
		}

		addr, err := ens.Resolve(arg.Value)
		if err != nil {
			return nil, err
		}

		address = &object.Address{Value: addr.String()}
	default:
		return nil, fmt.Errorf("not address type found")
	}

	return address, nil
}

func evalInstanceCall(instance *object.Instance, expr ast.Expression, env *object.Environment) object.Object {
	call, ok := expr.(*ast.CallExpression)
	if !ok {
		return newError("it is not a call")
	}

	name, ok := call.Function.(*ast.Identifier)
	if !ok {
		return newError("name not found")
	}

	args := evalExpressions(call.Arguments, env)
	if len(args) == 1 && isError(args[0]) {
		return args[0]
	}

	rpcEndpoint, err := env.GetRPCEndpoint()
	if err != nil {
		return newError(err.Error())
	}

	contract := ethereum.NewContract(instance.ABI, ethereum.NewClient(rpcEndpoint), instance.Address)

	result, err := contract.Call(context.Background(), name.Value, args)
	if err != nil {
		return newError(err.Error())
	}

	if len(result) > 1 {
		return &object.Multiple{Values: result}
	}

	return result[0]
}

// Private functions from here

func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range program.Statements {
		result = Eval(statement, env)

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value
		case *object.Error:
			return result
		}
	}

	return result
}

/*
Code like this: `fn a() {}; let x = a()` fails because the function is expected to return values always
if we introduce a new value object.Empty we have to make the special check for the empty function every
time on the eval function. For now, these type of code will throw a very ugly panic
*/

func evalBlockStatement(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = Eval(statement, env)

		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}

	return FALSE
}

func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right)
	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

func evalBangOperatorExpression(right object.Object) object.Object {
	switch right {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NULL:
		return TRUE
	default:
		return FALSE
	}
}

func evalMinusPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.INTEGER_OBJ {
		return newError("unknown operator: -%s", right.Type())
	}

	value := right.(*object.Integer).Value
	return &object.Integer{Value: big.NewInt(1).Mul(value, big.NewInt(-1))}
}

func evalInfixExpression(operator string, left, right object.Object) object.Object {
	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s", left.Type(), operator, right.Type())
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIntegerInfixExpression(operator string, left, right object.Object) object.Object {
	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Integer).Value

	switch operator {
	case "+":
		return &object.Integer{Value: big.NewInt(1).Add(leftVal, rightVal)}
	case "-":
		return &object.Integer{Value: big.NewInt(1).Sub(leftVal, rightVal)}
	case "*":
		return &object.Integer{Value: big.NewInt(1).Mul(leftVal, rightVal)}
	case "/":
		return &object.Integer{Value: big.NewInt(1).Div(leftVal, rightVal)}
	case "<":
		return nativeBoolToBooleanObject(leftVal.Cmp(rightVal) < 0)
	case ">":
		return nativeBoolToBooleanObject(leftVal.Cmp(rightVal) > 0)
	case "==":
		return nativeBoolToBooleanObject(leftVal.Cmp(rightVal) == 0)
	case "!=":
		return nativeBoolToBooleanObject(leftVal.Cmp(rightVal) != 0)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIfExpression(ie *ast.IfExpression, env *object.Environment) object.Object {
	condition := Eval(ie.Condition, env)

	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	} else {
		return NULL
	}
}

func evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}

	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}

	return newError("identifier not found: " + node.Value)
}

func evalExpressions(exps []ast.Expression, env *object.Environment) []object.Object {
	var result []object.Object

	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

func evalStringInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	if operator != "+" {
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}

	lv := left.(*object.String).Value
	rv := right.(*object.String).Value
	return &object.String{Value: lv + rv}
}

func evalArrayIndexExpression(array, index object.Object) object.Object {
	arrayObject := array.(*object.Array)
	idx := index.(*object.Integer).Value.Int64()
	max := int64(len(arrayObject.Elements) - 1)

	if idx < 0 || idx > max {
		return NULL
	}

	return arrayObject.Elements[idx]
}

func evalHashIndexExpression(hash, index object.Object) object.Object {
	hashObj := hash.(*object.Hash)
	k, ok := index.(object.Hashable)
	if !ok {
		return newError("unusable as hash key: %s", index.Type())
	}

	pair, ok := hashObj.Pairs[k.HashKey()]
	if !ok {
		return NULL
	}

	return pair.Value
}

func evalHashLiteral(
	node *ast.HashLiteral,
	env *object.Environment,
) object.Object {
	pairs := make(map[object.HashKey]object.HashPair)

	for k, v := range node.Pairs {
		key := Eval(k, env)
		if isError(key) {
			return key
		}

		hashKey, ok := key.(object.Hashable)
		if !ok {
			return newError("unusable as hash key: %s", key.Type())
		}

		value := Eval(v, env)
		if isError(value) {
			return value
		}

		hashed := hashKey.HashKey()
		pairs[hashed] = object.HashPair{Key: key, Value: value}
	}

	return &object.Hash{Pairs: pairs}
}

func isTruthy(obj object.Object) bool {
	switch obj {
	case NULL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
	default:
		return true
	}
}

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}

	return false
}

func encodeThisObject(log *types.Log, event object.Event) object.Object {
	pairs := make(map[object.HashKey]object.HashPair)

	encodePair := func(keyName string, value object.Object) {
		key := &object.String{Value: keyName}

		hashed := key.HashKey()
		pairs[hashed] = object.HashPair{Key: key, Value: value}
	}

	encodePair("blocknumber", &object.Integer{Value: big.NewInt(int64(log.BlockNumber))})
	encodePair("blockhash", &object.String{Value: hex.EncodeToString(log.BlockHash.Bytes())})
	encodePair("txhash", &object.String{Value: hex.EncodeToString(log.TxHash.Bytes())})
	encodePair("obj", &object.Instance{
		Name:    "", // dont need the name here
		Address: log.Address,
		ABI:     event.ABI,
	})

	return &object.Hash{Pairs: pairs}
}

// ApplyEvent runs the event. TODO. support this object
func ApplyEvent(event object.Event, args []object.Object, log *types.Log) (object.Object, error) {
	// extend env with args
	env := object.NewEnclosedEnvironment(event.Env)

	if len(event.Parameters) != len(args) {
		return nil, fmt.Errorf("event parameters dont match: %d and %d", len(event.Parameters), args)
	}

	for i, param := range event.Parameters {
		env.Set(param.Identifier.Value, args[i])
	}

	env.Set("this", encodeThisObject(log, event))

	// eval
	evaluated := Eval(event.Body, env)
	return evaluated, nil
}

// ApplyFunction applies a function. NOTE: The env is on the fn object
func ApplyFunction(env *object.Environment, fn object.Object, args []object.Object) object.Object {
	switch fn := fn.(type) {
	case *object.Function:
		extendedEnv := extendFunctionEnv(fn, args)
		evaluated := Eval(fn.Body, extendedEnv)
		return unwrapReturnValue(evaluated)

	case *object.Builtin:
		return fn.Fn(args...)

	case *object.Contract:
		// args 0 has to be an address
		if len(args) != 1 {
			return newError("expected 1 value, found %d", len(args))
		}

		address, err := evalAddress(env, args[0])
		if err != nil {
			return newError(err.Error())
		}

		return &object.Instance{
			Name:    fn.Name,
			Address: address.ToAddress(),
			ABI:     fn.ABI,
		}
	default:
		return newError("not a function: %s", fn.Type())
	}
}

func extendFunctionEnv(fn *object.Function, args []object.Object) *object.Environment {
	env := object.NewEnclosedEnvironment(fn.Env)

	for i, param := range fn.Parameters {
		env.Set(param.Value, args[i])
	}

	return env
}

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}

	return obj
}
