package eventcheck

import (
	"github.com/Fantom-foundation/go-lachesis/eventcheck/basiccheck"
	"github.com/Fantom-foundation/go-lachesis/eventcheck/epochcheck"
	"github.com/Fantom-foundation/go-lachesis/eventcheck/gaspowercheck"
	"github.com/Fantom-foundation/go-lachesis/eventcheck/heavycheck"
	"github.com/Fantom-foundation/go-lachesis/eventcheck/parentscheck"
	"github.com/Fantom-foundation/go-lachesis/inter/dag"
)

// Checkers is collection of all the checkers
type Checkers struct {
	Basiccheck    *basiccheck.Checker
	Epochcheck    *epochcheck.Checker
	Parentscheck  *parentscheck.Checker
	Gaspowercheck *gaspowercheck.Checker
	Heavycheck    *heavycheck.Checker
}

// Validate runs all the checks except Lachesis-related
func (v *Checkers) Validate(e dag.Event, parents []dag.Event) error {
	if err := v.Basiccheck.Validate(e); err != nil {
		return err
	}
	if err := v.Epochcheck.Validate(e); err != nil {
		return err
	}
	if err := v.Parentscheck.Validate(e, parents); err != nil {
		return err
	}
	var selfParent dag.Event
	if e.SelfParent() != nil {
		selfParent = parents[0]
	}
	if err := v.Gaspowercheck.Validate(e, selfParent); err != nil {
		return err
	}
	if err := v.Heavycheck.Validate(e); err != nil {
		return err
	}
	return nil
}
