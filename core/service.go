package core

import (
	"context"
	"sync"
	"time"

	"github.com/kowala-tech/kcoin/client/event"
	"github.com/kowala-tech/kcoin/client/log"
	"github.com/kowala-tech/p2p-poc/node"
	"github.com/kowala-tech/p2p-poc/p2p"
	pubsub "github.com/libp2p/go-floodsub"
	"github.com/libp2p/go-libp2p-net"
	inet "github.com/libp2p/go-libp2p-net"
	maddr "github.com/multiformats/go-multiaddr"
)

type Service struct {
	topic    string
	topicSub *pubsub.Subscription

	notifiee *net.NotifyBundle
	peers    *peerSet

	host p2p.Host

	globalEvents *event.TypeMux

	logger log.Logger

	wg sync.WaitGroup

	doneCh chan struct{}
}

func New(ctx *node.ServiceContext, cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = log.New()
	}

	service := &Service{
		topic:        protocolName,
		peers:        newPeerSet(),
		globalEvents: ctx.GlobalEvents,
		logger:       cfg.Logger,
		doneCh:       make(chan struct{}),
	}
	service.notifiee = &net.NotifyBundle{
		ListenF:      service.peerConnected,
		ListenCloseF: service.peerDisconnected,
	}

	return service, nil
}

func (s *Service) Start(host *p2p.Host) error {
	host.Network().Notify(s.notifiee)

	sub, err := host.Subscribe(s.topic)
	if err != nil {
		return err
	}

	go handleSubscription(sub)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.logger.Info("Sending Message")
				host.Publish(s.topic, []byte("teste"))
			case <-s.doneCh:
				return
			}
		}
	}()

	return nil
}

func handleSubscription(sub *pubsub.Subscription) {
	for {
		_, err := sub.Next(context.Background())
		if err != nil {
			return
		}
	}
}

func (s *Service) Stop() error {
	//host.Network().StopNotify(s.notifiee)
	s.topicSub.Cancel()
	return nil
}

func (s *Service) peerConnected(network inet.Network, addr maddr.Multiaddr) {
	newPeer(s.host, addr)
	s.wg.Add(1)
	defer s.wg.Done()
}

func (s *Service) peerDisconnected(network inet.Network, addr maddr.Multiaddr) {
	// @TODO
}

func (s *Service) handle(p *peer) error {
	/*
		// Ignore maxPeers if this is a trusted peer
		if pm.peers.Len() >= pm.maxPeers && !p.Peer.Info().Network.Trusted {
			return p2p.DiscTooManyPeers
		}
		p.Log().Debug("Kowala peer connected", "name", p.Name())
	*/
	s.logger.Debug("Kowala peer connected", "name", p.ID)

	/*
		// Execute the Kowala handshake
		var (
			genesis     = pm.blockchain.Genesis()
			head        = pm.blockchain.CurrentHeader()
			hash        = head.Hash()
			blockNumber = head.Number
		)
		if err := p.Handshake(pm.networkID, blockNumber, hash, genesis.Hash()); err != nil {
			p.Log().Debug("Kowala handshake failed", "err", err)
			return err
		}
		if rw, ok := p.rw.(*meteredMsgReadWriter); ok {
			rw.Init(p.version)
		}
	*/

	// Register the peer locally
	if err := s.peers.Register(p); err != nil {
		s.logger.Error("Kowala peer registration failed", "err", err)
		return err
	}
	defer s.removePeer(p.ID)

	/*
		// Register the peer in the downloader. If the downloader considers it banned, we disconnect
		if err := pm.downloader.RegisterPeer(p.id, p.version, p); err != nil {
			return err
		}

		// Propagate existing transactions. new transactions appearing
		// after this will be sent via broadcasts.
		pm.syncTransactions(p)
	*/

	// main loop. handle incoming messages.
	for {
		if err := s.handleMsg(p); err != nil {
			s.logger.Debug("Kowala message handling failed", "err", err)
			return err
		}
	}
}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (s *Service) handleMsg(p *peer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	// rawMsg, err := p.rw.ReadBytes('\n')
	_, err := p.rw.ReadBytes('\n')
	if err != nil {
		return err
	}

	/*
		if len(rawMsg) > MaxMsgSize {
			return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, protocol.Constants.MaxMsgSize)
		}

		defer msg.Discard()
	*/

	switch {
	}

	/*

		str, err := rw.ReadString('\n')
			if err != nil {
				log.Fatal(err)
			}

			if str == "" {
				return
			}
			if str != "\n" {
				chain := make([]Block, 0)
				if err := json.Unmarshal([]byte(str), &chain); err != nil {
					log.Fatal(err)
				}
	*/

	return nil
}

func (s *Service) removePeer(id string) {
	// Short circuit if the peer was already removed
	peer := s.peers.Peer(id)
	if peer == nil {
		return
	}
	log.Debug("Removing Kowala peer", "peer", id)

	// Unregister the peer from the downloader and Kowala peer set
	//pm.downloader.UnregisterPeer(id)
	if err := s.peers.Unregister(id); err != nil {
		log.Error("Peer removal failed", "peer", id, "err", err)
	}

	/*
		// Hard disconnect at the networking layer
		if peer != nil {
			peer.Peer.Disconnect(p2p.DiscUselessPeer)
		}
	*/
}
