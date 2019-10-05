package evaluator

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/umbracle/go-web3"
	"github.com/umbracle/go-web3/abi"
	"github.com/umbracle/go-web3/jsonrpc"

	"github.com/umbracle/heura/builtin"
	"github.com/umbracle/heura/builtin/ens"
	"github.com/umbracle/heura/heura/ast"
	"github.com/umbracle/heura/heura/encoding"
	"github.com/umbracle/heura/heura/ethereum"
	"github.com/umbracle/heura/heura/object"
	"github.com/umbracle/heura/heura/token"
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

	case *ast.ImportStatement:
		for _, imp := range node.Folders {
			name := imp.(*ast.StringLiteral).Value
			plugin, ok := builtin.BuiltinPlugins[name]
			if !ok {
				return newError("import %s not found", name)
			}
			env.Set(name, plugin())
		}

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

		event := &object.Event{
			Contract:   contract,
			Method:     method,
			Parameters: params,
			Body:       body,
			Env:        env,
		}

		switch obj := c.(type) {
		case *object.Contract:
			event.ABI = obj.ABI
		case *object.Instance:
			if node.Address != nil {
				return newError("cannot have address here")
			}

			event.ABI = obj.ABI
			event.Address = &obj.Address
		}

		m, ok := event.ABI.Events[method]
		if !ok {
			return newError("Method not found for event")
		}

		if len(m.Inputs) != len(params) {
			return newError("Event len different %d and %d", len(m.Inputs), len(params))
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

		objs := []object.Object{}
		args := abi.Arguments{}

		for indx, i := range m.Inputs {
			if i.Indexed {
				objs = append(objs, Eval(params[indx].Value, env))
				args = append(args, i)
			}
		}

		topics := []*web3.Hash{}
		for indx, arg := range args {
			if objs[indx] == nil {
				topics = append(topics, nil)
			} else {
				input, err := encoding.Decode(objs[indx], *arg.Type)
				if err != nil {
					panic(err)
				}
				topic, err := abi.EncodeTopic(arg.Type, input)
				if err != nil {
					panic(err)
				}
				topics = append(topics, &topic)
			}
		}

		fmt.Println("-- topics --")
		fmt.Println(topics)

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

		if node.Name != nil {
			env.Set(node.Name.Value, val)
		}
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
		case *ast.CallExpression:
			ff := left.(*object.Hash)
			call, ok := ff.GetString(obj.Function.String())
			if !ok {
				panic("Y")
			}
			args := evalExpressions(obj.Arguments, env)
			return ApplyFunction(env, call, args)
		}
		return newError("Dot access to hash object requires an identifier")

	case left.Type() == object.ACCOUNT_OBJ:
		return evalAccountCall(left.(*object.Account), index, env)

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
		var ok bool
		address, ok = ens.Resolve(arg).(*object.Address)
		if !ok {
			return nil, fmt.Errorf("failed to resolve ens")
		}

	default:
		return nil, fmt.Errorf("not address type found")
	}

	return address, nil
}

func evalAccountCall(account *object.Account, expr ast.Expression, env *object.Environment) object.Object {
	call, ok := expr.(*ast.CallExpression)
	if !ok {
		return newError("it is not a call")
	}

	name, ok := call.Function.(*ast.Identifier)
	if !ok {
		return newError("name not found")
	}

	rpcEndpoint, err := env.GetRPCEndpoint()
	if err != nil {
		return newError(err.Error())
	}

	c, _ := jsonrpc.NewClient(rpcEndpoint)

	switch name.Value {
	case "nonce":
		nonce, err := c.Eth().GetNonce(account.Addr, web3.Latest)
		if err != nil {
			return newError(err.Error())
		}
		return &object.Integer{Value: big.NewInt(int64(nonce))}

	case "balance":
		balance, err := c.Eth().GetBalance(account.Addr, web3.Latest)
		if err != nil {
			return newError(err.Error())
		}
		return &object.Integer{Value: balance}

	default:
		return newError("Unknown account method: %s", name.Value)
	}
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

	client, _ := jsonrpc.NewClient(rpcEndpoint)

	method, ok := instance.ABI.Methods[name.Value]
	if !ok {
		return newError(fmt.Sprintf("method %s not found", name.Value))
	}

	data, err := encoding.Pack(method.Inputs, args)
	if err != nil {
		return newError(err.Error())
	}

	msg := &web3.CallMsg{
		To:   instance.Address,
		Data: append(method.ID(), data...),
	}

	rawStr, err := client.Eth().Call(msg, web3.Latest)
	if err != nil {
		return newError(err.Error())
	}

	// Decode output
	raw, err := hex.DecodeString(rawStr[2:])
	if err != nil {
		return newError(err.Error())
	}
	result, err := encoding.Unpack(method.Outputs, raw)
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

	if builtin, ok := env.Builtins[node.Value]; ok {
		return builtin
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

func encodeThisObject(log *web3.Log, event object.Event) object.Object {
	pairs := make(map[object.HashKey]object.HashPair)

	encodePair := func(keyName string, value object.Object) {
		key := &object.String{Value: keyName}

		hashed := key.HashKey()
		pairs[hashed] = object.HashPair{Key: key, Value: value}
	}

	encodePair("blocknumber", &object.Integer{Value: big.NewInt(int64(log.BlockNumber))})
	encodePair("blockhash", &object.String{Value: hex.EncodeToString(log.BlockHash[:])})
	encodePair("txhash", &object.String{Value: hex.EncodeToString(log.TransactionHash[:])})
	encodePair("obj", &object.Instance{
		Name:    "", // dont need the name here
		Address: log.Address,
		ABI:     event.ABI,
	})

	return &object.Hash{Pairs: pairs}
}

// ApplyEvent runs the event. TODO. support this object
func ApplyEvent(event object.Event, args []object.Object, log *web3.Log) (object.Object, error) {
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
		extendedEnv, err := extendFunctionEnv(fn, args)
		if err != nil {
			return newError(err.Error())
		}
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

func extendFunctionEnv(fn *object.Function, args []object.Object) (*object.Environment, error) {
	env := object.NewEnclosedEnvironment(fn.Env)

	if len(args) != len(fn.Parameters) {
		return nil, fmt.Errorf("length or parameters not correct")
	}

	for i, param := range fn.Parameters {
		env.Set(param.Value, args[i])
	}
	return env, nil
}

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}

	return obj
}
