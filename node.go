package koorde

import (
	logger "log"
	"os"

	"github.com/holiman/uint256"
)

var (
	one = uint256.NewInt().SetOne()
	neg = uint256.NewInt().Not()
)

var log = logger.New(os.Stdout, "koorde: ", 0)

type node struct {
	cfg  koordeConfig
	id   *uint256.Int
	succ []*node
	d    []*node
}

func NewNode(config koordeConfig, id *uint256.Int) *node {
	n := new(node)
	n.cfg = config
	n.id = id.Clone()
	n.succ = make([]*node, n.cfg.backupSuccessors)
	n.d = make([]*node, n.cfg.degree)
	return n
}

// selects best starting i
// i should fulfill betweenEI(i, m.id, m.succ.id) but we want to swap low bits
// of i with high bits of k
// this still can be better (bit overlap, fitting (md.id, succ.id] better)
// Returns (kshift, i)
func (m node) bestStart(k *uint256.Int) (*uint256.Int, *uint256.Int) {
	i := uint256.NewInt()
	tmp := uint256.NewInt()
	//tmp.Sub(m.succ.id, m.id)
	//log.Printf("Id %064x\n", m.id.ToBig())
	//log.Printf("Sc %064x\n", m.succ.id.ToBig())

	// shift bits by degreeShift factor every time
	for j := KEY_SPACE - m.cfg.degreeShift; j >= 0; j = j - m.cfg.degreeShift {
		// mask out low bits of m.id
		i.Lsh(neg, uint(j)).And(i, m.id)
		// add one to the lowest bit of what is left of m.id
		tmp.Lsh(one, uint(j))
		i.Add(i, tmp)

		// put the key into low bits of i
		tmp.Rsh(k, uint(KEY_SPACE-j))
		i.Or(i, tmp)

		//log.Printf("i %064x\n", i.ToBig())
		//log.Printf("t %064x\n", tmp.ToBig())

		//log.Printf("r %064x\n", i.ToBig())

		// check if it still on path
		if betweenEI(i, m.id, m.succ[0].id) {
			log.Printf("Match at %d", KEY_SPACE-j)
			// return is kshift where parts in i are shifted out
			// and crafted i within our range
			return k.Lsh(k, uint(j)), i
		}
	}
	log.Printf("No match")
	i.Add(m.id.Clone(), one)
	return k, i
}

// Clean inteface function
func (m node) Lookup(k *uint256.Int) (*node, error) {
	log.Printf("[%064x] Looking up: %064x\n", m.id.ToBig(), k.ToBig())
	kshift, i := m.bestStart(k.Clone())
	return m.lookup(k, kshift, i)
}

// Performs lookup of node for key  k
// kshift is being shifted into virtual node i one step at the time
// i is virtual node on the path from starting node to k
func (m node) lookup(k, kshift, i *uint256.Int) (*node, error) {
	//log.Printf("Got: %064x @ %064x", i.ToBig(), m.id.ToBig())
	//log.Printf("Ksh: %064x", kshift.ToBig())

	// check if we are responsible for k
	if betweenEI(k, m.id, m.succ[0].id) {
		//log.Printf("[%x] Found %x\n", m.id, k)
		log.Printf("Found")
		return m.succ[0], nil
	}

	// check if one of our successors is responsible for k
	for i := 0; i < m.cfg.backupSuccessors-1; i++ {
		if betweenEI(k, m.succ[i].id, m.succ[i+1].id) {
			//log.Printf("[%x] Found %x\n", m.id, k)
			log.Printf("Found (%d)", i)
			return m.succ[i+1], nil
		}
	}

	// check if we are responsibe for path i -> k
	if betweenEI(i, m.id, m.succ[0].id) {
		//log.Printf("[%x] Forwarding %x\n", m.id, k)
		// forward the request
		// degreeShift highest bits of kshift
		topBits := uint256.NewInt().Rsh(kshift, KEY_SPACE-m.cfg.degreeShift)
		i = i.Lsh(i, m.cfg.degreeShift).Or(i, topBits) // i = (i << degreeShift) | topBits
		kshift = kshift.Lsh(kshift, m.cfg.degreeShift) // kshift = kshift << degreeShift

		// look for the best forwarder
		for j := 0; j < int(m.cfg.degree); j++ {
			if betweenEI(i, m.d[j].id, m.d[j].succ[0].id) {
				log.Printf("Forwarding (%d)", j)
				return m.d[j].lookup(k, kshift, i)
			}
		}
		// forward to furthers forwarder for correcting
		log.Printf("Forwarding far (%d)", m.cfg.degree-1)
		return m.d[int(m.cfg.degree-1)].lookup(k, kshift, i)
	} else {
		// correct if we are not responsibe for the path

		//log.Printf("[%x] Correcting %x\n", m.id, k)
		//log.Printf("Correcting delta %064x\n", tmp.ToBig())
		//tmp := uint256.NewInt()
		//tmp.Sub(i, m.succ[0].id)
		for j := m.cfg.backupSuccessors - 1; j >= 0; j-- {
			if betweenEI(i, m.succ[j].id, m.id) {
				log.Printf("Correcting (%d)", j+1)
				return m.succ[j].lookup(k, kshift, i)
			}
		}
	}
	panic("we shouldn't end up here")
}

// checks if x is in interval (min, max] mod 2^256
func betweenEI(x, start, end *uint256.Int) bool {
	reversed := false
	if end.Lt(start) {
		reversed = true
		start, end = end, start
	}
	return (x.Gt(start) && (x.Lt(end) || x.Eq(end))) != reversed

}

// checks if x is in interval [min, max) mod 2^256
func betweenIE(x, start, end *uint256.Int) bool {
	reversed := false
	if end.Lt(start) {
		reversed = true
		start, end = end, start
	}
	return ((x.Gt(start) || x.Eq(start)) && x.Lt(end)) != reversed

}
