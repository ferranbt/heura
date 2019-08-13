package builtin

import (
	"github.com/umbracle/heura/builtin/account"
	"github.com/umbracle/heura/builtin/ens"
	"github.com/umbracle/heura/builtin/etherscan"
	"github.com/umbracle/heura/heura/object"
)

// Factory is the factory method for the builtin backends
type Factory func(env *object.Environment) object.Object

// BuiltinPlugins are the builtin plugins that you can import
var BuiltinPlugins = map[string]Factory{
	"account":   account.Factory,
	"ens":       ens.Factory,
	"etherscan": etherscan.Factory,
}
