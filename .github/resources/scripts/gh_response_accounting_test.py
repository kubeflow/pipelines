#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import importlib.util
import json
import os
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).parent
spec = importlib.util.spec_from_file_location(
    "summarize", ROOT / "summarize-gh-responses.py"
)
summary = importlib.util.module_from_spec(spec)
spec.loader.exec_module(summary)


class GhResponseAccountingTest(unittest.TestCase):
    def test_paginated_headers_quota_and_body_are_separated(self):
        response = (
            "HTTP/2.0 200\r\nX-RateLimit-Limit: 15000\r\n"
            "X-RateLimit-Remaining: 00012\r\nX-RateLimit-Used: 14988\r\n"
            "X-RateLimit-Reset: 1794680873\r\nX-RateLimit-Resource: core\r\n"
            "Authorization: SECRET\r\n\r\nfirst\tin_progress\t\n"
            "HTTP/2.0 200\r\nX-RateLimit-Remaining: 11\r\n"
            "X-RateLimit-Resource: core\r\n\r\nsecond\tcompleted\tsuccess"
        )
        with tempfile.TemporaryDirectory() as directory:
            journal = Path(directory) / "accounting"
            result = subprocess.run(
                [
                    "bash",
                    "-c",
                    'source "$1"; output=$(record_gh_response "$2" jobs 0 "$RESPONSE" body); printf "%s" "$output"',
                    "test",
                    str(ROOT / "gh-response-accounting.sh"),
                    str(journal),
                ],
                env=dict(os.environ, RESPONSE=response),
                capture_output=True,
                text=True,
                check=True,
            )
            self.assertEqual(
                result.stdout, "first\tin_progress\t\nsecond\tcompleted\tsuccess"
            )
            report = summary.summarize(journal)
            self.assertEqual(report["invocations"], 1)
            self.assertEqual(report["responses"], 2)
            self.assertEqual(
                report["counts"], [{"category": "jobs", "status": 200, "count": 2}]
            )
            self.assertEqual(
                report["rate_limits"]["core"],
                {
                    "observations": 2,
                    "first": {
                        "limit": 15000,
                        "remaining": 12,
                        "used": 14988,
                        "reset": 1794680873,
                    },
                    "last": {"remaining": 11},
                },
            )
            self.assertNotIn("SECRET", journal.read_text())
            self.assertNotIn("first", journal.read_text())

    def test_304_error_and_transport_exits_remain_distinct(self):
        with tempfile.TemporaryDirectory() as directory:
            journal = Path(directory) / "accounting"
            for code, response in [
                (1, "HTTP/2.0 304\n\n"),
                (
                    1,
                    (
                        "HTTP/2.0 403\nX-RateLimit-Remaining: 0\n"
                        "X-RateLimit-Reset: SECRET\n"
                        "X-RateLimit-Resource: core\n\nSECRET"
                    ),
                ),
                (7, ""),
            ]:
                subprocess.run(
                    [
                        "bash",
                        "-c",
                        'source "$1"; record_gh_response "$2" artifacts "$3" "$RESPONSE"',
                        "test",
                        str(ROOT / "gh-response-accounting.sh"),
                        str(journal),
                        str(code),
                    ],
                    env=dict(os.environ, RESPONSE=response),
                    check=True,
                )
            report = summary.summarize(journal)
            self.assertEqual(report["invocations"], 3)
            self.assertEqual(report["responses"], 2)
            self.assertEqual(
                report["counts"],
                [
                    {"category": "artifacts", "status": 304, "count": 1},
                    {"category": "artifacts", "status": 403, "count": 1},
                ],
            )
            self.assertEqual(
                report["exits"],
                [
                    {"category": "artifacts", "exit_code": 1, "count": 2},
                    {"category": "artifacts", "exit_code": 7, "count": 1},
                ],
            )
            self.assertEqual(report["rate_limits"]["core"]["last"], {"remaining": 0})
            self.assertNotIn("SECRET", json.dumps(report))


if __name__ == "__main__":
    unittest.main()
