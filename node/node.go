package node

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"syscall"

	"github.com/kowala-tech/kcoin/client/event"
	"github.com/kowala-tech/kcoin/client/log"
	"github.com/kowala-tech/p2p-poc/p2p"
	"github.com/prometheus/prometheus/util/flock"
)

// Node represents a block chain node.
type Node struct {
	cfg Config

	serverCfg p2p.Config
	overlay   *p2p.Overlay

	lock         sync.RWMutex
	serviceFuncs []ServiceConstructor     // Service constructors (in dependency order)
	services     map[reflect.Type]Service // Currently running services

	globalEvents *event.TypeMux

	instanceDirLock flock.Releaser // prevents concurrent use of instance directory

	log log.Logger

	stop chan struct{} // Channel to wait for termination notifications
}

// New create a new Kowala node, ready for service registration.
func New(ctx context.Context, cfg Config) *Node {
	if cfg.Logger == nil {
		cfg.Logger = log.New()
	}

	return &Node{
		cfg:          cfg,
		serviceFuncs: []ServiceConstructor{},
		globalEvents: new(event.TypeMux),
		log:          cfg.Logger,
	}
}

// Register injects a new service into the node's stack. The service created by
// the passed constructor must be unique in its type with regard to sibling ones.
func (n *Node) Register(constructor ServiceConstructor) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.overlay != nil {
		return ErrNodeRunning
	}
	n.serviceFuncs = append(n.serviceFuncs, constructor)
	return nil
}

// Start create a live P2P node and starts running it.
func (n *Node) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	// Short circuit if the node's already running
	if n.overlay != nil {
		return ErrNodeRunning
	}
	if err := n.openDataDir(); err != nil {
		return err
	}

	// Initialize the p2p server. This creates the node key and
	// discovery databases.
	n.serverCfg = n.cfg.P2P
	//n.serverCfg.PrivateKey = n.config.NodeKey()
	n.serverCfg.Name = n.cfg.NodeName()
	n.serverCfg.Log = n.log

	if n.serverCfg.NodeDatabaseDir == "" {
		n.serverCfg.NodeDatabaseDir = n.cfg.NodeDB()
	}
	overlay, err := p2p.NewOverlay(n.serverCfg)
	if err != nil {
		return err
	}

	n.log.Info("Starting peer-to-peer node", "instance", n.serverCfg.Name)

	// Otherwise copy and specialize the P2P configuration
	services := make(map[reflect.Type]Service)
	for _, constructor := range n.serviceFuncs {
		// Create a new context for the particular service
		ctx := &ServiceContext{
			cfg:          &n.cfg,
			services:     make(map[reflect.Type]Service),
			GlobalEvents: n.globalEvents,
		}
		for kind, s := range services { // copy needed for threaded access
			ctx.services[kind] = s
		}
		// Construct and save the service
		service, err := constructor(ctx)
		if err != nil {
			return err
		}
		kind := reflect.TypeOf(service)
		if _, exists := services[kind]; exists {
			return &DuplicateServiceError{Kind: kind}
		}
		services[kind] = service
	}

	/*
		if err := srv.Start(); err != nil {
			return convertFileLockError(err)
		}
	*/
	// Start each of the services
	started := []reflect.Type{}
	for kind, service := range services {
		// Start the next service, stopping all previous upon failure
		if err := service.Start(overlay); err != nil {
			for _, kind := range started {
				services[kind].Stop()
			}

			// @TODO
			//srv.Stop()

			return err
		}
		// Mark the service started for potential cleanup
		started = append(started, kind)
	}

	// Finish initializing the startup
	n.services = services
	n.overlay = overlay

	return nil
}

func (n *Node) openDataDir() error {
	if n.cfg.DataDir == "" {
		return nil // ephemeral
	}

	instdir := filepath.Join(n.cfg.DataDir, n.cfg.name())
	if err := os.MkdirAll(instdir, 0700); err != nil {
		return err
	}
	// Lock the instance directory to prevent concurrent use by another instance as well as
	// accidental use of the instance directory as a database.
	release, _, err := flock.New(filepath.Join(instdir, "LOCK"))
	if err != nil {
		return convertFileLockError(err)
	}
	n.instanceDirLock = release
	return nil
}

// Stop terminates a running node along with all it's services. In the node was
// not started, an error is returned.
func (n *Node) Stop() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	// Short circuit if the node's not running
	if n.overlay == nil {
		return ErrNodeStopped
	}

	// Terminate the API, services and the p2p server.
	//n.stopWS()
	//n.stopHTTP()
	//n.stopIPC()
	//n.rpcAPIs = nil
	failure := &StopError{
		Services: make(map[reflect.Type]error),
	}
	for kind, service := range n.services {
		if err := service.Stop(); err != nil {
			failure.Services[kind] = err
		}
	}
	//n.server.Stop()
	n.services = nil
	n.overlay = nil

	// Release instance directory lock.
	if n.instanceDirLock != nil {
		if err := n.instanceDirLock.Release(); err != nil {
			n.log.Error("Can't release datadir lock", "err", err)
		}
		n.instanceDirLock = nil
	}

	// unblock n.Wait
	close(n.stop)

	// Remove the keystore if it was created ephemerally.
	//var keystoreErr error
	//if n.ephemeralKeystore != "" {
	//	keystoreErr = os.RemoveAll(n.ephemeralKeystore)
	//}

	if len(failure.Services) > 0 {
		return failure
	}
	/*
		if keystoreErr != nil {
			return keystoreErr
		}
	*/
	return nil
}

func convertFileLockError(err error) error {
	if errno, ok := err.(syscall.Errno); ok && datadirInUseErrnos[uint(errno)] {
		return ErrDatadirUsed
	}
	return err
}

// Wait blocks the thread until the node is stopped. If the node is not running
// at the time of invocation, the method immediately returns.
func (n *Node) Wait() {
	n.lock.RLock()
	if n.overlay == nil {
		n.lock.RUnlock()
		return
	}
	stop := n.stop
	n.lock.RUnlock()

	<-stop
}
