package p2p

import (
	"github.com/kowala-tech/kcoin/client/log"
	"github.com/kowala-tech/p2p-poc/params"
	crypto "github.com/libp2p/go-libp2p-crypto"
)

var DefaultConfig = Config{
	ListenAddr:     "/ip4/127.0.0.1/tcp/10000",
	IsBootnode:     false,
	BootstrapNodes: params.NetworkBootnodes,
}

type Config struct {
	Identity *crypto.PrivKey

	Name string

	BootstrapNodes []string

	ListenAddr string

	NodeDatabaseDir string

	Logger log.Logger

	IsBootnode bool

	IDGenerationSeed int
}
