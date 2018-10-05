package node

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrDatadirUsed    = errors.New("datadir already used by another process")
	ErrNodeRunning    = errors.New("node already running")
	ErrNodeStopped    = errors.New("node not started")
	ErrServiceUnknown = errors.New("unknown service")

	datadirInUseErrnos = map[uint]bool{11: true, 32: true, 35: true}
)

// DuplicateServiceError is returned during Node startup if a registered service
// constructor returns a service of the same type that was already started.
type DuplicateServiceError struct {
	Kind reflect.Type
}

// Error generates a textual representation of the duplicate service error.
func (e *DuplicateServiceError) Error() string {
	return fmt.Sprintf("duplicate service: %v", e.Kind)
}

// StopError is returned if a Node fails to stop either any of its registered
// services or itself.
type StopError struct {
	Server   error
	Services map[reflect.Type]error
}

// Error generates a textual representation of the stop error.
func (e *StopError) Error() string {
	return fmt.Sprintf("server: %v, services: %v", e.Server, e.Services)
}
