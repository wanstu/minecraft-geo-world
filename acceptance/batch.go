package acceptance

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var FixedAcceptanceSeeds = []int64{
	123456,
	-4852173505788415804,
	-563659577384754347,
	4034658472718554561,
}

var RepresentativePresets = []string{
	"standard",
	"natural",
	"civilization",
	"epic",
}

// StressProfiles are acceptance-only parameter snapshots. They exercise the
// v1.1 requirement that the same causal model survives both large-scale and
// small-scale spatial pressure; they are not additional public generators.
var StressProfiles = []string{
	"stress_large",
	"stress_small",
}

func RequiredAcceptanceProfiles() []string {
	out := make([]string, 0, len(RepresentativePresets)+len(StressProfiles))
	out = append(out, RepresentativePresets...)
	out = append(out, StressProfiles...)
	return out
}

type BatchCoverageStatus string

const (
	BatchCovered          BatchCoverageStatus = "COVERED"
	BatchVerified         BatchCoverageStatus = "VERIFIED"
	BatchMissingSample    BatchCoverageStatus = "MISSING_SAMPLE"
	BatchNotImplemented   BatchCoverageStatus = "NOT_IMPLEMENTED"
	BatchVerifierRequired BatchCoverageStatus = "VERIFIER_REQUIRED"
	BatchMissingMatrix    BatchCoverageStatus = "MISSING_MATRIX"
)

type BatchModel struct {
	Seed                    int64    `json:"seed"`
	SeedText                string   `json:"seedText"`
	Preset                  string   `json:"preset"`
	WorldProfileVersion     string   `json:"worldProfileVersion"`
	GenerationRulesVersion  string   `json:"generationRulesVersion"`
	WorldModelVersion       string   `json:"worldModelVersion"`
	ParameterSnapshotSHA256 string   `json:"parameterSnapshotSha256"`
	FailureIDs              []string `json:"failureIds,omitempty"`
}

type BatchMissingModel struct {
	Seed     int64  `json:"seed"`
	SeedText string `json:"seedText"`
	Preset   string `json:"preset"`
}

type BatchResult struct {
	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	Requirement         RequirementKind     `json:"requirement"`
	Inspection          InspectionKind      `json:"inspection"`
	CoverageStatus      BatchCoverageStatus `json:"coverageStatus"`
	ModelCount          int                 `json:"modelCount"`
	CandidateModelCount int                 `json:"candidateModelCount"`
	MissingSamples      []BatchMissingModel `json:"missingSamples,omitempty"`
	Note                string              `json:"note,omitempty"`
}

type BatchReport struct {
	RegistryVersion        string            `json:"registryVersion"`
	RequiredPresets        []string          `json:"requiredPresets"`
	RequiredStressProfiles []string          `json:"requiredStressProfiles"`
	FixedSeeds             []int64           `json:"fixedSeeds"`
	FixedSeedTexts         []string          `json:"fixedSeedTexts"`
	RandomSeed             *int64            `json:"randomSeed,omitempty"`
	RandomSeedText         string            `json:"randomSeedText,omitempty"`
	MatrixComplete         bool              `json:"matrixComplete"`
	MissingCombinations    []string          `json:"missingCombinations,omitempty"`
	Anomalies              []string          `json:"anomalies,omitempty"`
	Models                 []BatchModel      `json:"models"`
	Results                []BatchResult     `json:"results"`
	VerifierEvidence       *VerifierEvidence `json:"verifierEvidence,omitempty"`
}

func BuildBatch(reports []Report) (BatchReport, error) {
	if len(reports) == 0 {
		return BatchReport{}, fmt.Errorf("acceptance: at least one report is required")
	}
	entries, err := Entries()
	if err != nil {
		return BatchReport{}, err
	}

	out := BatchReport{
		RegistryVersion:        reports[0].RegistryVersion,
		RequiredPresets:        append([]string(nil), RepresentativePresets...),
		RequiredStressProfiles: append([]string(nil), StressProfiles...),
		FixedSeeds:             append([]int64(nil), FixedAcceptanceSeeds...),
		Models:                 make([]BatchModel, 0, len(reports)),
		Results:                make([]BatchResult, 0, len(entries)),
	}
	for _, seed := range FixedAcceptanceSeeds {
		out.FixedSeedTexts = append(out.FixedSeedTexts, strconv.FormatInt(seed, 10))
	}

	type comboKey struct {
		preset string
		seed   int64
	}
	seen := make(map[comboKey]int)
	presetSeeds := make(map[string]map[int64]bool)
	presetSnapshot := make(map[string]string)
	resultByModel := make([]map[string]Result, 0, len(reports))
	expectedWorldProfileVersion := reports[0].WorldProfileVersion
	expectedGenerationRulesVersion := reports[0].GenerationRulesVersion
	expectedWorldModelVersion := reports[0].WorldModelVersion
	identityComplete := true
	sharedModelContract := true

	for index, report := range reports {
		if report.RegistryVersion != out.RegistryVersion {
			out.Anomalies = append(out.Anomalies, fmt.Sprintf(
				"registry version mismatch: model %d has %s, expected %s",
				index+1,
				report.RegistryVersion,
				out.RegistryVersion,
			))
		}
		if len(report.Results) != len(entries) {
			out.Anomalies = append(out.Anomalies, fmt.Sprintf(
				"%s/%s has %d acceptance items, expected %d",
				report.Preset,
				strconv.FormatInt(report.Seed, 10),
				len(report.Results),
				len(entries),
			))
		}
		if report.WorldProfileVersion == "" ||
			report.GenerationRulesVersion == "" ||
			report.WorldModelVersion == "" ||
			report.Preset == "" ||
			report.ParameterSnapshotSHA256 == "" {
			identityComplete = false
		}
		if report.WorldProfileVersion != expectedWorldProfileVersion ||
			report.GenerationRulesVersion != expectedGenerationRulesVersion ||
			report.WorldModelVersion != expectedWorldModelVersion {
			sharedModelContract = false
		}

		key := comboKey{preset: report.Preset, seed: report.Seed}
		seen[key]++
		if seen[key] > 1 {
			out.Anomalies = append(out.Anomalies, fmt.Sprintf(
				"duplicate model for preset=%s seed=%s",
				report.Preset,
				strconv.FormatInt(report.Seed, 10),
			))
		}
		if presetSeeds[report.Preset] == nil {
			presetSeeds[report.Preset] = make(map[int64]bool)
		}
		presetSeeds[report.Preset][report.Seed] = true

		if previous, ok := presetSnapshot[report.Preset]; ok &&
			previous != report.ParameterSnapshotSHA256 {
			out.Anomalies = append(out.Anomalies, fmt.Sprintf(
				"preset=%s uses inconsistent parameter snapshots: %s vs %s",
				report.Preset,
				previous,
				report.ParameterSnapshotSHA256,
			))
		} else if !ok {
			presetSnapshot[report.Preset] = report.ParameterSnapshotSHA256
		}

		failures := make([]string, 0)
		indexed := make(map[string]Result, len(report.Results))
		for _, result := range report.Results {
			indexed[result.ID] = result
			if result.Status == StatusFail {
				failures = append(failures, result.ID)
			}
		}
		sort.Strings(failures)
		resultByModel = append(resultByModel, indexed)
		out.Models = append(out.Models, BatchModel{
			Seed:                    report.Seed,
			SeedText:                strconv.FormatInt(report.Seed, 10),
			Preset:                  report.Preset,
			WorldProfileVersion:     report.WorldProfileVersion,
			GenerationRulesVersion:  report.GenerationRulesVersion,
			WorldModelVersion:       report.WorldModelVersion,
			ParameterSnapshotSHA256: report.ParameterSnapshotSHA256,
			FailureIDs:              failures,
		})
	}

	sort.Slice(out.Models, func(i, j int) bool {
		if out.Models[i].Preset != out.Models[j].Preset {
			return out.Models[i].Preset < out.Models[j].Preset
		}
		return out.Models[i].Seed < out.Models[j].Seed
	})

	fixed := make(map[int64]bool, len(FixedAcceptanceSeeds))
	for _, seed := range FixedAcceptanceSeeds {
		fixed[seed] = true
	}
	requiredProfiles := RequiredAcceptanceProfiles()
	var sharedRandom []int64
	for seed := range presetSeeds[requiredProfiles[0]] {
		if fixed[seed] {
			continue
		}
		presentEverywhere := true
		for _, profile := range requiredProfiles[1:] {
			if !presetSeeds[profile][seed] {
				presentEverywhere = false
				break
			}
		}
		if presentEverywhere {
			sharedRandom = append(sharedRandom, seed)
		}
	}
	sort.Slice(sharedRandom, func(i, j int) bool { return sharedRandom[i] < sharedRandom[j] })
	if len(sharedRandom) > 0 {
		randomSeed := sharedRandom[0]
		out.RandomSeed = &randomSeed
		out.RandomSeedText = strconv.FormatInt(randomSeed, 10)
	}

	for _, profile := range requiredProfiles {
		for _, seed := range FixedAcceptanceSeeds {
			if !presetSeeds[profile][seed] {
				out.MissingCombinations = append(out.MissingCombinations, fmt.Sprintf(
					"%s/%s",
					profile,
					strconv.FormatInt(seed, 10),
				))
			}
		}
	}
	if out.RandomSeed == nil {
		out.MissingCombinations = append(
			out.MissingCombinations,
			"shared-random-seed across representative presets and scale stress profiles",
		)
	} else {
		for _, profile := range requiredProfiles {
			if !presetSeeds[profile][*out.RandomSeed] {
				out.MissingCombinations = append(out.MissingCombinations, fmt.Sprintf(
					"%s/%s",
					profile,
					out.RandomSeedText,
				))
			}
		}
	}
	out.MatrixComplete = len(out.MissingCombinations) == 0 && len(out.Anomalies) == 0

	for _, entry := range entries {
		item := BatchResult{
			ID:          entry.ID,
			Name:        entry.Name,
			Requirement: entry.Requirement,
			Inspection:  entry.Inspection,
			ModelCount:  len(reports),
		}
		implementedLater := entry.Requirement == RequirementLater &&
			laterFeatureImplemented(entry.ID)
		if entry.Requirement == RequirementLater && !implementedLater {
			item.CoverageStatus = BatchNotImplemented
			item.Note = "v1.1 Later 项；保持 NOT_IMPLEMENTED。"
			out.Results = append(out.Results, item)
			continue
		}
		if entry.ID == "PARAM-09" {
			if out.MatrixComplete {
				item.CoverageStatus = BatchCovered
				item.Note = "4 个代表预设 + 2 个尺度压力配置 × 固定 4 Seed × 共享随机 Seed 矩阵完整；功能结论仍需各项自动/人工验收。"
			} else {
				item.CoverageStatus = BatchMissingMatrix
				item.Note = "代表预设 + 大/小尺度压力配置完整回归矩阵尚不完整。"
			}
			out.Results = append(out.Results, item)
			continue
		}
		if entry.ID == "PROFILE-05" {
			if identityComplete {
				item.CoverageStatus = BatchVerified
				item.Note = "批量验收器已对每个模型记录 Seed、Profile/Rules/Model 版本、Preset 与参数快照哈希。"
			} else {
				item.CoverageStatus = BatchVerifierRequired
				item.Note = "存在模型身份或参数快照信息不完整。"
			}
			out.Results = append(out.Results, item)
			continue
		}
		if entry.ID == "PARAM-08" {
			if out.MatrixComplete && identityComplete && sharedModelContract {
				item.CoverageStatus = BatchVerified
				item.Note = fmt.Sprintf(
					"4 个官方代表预设 × 5 Seed 共用 World Model %s / Rules %s；差异由各预设参数快照表达。",
					expectedWorldModelVersion,
					expectedGenerationRulesVersion,
				)
			} else {
				item.CoverageStatus = BatchVerifierRequired
				item.Note = "官方预设矩阵尚不能证明共享同一 World Model/Rules 契约。"
			}
			out.Results = append(out.Results, item)
			continue
		}
		if entry.ID == "PARAM-03" {
			item.CoverageStatus = BatchVerifierRequired
			item.Note = "本轮需人工视觉验收：参数调高/调低后，预览与实际世界效果应稳定、方向正确且肉眼可辨识。"
			out.Results = append(out.Results, item)
			continue
		}
		if entry.ID == "PROFILE-08" {
			item.CoverageStatus = BatchVerifierRequired
			item.Note = "该项由同 Minecraft 版本的 Fabric/NeoForge 构建分别执行锁定的 GeoWorld biome-decision fingerprint；批量 World Model 坐标矩阵本身不替代 Loader 侧自动门禁。"
			out.Results = append(out.Results, item)
			continue
		}
		if !entryUsesWorldSamples(entry) {
			item.CoverageStatus = BatchVerifierRequired
			item.Note = "该项不以世界坐标候选覆盖判断，需对应架构/API/性能或专用 verifier。"
			out.Results = append(out.Results, item)
			continue
		}

		for i, report := range reports {
			result, ok := resultByModel[i][entry.ID]
			if !ok {
				if !implementedLater {
					item.MissingSamples = append(item.MissingSamples, BatchMissingModel{
						Seed:     report.Seed,
						SeedText: strconv.FormatInt(report.Seed, 10),
						Preset:   report.Preset,
					})
				}
				continue
			}
			if len(result.Points) > 0 {
				item.CandidateModelCount++
			}
			if !implementedLater && result.Status == StatusFail &&
				strings.Contains(result.Note, "MISSING_SAMPLE") {
				item.MissingSamples = append(item.MissingSamples, BatchMissingModel{
					Seed:     report.Seed,
					SeedText: strconv.FormatInt(report.Seed, 10),
					Preset:   report.Preset,
				})
			}
		}
		if implementedLater {
			if item.CandidateModelCount > 0 {
				item.CoverageStatus = BatchCovered
				item.Note = fmt.Sprintf(
					"后期功能已实现；代表矩阵中 %d/%d 个模型实例化了可定位候选。稀有功能允许部分 Seed 缺席。",
					item.CandidateModelCount,
					item.ModelCount,
				)
			} else {
				item.CoverageStatus = BatchVerifierRequired
				item.Note = "后期功能代码已实现，但当前代表矩阵未实例化任何可定位候选；需要扩大 Seed 矩阵或检查候选定位器。"
			}
		} else if len(item.MissingSamples) > 0 {
			item.CoverageStatus = BatchMissingSample
			item.Note = "至少一个模型缺少该普通验收项的候选样本。"
		} else {
			item.CoverageStatus = BatchCovered
			item.Note = "全部已提供模型均有该验收项候选样本；仍需按通过标准进行自动/人工判定。"
		}
		out.Results = append(out.Results, item)
	}

	return out, nil
}
