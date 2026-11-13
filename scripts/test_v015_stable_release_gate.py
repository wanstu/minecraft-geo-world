#!/usr/bin/env python3
"""v0.1.5 stable release scope / CI safety gate, independent of experimental branches."""
from pathlib import Path
import re
import unittest

ROOT=Path(__file__).resolve().parent.parent
WORKFLOW=(ROOT/".github/workflows/release.yml").read_text(encoding="utf-8")
PROFILE=(ROOT/"internal/generator/profile.go").read_text(encoding="utf-8")
ENGINE=(ROOT/"java/minecraft-common/src/main/java/geoworld/minecraft/common/engine/GeoWorldEngineBridge.java").read_text(encoding="utf-8")

class StableReleaseGate(unittest.TestCase):
    def test_stable_rules_not_unreviewed_sinkhole(self):
        self.assertRegex(PROFILE,r'CurrentGenerationRulesVersion\s*=\s*"geoworld-rules-v1\.1\.22"')
        self.assertRegex(ENGINE,r'EXPECTED_GENERATION_RULES_VERSION\s*=\s*"geoworld-rules-v1\.1\.22"')
        self.assertNotIn("sinkholeCompoundReliefRadius", (ROOT/"internal/specialfeatures/model.go").read_text())
        self.assertNotIn("sinkholeFlowAlignedTransfer", (ROOT/"internal/specialfeatures/model.go").read_text())
        self.assertNotIn("geoworld-rules-v1.1.31",PROFILE)
        self.assertNotIn("geoworld-rules-v1.1.30",PROFILE)

    def test_ci_matrix_is_same_nine_targets(self):
        def matrix(job, next_job):
            section=WORKFLOW.split(f"  {job}:\n",1)[1].split(f"  {next_job}:\n",1)[0]
            return re.findall(
                r"\{ minecraft: '([^']+)', loader: ([a-z]+), java: '([0-9]+)', module: '([^']+)' \}",section)
        verify=matrix("implemented-adapters","release-build")
        release=matrix("release-build","publish")
        self.assertEqual(len(verify),9)
        self.assertEqual(len(set(verify)),9)
        self.assertEqual(verify,release)
        self.assertIn(("1.20.1","fabric","17","java/adapters/1.20.1-fabric"),verify)
        self.assertIn(("1.21.1","neoforge","21","java/adapters/1.21.1-neoforge"),verify)
        self.assertIn(("26.3","neoforge","25","java/neoforge-mod"),verify)
        for _,_,_,module in verify:
            self.assertTrue((ROOT/module/"gradle.properties").is_file(),module)

    def test_manual_and_pr_never_publish_or_trigger_release_build(self):
        self.assertIn("if: ${{ github.event_name != 'push' || !startsWith(github.ref, 'refs/tags/v') }}",WORKFLOW)
        self.assertEqual(WORKFLOW.count("if: github.event_name == 'push' && startsWith(github.ref, 'refs/tags/v')"),2)
        self.assertIn("branches: [master]",WORKFLOW)
        self.assertIn("tags: ['v*']",WORKFLOW)
        self.assertIn("workflow_dispatch:",WORKFLOW)
        self.assertIn("pull_request:",WORKFLOW)

    def test_published_files_are_built_from_tagged_master_with_version(self):
        self.assertIn("Release tag must point to current master.",WORKFLOW)
        self.assertIn("-Pmod_version=$env:MOD_VERSION",WORKFLOW)
        self.assertIn("Verify Java sampling against all committed Go golden models",WORKFLOW)
        self.assertIn("Test all GeoWorld Go packages",WORKFLOW)

if __name__=="__main__":
    unittest.main()
