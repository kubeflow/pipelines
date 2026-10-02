#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Exercise shared CI eligibility against authoritative membership fixtures."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

MODULE = Path(__file__).with_name('ci_eligibility.js')


def evaluate(body, members_file=None):
    environment = os.environ.copy()
    environment.pop('KUBEFLOW_MEMBERS_FILE', None)
    if members_file is not None:
        environment['KUBEFLOW_MEMBERS_FILE'] = str(members_file)
    script = f'''
const {{loadMembers, eligible}} = require({json.dumps(str(MODULE))});
try {{
  const result = (() => {{ {body} }})();
  console.log(JSON.stringify({{result}}));
}} catch (error) {{
  console.log(JSON.stringify({{error: error.message}}));
}}
'''
    result = subprocess.run(['node'],
                            input=script,
                            capture_output=True,
                            text=True,
                            check=True,
                            env=environment)
    return json.loads(result.stdout)


def pull_request(login='contributor', association='CONTRIBUTOR', labels=()):
    return {
        'user': {
            'login': login
        },
        'author_association': association,
        'labels': [{
            'name': label
        } for label in labels],
    }


class CIEligibilityTest(unittest.TestCase):

    def test_membership_is_case_insensitive_and_ignores_association(self):
        with tempfile.TemporaryDirectory() as directory:
            members_file = Path(directory) / 'members.json'
            members_file.write_text('["Contributor", "OrgAdmin"]')
            fixtures = [
                (pull_request(), True),
                (pull_request(login='CONTRIBUTOR', association='NONE'), True),
                (pull_request(login='orgadmin',
                              association='CONTRIBUTOR'), True),
                (pull_request(login='outsider', association='MEMBER'), False),
                (pull_request(login='outsider', association='OWNER'), False),
                (pull_request(login='outsider',
                              association='COLLABORATOR'), False),
                (pull_request(login='dependabot[bot]',
                              association='NONE'), True),
                (pull_request(
                    login='copybara-service[bot]', association='NONE'), False),
            ]
            for pr, expected in fixtures:
                with self.subTest(pr=pr):
                    body = f'return eligible({json.dumps(pr)}, loadMembers());'
                    self.assertEqual(
                        evaluate(body, members_file), {'result': expected})

    def test_explicit_approval_and_blocking_labels(self):
        with tempfile.TemporaryDirectory() as directory:
            members_file = Path(directory) / 'members.json'
            members_file.write_text('["contributor"]')
            for login in ['contributor', 'outsider', 'dependabot[bot]']:
                for labels, expected in [
                    (['ok-to-test'], True),
                    (['needs-ok-to-test'], False),
                    (['ok-to-test', 'needs-ok-to-test'], False),
                ]:
                    with self.subTest(login=login, labels=labels):
                        pr = pull_request(login=login, labels=labels)
                        body = f'return eligible({json.dumps(pr)});'
                        self.assertEqual(
                            evaluate(body, members_file), {'result': expected})

    def test_loader_returns_normalized_deduplicated_set(self):
        with tempfile.TemporaryDirectory() as directory:
            members_file = Path(directory) / 'members.json'
            members_file.write_text('["Member", "member", "ADMIN"]')
            body = f'return [...loadMembers({json.dumps(str(members_file))})];'
            self.assertEqual(evaluate(body), {'result': ['member', 'admin']})

    def test_missing_file_configuration_fails_clearly(self):
        result = evaluate('return [...loadMembers()];')
        self.assertIn('KUBEFLOW_MEMBERS_FILE', result['error'])
        with tempfile.TemporaryDirectory() as directory:
            result = evaluate('return [...loadMembers()];',
                              Path(directory) / 'missing.json')
            self.assertIn('Could not load Kubeflow ACL membership file',
                          result['error'])

    def test_invalid_membership_fails_instead_of_classifying_an_outsider(self):
        invalid_values = [
            '', '{', 'null', '{}', '[]', '"member"', '[123]', '[null]',
            '["member", false]', '[""]', '["invalid user"]'
        ]
        with tempfile.TemporaryDirectory() as directory:
            members_file = Path(directory) / 'members.json'
            for value in invalid_values:
                with self.subTest(value=value):
                    members_file.write_text(value)
                    pr = pull_request(login='outsider')
                    result = evaluate(f'return eligible({json.dumps(pr)});',
                                      members_file)
                    self.assertIn('Kubeflow ACL membership file',
                                  result['error'])
                    self.assertNotIn('result', result)


if __name__ == '__main__':
    unittest.main()
