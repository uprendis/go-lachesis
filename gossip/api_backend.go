package gossip

import (
	"errors"
)

var ErrNotImplemented = func(name string) error { return errors.New(name + " method is not implemented yet") }

// APIBackend implements ethapi.Backend.
type APIBackend struct {
	extRPCEnabled bool
	svc           *Service
}
