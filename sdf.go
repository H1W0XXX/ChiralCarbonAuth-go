// File: sdf.go
package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

type Atom struct {
	X, Y    float64
	Element string
	HCount  int
}

type Bond struct {
	From, To, Order int
}

type Molecule struct {
	Atoms []Atom
	Bonds []Bond
}

func pickRandomOffset(offsets []int64) int64 {
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	return offsets[r.Intn(len(offsets))]
}

// ParseSDF 只读第一个分子，支持 V2000 格式
func ParseSDF(path string) (*Molecule, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	// 跳过三行 header
	for i := 0; i < 3; i++ {
		if !sc.Scan() {
			return nil, fmt.Errorf("unexpected EOF in header")
		}
	}
	// counts line
	if !sc.Scan() {
		return nil, fmt.Errorf("missing counts line")
	}
	cnt := sc.Text()
	atomCount, _ := strconv.Atoi(strings.TrimSpace(cnt[0:3]))
	bondCount, _ := strconv.Atoi(strings.TrimSpace(cnt[3:6]))

	mol := &Molecule{
		Atoms: make([]Atom, atomCount),
		Bonds: make([]Bond, bondCount),
	}
	// atom block
	for i := 0; i < atomCount; i++ {
		sc.Scan()
		line := sc.Text()
		x, _ := strconv.ParseFloat(strings.TrimSpace(line[0:10]), 64)
		y, _ := strconv.ParseFloat(strings.TrimSpace(line[10:20]), 64)
		elem := strings.TrimSpace(line[31:34])
		mol.Atoms[i] = Atom{X: x, Y: y, Element: elem, HCount: 0}
	}
	// bond block
	for i := 0; i < bondCount; i++ {
		sc.Scan()
		line := sc.Text()
		fIdx, _ := strconv.Atoi(strings.TrimSpace(line[0:3]))
		tIdx, _ := strconv.Atoi(strings.TrimSpace(line[3:6]))
		order, _ := strconv.Atoi(strings.TrimSpace(line[6:9]))
		mol.Bonds[i] = Bond{From: fIdx - 1, To: tIdx - 1, Order: order}
	}
	return mol, nil
}
func parseRandomMolFromFile(sdfPath string) (*Molecule, error) {
	idxPath := strings.TrimSuffix(sdfPath, ".sdf") + ".index"
	offsets, err := loadIndex(idxPath)
	if err != nil {
		return nil, err
	}
	if len(offsets) == 0 {
		return nil, fmt.Errorf("index is empty")
	}
	off := pickRandomOffset(offsets)
	return ParseMolAtOffset(sdfPath, off)
}

// ParseMolString: 解析单个 mol 字符串
func ParseMolString(str string) (*Molecule, error) {
	// 简单判断：如果传进来的字符串很短，而且是 .sdf 文件路径
	if strings.HasSuffix(str, ".sdf") && len(str) < 300 {
		return parseRandomMolFromFile(str)
	}

	// 否则，按原来解析 mol block 的逻辑
	lines := strings.Split(strings.ReplaceAll(str, "\r\n", "\n"), "\n")
	if len(lines) < 4 {
		return nil, fmt.Errorf("invalid mol: too few lines")
	}

	var atoms []Atom
	var bonds []Bond

	var countsLine string
	for i, line := range lines {
		if len(line) >= 39 && strings.Contains(line[30:39], "V2000") {
			countsLine = lines[i]
			lines = lines[i+1:]
			break
		}
	}
	if countsLine == "" {
		return nil, fmt.Errorf("invalid mol: V2000 not found")
	}

	numAtoms := parseIntSafe(countsLine[:3])
	numBonds := parseIntSafe(countsLine[3:6])

	if len(lines) < numAtoms+numBonds {
		return nil, fmt.Errorf("invalid mol: lines too short for atoms+bonds")
	}

	for i := 0; i < numAtoms; i++ {
		l := lines[i]
		if len(l) < 39 {
			continue
		}
		atoms = append(atoms, Atom{
			X:       parseFloatSafe(l[0:10]),
			Y:       parseFloatSafe(l[10:20]),
			Element: strings.TrimSpace(l[31:34]),
		})
	}

	for i := 0; i < numBonds; i++ {
		l := lines[numAtoms+i]
		if len(l) < 12 {
			continue
		}
		bonds = append(bonds, Bond{
			From:  parseIntSafe(l[0:3]) - 1,
			To:    parseIntSafe(l[3:6]) - 1,
			Order: parseIntSafe(l[6:9]),
		})
	}

	return &Molecule{
		Atoms: atoms,
		Bonds: bonds,
	}, nil
}

func parseIntSafe(s string) int {
	n := 0
	fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	return n
}

func parseFloatSafe(s string) float64 {
	f := 0.0
	fmt.Sscanf(strings.TrimSpace(s), "%f", &f)
	return f
}

// Hydrogenate 填充隐式氢到 Atom.HCount
func Hydrogenate(mol *Molecule) {
	for ai := range mol.Atoms {
		atom := &mol.Atoms[ai]
		// 先把显式氢数算到 hcnt
		hcnt := atom.HCount
		// 计算已有键的键阶之和
		totalBond := 0
		for _, b := range mol.GetAtomDeclaredBonds(ai + 1) {
			totalBond += b.Order
		}
		switch atom.Element {
		case "C":
			atom.HCount = max(0, 4-totalBond)
		case "O", "S":
			atom.HCount = max(0, 2-totalBond)
		case "N", "P":
			atom.HCount = max(0, 3-totalBond)
		// …按 Java initOnce 里相同的逻辑
		default:
			atom.HCount = hcnt
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// loadIndex 读取索引文件，每行一个偏移量（ASCII 格式），返回 []int64
func loadIndex(idxPath string) ([]int64, error) {
	file, err := os.Open(idxPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var offsets []int64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		off, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse offset %q: %w", line, err)
		}
		offsets = append(offsets, off)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return offsets, nil
}

// readMolAt 偏移 off 处开始读，一直读到下一个 "$$$$"（含），返回这一块的文本（不含终结符行）
func readMolAt(sdfPath string, off int64) (string, error) {
	f, err := os.Open(sdfPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// 定位到分子块开头
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return "", err
	}

	reader := bufio.NewReader(f)
	var sb strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "$$$$" {
			// 遇到块结束符，停止（不包含这行）
			break
		}
		sb.WriteString(line)
		if err == io.EOF {
			// 文件末尾
			break
		}
	}
	return sb.String(), nil
}
func ParseMolAtOffset(sdfPath string, offset int64) (*Molecule, error) {
	f, err := os.Open(sdfPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	reader := bufio.NewReader(f)
	var sb strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "$$$$" {
			break // 读到 "$$$$" 结束（不包含）
		}
		sb.WriteString(line)
		// 保持原来的换行
		if !strings.HasSuffix(line, "\n") {
			sb.WriteString("\n")
		}
		if err == io.EOF {
			break
		}
	}

	// 现在 sb.String() 是这个分子的 mol 字符串
	molStr := sb.String()
	return ParseMolString(molStr)
}

func pickRandomMoleculeFromIndexed(sdfPath, idxPath string) (*Molecule, error) {
	offsets, err := loadIndex(idxPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}
	if len(offsets) == 0 {
		return nil, fmt.Errorf("index is empty")
	}
	off := offsets[rand.Intn(len(offsets))]
	return ParseMolAtOffset(sdfPath, off)
}
