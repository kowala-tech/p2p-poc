package p2p

import (
	host "github.com/libp2p/go-libp2p-host"
)

type Server struct {
	host host.Host
}

func NewServer(cfg Config) *Server {
	return &Server{}
}
