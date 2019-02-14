package koorde

import (
	"fmt"
	"io"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestBetweenEI(t *testing.T) {
	one := uint256.NewInt().SetUint64(1)
	ten := uint256.NewInt().SetUint64(10)
	big := uint256.NewInt()
	big.Sub(one, ten)

	assert.True(t, betweenEI(ten, one, big), "1 < 10 <= a lot")
	assert.True(t, betweenEI(ten, one, ten), "1 < 10 <= 10")
	assert.False(t, betweenEI(one, one, ten), "1 < 1 <= 10")
	assert.False(t, betweenEI(one, ten, big), "10 < 1 <= 10")

	// wrapping
	assert.True(t, betweenEI(one, big, ten), "a lot < 1 <= 10")
	assert.False(t, betweenEI(big, big, ten), "a lot < 1 <= 10")
	assert.True(t, betweenEI(ten, big, ten), "a lot < 1 <= 10")
}

func TestResolve(t *testing.T) {
	for N := 128; N <= 1<<14; N = N << 1 {
		t.Run(fmt.Sprintf("nodes-%d", N), func(t *testing.T) {
			testResolveN(N, t)
		})
	}
}

func testResolveN(N int, t *testing.T) {
	seed := time.Now().UnixNano()
	t.Logf("Seed: %d", seed)
	rnd := rand.New(rand.NewSource(seed))
	//rnd := rand.New(rand.NewSource(5))
	nodes := make([]*node, 0, N)
	tmp := uint256.NewInt()
	tmpBuf := make([]byte, 256/8)
	for i := 0; i < N; i++ {
		_, err := io.ReadFull(rnd, tmpBuf)
		assert.NoError(t, err, "in random")
		tmp = tmp.SetBytes(tmpBuf)
		nodes = append(nodes, NewNode(tmp))
	}
	sort.Slice(nodes, func(a, b int) bool {
		return nodes[a].id.Lt(nodes[b].id)
	})

	// create successor paths
	for i := 0; i < N; i++ {
		nodes[i].succ = nodes[(i+1)%N]
	}
	// create de Burjin paths
	for i := 0; i < N; i++ {
		did := nodes[i].id.Clone()
		did.Lsh(did, 1) // 2m
		prev := nodes[i]
		curr := prev.succ

		for {
			if betweenEI(did, curr.id, curr.succ.id) {
				nodes[i].d = prev
				break
			}
			prev, curr = curr, curr.succ
		}
	}
	for i := 0; i < N; i++ {
		assert.NotNil(t, nodes[i].succ, "successor nil at %d", i)
		assert.NotNil(t, nodes[i].d, "d nil at %d", i)
	}

	runs := 10000
	for i := 0; i < runs; i++ {
		_, err := io.ReadFull(rnd, tmpBuf)
		assert.NoError(t, err, "in random")
		tmp = tmp.SetBytes(tmpBuf)
		//tmp.Or(tmp, neg).Rsh(tmp, 16)

		_, err = nodes[rnd.Intn(N)].Lookup(tmp)
		assert.NoError(t, err, "lookup doesn't error")
		//t.Logf("Key [%x] found at [%x]", tmp, n.id)
	}

}

func BenchmarkLookup(b *testing.B) {
	N := 60000
	rnd := rand.New(rand.NewSource(1))
	nodes := make([]*node, 0, N)
	tmp := uint256.NewInt()
	tmpBuf := make([]byte, 256/8)
	for i := 0; i < N; i++ {
		_, err := io.ReadFull(rnd, tmpBuf)
		assert.NoError(b, err, "in random")
		tmp = tmp.SetBytes(tmpBuf)
		nodes = append(nodes, NewNode(tmp))
	}
	sort.Slice(nodes, func(a, b int) bool {
		return nodes[a].id.Lt(nodes[b].id)
	})

	for i := 0; i < N; i++ {
		nodes[i].succ = nodes[(i+1)%N]
	}
	// create de Burjin paths
	for i := 0; i < N; i++ {
		did := nodes[i].id.Clone()
		did.Lsh(did, 1) // 2m

		prev := nodes[((i*4)/10)%N]
		curr := prev.succ

		for {
			if betweenEI(did, curr.id, curr.succ.id) {
				nodes[i].d = prev
				break
			}
			prev, curr = curr, curr.succ
		}
	}
	for i := 0; i < N; i++ {
		assert.NotNil(b, nodes[i].succ, "successor nil at %d", i)
		assert.NotNil(b, nodes[i].d, "d nil at %d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := io.ReadFull(rnd, tmpBuf)
		assert.NoError(b, err, "in random")
		tmp = tmp.SetBytes(tmpBuf)

		_, err = nodes[rnd.Intn(N)].Lookup(tmp)
		assert.NoError(b, err, "lookup doesn't error")
		//b.Logf("Key [%x] found at [%x]", tmp, n.id)

	}
}
