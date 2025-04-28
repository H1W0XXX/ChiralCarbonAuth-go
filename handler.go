// File: handler.go
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
)

var (
	mu         sync.Mutex
	challenges = make(map[string]Challenge)
)

func ParseSDFMulti(path string) ([]*Molecule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	contents := string(data)
	parts := strings.Split(contents, "$$$$")
	var molecules []*Molecule
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		mol, err := ParseMolString(part)
		if err != nil {
			continue // 有坏的就跳过
		}
		molecules = append(molecules, mol)
	}
	return molecules, nil
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	// 尝试多次，确保至少有 3 个手性碳
	var mol *Molecule
	var chiral []int
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		mol, err = pickRandomMoleculeFromIndexed("Compound_156500001_157000000.sdf", "Compound_156500001_157000000.index")
		if err != nil {
			continue
		}
		Hydrogenate(mol)
		chiral = GetMoleculeChiralCarbons(mol)
		if len(chiral) >= 3 {
			break
		}
	}
	if len(chiral) < 3 {
		http.Error(w, "not enough chiral carbons, try again", http.StatusInternalServerError)
		return
	}

	// 3) 自动网格
	cols, rows := AutoGrid(len(chiral))

	// 4) 渲染配置（动态计算）
	renderCfg, err := CalculateRenderConfig(mol, 600, cols, rows)
	if err != nil {
		http.Error(w, "failed to calculate render config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// 标记手性碳
	for _, idx := range chiral {
		renderCfg.ShownChiral[idx] = true
	}

	// 5) 绘制分子并拿到 regions
	molBytes, regions, err := RenderMoleculeImage(mol, renderCfg)
	if err != nil {
		http.Error(w, "failed to draw molecule: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 6) 计算答案：用同一个 renderCfg
	cellW := float64(renderCfg.Width) / float64(renderCfg.GridCountX)
	cellH := float64(renderCfg.Height) / float64(renderCfg.GridCountY)

	answersSet := make(map[string]struct{}, len(chiral))
	for _, idx := range chiral {
		a := mol.Atoms[idx-1]
		// 原子中心在画布上的像素坐标
		px := renderCfg.FontSize + renderCfg.ScaleFactor*(a.X-mol.MinX())
		py := float64(renderCfg.Height) - renderCfg.FontSize - renderCfg.ScaleFactor*(a.Y-mol.MinY())

		col := int(px / cellW)
		row := int(py / cellH)
		// 边界保护
		if col < 0 {
			col = 0
		} else if col >= renderCfg.GridCountX {
			col = renderCfg.GridCountX - 1
		}
		if row < 0 {
			row = 0
		} else if row >= renderCfg.GridCountY {
			row = renderCfg.GridCountY - 1
		}
		label := fmt.Sprintf("%c%d", 'A'+col, row+1)
		answersSet[label] = struct{}{}
	}

	// 7) 转为切片并排序
	answers := make([]string, 0, len(answersSet))
	for lbl := range answersSet {
		answers = append(answers, lbl)
	}
	sort.Strings(answers)

	// 8) 存储并返回
	id := uuid.New().String()
	log.Printf("Challenge %s Correct Answers: %v", id, answers)
	mu.Lock()
	challenges[id] = Challenge{Regions: regions, Answers: answers}
	mu.Unlock()

	rsp := StartResponse{
		UUID:    id,
		Image:   "data:image/png;base64," + base64.StdEncoding.EncodeToString(molBytes),
		Regions: regions,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rsp)
}

func handleVerify(w http.ResponseWriter, r *http.Request) {
	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	mu.Lock()
	chal, ok := challenges[req.UUID]
	mu.Unlock()
	if !ok {
		http.Error(w, "uuid not found", http.StatusNotFound)
		return
	}

	// 对比答案
	ansMap := make(map[string]bool, len(chal.Answers))
	for _, a := range chal.Answers {
		ansMap[a] = true
	}
	if len(req.Selections) != len(chal.Answers) {
		json.NewEncoder(w).Encode(VerifyResponse{false, "验证失败"})
		return
	}
	for _, sel := range req.Selections {
		if !ansMap[sel] {
			json.NewEncoder(w).Encode(VerifyResponse{false, "验证失败"})
			return
		}
	}
	json.NewEncoder(w).Encode(VerifyResponse{true, "验证通过"})
}
