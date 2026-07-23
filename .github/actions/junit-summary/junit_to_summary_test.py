#!/usr/bin/env python3
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

import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

from junit_to_summary import TestCase, TestReport, TestSuite, generate_markdown_summary


class JunitToSummaryTest(unittest.TestCase):

    def setUp(self):
        self.temp_dir = tempfile.TemporaryDirectory()
        self.xml_path = Path(self.temp_dir.name) / "sample_junit.xml"
        self.output_path = Path(self.temp_dir.name) / "summary.md"

        sample_xml = """<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="SampleSuite" tests="1" failures="0" errors="0" skipped="0" time="1.23">
    <testcase name="test_foo" classname="FooTest" time="1.23" status="passed"/>
</testsuite>
"""
        self.xml_path.write_text(sample_xml, encoding="utf-8")
        self.script_path = Path(__file__).parent / "junit_to_summary.py"

    def tearDown(self):
        self.temp_dir.cleanup()

    def test_generate_markdown_summary_with_nonempty_artifact_url(self):
        report = TestReport(
            total_tests=1,
            total_failures=0,
            total_errors=0,
            total_skipped=0,
            total_passed=1,
            total_time=1.23,
            test_suites=[
                TestSuite(
                    name="SampleSuite",
                    tests=1,
                    failures=0,
                    errors=0,
                    skipped=0,
                    time=1.23,
                    test_cases=[
                        TestCase(name="test_foo", classname="FooTest", time=1.23, status="passed")
                    ]
                )
            ]
        )
        artifact_url = "https://github.com/kubeflow/pipelines/actions/runs/12345/artifacts/67890"
        custom_data_json = f'{{"HTML Report": "{artifact_url}"}}'
        custom_data = json.loads(custom_data_json)

        markdown = generate_markdown_summary(report, custom_data)

        self.assertIn(f"**HTML Report**: {artifact_url}", markdown)
        self.assertNotIn('\\"', markdown)
        self.assertNotIn('"', markdown.splitlines()[0])

    def test_cli_custom_data_argument_parsing(self):
        """Test passing nonempty JSON through the CLI argument path."""
        artifact_url = "https://github.com/kubeflow/pipelines/actions/runs/12345/artifacts/67890"
        custom_data_json = f'{{"HTML Report": "{artifact_url}"}}'

        cmd = [
            sys.executable,
            str(self.script_path),
            str(self.xml_path),
            "--custom-data",
            custom_data_json,
            "--output",
            str(self.output_path),
            "--no-fail-on-test-failures",
        ]

        result = subprocess.run(cmd, capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, msg=f"CLI failed: {result.stderr}")

        markdown = self.output_path.read_text(encoding="utf-8")
        self.assertIn(f"**HTML Report**: {artifact_url}", markdown)
        self.assertNotIn('\\"', markdown)

    def test_composite_action_input_env_var_to_cli_argument_path(self):
        """Test composite action input -> env var -> CLI argument array path."""
        artifact_url = "https://github.com/kubeflow/pipelines/actions/runs/12345/artifacts/67890"
        custom_data_input = f'{{"HTML Report": "{artifact_url}"}}'

        # Simulate composite action env var step mapping (CUSTOM_DATA_INPUT: ${{ inputs.custom_data }})
        env = os.environ.copy()
        env["CUSTOM_DATA_INPUT"] = custom_data_input

        # Simulate bash array construction from composite action action.yml
        args = []
        if env.get("CUSTOM_DATA_INPUT"):
            args.extend(["--custom-data", env["CUSTOM_DATA_INPUT"]])

        cmd = [
            sys.executable,
            str(self.script_path),
            str(self.xml_path),
            "--output",
            str(self.output_path),
            "--no-fail-on-test-failures",
        ] + args

        result = subprocess.run(cmd, capture_output=True, text=True, env=env)
        self.assertEqual(result.returncode, 0, msg=f"Execution failed: {result.stderr}")

        markdown = self.output_path.read_text(encoding="utf-8")
        self.assertIn(f"**HTML Report**: {artifact_url}", markdown)
        self.assertNotIn('\\"', markdown)

    def test_bash_composite_action_execution(self):
        """Test execution of the shared parse_junit_summary.sh script used by action.yml."""
        if sys.platform == "win32":
            self.skipTest("bash shell integration test runs on Linux CI environments")

        bash_executable = shutil.which("bash")
        if not bash_executable:
            self.skipTest("bash is not available in environment")

        runner_script = Path(__file__).parent / "parse_junit_summary.sh"
        artifact_url = "https://github.com/kubeflow/pipelines/actions/runs/12345/artifacts/67890"
        custom_data_input = f'{{"HTML Report": "{artifact_url}"}}'

        env = os.environ.copy()
        env.update({
            "CUSTOM_DATA_INPUT": custom_data_input,
            "FAIL_ON_TEST_FAILURES": "false",
            "OUTPUT_PATH": str(self.output_path),
            "PYTHON_EXEC": sys.executable,
            "JUNIT_SCRIPT": str(self.script_path),
            "XML_FILES": str(self.xml_path),
        })

        cmd = [bash_executable, str(runner_script)]

        result = subprocess.run(cmd, capture_output=True, text=True, env=env)
        self.assertEqual(result.returncode, 0, msg=f"parse_junit_summary.sh failed: {result.stderr}")

        markdown = self.output_path.read_text(encoding="utf-8")
        self.assertIn(f"**HTML Report**: {artifact_url}", markdown)
        self.assertNotIn('\\"', markdown)

    def test_missing_xml_files_unconditional_error(self):
        """Test missing XML files returns non-zero exit unconditionally."""
        empty_dir = Path(self.temp_dir.name) / "empty_dir"
        empty_dir.mkdir()

        cmd = [
            sys.executable,
            str(self.script_path),
            str(empty_dir),
            "--no-fail-on-test-failures",
        ]

        result = subprocess.run(cmd, capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0, msg="Missing XML files must be an unconditional error")
        self.assertIn("Error: No XML files found", result.stderr)


if __name__ == "__main__":
    unittest.main()
