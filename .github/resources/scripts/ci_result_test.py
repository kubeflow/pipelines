#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import argparse
import json
import os
import tempfile
import unittest

import ci_result


class CiResultTest(unittest.TestCase):
    def test_aggregate_junit_provides_true_test_denominators(self):
        with tempfile.TemporaryDirectory() as directory:
            with open(os.path.join(directory, "junit.xml"), "w", encoding="utf-8") as report:
                report.write(
                    '<testsuite><testcase classname="Suite" name="green"/>'
                    '<testcase classname="Suite" name="flaky"><failure/></testcase>'
                    '<testcase classname="Suite" name="skip"><skipped/></testcase>'
                    "</testsuite>"
                )
            # A shard beside the aggregate must not double-count the same run.
            with open(os.path.join(directory, "shard.xml"), "w", encoding="utf-8") as report:
                report.write('<testsuite><testcase classname="Suite" name="flaky"/></testsuite>')
            tests, errors = ci_result.read_test_results(directory)

        self.assertEqual(errors, 0)
        self.assertEqual(
            tests,
            [
                {"id": "Suite :: flaky", "executions": 1, "failures": 1, "skipped": 0},
                {"id": "Suite :: green", "executions": 1, "failures": 0, "skipped": 0},
                {"id": "Suite :: skip", "executions": 0, "failures": 0, "skipped": 1},
            ],
        )

    def test_classification_separates_test_infrastructure_and_reporting(self):
        failed_test = [{"id": "t", "executions": 1, "failures": 1, "skipped": 0}]
        self.assertEqual(
            ci_result.classify("failure", ["success"], failed_test, {}),
            "test_failure",
        )
        self.assertEqual(
            ci_result.classify(
                "failure", ["success"], failed_test, {"client_timeout": 1}
            ),
            "infrastructure_failure",
        )
        self.assertEqual(
            ci_result.classify("failure", ["success"], [], {"clusterip_dial_timeout": 2}),
            "infrastructure_failure",
        )
        self.assertEqual(
            ci_result.classify("success", ["failure"], [], {}),
            "reporting_failure",
        )
        self.assertEqual(ci_result.classify("success", ["success"], [], {}), "success")

    def test_go_json_uses_package_qualified_terminal_events(self):
        with tempfile.TemporaryDirectory() as directory:
            path = os.path.join(directory, "go-test.json")
            with open(path, "w", encoding="utf-8") as output:
                output.write(
                    '{"Action":"run","Package":"example/a","Test":"TestThing"}\n'
                    '{"Action":"fail","Package":"example/a","Test":"TestThing"}\n'
                    '{"Action":"pass","Package":"example/b","Test":"TestThing"}\n'
                )
            tests, errors = ci_result.read_go_test_results(path)

        self.assertEqual(errors, 0)
        self.assertEqual(
            tests,
            [
                {"id": "example/a :: TestThing", "executions": 1, "failures": 1, "skipped": 0},
                {"id": "example/b :: TestThing", "executions": 1, "failures": 0, "skipped": 0},
            ],
        )

    def test_build_result_includes_durations_dimensions_and_signatures(self):
        with tempfile.TemporaryDirectory() as directory:
            paths = {}
            for name, value in (
                ("action", "10"), ("test_start", "20"), ("test_end", "50"), ("report_end", "55")
            ):
                paths[name] = os.path.join(directory, name)
                with open(paths[name], "w", encoding="utf-8") as marker:
                    marker.write(value)
            signatures = os.path.join(directory, "signatures.json")
            with open(signatures, "w", encoding="utf-8") as output:
                json.dump({"connection_reset": 3}, output)
            args = argparse.Namespace(
                reports_dir=directory, signatures=signatures,
                go_json="",
                action_start=paths["action"], test_start=paths["test_start"],
                test_end=paths["test_end"], report_end=paths["report_end"],
                repository="o/r", workflow="wf", job="job", report_name="lane",
                run_id="7", run_attempt="2", sha="abc", test_outcome="failure",
                install_outcome="success", html_outcome="success",
                upload_outcome="success", publish_outcome="success",
                pipeline_store="database", proxy="false", cache_enabled="true",
                multi_user="false", deployment_mode="direct", mlflow_enabled="false",
                test_label="Smoke", parallel_nodes="2",
            )
            result = ci_result.build_result(args)

        self.assertEqual(result["result"], "infrastructure_failure")
        self.assertEqual(result["durations_seconds"], {"setup": 10.0, "test": 30.0, "report": 5.0})
        self.assertEqual(result["dimensions"]["parallel_nodes"], "2")
        self.assertEqual(result["signatures"]["connection_reset"], 3)


if __name__ == "__main__":
    unittest.main()
