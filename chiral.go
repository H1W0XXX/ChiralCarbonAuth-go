// File: chiral.go
package main

import (
	"fmt"
	"math"
)

// Molecule 中添加的缓存字段示例：
// bondIDMap map[Bond]int
// atomBondMap map[int][]int
// chainTTL    int

// GetMoleculeChiralCarbons returns all chiral carbon atom indices (1-based).
func GetMoleculeChiralCarbons(m *Molecule) []int {
	Hydrogenate(m)  // 确保隐式 HCount 正确
	m.buildCaches() // 初始化缓存

	var out []int
	//fmt.Printf("→ Molecule: %d atoms, %d bonds\n", len(m.Atoms), len(m.Bonds))
	for zero := range m.Atoms {
		// 打印每个原子的初始信息
		//atom := &m.Atoms[zero]
		//bondIDs := m.atomBondMap[zero]
		//fmt.Printf("Atom %2d (%s): bonds=%v, HCount=%d\n",
		//	zero+1, atom.Element, bondIDs, atom.HCount,
		//)
		if m.isChiralCarbon0(zero) {
			//fmt.Printf("  -> CHIRAL!\n")
			out = append(out, zero+1)
		}
	}
	//fmt.Printf("=> Chiral Carbons: %v\n", out)
	return out
}

// GetAtomDeclaredBonds returns all bonds connected to atom at 1-based index idx.
// 兼容旧代码调用：Hydrogenate 等也可继续使用。
func (m *Molecule) GetAtomDeclaredBonds(idx int) []Bond {
	m.buildCaches()
	zero := idx - 1
	ids := m.atomBondMap[zero]
	out := make([]Bond, 0, len(ids))
	for _, id := range ids {
		out = append(out, m.Bonds[id-1])
	}
	return out
}

// isChiralCarbon0 determines if the atom at zero-based index c0 is a chiral carbon.
// Assumes m.buildCaches() has been called so that m.atomBondMap and m.chainTTL are initialized.
// isChiralCarbon0 determines if the atom at zero-based index c0 is a chiral carbon.
func (m *Molecule) isChiralCarbon0(c0 int) bool {
	a := &m.Atoms[c0]
	// 跳过非碳
	if a.Element != "C" {
		return false
	}

	bondIDs := m.atomBondMap[c0]
	hcnt := a.HCount
	var nonHBonds []int

	//fmt.Printf("  Checking C atom %d: initial H=%d, bonds=%v\n", c0+1, hcnt, bondIDs)
	// 分类 H / 非-H
	for _, bid := range bondIDs {
		b := m.Bonds[bid-1]
		other0 := b.From // 直接取 0-based
		if other0 == c0 {
			other0 = b.To
		}
		other := &m.Atoms[other0]
		declIDs := m.atomBondMap[other0]
		if other.Element == "H" && len(declIDs) == 1 {
			hcnt++
		} else {
			nonHBonds = append(nonHBonds, bid)
		}
	}
	//fmt.Printf("    after classify: H=%d, nonHBonds=%v\n", hcnt, nonHBonds)

	var pairs [][2]int
	switch {
	case len(nonHBonds) == 4 && hcnt == 0:
		pairs = [][2]int{{0, 1}, {0, 2}, {0, 3}, {1, 2}, {1, 3}, {2, 3}}
	case len(nonHBonds) == 3 && hcnt == 1:
		pairs = [][2]int{{0, 1}, {0, 2}, {1, 2}}
	default:
		//fmt.Printf("    → not 4+0/3+1 pattern\n")
		return false
	}

	for _, p := range pairs {
		visited := make(map[[4]int]bool)
		if m.compareChainRec(c0, c0, nonHBonds[p[0]], nonHBonds[p[1]], m.chainTTL, visited) {
			//fmt.Printf("    → chains %v match, not chiral\n", p)
			return false
		}
	}
	return true
}

// buildCaches initializes caching structures for quick lookups
func (m *Molecule) buildCaches() {
	if m.bondIDMap != nil {
		return
	}
	fmt.Println("Building caches...")
	m.bondIDMap = make(map[Bond]int, len(m.Bonds))
	m.atomBondMap = make(map[int][]int, len(m.Atoms))
	m.chainTTL = 3 + int(math.Sqrt(float64(len(m.Atoms))))

	for i, b := range m.Bonds {
		id := i + 1
		m.bondIDMap[b] = id
		// Bond.From/To are zero-based indices
		from0 := int(b.From)
		to0 := int(b.To)
		if from0 < 0 || from0 >= len(m.Atoms) || to0 < 0 || to0 >= len(m.Atoms) {
			panic(fmt.Sprintf("buildCaches: invalid bond %v endpoints", b))
		}
		m.atomBondMap[from0] = append(m.atomBondMap[from0], id)
		m.atomBondMap[to0] = append(m.atomBondMap[to0], id)
	}
}

// compareChainRec recursively compares two substituent chains for identity, with cycle detection
func (m *Molecule) compareChainRec(atom1, atom2, chain1, chain2, ttl int, visited map[[4]int]bool) bool {
	key := [4]int{atom1, atom2, chain1, chain2}
	if visited[key] {
		return true
	}
	visited[key] = true

	if ttl < 0 {
		return true
	}
	if chain1 < 1 || chain2 < 1 || chain1 > len(m.Bonds) || chain2 > len(m.Bonds) {
		return true
	}

	b1 := m.Bonds[chain1-1]
	b2 := m.Bonds[chain2-1]

	// Find next atoms
	var next1, next2 int
	if int(b1.From) == atom1 {
		next1 = int(b1.To)
	} else {
		next1 = int(b1.From)
	}
	if int(b2.From) == atom2 {
		next2 = int(b2.To)
	} else {
		next2 = int(b2.From)
	}

	// Compare bond order and element
	if b1.Order != b2.Order {
		return false
	}
	a1 := &m.Atoms[next1]
	a2 := &m.Atoms[next2]
	if a1.Element != a2.Element {
		return false
	}

	// Count implicit H and gather substituents
	h1, h2 := a1.HCount, a2.HCount
	bonds1 := m.atomBondMap[next1]
	bonds2 := m.atomBondMap[next2]
	var subs1, subs2 []int
	for _, bid := range bonds1 {
		if bid == chain1 {
			continue
		}
		b := m.Bonds[bid-1]
		other0 := int(b.From)
		if other0 == next1 {
			other0 = int(b.To)
		}
		other := &m.Atoms[other0]
		declIDs := m.atomBondMap[other0]
		if other.Element == "H" && len(declIDs) == 1 {
			h1++
		} else {
			subs1 = append(subs1, bid)
		}
	}
	for _, bid := range bonds2 {
		if bid == chain2 {
			continue
		}
		b := m.Bonds[bid-1]
		other0 := int(b.From)
		if other0 == next2 {
			other0 = int(b.To)
		}
		other := &m.Atoms[other0]
		declIDs := m.atomBondMap[other0]
		if other.Element == "H" && len(declIDs) == 1 {
			h2++
		} else {
			subs2 = append(subs2, bid)
		}
	}
	if h1 != h2 || len(subs1) != len(subs2) {
		return false
	}
	if len(subs1) == 0 {
		return true
	}

	ttl--
	for _, id1 := range subs1 {
		matched := false
		for _, id2 := range subs2 {
			if m.compareChainRec(next1, next2, id1, id2, ttl, visited) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// CompareChain provides a public wrapper for chain comparison if needed
func CompareChain(m *Molecule, center, c1, c2 int) bool {
	m.buildCaches()
	visited := make(map[[4]int]bool)
	return m.compareChainRec(center, center, c1, c2, m.chainTTL, visited)
}
