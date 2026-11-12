#!/usr/bin/env python3
"""Guard GeoWorld's non-publishing Windows verification path.

Static workflow contract checks, not evidence that GitHub Windows CI has run.
"""
from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parent.parent
WORKFLOW = (ROOT / ".github/workflows/release.yml").read_text(encoding="utf-8")
TARGET_PATTERN = re.compile(
    r"- \{ minecraft: '([^']+)', loader: (fabric|neoforge), "
    r"java: '([^']+)', module: '([^']+)' \}"
)


def job_block(job_name):
    lines = WORKFLOW.splitlines()
    marker = f"  {job_name}:"
    try:
        start = lines.index(marker)
    except ValueError as exc:
        raise AssertionError(f"Missing CI job {job_name}") from exc
    end = next(
        (i for i in range(start + 1, len(lines))
         if re.match(r"^  [a-z][a-z0-9-]*:$", lines[i])),
        len(lines),
    )
    return "\n".join(lines[start:end])


def job_condition(job_name):
    match = re.search(r"^    if: (.+)$", job_block(job_name), re.MULTILINE)
    if not match:
        raise AssertionError(f"Missing condition for job {job_name}")
    return match.group(1).strip()


class WindowsVerificationReleaseIsolation(unittest.TestCase):
    def test_manual_dispatch_supported(self):
        self.assertRegex(WORKFLOW, r"(?m)^  workflow_dispatch:\s*$")

    def test_windows_verification_only_for_non_release_tag_push(self):
        self.assertEqual(
            job_condition("implemented-adapters"),
            "${{ github.event_name != 'push' || !startsWith(github.ref, 'refs/tags/v') }}",
        )

    def test_release_build_requires_tag_push(self):
        self.assertEqual(
            job_condition("release-build"),
            "github.event_name == 'push' && startsWith(github.ref, 'refs/tags/v')",
        )

    def test_publish_requires_tag_push(self):
        self.assertEqual(
            job_condition("publish"),
            "github.event_name == 'push' && startsWith(github.ref, 'refs/tags/v')",
        )
        self.assertIn("needs: [core-tests, release-build]", job_block("publish"))

    def test_windows_verification_and_release_target_matrix_identical(self):
        verify = TARGET_PATTERN.findall(job_block("implemented-adapters"))
        release = TARGET_PATTERN.findall(job_block("release-build"))
        self.assertEqual(len(verify), 9)
        self.assertEqual(verify, release)
        self.assertEqual(len(set(verify)), 9)
        for mc, loader, java, module in verify:
            self.assertIn(java, ("17", "21", "25"))
            self.assertTrue((ROOT / module / "gradlew.bat").is_file())
            props = (ROOT / module / "gradle.properties").read_text()
            self.assertIn(f"minecraft_version={mc}", props)

    def test_verification_does_not_upload_or_publish(self):
        verified = job_block("implemented-adapters")
        self.assertNotIn("gh release create", verified)
        self.assertNotIn("actions/upload-artifact@", verified)
        self.assertNotIn("contents: write", verified)
        self.assertIn("runs-on: windows-latest", verified)

    def test_release_write_permission_is_publish_scoped(self):
        self.assertIn("permissions:\n  contents: read", WORKFLOW)
        self.assertIn("    permissions:\n      contents: write", job_block("publish"))
        self.assertNotIn("contents: write", job_block("release-build"))

    def test_event_truth_table(self):
        # Contract for the exact expressions above. This is NOT GitHub CI execution.
        for event, ref, verify_expected, release_expected in (
            ("workflow_dispatch", "refs/heads/qa", True, False),
            ("workflow_dispatch", "refs/tags/v0.1.5-rc.1", True, False),
            ("pull_request", "refs/pull/10/merge", True, False),
            ("push", "refs/heads/master", True, False),
            ("push", "refs/tags/v0.1.5-rc.1", False, True),
            ("push", "refs/tags/other", True, False),
        ):
            is_release_push = event == "push" and ref.startswith("refs/tags/v")
            self.assertEqual(not is_release_push, verify_expected)
            self.assertEqual(is_release_push, release_expected)


if __name__ == "__main__":
    unittest.main()
