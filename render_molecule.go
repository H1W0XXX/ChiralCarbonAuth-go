// File: render_molecule.go
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image/png"
	"math"
	"math/rand"
	"os"

	"github.com/fogleman/gg"
)

// MoleculeRenderConfig 与 Java 版 MoleculeRenderConfig 对应
type MoleculeRenderConfig struct {
	Width, Height          int     // 画布尺寸
	FontSize               float64 // 字体大小
	ScaleFactor            float64 // 缩放因子
	GridCountX, GridCountY int     // 网格行列数
	DrawGrid               bool    // 是否绘制背景网格

	// 每个原子文字或符号的边界
	LabelLeft, LabelRight, LabelTop, LabelBottom []float64

	// 需标记的手性碳（1-based 索引）
	ShownChiral map[int]bool
}

// CalculateRenderConfig 根据 Java 版逻辑，计算 fontSize, scaleFactor 并确定画布大小
// maxSize 对应 "最大边长"，gridX/gridY 对应网格行列数
func CalculateRenderConfig(mol *Molecule, maxSize, gridX, gridY int) (*MoleculeRenderConfig, error) {
	rx := mol.RangeX()
	ry := mol.RangeY()
	if rx == 0 || ry == 0 {
		return nil, fmt.Errorf("molecule has no range")
	}
	// Java: scaleFactor = min(maxSize/rx, maxSize/ry)
	scale := math.Min(float64(maxSize)/rx, float64(maxSize)/ry)
	// Java: fontSize = avgBondLength/1.8*scale, 并 cap 到 maxSize/16
	avgBond := mol.AverageBondLength()
	fontSize := avgBond / 1.8 * scale
	if fontSize > float64(maxSize)/16.0 {
		fontSize = float64(maxSize) / 16.0
	}
	// Java: width = rx*scale, height = ry*scale
	w := int(rx * scale)
	h := int(ry * scale)

	cfg := &MoleculeRenderConfig{
		Width:       w + 2*int(fontSize),
		Height:      h + 2*int(fontSize),
		FontSize:    fontSize,
		ScaleFactor: scale,
		GridCountX:  gridX,
		GridCountY:  gridY,
		DrawGrid:    true,
		ShownChiral: make(map[int]bool),
	}
	// 初始化 label 数组
	n := len(mol.Atoms)
	cfg.LabelLeft = make([]float64, n)
	cfg.LabelRight = make([]float64, n)
	cfg.LabelTop = make([]float64, n)
	cfg.LabelBottom = make([]float64, n)
	return cfg, nil
}

// RenderMoleculeImage 按 Java renderMoleculeAsImage 逻辑绘制并返回 PNG 字节 + 区域标签列表
func RenderMoleculeImage(mol *Molecule, cfg *MoleculeRenderConfig) ([]byte, []string, error) {
	dc := gg.NewContext(cfg.Width, cfg.Height)
	// 白底
	dc.SetRGB(1, 1, 1)
	dc.Clear()
	if cfg.DrawGrid {
		drawGridBackground(dc, cfg)
	}
	doDrawMolecule(dc, mol, cfg)

	// 输出 PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, dc.Image()); err != nil {
		return nil, nil, err
	}
	// 区域标签 A1, A2, ... B1...
	regions := make([]string, 0, cfg.GridCountX*cfg.GridCountY)
	for i := 0; i < cfg.GridCountX; i++ {
		for j := 0; j < cfg.GridCountY; j++ {
			regions = append(regions, fmt.Sprintf("%c%d", 'A'+i, j+1))
		}
	}
	return buf.Bytes(), regions, nil
}

// drawGridBackground 对应 doDrawGridTagForBackground
func drawGridBackground(dc *gg.Context, cfg *MoleculeRenderConfig) {
	unitX := float64(cfg.Width) / float64(cfg.GridCountX)
	unitY := float64(cfg.Height) / float64(cfg.GridCountY)
	// 绘制双色棋盘格
	for i := 0; i < cfg.GridCountX; i++ {
		for j := 0; j < cfg.GridCountY; j++ {
			if (i+j)%2 == 0 {
				dc.SetHexColor("#FFFFFF")
			} else {
				dc.SetHexColor("#E0E0E0")
			}
			dc.DrawRectangle(float64(i)*unitX, float64(j)*unitY, unitX, unitY)
			dc.Fill()
		}
	}
	// 标签文字
	labelSize := math.Min(math.Min(unitX, unitY)/2.0, cfg.FontSize)
	dc.SetRGB(0.627, 0.627, 0.627)
	dc.LoadFontFace("Roboto-Regular.ttf", labelSize)
	for i := 0; i < cfg.GridCountX; i++ {
		for j := 0; j < cfg.GridCountY; j++ {
			tag := fmt.Sprintf("%c%d", 'A'+i, j+1)
			x := float64(i)*unitX + labelSize*0.25
			y := float64(j+1)*unitY - dc.FontHeight()/2
			dc.DrawString(tag, x, y)
		}
	}
}

// doDrawMolecule 对应 Java doDrawMolecule（简化版）
func doDrawMolecule(dc *gg.Context, mol *Molecule, cfg *MoleculeRenderConfig) {
	dc.SetLineWidth(cfg.FontSize / 12)
	dc.SetRGB(0, 0, 0)
	dc.LoadFontFace("Roboto-Regular.ttf", cfg.FontSize)

	// 1) 绘制原子标签 & 计算 padding
	for i, a := range mol.Atoms {
		x := cfg.FontSize + cfg.ScaleFactor*(a.X-mol.MinX())
		y := float64(cfg.Height) - cfg.FontSize - cfg.ScaleFactor*(a.Y-mol.MinY())
		// 碳原子不画元素符号，只画手性★
		if a.Element == "C" {
			cfg.LabelLeft[i] = 0
			cfg.LabelRight[i] = 0
			cfg.LabelTop[i] = 0
			cfg.LabelBottom[i] = 0
			if cfg.ShownChiral[i+1] {
				s := "*"
				w, _ := dc.MeasureString(s)
				r := w/4 + cfg.FontSize/4
				dc.DrawStringAnchored(s, x+r, y-r, 0.5, 0.5)
			}
		} else {
			// 非碳元素：先测文字宽度，设置 padding
			w, _ := dc.MeasureString(a.Element)
			cfg.LabelLeft[i] = w / 2
			cfg.LabelRight[i] = w / 2
			cfg.LabelTop[i] = cfg.FontSize / 2
			cfg.LabelBottom[i] = cfg.FontSize / 2
			// 元素文字居中绘制
			dc.DrawStringAnchored(a.Element, x, y, 0.5, 0.5)
			// 手性星号靠左
			if cfg.ShownChiral[i+1] {
				s := "*"
				w2, _ := dc.MeasureString(s)
				dc.DrawString(s, x-cfg.LabelLeft[i]-w2/2, y)
				cfg.LabelLeft[i] += w2
			}
			// **可按需扩展：电荷、显式氢等，参考 Java 版**
		}
	}

	// 2) 绘制键（Bond）
	for _, b := range mol.Bonds {
		x1 := cfg.FontSize + cfg.ScaleFactor*(mol.Atoms[b.From].X-mol.MinX())
		y1 := float64(cfg.Height) - cfg.FontSize - cfg.ScaleFactor*(mol.Atoms[b.From].Y-mol.MinY())
		x2 := cfg.FontSize + cfg.ScaleFactor*(mol.Atoms[b.To].X-mol.MinX())
		y2 := float64(cfg.Height) - cfg.FontSize - cfg.ScaleFactor*(mol.Atoms[b.To].Y-mol.MinY())
		p1 := calcLinePointConfined(x1, y1, x2, y2,
			cfg.LabelLeft[b.From], cfg.LabelRight[b.From], cfg.LabelTop[b.From], cfg.LabelBottom[b.From])
		p2 := calcLinePointConfined(x2, y2, x1, y1,
			cfg.LabelLeft[b.To], cfg.LabelRight[b.To], cfg.LabelTop[b.To], cfg.LabelBottom[b.To])
		rad := math.Atan2(y2-y1, x2-x1)
		delta := cfg.FontSize / 6
		dxOff := math.Sin(rad) * delta
		dyOff := -math.Cos(rad) * delta
		switch b.Order {
		case 1:
			dc.DrawLine(p1.X, p1.Y, p2.X, p2.Y)
		case 2:
			dc.DrawLine(p1.X+dxOff/2, p1.Y+dyOff/2, p2.X+dxOff/2, p2.Y+dyOff/2)
			dc.DrawLine(p1.X-dxOff/2, p1.Y-dyOff/2, p2.X-dxOff/2, p2.Y-dyOff/2)
		case 3:
			dc.DrawLine(p1.X, p1.Y, p2.X, p2.Y)
			dc.DrawLine(p1.X+dxOff, p1.Y+dyOff, p2.X+dxOff, p2.Y+dyOff)
			dc.DrawLine(p1.X-dxOff, p1.Y-dyOff, p2.X-dxOff, p2.Y-dyOff)
		}
		dc.Stroke()
	}
}

// 点结构体，用于裁剪线段端点
type Point struct{ X, Y float64 }

// calcLinePointConfined 对应 Java calcLinePointConfined
func calcLinePointConfined(x, y, x2, y2, left, right, top, bottom float64) Point {
	// w/h 同 Java 版
	w := right
	if x2 <= x {
		w = left
	}
	h := top
	if y2 < y {
		h = bottom
	}
	k := math.Atan2(h, w)
	sigx := math.Copysign(1, x2-x)
	sigy := math.Copysign(1, y2-y)
	absRad := math.Atan2(math.Abs(y2-y), math.Abs(x2-x))
	if absRad > k {
		return Point{X: x + sigx*h/math.Tan(absRad), Y: y + sigy*h}
	}
	return Point{X: x + sigx*w, Y: y + sigy*w*math.Tan(absRad)}
}

// 以下是 Molecule 辅助方法，请确保与你的 Molecule 定义一致
func (m *Molecule) MinX() float64 {
	min := math.MaxFloat64
	for _, a := range m.Atoms {
		if a.X < min {
			min = a.X
		}
	}
	return min
}
func (m *Molecule) MinY() float64 {
	min := math.MaxFloat64
	for _, a := range m.Atoms {
		if a.Y < min {
			min = a.Y
		}
	}
	return min
}
func (m *Molecule) RangeX() float64 {
	return m.MaxX() - m.MinX()
}
func (m *Molecule) RangeY() float64 {
	return m.MaxY() - m.MinY()
}
func (m *Molecule) MaxX() float64 {
	max := -math.MaxFloat64
	for _, a := range m.Atoms {
		if a.X > max {
			max = a.X
		}
	}
	return max
}
func (m *Molecule) MaxY() float64 {
	max := -math.MaxFloat64
	for _, a := range m.Atoms {
		if a.Y > max {
			max = a.Y
		}
	}
	return max
}
func (m *Molecule) AverageBondLength() float64 {
	total := 0.0
	for _, b := range m.Bonds {
		a1 := m.Atoms[b.From]
		a2 := m.Atoms[b.To]
		total += math.Hypot(a1.X-a2.X, a1.Y-a2.Y)
	}
	if len(m.Bonds) == 0 {
		return 0
	}
	return total / float64(len(m.Bonds))
}

// pickRandomMolecule 从 .sdf.gz 文件中，用水塘抽样随机选一条分子记录
func pickRandomMolecule(path string) (*Molecule, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var buf bytes.Buffer
	var chosen string
	count := 0

	for scanner.Scan() {
		line := scanner.Text()
		if line == "$$$$" {
			// 一个分子记录结束
			count++
			// 水塘抽样：以 1/count 概率选中
			if rand.Intn(count) == 0 {
				chosen = buf.String()
			}
			buf.Reset()
		} else {
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	}
	// 如果文件最后没以 $$$$ 结尾，处理最后一段
	if buf.Len() > 0 {
		count++
		if rand.Intn(count) == 0 {
			chosen = buf.String()
		}
	}

	if chosen == "" {
		return nil, fmt.Errorf("no molecule found in %s", path)
	}
	// 解析单条 Mol 文本
	return ParseMolString(chosen)
}
