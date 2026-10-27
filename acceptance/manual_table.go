package acceptance

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

const ManualTableFormat = "geoworld-manual-acceptance-table"

type ManualTable struct {
	Format                  string              `json:"format"`
	Version                 int                 `json:"version"`
	RegistryVersion         string              `json:"registryVersion"`
	Seed                    int64               `json:"seed"`
	WorldProfileVersion     string              `json:"worldProfileVersion"`
	GenerationRulesVersion  string              `json:"generationRulesVersion"`
	WorldModelVersion       string              `json:"worldModelVersion"`
	Preset                  string              `json:"preset"`
	ParameterSnapshotSHA256 string              `json:"parameterSnapshotSha256"`
	BigCategories           []ManualBigCategory `json:"bigCategories"`
}

type ManualBigCategory struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	SmallCategories []ManualSmallCategory `json:"smallCategories"`
}

type ManualSmallCategory struct {
	ID    string       `json:"id"`
	Name  string       `json:"name"`
	Items []ManualItem `json:"items"`
}

type ManualItem struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Requirement RequirementKind `json:"requirement"`
	Inspection  InspectionKind  `json:"inspection"`
	Method      string          `json:"method"`
	Status      ResultStatus    `json:"status"`
	Expected    string          `json:"expected"`
	Failure     string          `json:"failure"`
	Points      []Point         `json:"points,omitempty"`
	Note        string          `json:"note,omitempty"`
}

type SpotTable struct {
	Format                  string            `json:"format"`
	Version                 int               `json:"version"`
	SourceFormat            string            `json:"sourceFormat"`
	SourceVersion           int               `json:"sourceVersion"`
	Seed                    int64             `json:"seed"`
	WorldProfileVersion     string            `json:"worldProfileVersion"`
	GenerationRulesVersion  string            `json:"generationRulesVersion"`
	WorldModelVersion       string            `json:"worldModelVersion"`
	Preset                  string            `json:"preset"`
	ParameterSnapshotSHA256 string            `json:"parameterSnapshotSha256"`
	PerBigCategory          int               `json:"perBigCategory"`
	BigCategories           []SpotBigCategory `json:"bigCategories"`
}

type SpotBigCategory struct {
	ID    string     `json:"id"`
	Name  string     `json:"name"`
	Items []SpotItem `json:"items"`
}

type SpotItem struct {
	SmallCategoryID   string     `json:"smallCategoryId"`
	SmallCategoryName string     `json:"smallCategoryName"`
	Item              ManualItem `json:"item"`
}

type categorySpec struct {
	ID       string
	Name     string
	Prefixes []string
}

var manualCategorySpecs = []categorySpec{
	{ID: "CORE", Name: "核心边界与原版兼容", Prefixes: []string{"CORE", "VSTRUCT"}},
	{ID: "MACRO", Name: "宏观大陆、出生与外围", Prefixes: []string{"MACRO", "SPAWN", "EDGE", "ISLAND"}},
	{ID: "TERRAIN", Name: "陆地地形与交通", Prefixes: []string{"MTN", "HILL", "PLATEAU", "BASIN", "PLAIN", "RARE", "CORRIDOR"}},
	{ID: "WATER", Name: "水系、湿地、海岸与海洋", Prefixes: []string{"RIVER", "FLOOD", "LAKE", "WET", "EST", "COAST", "SEA", "NODE"}},
	{ID: "ECOLOGY", Name: "气候、群系与地表生态", Prefixes: []string{"CLIMATE", "BIOME", "FOREST", "POLAR", "SURFACE"}},
	{ID: "UNDERGROUND", Name: "地下、洞穴与矿产", Prefixes: []string{"ORE", "CAVE"}},
	{ID: "PROFILE", Name: "参数、预设与版本锁定", Prefixes: []string{"PARAM", "PROFILE"}},
	{ID: "RUNTIME", Name: "模型查询与运行时性能", Prefixes: []string{"MODEL", "RUNTIME"}},
}

var smallCategoryNames = map[string]string{
	"CORE": "核心范围与兼容", "VSTRUCT": "原版结构兼容",
	"MACRO": "宏观大陆", "SPAWN": "出生点", "EDGE": "世界外围", "ISLAND": "岛屿",
	"MTN": "山系", "HILL": "丘陵", "PLATEAU": "高原", "BASIN": "盆地",
	"PLAIN": "平原", "RARE": "稀有地貌", "CORRIDOR": "自然交通走廊",
	"RIVER": "河流", "FLOOD": "河漫滩与河阶", "LAKE": "湖泊", "WET": "湿地",
	"EST": "河口", "COAST": "海岸", "SEA": "海洋与海底", "NODE": "关键地理节点",
	"CLIMATE": "气候", "BIOME": "群系", "FOREST": "森林", "POLAR": "极地",
	"SURFACE": "地表材质", "ORE": "矿产", "CAVE": "洞穴与地下空间",
	"PARAM": "参数", "PROFILE": "世界规则锁定", "MODEL": "Natural Model 查询",
	"RUNTIME": "运行时性能",
}

func BuildManualTable(report Report) (ManualTable, error) {
	if len(report.Results) == 0 {
		return ManualTable{}, fmt.Errorf("acceptance: empty report")
	}
	prefixToBig := make(map[string]int)
	for i, spec := range manualCategorySpecs {
		for _, prefix := range spec.Prefixes {
			if _, exists := prefixToBig[prefix]; exists {
				return ManualTable{}, fmt.Errorf("acceptance: duplicate category prefix %s", prefix)
			}
			prefixToBig[prefix] = i
		}
	}
	table := ManualTable{
		Format: ManualTableFormat, Version: 1,
		RegistryVersion:         report.RegistryVersion,
		Seed:                    report.Seed,
		WorldProfileVersion:     report.WorldProfileVersion,
		GenerationRulesVersion:  report.GenerationRulesVersion,
		WorldModelVersion:       report.WorldModelVersion,
		Preset:                  report.Preset,
		ParameterSnapshotSHA256: report.ParameterSnapshotSHA256,
		BigCategories:           make([]ManualBigCategory, len(manualCategorySpecs)),
	}
	smallIndex := make([]map[string]int, len(manualCategorySpecs))
	for i, spec := range manualCategorySpecs {
		table.BigCategories[i] = ManualBigCategory{ID: spec.ID, Name: spec.Name}
		smallIndex[i] = make(map[string]int)
	}
	seen := make(map[string]bool, len(report.Results))
	for _, result := range report.Results {
		if seen[result.ID] {
			return ManualTable{}, fmt.Errorf("acceptance: duplicate result ID %s", result.ID)
		}
		seen[result.ID] = true
		prefix := strings.SplitN(result.ID, "-", 2)[0]
		bigIndex, ok := prefixToBig[prefix]
		if !ok {
			return ManualTable{}, fmt.Errorf("acceptance: unclassified prefix %s for %s", prefix, result.ID)
		}
		si, ok := smallIndex[bigIndex][prefix]
		if !ok {
			si = len(table.BigCategories[bigIndex].SmallCategories)
			smallIndex[bigIndex][prefix] = si
			name := smallCategoryNames[prefix]
			if name == "" {
				name = prefix
			}
			table.BigCategories[bigIndex].SmallCategories = append(
				table.BigCategories[bigIndex].SmallCategories,
				ManualSmallCategory{ID: prefix, Name: name},
			)
		}
		small := &table.BigCategories[bigIndex].SmallCategories[si]
		small.Items = append(small.Items, ManualItem{
			ID: result.ID, Name: result.Name, Requirement: result.Requirement,
			Inspection: result.Inspection, Method: result.Method, Status: result.Status,
			Expected: result.Expected, Failure: result.Failure,
			Points: append([]Point(nil), result.Points...), Note: result.Note,
		})
	}
	entries, err := Entries()
	if err != nil {
		return ManualTable{}, err
	}
	if len(seen) != len(entries) {
		return ManualTable{}, fmt.Errorf(
			"acceptance: manual table contains %d registry IDs, want %d", len(seen), len(entries),
		)
	}
	for _, entry := range entries {
		if !seen[entry.ID] {
			return ManualTable{}, fmt.Errorf("acceptance: manual table missing registry ID %s", entry.ID)
		}
	}
	return table, nil
}

func BuildSpotTable(full ManualTable, perBig int) (SpotTable, error) {
	if full.Format != ManualTableFormat || full.Version != 1 {
		return SpotTable{}, fmt.Errorf("acceptance: unsupported manual table %q v%d", full.Format, full.Version)
	}
	if perBig <= 0 {
		return SpotTable{}, fmt.Errorf("acceptance: perBig must be positive")
	}
	spot := SpotTable{
		Format: "geoworld-spot-acceptance-table", Version: 1,
		SourceFormat: full.Format, SourceVersion: full.Version,
		Seed: full.Seed, WorldProfileVersion: full.WorldProfileVersion,
		GenerationRulesVersion: full.GenerationRulesVersion,
		WorldModelVersion:      full.WorldModelVersion, Preset: full.Preset,
		ParameterSnapshotSHA256: full.ParameterSnapshotSHA256,
		PerBigCategory:          perBig,
	}
	identity := fmt.Sprintf("%s|%d|%s", full.GenerationRulesVersion, full.Seed, full.ParameterSnapshotSHA256)
	for _, big := range full.BigCategories {
		candidates := make([]SpotItem, 0)
		for _, small := range big.SmallCategories {
			for _, item := range small.Items {
				if item.Status == StatusNotImplemented {
					continue
				}
				candidates = append(candidates, SpotItem{
					SmallCategoryID: small.ID, SmallCategoryName: small.Name, Item: item,
				})
			}
		}
		if len(candidates) < perBig {
			return SpotTable{}, fmt.Errorf(
				"acceptance: big category %s has only %d implemented items, want %d",
				big.ID, len(candidates), perBig,
			)
		}
		sort.SliceStable(candidates, func(i, j int) bool {
			pi := spotPriority(candidates[i].Item)
			pj := spotPriority(candidates[j].Item)
			if pi != pj {
				return pi < pj
			}
			return stableSpotHash(identity, big.ID, candidates[i].Item.ID) <
				stableSpotHash(identity, big.ID, candidates[j].Item.ID)
		})
		selected := make([]SpotItem, 0, perBig)
		usedSmall := map[string]bool{}
		for _, candidate := range candidates {
			if len(selected) == perBig {
				break
			}
			if usedSmall[candidate.SmallCategoryID] {
				continue
			}
			selected = append(selected, candidate)
			usedSmall[candidate.SmallCategoryID] = true
		}
		for _, candidate := range candidates {
			if len(selected) == perBig {
				break
			}
			already := false
			for _, chosen := range selected {
				if chosen.Item.ID == candidate.Item.ID {
					already = true
					break
				}
			}
			if !already {
				selected = append(selected, candidate)
			}
		}
		if len(selected) != perBig {
			return SpotTable{}, fmt.Errorf("acceptance: could not select %d items for %s", perBig, big.ID)
		}
		spot.BigCategories = append(spot.BigCategories, SpotBigCategory{
			ID: big.ID, Name: big.Name, Items: selected,
		})
	}
	return spot, nil
}

func spotPriority(item ManualItem) int {
	if len(item.Points) > 0 {
		return 0
	}
	switch item.Inspection {
	case InspectionAutomaticAndManual, InspectionPoint, InspectionRoute,
		InspectionRegion, InspectionRelation, InspectionUI:
		return 1
	default:
		return 2
	}
}

func stableSpotHash(identity, bigID, itemID string) string {
	sum := sha256.Sum256([]byte(identity + "|" + bigID + "|" + itemID))
	return hex.EncodeToString(sum[:])
}

func ManualTableJSON(table ManualTable) ([]byte, error) {
	data, err := json.MarshalIndent(table, "", "  ")
	if err == nil {
		data = append(data, '\n')
	}
	return data, err
}

func SpotTableJSON(table SpotTable) ([]byte, error) {
	data, err := json.MarshalIndent(table, "", "  ")
	if err == nil {
		data = append(data, '\n')
	}
	return data, err
}

func ManualTableMarkdown(table ManualTable) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# GeoWorld 完整人工验收表\n\n")
	fmt.Fprintf(&out, "- Seed: %d\n- World Profile: %s\n- Generation Rules: %s\n- World Model: %s\n- Preset: %s\n- Registry: %s\n\n",
		table.Seed, table.WorldProfileVersion, table.GenerationRulesVersion,
		table.WorldModelVersion, table.Preset, table.RegistryVersion)
	for _, big := range table.BigCategories {
		fmt.Fprintf(&out, "## %s · %s\n\n", big.ID, big.Name)
		for _, small := range big.SmallCategories {
			fmt.Fprintf(&out, "### %s · %s\n\n", small.ID, small.Name)
			fmt.Fprintf(&out, "| ID | 具体条目 | 验收方式 | Seed / TP | 通过标准 |\n|---|---|---|---|---|\n")
			for _, item := range small.Items {
				fmt.Fprintf(&out, "| %s | %s | %s | %s | %s |\n",
					md(item.ID), md(item.Name), md(item.Method),
					md(seedTP(table.Seed, item.Points)), md(item.Expected))
			}
			out.WriteString("\n")
		}
	}
	return out.String()
}

func SpotTableMarkdown(table SpotTable) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# GeoWorld 人工抽检表\n\n")
	fmt.Fprintf(&out, "- 来源: 完整验收表 %s v%d\n- Seed: %d\n- Generation Rules: %s\n- 每个大类抽检: %d 条\n\n",
		table.SourceFormat, table.SourceVersion, table.Seed,
		table.GenerationRulesVersion, table.PerBigCategory)
	fmt.Fprintf(&out, "| 大类 | 小类 | 具体条目 | Seed / TP | 简单描述 |\n|---|---|---|---|---|\n")
	for _, big := range table.BigCategories {
		for _, selected := range big.Items {
			item := selected.Item
			desc := item.Expected
			if item.Note != "" {
				desc += "；" + item.Note
			}
			fmt.Fprintf(&out, "| %s | %s | %s · %s | %s | %s |\n",
				md(big.Name), md(selected.SmallCategoryName), md(item.ID), md(item.Name),
				md(seedTP(table.Seed, item.Points)), md(desc))
		}
	}
	return out.String()
}

func WriteManualTableCSV(w io.Writer, table ManualTable) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()
	if err := cw.Write([]string{"大类", "小类", "ID", "具体条目", "验收方式", "Seed", "TP/路线", "通过标准", "状态"}); err != nil {
		return err
	}
	for _, big := range table.BigCategories {
		for _, small := range big.SmallCategories {
			for _, item := range small.Items {
				if err := cw.Write([]string{
					big.Name, small.Name, item.ID, item.Name, item.Method,
					strconv.FormatInt(table.Seed, 10), joinTP(item.Points),
					item.Expected, string(item.Status),
				}); err != nil {
					return err
				}
			}
		}
	}
	return cw.Error()
}

func WriteSpotTableCSV(w io.Writer, table SpotTable) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()
	if err := cw.Write([]string{"大类", "小类", "ID", "具体条目", "Seed", "TP/路线", "简单描述"}); err != nil {
		return err
	}
	for _, big := range table.BigCategories {
		for _, selected := range big.Items {
			item := selected.Item
			if err := cw.Write([]string{
				big.Name, selected.SmallCategoryName, item.ID, item.Name,
				strconv.FormatInt(table.Seed, 10), joinTP(item.Points), item.Expected,
			}); err != nil {
				return err
			}
		}
	}
	return cw.Error()
}

func joinTP(points []Point) string {
	if len(points) == 0 {
		return "无 TP；按验收方式/自动检查执行"
	}
	tps := make([]string, 0, len(points))
	for _, point := range points {
		tps = append(tps, point.TP)
	}
	return strings.Join(tps, " | ")
}

func seedTP(seed int64, points []Point) string {
	return fmt.Sprintf("Seed %d；%s", seed, joinTP(points))
}

func md(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", "<br>")
	return s
}
