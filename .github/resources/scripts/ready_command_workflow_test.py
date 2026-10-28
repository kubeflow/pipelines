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
"""Exercise the /ready issue command against a stubbed GitHub CLI."""

import os
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = ROOT / '.github/workflows/ready-command.yml'

# Logs each invocation and answers the permission lookup from FAKE_ROLE.
FAKE_GH = textwrap.dedent(
    """\
    #!/usr/bin/env bash
    printf '%s\\n' "$*" >> "${GH_CALL_LOG}"
    if [[ "$*" == *"/permission"* ]]; then
      if [[ -n "${FAKE_PERMISSION_FAILURE:-}" ]]; then
        echo 'HTTP 403: Resource not accessible by integration' >&2
        exit 1
      fi
      printf '%s\\n' "${FAKE_ROLE}"
    fi
    """
)

LABEL_CALL = (
    'api --method POST repos/kubeflow/pipelines/issues/42/labels '
    '-f labels[]=ready'
)
REACTION_CALL = (
    'api --method POST repos/kubeflow/pipelines/issues/comments/7/reactions '
    '-f content=+1'
)


def load_job() -> dict:
    """Return the single job defined by the ready command workflow."""
    workflow = yaml.safe_load(WORKFLOW.read_text(encoding='utf-8'))
    return workflow['jobs']['ready']


class ReadyCommandWorkflowTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.job = load_job()
        (cls.step,) = cls.job['steps']

    def run_command(
        self,
        body: str,
        role: str = 'triage',
        permission_failure: bool = False,
    ) -> tuple[subprocess.CompletedProcess, list[str]]:
        """Run the step script and return its result and the gh calls."""
        with tempfile.TemporaryDirectory() as temp_dir:
            bin_dir = Path(temp_dir) / 'bin'
            bin_dir.mkdir()
            fake_gh = bin_dir / 'gh'
            fake_gh.write_text(FAKE_GH, encoding='utf-8')
            fake_gh.chmod(0o755)
            call_log = Path(temp_dir) / 'calls.log'
            call_log.touch()

            env = {
                'PATH': f'{bin_dir}{os.pathsep}{os.environ["PATH"]}',
                'GH_CALL_LOG': str(call_log),
                'GH_REPO': 'kubeflow/pipelines',
                'COMMENT_BODY': body,
                'COMMENT_ID': '7',
                'COMMENTER': 'octocat',
                'ISSUE_NUMBER': '42',
                'FAKE_ROLE': role,
            }
            if permission_failure:
                env['FAKE_PERMISSION_FAILURE'] = '1'
            result = subprocess.run(
                ['bash', '-c', self.step['run']],
                env=env,
                capture_output=True,
                text=True,
                check=False,
            )
            return result, call_log.read_text(encoding='utf-8').splitlines()

    def test_runs_only_for_issue_comments_in_upstream_repository(self):
        self.assertEqual(
            yaml.safe_load(WORKFLOW.read_text(encoding='utf-8'))[True],
            {'issue_comment': {'types': ['created']}},
        )
        condition = self.job['if']
        self.assertIn("github.repository == 'kubeflow/pipelines'", condition)
        self.assertIn('!github.event.issue.pull_request', condition)

    def test_comment_body_is_not_interpolated_into_script(self):
        self.assertNotIn('${{', self.step['run'])
        self.assertEqual(
            self.step['env']['COMMENT_BODY'],
            '${{ github.event.comment.body }}',
        )

    def test_triage_or_higher_applies_label_and_reacts(self):
        for role in ('admin', 'maintain', 'write', 'triage'):
            with self.subTest(role=role):
                result, calls = self.run_command('/ready', role=role)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(calls[1:], [LABEL_CALL, REACTION_CALL])

    def test_command_matches_own_line_with_surrounding_whitespace(self):
        body = 'Looks good to me.\r\n  /ready \r\nThanks!'
        result, calls = self.run_command(body)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn(LABEL_CALL, calls)

    def test_mentions_inside_prose_are_ignored(self):
        for body in ('is this /ready?', '/ready-ish', 'not /ready', '/readyy'):
            with self.subTest(body=body):
                result, calls = self.run_command(body)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(calls, [])

    def test_users_below_triage_are_denied_without_labeling(self):
        for role in ('read', 'none', ''):
            with self.subTest(role=role):
                result, calls = self.run_command('/ready', role=role)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(len(calls), 2)
                self.assertIn('issues/42/comments', calls[1])
                self.assertNotIn(LABEL_CALL, calls)

    def test_permission_lookup_failure_fails_without_labeling(self):
        result, calls = self.run_command('/ready', permission_failure=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(len(calls), 1)
        self.assertIn('/permission', calls[0])


if __name__ == '__main__':
    unittest.main()
