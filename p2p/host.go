package p2p

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"

	dstore "github.com/ipfs/go-datastore"
	ipfssync "github.com/ipfs/go-datastore/sync"
	"github.com/kowala-tech/kcoin/client/log"
	pubsub "github.com/libp2p/go-floodsub"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	maddr "github.com/multiformats/go-multiaddr"
)

type Host struct {
	*pubsub.PubSub
	host.Host

	logger log.Logger
}

func NewHost(cfg Config) (*Host, error) {
	if cfg.Logger == nil {
		cfg.Logger = log.New()
	}

	ctx := context.Background()

	privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.New(rand.NewSource(0)))
	if err != nil {
		return nil, err
	}

	host, err := libp2p.New(ctx, libp2p.Identity(privKey), libp2p.ListenAddrStrings(cfg.ListenAddr))
	if err != nil {
		return nil, err
	}

	dht := dht.NewDHT(ctx, host, ipfssync.MutexWrap(dstore.NewMapDatastore()))

	if len(cfg.BootstrapNodes) > 0 {
		bootnodes := make([]pstore.PeerInfo, len(cfg.BootstrapNodes))
		for i, addr := range cfg.BootstrapNodes {
			multiAddr := maddr.StringCast(addr)
			p, err := pstore.InfoFromP2pAddr(multiAddr)
			if err != nil {
				return nil, err
			}
			bootnodes[i] = *p
		}
		routedHost := rhost.Wrap(host, dht)
		bootstrapConnect(ctx, routedHost, bootnodes, cfg.Logger)
	}

	if cfg.IsBootnode {
		if err := dht.Bootstrap(ctx); err != nil {
			return nil, err
		}
	}

	gossip, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		return nil, err
	}

	return &Host{
		Host:   host,
		PubSub: gossip,
		logger: cfg.Logger,
	}, nil
}

func bootstrapConnect(ctx context.Context, ph host.Host, peers []pstore.PeerInfo, log log.Logger) error {
	if len(peers) < 1 {
		return errors.New("not enough bootstrap peers")
	}

	errs := make(chan error, len(peers))
	var wg sync.WaitGroup
	for _, p := range peers {

		// performed asynchronously because when performed synchronously, if
		// one `Connect` call hangs, subsequent calls are more likely to
		// fail/abort due to an expiring context.
		// Also, performed asynchronously for dial speed.

		wg.Add(1)
		go func(p pstore.PeerInfo) {
			defer wg.Done()
			defer log.Debug("bootstrapDial", ph.ID(), p.ID)
			log.Debug("from", ph.ID(), "bootstrapping to", p.ID)

			ph.Peerstore().AddAddrs(p.ID, p.Addrs, pstore.PermanentAddrTTL)
			if err := ph.Connect(ctx, p); err != nil {
				log.Error("Failed to bootstrap with", p.ID, err)
				errs <- err
				return
			}
			log.Debug("bootstrapDialSuccess", p.ID)
			log.Info("Bootstrapped with", p.ID)
		}(p)
	}
	wg.Wait()

	// our failure condition is when no connection attempt succeeded.
	// So drain the errs channel, counting the results.
	close(errs)
	count := 0
	var err error
	for err = range errs {
		if err != nil {
			count++
		}
	}
	if count == len(peers) {
		return fmt.Errorf("Failed to bootstrap. %s", err)
	}
	return nil
}
