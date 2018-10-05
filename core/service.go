package core

import (
	"context"
	"time"

	"github.com/kowala-tech/kcoin/client/event"
	"github.com/kowala-tech/kcoin/client/log"
	"github.com/kowala-tech/p2p-poc/node"
	"github.com/kowala-tech/p2p-poc/p2p"
	pubsub "github.com/libp2p/go-floodsub"
)

type Service struct {
	topic    string
	topicSub *pubsub.Subscription

	globalEvents *event.TypeMux
	overlay      *pubsub.PubSub

	log log.Logger

	doneCh chan struct{}
}

func New(ctx *node.ServiceContext, cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = log.New()
	}

	service := &Service{
		topic:        protocolName,
		globalEvents: ctx.GlobalEvents,
		log:          cfg.Logger,
		doneCh:       make(chan struct{}),
	}

	return service, nil
}

func (s *Service) Start(overlay *p2p.Overlay) error {
	sub, err := overlay.Subscribe(s.topic)
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
				overlay.Publish(s.topic, []byte("teste"))
			case <-s.doneCh:
				return
			}
		}
	}()

	return nil
}

func (s *Service) Stop() error {
	s.topicSub.Cancel()
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
