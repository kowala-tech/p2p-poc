package node

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kowala-tech/kcoin/client/log"
	"github.com/kowala-tech/p2p-poc/p2p"
)

const (
	datadirNodeDatabase = "nodes" // Path within the datadir to store the node infos
)

type Config struct {
	Name      string
	UserIdent string
	Version   string `toml:"-"`
	DataDir   string

	P2P p2p.Config

	Logger log.Logger
}

// NodeName returns the devp2p node identifier.
func (c *Config) NodeName() string {
	name := c.name()
	// Backwards compatibility: previous versions used title-cased "Kusd", keep that.
	if name == "kcoin" || name == "kcoin-testnet" {
		name = "Kusd"
	}
	if c.UserIdent != "" {
		name += "/" + c.UserIdent
	}
	if c.Version != "" {
		name += "/v" + c.Version
	}
	name += "/" + runtime.GOOS + "-" + runtime.GOARCH
	name += "/" + runtime.Version()
	return name
}

func (c *Config) name() string {
	if c.Name == "" {
		progname := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
		if progname == "" {
			panic("empty executable name, set Config.Name")
		}
		return progname
	}
	return c.Name
}

// resolvePath resolves path in the instance directory.
func (c *Config) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if c.DataDir == "" {
		return ""
	}

	return filepath.Join(c.instanceDir(), path)
}

// NodeDB returns the path to the discovery node database.
func (c *Config) NodeDB() string {
	if c.DataDir == "" {
		return "" // ephemeral
	}
	return c.resolvePath(datadirNodeDatabase)
}

func (c *Config) instanceDir() string {
	if c.DataDir == "" {
		return ""
	}
	return filepath.Join(c.DataDir, c.name())
}
