package core

import (
	"bufio"
	"context"
	"errors"
	"log"
	"math/big"
	"sync"

	"github.com/kowala-tech/p2p-poc/p2p"
	libpeer "github.com/libp2p/go-libp2p-peer"
	maddr "github.com/multiformats/go-multiaddr"
)

var (
	errClosed            = errors.New("peer set is closed")
	errAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
)

type peer struct {
	ID string

	rw *bufio.ReadWriter

	blockNumber *big.Int
	doneCh      chan struct{}
}

func newPeer(host p2p.Host, addr maddr.Multiaddr) *peer {
	pid, err := addr.ValueForProtocol(maddr.P_IPFS)
	if err != nil {
		log.Fatalln(err)
	}

	peerID, err := libpeer.IDB58Decode(pid)
	if err != nil {
		log.Fatalln(err)
	}

	s, err := host.NewStream(context.Background(), peerID, "/core/1.0.0")
	if err != nil {
		log.Fatalln(err)
	}

	// Create a buffered stream so that read and writes are non blocking.
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	return &peer{
		rw:     rw,
		doneCh: make(chan struct{}),
	}
}

/*
// broadcast is a write loop that multiplexes block propagations, announcements
// and transaction broadcasts into the remote peer. The goal is to have an async
// writer that does not lock up node internals.
func (p *peer) broadcast() {
	for {
		select {
		case txs := <-p.queuedTxs:
			if err := p.SendTransactions(txs); err != nil {
				return
			}
			p.Log().Trace("Broadcast transactions", "count", len(txs))

		case prop := <-p.queuedProps:
			if err := p.SendNewBlock(prop.block); err != nil {
				return
			}
			p.Log().Trace("Propagated block", "number", prop.block.Number(), "hash", prop.block.Hash())

		case block := <-p.queuedAnns:
			if err := p.SendNewBlockHashes([]common.Hash{block.Hash()}, []uint64{block.NumberU64()}); err != nil {
				return
			}
			p.Log().Trace("Announced block", "number", block.Number(), "hash", block.Hash())

		case <-p.term:
			return
		}
	}
}
*/

// peerSet represents the collection of active peers currently participating in
// the Ethereum sub-protocol.
type peerSet struct {
	peers  map[string]*peer
	lock   sync.RWMutex
	closed bool
}

// newPeerSet creates a new peer set to track the active participants.
func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peer),
	}
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known. If a new peer it registered, its broadcast loop is also
// started.
func (ps *peerSet) Register(p *peer) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if ps.closed {
		return errClosed
	}
	if _, ok := ps.peers[p.ID]; ok {
		return errAlreadyRegistered
	}
	ps.peers[p.ID] = p
	//go p.broadcast()

	return nil
}

// Unregister removes a remote peer from the active set, disabling any further
// actions to/from that particular entity.
func (ps *peerSet) Unregister(ID string) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	_, ok := ps.peers[ID]
	if !ok {
		return errNotRegistered
	}
	delete(ps.peers, ID)
	//p.close()

	return nil
}

// Peer retrieves the registered peer with the given id.
func (ps *peerSet) Peer(ID string) *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[ID]
}

// Len returns if the current number of peers in the set.
func (ps *peerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}
