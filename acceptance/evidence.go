package acceptance

import (
	"fmt"
	"sort"
)

const (
	VerifierEvidenceFormat         = "geoworld-model-verifier-evidence"
	CombinedVerifierEvidenceFormat = "geoworld-verifier-evidence"
)

type VerifierEvidence struct {
	Format     string            `json:"format"`
	Version    int               `json:"version"`
	ModelCount int               `json:"modelCount"`
	PassedIDs  []string          `json:"passedIds"`
	Notes      map[string]string `json:"notes,omitempty"`
}

func (report *BatchReport) ApplyVerifierEvidence(evidence VerifierEvidence) error {
	if report == nil {
		return fmt.Errorf("acceptance: batch report is required")
	}
	if (evidence.Format != VerifierEvidenceFormat &&
		evidence.Format != CombinedVerifierEvidenceFormat) ||
		evidence.Version != 1 {
		return fmt.Errorf(
			"acceptance: unsupported verifier evidence %q v%d",
			evidence.Format,
			evidence.Version,
		)
	}
	if evidence.ModelCount != len(report.Models) {
		return fmt.Errorf(
			"acceptance: verifier evidence model count=%d want=%d",
			evidence.ModelCount,
			len(report.Models),
		)
	}

	index := make(map[string]int, len(report.Results))
	for i := range report.Results {
		index[report.Results[i].ID] = i
	}
	seen := make(map[string]bool, len(evidence.PassedIDs))
	passed := append([]string(nil), evidence.PassedIDs...)
	sort.Strings(passed)
	for _, id := range passed {
		if seen[id] {
			return fmt.Errorf("acceptance: duplicate verifier evidence ID %s", id)
		}
		seen[id] = true
		i, ok := index[id]
		if !ok {
			return fmt.Errorf("acceptance: verifier evidence references unknown ID %s", id)
		}
		item := &report.Results[i]
		if item.CoverageStatus != BatchVerifierRequired {
			return fmt.Errorf(
				"acceptance: verifier evidence ID %s has incompatible status %s",
				id,
				item.CoverageStatus,
			)
		}
		item.CoverageStatus = BatchVerified
		if note := evidence.Notes[id]; note != "" {
			item.Note = note
		} else if evidence.Format == VerifierEvidenceFormat {
			item.Note = fmt.Sprintf(
				"专用黑盒 verifier 已在 %d 个模型上通过。",
				evidence.ModelCount,
			)
		} else {
			item.Note = "专用自动 verifier/test 已通过。"
		}
	}
	report.VerifierEvidence = &VerifierEvidence{
		Format:     evidence.Format,
		Version:    evidence.Version,
		ModelCount: evidence.ModelCount,
		PassedIDs:  passed,
		Notes:      cloneEvidenceNotes(evidence.Notes),
	}
	return nil
}

func cloneEvidenceNotes(notes map[string]string) map[string]string {
	if len(notes) == 0 {
		return nil
	}
	out := make(map[string]string, len(notes))
	for id, note := range notes {
		out[id] = note
	}
	return out
}
