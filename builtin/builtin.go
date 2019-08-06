package builtin

import (
	"github.com/umbracle/heura/builtin/ens"
	"github.com/umbracle/heura/builtin/etherscan"
	"github.com/umbracle/heura/heura/object"
)

// Factory is the factory method for the builtin backends
type Factory func() object.Object

// BuiltinPlugins are the builtin plugins that you can import
var BuiltinPlugins = map[string]Factory{
	"ens":       ens.Factory,
	"etherscan": etherscan.Factory,
}
