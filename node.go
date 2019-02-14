package koorde

import (
	"log"

	"github.com/holiman/uint256"
)

var (
	one = uint256.NewInt().SetOne()
	neg = uint256.NewInt().Not()
)

type node struct {
	id   *uint256.Int
	succ *node
	d    *node
}

func NewNode(id *uint256.Int) *node {
	n := new(node)
	n.id = id.Clone()
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

	for j := 255; j >= 0; j-- {
		// mask out low bits of m.id
		i.Lsh(neg, uint(j)).And(i, m.id)
		// add one to the lowest bit of what is left of m.id
		tmp.Lsh(one, uint(j))
		i.Add(i, tmp)

		// put the key into low bits of i
		tmp.Rsh(k, uint(256-j))
		i.Or(i, tmp)

		//log.Printf("i %064x\n", i.ToBig())
		//log.Printf("t %064x\n", tmp.ToBig())

		//log.Printf("r %064x\n", i.ToBig())

		// check if it still on path
		if betweenEI(i, m.id, m.succ.id) {
			log.Printf("Match at %d", j)
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
	if betweenEI(k, m.id, m.succ.id) {
		//log.Printf("[%x] Found %x\n", m.id, k)
		log.Printf("Found")
		return m.succ, nil
	}
	kshift, i := m.bestStart(k.Clone())
	return m.lookup(k, kshift, i)
}

// Performs lookup of node for key  k
// kshift is being shifted into virtual node i one step at the time
// i is virtual node on the path from starting node to k
func (m node) lookup(k, kshift, i *uint256.Int) (*node, error) {
	// check if our successor is responsible for k
	//log.Printf("Got: %064x @ %064x", i.ToBig(), m.id.ToBig())
	//log.Printf("Ksh: %064x", kshift.ToBig())
	if betweenEI(k, m.id, m.succ.id) {
		//log.Printf("[%x] Found %x\n", m.id, k)
		log.Printf("Found")
		return m.succ, nil
	}

	// check if we are responsibe for path i -> k
	if betweenEI(i, m.id, m.succ.id) {
		//log.Printf("[%x] Forwarding %x\n", m.id, k)
		log.Printf("Forwarding")

		// forward the request
		topBit := uint256.NewInt().Rsh(k, 255) // topBit.bit(255)
		i = i.Lsh(i, 1).Or(i, topBit)          // i = (i << 1) | topBit
		kshift = kshift.Lsh(kshift, 1)         // kshift = kshift << 1

		return m.d.lookup(k, kshift, i)
	} else {
		//log.Printf("[%x] Correcting %x\n", m.id, k)
		tmp := uint256.NewInt()
		tmp.Sub(i, m.succ.id)
		log.Printf("Correcting")
		//log.Printf("Correcting delta %064x\n", tmp.ToBig())
		// correct if we are not responsibe for the path
		return m.succ.lookup(k, kshift, i)
	}
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
