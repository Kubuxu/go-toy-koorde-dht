package koorde

import (
	"flag"
	"fmt"
	"io/ioutil"
	logger "log"
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

type kRnd struct {
	rand.Rand
}

func (rnd *kRnd) Uint256(u *uint256.Int) {
	buf := make([]byte, KEY_SPACE/8)
	rnd.Read(buf)
	u.SetBytes(buf)
}

func setupNetwork(t testing.TB, rnd *kRnd, cfg koordeConfig, N int) []*node {
	assert := assert.New(t)

	nodes := make([]*node, 0, N)
	tmp := uint256.NewInt()
	for i := 0; i < N; i++ {
		rnd.Uint256(tmp)
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
		did.Lsh(did, cfg.degreeShift) // N * m
		prev := nodes[i]
		curr := prev.succ[0]

		for {
			if betweenEI(did, curr.id, curr.succ[0].id) {
				nodes[i].d[0] = prev
				for j := 1; j < int(cfg.degree); j++ {
					nodes[i].d[j] = nodes[i].d[j-1].succ[0]
				}
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

func setupTest(t testing.TB, cfg koordeConfig, N int, setSeed int64) (*assert.Assertions, *kRnd, []*node) {
	assert := assert.New(t)
	seed := setSeed
	if seed < 0 {
		seed = time.Now().Unix()
		t.Logf("Seed: %d", seed)
	}
	rnd := &kRnd{*rand.New(rand.NewSource(seed))}
	nodes := setupNetwork(t, rnd, cfg, N)

	return assert, rnd, nodes

}

func disableLog() func() {
	oldLog := log
	log = logger.New(ioutil.Discard, "", 0)
	return func() {
		log = oldLog
	}
}

func TestResolveIsConsistent(t *testing.T) {
	defer disableLog()()

	cfg, err := Config(16, 16)
	assert.NoError(t, err, "Config returned error")
	assert, rnd, nodes := setupTest(t, cfg, 1000, -1)

	k := uint256.NewInt()
	rnd.Uint256(k)

	nForK, err := nodes[0].Lookup(k)
	assert.NoError(err, "lookup returned error")

	for _, n := range nodes {
		m, err := n.Lookup(k)
		assert.NoError(err, "lookup returned error")
		assert.Equal(nForK.id, m.id, "different node for key")
	}
}

var long = flag.Bool("long", false, "run long tests")

func TestResolve(t *testing.T) {
	Nmax := 1 << 10
	if *long {
		Nmax = 1 << 15
	}

	for N := 1 << 8; N <= Nmax; N = N << 1 {
		t.Run(fmt.Sprintf("nodes-%d", N), func(t *testing.T) {
			testResolveN(N, t)
		})
	}
}

func testResolveN(N int, t *testing.T) {
	cfg, err := Config(16, 16)
	assert.NoError(t, err, "Config returned error")
	assert, rnd, nodes := setupTest(t, cfg, N, -1)
	//assert, rnd, nodes := setupTest(t, cfg, N, 1550215297)

	k := uint256.NewInt()

	runs := 1000
	if *long {
		runs := 100000
	}
	for i := 0; i < runs; i++ {
		rnd.Uint256(k)
		_, err = nodes[rnd.Intn(N)].Lookup(k)
		assert.NoError(err, "lookup doesn't error")
		//t.Logf("Key [%x] found at [%x]", tmp, n.id)
	}
}

func BenchmarkLookup(b *testing.B) {
	N := 60000
	cfg, err := Config(2, 8)
	assert.NoError(b, err, "Config returned error")
	assert, rnd, nodes := setupTest(b, cfg, N, -1)

	k := uint256.NewInt()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rnd.Uint256(k)
		_, err = nodes[rnd.Intn(N)].Lookup(tmp)
		assert.NoError(err, "lookup doesn't error")
		//b.Logf("Key [%x] found at [%x]", tmp, n.id)
	}
}
