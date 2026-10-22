# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""CI component SDK installs must follow the checkout, not ephemeral PR
refs."""

import os
import re
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]


class SDKSourceCheckoutTest(unittest.TestCase):
    def test_action_passes_actual_checkout_sha(self):
        action = (ROOT / ".github/actions/test-and-report/action.yml").read_text()
        assignment = re.search(r"^        SDK_SOURCE_SHA=.*$", action, re.MULTILINE)
        self.assertIsNotNone(
            assignment, "Test action does not capture an immutable SDK source"
        )
        command = next(
            line.strip()
            for line in action.splitlines()
            if "go run github.com/onsi/ginkgo" in line
        )
        argument = re.search(
            r'-sdkSourceSHA=(?:"\$SDK_SOURCE_SHA"|\$SDK_SOURCE_SHA)', command
        )
        self.assertIsNotNone(
            argument, "Test action does not pass the captured source to Ginkgo"
        )
        with tempfile.TemporaryDirectory() as directory:

            def git(*args):
                return subprocess.check_output(
                    ["git", *args], cwd=directory, text=True
                ).strip()

            git("init", "-q")
            git(
                "-c",
                "user.name=Test",
                "-c",
                "user.email=test@example.com",
                "commit",
                "--allow-empty",
                "-qm",
                "fixture",
            )
            expected = git("rev-parse", "HEAD")
            for event in ("pull_request", "workflow_dispatch"):
                with self.subTest(event=event):
                    script = (
                        assignment.group().strip()
                        + '\nprintf "%s" '
                        + argument.group().split("=", 1)[1]
                    )
                    actual = subprocess.check_output(
                        ["bash", "-eu", "-c", script],
                        cwd=directory,
                        text=True,
                        env={
                            **os.environ,
                            "GITHUB_SHA": "b" * 40,
                            "GITHUB_EVENT_NAME": event,
                        },
                    )
                    self.assertEqual(actual, expected)


if __name__ == "__main__":
    unittest.main()
