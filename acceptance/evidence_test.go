package acceptance

import "testing"

func TestCombinedVerifierEvidenceKeepsPerIDNotes(t *testing.T) {
	report := BatchReport{
		Models: []BatchModel{{}, {}},
		Results: []BatchResult{
			{ID: "PARAM-01", CoverageStatus: BatchVerifierRequired},
			{ID: "PROFILE-03", CoverageStatus: BatchVerifierRequired},
		},
	}
	evidence := VerifierEvidence{
		Format:     CombinedVerifierEvidenceFormat,
		Version:    1,
		ModelCount: 2,
		PassedIDs:  []string{"PROFILE-03", "PARAM-01"},
		Notes: map[string]string{
			"PARAM-01":   "creation verifier pass",
			"PROFILE-03": "preview verifier pass",
		},
	}
	if err := report.ApplyVerifierEvidence(evidence); err != nil {
		t.Fatal(err)
	}
	for _, result := range report.Results {
		if result.CoverageStatus != BatchVerified {
			t.Fatalf("%s status=%s want=%s", result.ID, result.CoverageStatus, BatchVerified)
		}
		if result.Note != evidence.Notes[result.ID] {
			t.Fatalf("%s note=%q want=%q", result.ID, result.Note, evidence.Notes[result.ID])
		}
	}
	if report.VerifierEvidence == nil ||
		report.VerifierEvidence.Format != CombinedVerifierEvidenceFormat ||
		len(report.VerifierEvidence.PassedIDs) != 2 {
		t.Fatalf("stored evidence=%+v", report.VerifierEvidence)
	}

	evidence.Notes["PARAM-01"] = "mutated"
	if report.VerifierEvidence.Notes["PARAM-01"] != "creation verifier pass" {
		t.Fatal("stored verifier evidence notes were not cloned")
	}
}

func TestLegacyModelVerifierEvidenceStillUsesModelCountNote(t *testing.T) {
	report := BatchReport{
		Models:  []BatchModel{{}, {}, {}},
		Results: []BatchResult{{ID: "MODEL-01", CoverageStatus: BatchVerifierRequired}},
	}
	if err := report.ApplyVerifierEvidence(VerifierEvidence{
		Format:     VerifierEvidenceFormat,
		Version:    1,
		ModelCount: 3,
		PassedIDs:  []string{"MODEL-01"},
	}); err != nil {
		t.Fatal(err)
	}
	if got := report.Results[0].Note; got != "专用黑盒 verifier 已在 3 个模型上通过。" {
		t.Fatalf("legacy note=%q", got)
	}
}
