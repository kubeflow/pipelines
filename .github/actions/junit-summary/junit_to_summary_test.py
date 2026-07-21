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
import tempfile
import unittest
from junit_to_summary import TestReport, TestSuite, TestCase, generate_markdown_summary


class JunitToSummaryTest(unittest.TestCase):

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


if __name__ == '__main__':
    unittest.main()
