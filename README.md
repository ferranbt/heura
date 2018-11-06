# Heura

Ethereum DSL to interact with smart contracts, call and listen for events.

## Install

```
go get github.com/umbracle/heura/heura
```

## Usage

Start the REPL:

```
go run main.go
```

Run a specific file:

```
go run main.go source.hra
```

By default, Heura uses the Infura mainnet nodes to make calls to contracts (https) and listen for events (websockets). Those values can be modified by setting the next flags:

```
go run main.go --endpoint <endpoint> --wsendpoint <websocket endpoint> <file.hra>
```

## Syntax

Heura is an interpreted language. It is still a work in progress and the syntax is expected to change.

### Artifacts

Load artifacts from folder or files

```
artifact (
    "./abis",
    "./artifacts/0x.json"
)
```

or use the builtin ones:

```
artifacts (
    ERC20
)
```

### Calls

Once the artifact is loaded, you can instantiate contracts at specific addresses:

```
let token = ERC20(0x...)
```

or use ENS:

```
let token = ERC20("sometoken.eth")
```

After this, the call functions specified in the ABI are available:

```
token.decimals();
```

Transactions are not supported.

### Events

Listen for ethereum events:

```
on ERC20.Transfer (from, to, value) {
    print (from)
}
```

or filter for an specific address (with either the address or an ENS name):

```
on ERC20("somename.eth").Transfer (from, to, value)
```

Inside the scope of the event callback there is a special 'this' variable with information about the transaction that executed the event: blocknumber, blockhash, transaction hash and an instance of the contract that emit the event.

```
on ERC20.Transfer (from, to, value) {
    print (this.blocknumber)
    print (this.obj.symbol())
} 
```

It is possible to filter by specific topic values in the event:

```
let FROM=0xB9536d30A25466a909563EE4f35fE3c158fc2964

on ERC20.Transfer(from=FROM, to, value) {
```

Note that is only possible with parameters that are indexed on the event.

### Functions

Functions are declared with the keyword 'fn' and can return multiple values.

```
fn some_function() {
    return 1, 2
}

let x, y := some_function()
```
