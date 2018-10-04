package core

import (
	"log"

	"github.com/kowala-tech/p2p-poc/p2p"
)

type Service struct {
	log log.Logger
}

func New() *Service {
	return &Service{}
}

func (s *Service) Start(server *p2p.Server) error { return nil }
func (s *Service) Stop() error                    { return nil }
