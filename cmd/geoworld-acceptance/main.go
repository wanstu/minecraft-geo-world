package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"geoworld/acceptance"
	"geoworld/internal/worldmodelio"
)

func main() {
	modelPath := flag.String("model", "", "locked .geoworld model path")
	modelsDir := flag.String("models-dir", "", "recursively aggregate all .geoworld models under this directory")
	format := flag.String("format", "markdown", "output format: markdown or json")
	output := flag.String("output", "", "optional output file; stdout when omitted")
	verifierEvidence := flag.String(
		"verifier-evidence",
		"",
		"optional machine-readable verifier evidence JSON for batch reports",
	)
	flag.Parse()

	if (*modelPath == "") == (*modelsDir == "") {
		fatal(fmt.Errorf("exactly one of -model or -models-dir is required"))
	}
	if *modelsDir != "" {
		runBatch(*modelsDir, *format, *output, *verifierEvidence)
		return
	}
	if *verifierEvidence != "" {
		fatal(fmt.Errorf("-verifier-evidence is only supported with -models-dir"))
	}

	model, err := worldmodelio.Load(*modelPath)
	if err != nil {
		fatal(err)
	}
	report, err := acceptance.Plan(model)
	if err != nil {
		fatal(err)
	}

	var data []byte
	switch strings.ToLower(strings.TrimSpace(*format)) {
	case "json":
		data, err = json.MarshalIndent(report, "", "  ")
		if err == nil {
			data = append(data, '\n')
		}
	case "markdown", "md":
		data = []byte(markdown(report))
	default:
		err = fmt.Errorf("unsupported -format %q", *format)
	}
	if err != nil {
		fatal(err)
	}
	if *output == "" {
		_, _ = os.Stdout.Write(data)
		return
	}
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("acceptance report: %s (%d items)\n", *output, len(report.Results))
}

func markdown(report acceptance.Report) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# GeoWorld v1.1 验收清单\n\n")
	fmt.Fprintf(&out, "- Seed: %d\n", report.Seed)
	fmt.Fprintf(&out, "- World Profile: %s\n", report.WorldProfileVersion)
	fmt.Fprintf(&out, "- Generation Rules: %s\n", report.GenerationRulesVersion)
	fmt.Fprintf(&out, "- World Model: %s\n", report.WorldModelVersion)
	fmt.Fprintf(&out, "- Preset: %s\n", report.Preset)
	fmt.Fprintf(&out, "- Parameter Snapshot SHA-256: %s\n", report.ParameterSnapshotSHA256)
	fmt.Fprintf(&out, "- Acceptance Registry: %s\n\n", report.RegistryVersion)

	for _, result := range report.Results {
		fmt.Fprintf(
			&out,
			"## %s · %s · %s\n\n",
			result.ID,
			result.Name,
			result.Status,
		)
		fmt.Fprintf(&out, "- 类型：%s / %s\n", result.Requirement, result.Inspection)
		fmt.Fprintf(&out, "- 文档验收方式：%s\n", result.Method)
		fmt.Fprintf(
			&out,
			"- 验收配置：Seed %d / Profile %s / Preset %s / Snapshot %s\n",
			report.Seed,
			report.WorldProfileVersion,
			report.Preset,
			report.ParameterSnapshotSHA256,
		)
		fmt.Fprintf(&out, "- 观察：%s\n", result.Observe)
		fmt.Fprintf(&out, "- 通过：%s\n", result.Expected)
		fmt.Fprintf(&out, "- 失败：%s\n", result.Failure)
		if result.Note != "" {
			fmt.Fprintf(&out, "- 说明：%s\n", result.Note)
		}
		if len(result.Points) == 0 {
			if result.Status == acceptance.StatusFail {
				fmt.Fprintf(&out, "- 坐标：未定位到人工样本；这是当前验收失败的一部分。\n\n")
			} else {
				fmt.Fprintf(&out, "- 坐标：此项由架构/API/性能检查或专用 Minecraft Verifier 验证。\n\n")
			}
			continue
		}
		if result.Inspection == acceptance.InspectionRoute {
			roles := make([]string, 0, len(result.Points))
			for _, point := range result.Points {
				roles = append(roles, point.Role)
			}
			fmt.Fprintf(&out, "- 观察顺序：%s\n", strings.Join(roles, " → "))
		}
		fmt.Fprintf(&out, "- 候选坐标：\n")
		for i, point := range result.Points {
			fmt.Fprintf(
				&out,
				"  %d. [%s] %s — %s；观察「%s」；Water=%s / Biome=%s / Terrain=%s\n",
				i+1,
				point.Role,
				point.TP,
				point.Reason,
				result.Name,
				point.Water,
				point.Biome,
				point.Terrain,
			)
		}
		out.WriteString("\n")
	}
	return out.String()
}

func runBatch(modelsDir, format, output, verifierEvidence string) {
	paths := make([]string, 0, 32)
	err := filepath.WalkDir(modelsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".geoworld") {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		fatal(err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		fatal(fmt.Errorf("no .geoworld models found under %s", modelsDir))
	}

	reports := make([]acceptance.Report, 0, len(paths))
	for _, path := range paths {
		model, loadErr := worldmodelio.Load(path)
		if loadErr != nil {
			fatal(fmt.Errorf("%s: %w", path, loadErr))
		}
		report, planErr := acceptance.Plan(model)
		if planErr != nil {
			fatal(fmt.Errorf("%s: %w", path, planErr))
		}
		reports = append(reports, report)
	}

	batch, err := acceptance.BuildBatch(reports)
	if err != nil {
		fatal(err)
	}
	if verifierEvidence != "" {
		rawEvidence, readErr := os.ReadFile(verifierEvidence)
		if readErr != nil {
			fatal(readErr)
		}
		var evidence acceptance.VerifierEvidence
		if decodeErr := json.Unmarshal(rawEvidence, &evidence); decodeErr != nil {
			fatal(fmt.Errorf("decode verifier evidence: %w", decodeErr))
		}
		if applyErr := batch.ApplyVerifierEvidence(evidence); applyErr != nil {
			fatal(applyErr)
		}
	}
	var data []byte
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		data, err = json.MarshalIndent(batch, "", "  ")
		if err == nil {
			data = append(data, '\n')
		}
	case "markdown", "md":
		data = []byte(batchMarkdown(batch))
	default:
		err = fmt.Errorf("unsupported -format %q", format)
	}
	if err != nil {
		fatal(err)
	}
	if output == "" {
		_, _ = os.Stdout.Write(data)
		return
	}
	if err := os.WriteFile(output, data, 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf(
		"batch acceptance report: %s (%d models, %d items)\n",
		output,
		len(batch.Models),
		len(batch.Results),
	)
}

func batchMarkdown(report acceptance.BatchReport) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# GeoWorld v1.1 批量验收覆盖报告\n\n")
	fmt.Fprintf(&out, "- Acceptance Registry: %s\n", report.RegistryVersion)
	fmt.Fprintf(&out, "- 模型数: %d\n", len(report.Models))
	fmt.Fprintf(&out, "- 代表预设: %s\n", strings.Join(report.RequiredPresets, ", "))
	fmt.Fprintf(&out, "- 固定 Seed: %s\n", strings.Join(report.FixedSeedTexts, ", "))
	if report.RandomSeed != nil {
		fmt.Fprintf(&out, "- 本轮共享随机 Seed: %s\n", report.RandomSeedText)
	} else {
		fmt.Fprintf(&out, "- 本轮共享随机 Seed: 缺失\n")
	}
	fmt.Fprintf(&out, "- 代表预设回归矩阵完整: %t\n", report.MatrixComplete)
	if report.VerifierEvidence != nil {
		fmt.Fprintf(
			&out,
			"- Verifier Evidence: %s v%d / %d models / %d IDs\n",
			report.VerifierEvidence.Format,
			report.VerifierEvidence.Version,
			report.VerifierEvidence.ModelCount,
			len(report.VerifierEvidence.PassedIDs),
		)
	}
	out.WriteString("\n")

	if len(report.MissingCombinations) > 0 {
		fmt.Fprintf(&out, "## 缺失组合\n\n")
		for _, missing := range report.MissingCombinations {
			fmt.Fprintf(&out, "- %s\n", missing)
		}
		out.WriteString("\n")
	}
	if len(report.Anomalies) > 0 {
		fmt.Fprintf(&out, "## 异常\n\n")
		for _, anomaly := range report.Anomalies {
			fmt.Fprintf(&out, "- %s\n", anomaly)
		}
		out.WriteString("\n")
	}

	counts := make(map[acceptance.BatchCoverageStatus]int)
	for _, result := range report.Results {
		counts[result.CoverageStatus]++
	}
	fmt.Fprintf(&out, "## 覆盖摘要\n\n")
	for _, status := range []acceptance.BatchCoverageStatus{
		acceptance.BatchCovered,
		acceptance.BatchVerified,
		acceptance.BatchMissingSample,
		acceptance.BatchVerifierRequired,
		acceptance.BatchNotImplemented,
		acceptance.BatchMissingMatrix,
	} {
		fmt.Fprintf(&out, "- %s: %d\n", status, counts[status])
	}
	out.WriteString("\n")

	fmt.Fprintf(&out, "## 模型矩阵\n\n")
	for _, model := range report.Models {
		fmt.Fprintf(
			&out,
			"- %s / Seed %s / Snapshot %s",
			model.Preset,
			model.SeedText,
			model.ParameterSnapshotSHA256,
		)
		if len(model.FailureIDs) > 0 {
			fmt.Fprintf(&out, " / FAIL: %s", strings.Join(model.FailureIDs, ", "))
		}
		out.WriteString("\n")
	}
	out.WriteString("\n")

	fmt.Fprintf(&out, "## 177 项自动覆盖分析\n\n")
	for _, result := range report.Results {
		fmt.Fprintf(
			&out,
			"- %s · %s — %s",
			result.ID,
			result.Name,
			result.CoverageStatus,
		)
		if result.ModelCount > 0 && result.CandidateModelCount > 0 {
			fmt.Fprintf(
				&out,
				"（候选覆盖 %d/%d 模型）",
				result.CandidateModelCount,
				result.ModelCount,
			)
		}
		if len(result.MissingSamples) > 0 {
			missing := make([]string, 0, len(result.MissingSamples))
			for _, model := range result.MissingSamples {
				missing = append(missing, model.Preset+"/"+model.SeedText)
			}
			fmt.Fprintf(&out, "；缺样本 %s", strings.Join(missing, ", "))
		}
		if result.Note != "" {
			fmt.Fprintf(&out, "；%s", result.Note)
		}
		out.WriteString("\n")
	}
	return out.String()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "geoworld-acceptance:", err)
	os.Exit(1)
}
