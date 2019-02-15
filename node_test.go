package koorde

import (
	"flag"
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
	assert := assert.New(t)
	one := uint256.NewInt().SetUint64(1)
	ten := uint256.NewInt().SetUint64(10)
	big := uint256.NewInt()
	big.Sub(one, ten)

	assert.True(betweenEI(ten, one, big), "1 < 10 <= a lot")
	assert.True(betweenEI(ten, one, ten), "1 < 10 <= 10")
	assert.False(betweenEI(one, one, ten), "1 < 1 <= 10")
	assert.False(betweenEI(one, ten, big), "10 < 1 <= 10")

	// wrapping
	assert.True(betweenEI(one, big, ten), "a lot < 1 <= 10")
	assert.False(betweenEI(big, big, ten), "a lot < 1 <= 10")
	assert.True(betweenEI(ten, big, ten), "a lot < 1 <= 10")
}

func setupNetwork(t testing.TB, rnd *rand.Rand, cfg kordeConfig, N int) []*node {
	assert := assert.New(t)

	nodes := make([]*node, 0, N)
	tmp := uint256.NewInt()
	tmpBuf := make([]byte, 256/8)
	for i := 0; i < N; i++ {
		_, err := io.ReadFull(rnd, tmpBuf)
		assert.NoError(err, "in random")
		tmp = tmp.SetBytes(tmpBuf)
		nodes = append(nodes, NewNode(cfg, tmp))
	}
	sort.Slice(nodes, func(a, b int) bool {
		return nodes[a].id.Lt(nodes[b].id)
	})

	// create successor paths
	for i := 0; i < N; i++ {
		for j := 0; j < cfg.backupSuccessors; j++ {
			nodes[i].succ[j] = nodes[(i+j+1)%N]
		}
	}
	// create de Burjin paths
	for i := 0; i < N; i++ {
		did := nodes[i].id.Clone()
		did.Lsh(did, 1) // 2m
		prev := nodes[i]
		curr := prev.succ[0]

		for {
			if betweenEI(did, curr.id, curr.succ[0].id) {
				nodes[i].d = prev
				break
			}
			prev, curr = curr, curr.succ[0]
		}
	}
	for i := 0; i < N; i++ {
		assert.NotContains(nodes[i].succ, nil, "succesor nil at %d", i)
		assert.NotNil(nodes[i].d, "d nil at %d", i)
	}

	return nodes

}

func setupTest(t testing.TB, cfg kordeConfig, N int, setSeed int64) (*assert.Assertions, *rand.Rand, []*node) {
	assert := assert.New(t)
	seed := setSeed
	if seed < 0 {
		seed = time.Now().Unix()
		t.Logf("Seed: %d", seed)
	}
	rnd := rand.New(rand.NewSource(seed))
	nodes := setupNetwork(t, rnd, cfg, N)

	return assert, rnd, nodes

}

var long = flag.Bool("long", false, "run long tests")

func TestResolve(t *testing.T) {
	Nmax := 1 << 12
	if *long {
		Nmax = 1 << 15
	}

	for N := 1 << 7; N <= Nmax; N = N << 1 {
		t.Run(fmt.Sprintf("nodes-%d", N), func(t *testing.T) {
			testResolveN(N, t)
		})
	}
}

func testResolveN(N int, t *testing.T) {
	cfg := kordeConfig{backupSuccessors: 8}
	assert, rnd, nodes := setupTest(t, cfg, N, -1)

	tmp := uint256.NewInt()
	tmpBuf := make([]byte, 256/8)

	runs := 10000
	for i := 0; i < runs; i++ {
		_, err := io.ReadFull(rnd, tmpBuf)
		assert.NoError(err, "in random")
		tmp = tmp.SetBytes(tmpBuf)
		//tmp.Or(tmp, neg).Rsh(tmp, 16)

		_, err = nodes[rnd.Intn(N)].Lookup(tmp)
		assert.NoError(err, "lookup doesn't error")
		//t.Logf("Key [%x] found at [%x]", tmp, n.id)
	}

}

func BenchmarkLookup(b *testing.B) {
	N := 60000
	cfg := kordeConfig{backupSuccessors: 8}
	assert, rnd, nodes := setupTest(b, cfg, N, -1)

	tmp := uint256.NewInt()
	tmpBuf := make([]byte, 256/8)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := io.ReadFull(rnd, tmpBuf)
		assert.NoError(err, "in random")
		tmp = tmp.SetBytes(tmpBuf)

		_, err = nodes[rnd.Intn(N)].Lookup(tmp)
		assert.NoError(err, "lookup doesn't error")
		//b.Logf("Key [%x] found at [%x]", tmp, n.id)

	}
}
