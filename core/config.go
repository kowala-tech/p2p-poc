package core

import "github.com/kowala-tech/kcoin/client/log"

var DefaultConfig = Config{}

type Config struct {
	Logger log.Logger
}
