package basiccheck

import (
	"errors"
	base "github.com/Fantom-foundation/lachesis-base/eventcheck/basiccheck"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"

	"github.com/Fantom-foundation/go-lachesis/inter"
	"github.com/Fantom-foundation/go-lachesis/benchopera"
)

var (
	ErrVersion        = errors.New("event has wrong version")
	ErrTooManyParents = errors.New("event has too many parents")
	ErrZeroTime       = errors.New("event has zero timestamp")
)

type Checker struct {
	base   *base.Checker
	config *benchopera.DagConfig
}

// New validator which performs checks which don't require anything except event
func New(config *benchopera.DagConfig) *Checker {
	return &Checker{
		config: config,
		base:   &base.Checker{},
	}
}

// Validate event
func (v *Checker) Validate(de dag.Event) error {
	if err := v.base.Validate(de); err != nil {
		return err
	}
	e := de.(*inter.Event)
	if e.Version() != 0 {
		return ErrVersion
	}
	if e.CreationTime() <= 0 {
		return ErrZeroTime
	}
	if len(e.Payload()) > v.config.MaxParents {
		return ErrTooManyParents
	}

	return nil
}
