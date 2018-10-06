package p2p

import (
	"github.com/kowala-tech/kcoin/client/log"
	crypto "github.com/libp2p/go-libp2p-crypto"
)

var DefaultConfig = Config{
	MaxPeers: 15,
}

type Config struct {
	Identity *crypto.PrivKey

	// MaxPeers is the maximum number of peers that can be
	// connected. It must be greater than zero.
	MaxPeers int

	// Name sets the node name of this host.
	// Use common.MakeName to create a name that follows existing conventions.
	Name string

	BootstrapNodes []string

	ListenAddr string

	NodeDatabaseDir string

	Logger log.Logger

	IsBootnode bool
}
