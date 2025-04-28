// File: chiral.go
package main

import "math"

// GetMoleculeChiralCarbons returns all chiral carbon atom indices (1-based).
func GetMoleculeChiralCarbons(mol *Molecule) []int {
	var result []int
	for i := 1; i <= len(mol.Atoms); i++ {
		if IsChiralCarbon(mol, i) {
			result = append(result, i)
		}
	}
	return result
}

func (m *Molecule) GetAtomDeclaredBonds(idx int) []Bond {
	zeroIdx := idx - 1
	var ret []Bond
	for _, b := range m.Bonds {
		if int(b.From) == zeroIdx || int(b.To) == zeroIdx {
			ret = append(ret, b)
		}
	}
	return ret
}

// GetBondID 返回给定 Bond 在 Molecule.Bonds 切片中的 1-based 索引，找不到返回 -1
func (m *Molecule) GetBondID(target Bond) int {
	for i, b := range m.Bonds {
		if b == target {
			return i + 1
		}
	}
	return -1
}

// GetAtom 返回指定索引的原子 (1-based 索引)
func (m *Molecule) GetAtom(idx int) Atom {
	return m.Atoms[idx-1]
}

// GetBond 返回指定索引的键 (1-based 索引)
func (m *Molecule) GetBond(idx int) Bond {
	return m.Bonds[idx-1]
}

// IsChiralCarbon determines if the atom at the given 1-based index is a chiral carbon.
func IsChiralCarbon(mol *Molecule, index int) bool {
	if mol.GetAtom(index).Element != "C" {
		return false
	}
	zeroIdx := index - 1
	bonds := mol.GetAtomDeclaredBonds(index)

	hcnt := mol.GetAtom(index).HCount
	var nonHBonds []Bond

	for _, b := range bonds {
		// 先算出另一个原子的 zero-based 索引
		var nbZero int
		if int(b.From) == zeroIdx {
			nbZero = int(b.To)
		} else {
			nbZero = int(b.From)
		}
		// 再转成 one-based
		otherIdx := nbZero + 1

		other := mol.GetAtom(otherIdx)
		decl := mol.GetAtomDeclaredBonds(otherIdx)

		if other.Element == "H" && len(decl) == 1 {
			hcnt++
		} else {
			nonHBonds = append(nonHBonds, b)
		}
	}
	// Check 4+0 or 3+1 substitution pattern
	if len(nonHBonds) == 4 && hcnt == 0 {
		// compare all 6 pairs
		ids := make([]int, 4)
		for i := 0; i < 4; i++ {
			ids[i] = mol.GetBondID(nonHBonds[i])
		}
		pairs := [][2]int{{0, 1}, {0, 2}, {0, 3}, {1, 2}, {1, 3}, {2, 3}}
		for _, p := range pairs {
			if CompareChain(mol, index, ids[p[0]], ids[p[1]]) {
				return false
			}
		}
		return true
	} else if len(nonHBonds) == 3 && hcnt == 1 {
		// compare 3 pairs
		ids := make([]int, 3)
		for i := 0; i < 3; i++ {
			ids[i] = mol.GetBondID(nonHBonds[i])
		}
		pairs := [][2]int{{0, 1}, {0, 2}, {1, 2}}
		for _, p := range pairs {
			if CompareChain(mol, index, ids[p[0]], ids[p[1]]) {
				return false
			}
		}
		return true
	}
	return false
}

// CompareChain checks if two chains from the center bond outwards are identical.
func CompareChain(mol *Molecule, center, c1, c2 int) bool {
	ttl := int(3 + math.Sqrt(float64(len(mol.Atoms))))
	return compareChainRec(mol, center, center, c1, c2, ttl)
}

func compareChainRec(mol *Molecule, atom1, atom2, chain1, chain2, ttl int) bool {
	if ttl < 0 {
		return true
	}
	// Invalid bond ID => treat as matching end
	if chain1 < 1 || chain2 < 1 || chain1 > len(mol.Bonds) || chain2 > len(mol.Bonds) {
		return true
	}
	b1 := mol.GetBond(chain1)
	b2 := mol.GetBond(chain2)
	// —— 1. 把 atom1/atom2 转成 0-based
	zeroAtom1 := atom1 - 1
	zeroAtom2 := atom2 - 1
	// —— 2. 找到下一个原子的 0-based 索引
	var nextZero1, nextZero2 int
	if int(b1.From) == zeroAtom1 {
		nextZero1 = int(b1.To)
	} else {
		nextZero1 = int(b1.From)
	}
	if int(b2.From) == zeroAtom2 {
		nextZero2 = int(b2.To)
	} else {
		nextZero2 = int(b2.From)
	}

	// —— 3. 转回 1-based
	n1 := nextZero1 + 1
	n2 := nextZero2 + 1

	if b1.Order != b2.Order {
		return false
	}
	// Element check
	a1 := mol.GetAtom(n1)
	a2 := mol.GetAtom(n2)
	if a1.Element != a2.Element {
		return false
	}
	// Count H and collect substituents
	h1, h2 := a1.HCount, a2.HCount
	bonds1 := mol.GetAtomDeclaredBonds(n1)
	bonds2 := mol.GetAtomDeclaredBonds(n2)
	var subs1, subs2 []int
	for _, b := range bonds1 {
		if b == b1 {
			continue
		}
		other := b.From
		if other == n1 {
			other = b.To
		}
		if other < 1 || other > len(mol.Atoms) {
			continue
		}
		oa := mol.GetAtom(other)
		decl := mol.GetAtomDeclaredBonds(other)
		if oa.Element == "H" && len(decl) == 1 {
			h1++
		} else {
			subs1 = append(subs1, mol.GetBondID(b))
		}
	}
	for _, b := range bonds2 {
		if b == b2 {
			continue
		}
		other := b.From
		if other == n2 {
			other = b.To
		}
		if other < 1 || other > len(mol.Atoms) {
			continue
		}
		oa := mol.GetAtom(other)
		decl := mol.GetAtomDeclaredBonds(other)
		if oa.Element == "H" && len(decl) == 1 {
			h2++
		} else {
			subs2 = append(subs2, mol.GetBondID(b))
		}
	}
	if h1 != h2 || len(subs1) != len(subs2) {
		return false
	}
	if len(subs1) == 0 {
		return true
	}
	ttl--
	// Recursively compare substituent chains
	for _, id1 := range subs1 {
		matched := false
		for _, id2 := range subs2 {
			if compareChainRec(mol, n1, n2, id1, id2, ttl) {
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
