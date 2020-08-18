package inter

import (
	"fmt"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

// GasPowerLeft is long-term gas power left and short-term gas power left
type GasPowerLeft struct {
	Gas [2]uint64
}

// Add add to all gas power lefts
func (g *GasPowerLeft) Add(diff uint64) {
	for i := range g.Gas {
		g.Gas[i] += diff
	}
}

// Min returns minimum within long-term gas power left and short-term gas power left
func (g *GasPowerLeft) Min() uint64 {
	min := g.Gas[0]
	for _, gas := range g.Gas {
		if min > gas {
			min = gas
		}
	}
	return min
}

// Max returns maximum within long-term gas power left and short-term gas power left
func (g *GasPowerLeft) Max() uint64 {
	max := g.Gas[0]
	for _, gas := range g.Gas {
		if max < gas {
			max = gas
		}
	}
	return max
}

// Sub subtracts from all gas power lefts
func (g *GasPowerLeft) Sub(diff uint64) *GasPowerLeft {
	for i := range g.Gas {
		g.Gas[i] -= diff
	}
	return g
}

// String returns string representation.
func (g *GasPowerLeft) String() string {
	return fmt.Sprintf("{short=%d, long=%d}", g.Gas[idx.ShortTermGas], g.Gas[idx.LongTermGas])
}
