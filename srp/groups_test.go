package srp

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupEqual(t *testing.T) {
	for _, g := range RFCGroups {
		if !g.Equal(g) {
			t.Error()
		}
	}
}

func TestGroupsContains(t *testing.T) {
	G := new(big.Int).Set(Group4096.G)
	N := new(big.Int).Set(Group4096.N)

	G = G.Add(G, big.NewInt(10))
	g2 := Group{
		G: G,
		N: N,
	}

	var groups Groups
	assert.True(t, groups.Contains(Group4096))

	groups = RFCGroups
	assert.True(t, groups.Contains(Group4096))

	assert.False(t, groups.Contains(g2))
}
