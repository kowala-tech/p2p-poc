package p2p

import (
	"context"
	"math/rand"

	dstore "github.com/ipfs/go-datastore"
	ipfssync "github.com/ipfs/go-datastore/sync"
	pubsub "github.com/libp2p/go-floodsub"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

type Overlay struct {
	*pubsub.PubSub
	host host.Host
}

func NewOverlay(cfg Config) (*Overlay, error) {
	ctx := context.Background()

	privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.New(rand.NewSource(0)))
	if err != nil {
		return nil, err
	}

	host, err := libp2p.New(ctx, libp2p.Identity(privKey), libp2p.ListenAddrStrings(cfg.ListenAddr))
	if err != nil {
		return nil, err
	}

	dhtClient := dht.NewDHT(ctx, host, ipfssync.MutexWrap(dstore.NewMapDatastore()))
	if err := dhtClient.Bootstrap(ctx); err != nil {
		return nil, err
	}

	overlay, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		return nil, err
	}

	return &Overlay{
		host:   host,
		PubSub: overlay,
	}, nil
}
