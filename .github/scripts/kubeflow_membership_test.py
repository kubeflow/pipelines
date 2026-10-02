from __future__ import annotations

import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

import kubeflow_membership as membership

ACL = '''
orgs:
    kubeflow:
        admins:
        - OrgAdmin
        billing_email: admin@example.com
        members:
        - JerT33
        - AnotherMember # inline comment
        teams:
            team-a:
                members:
                - TeamOnlyUser
    another-org:
        admins:
        - OtherOrgAdmin
        members:
        - OtherOrgMember
'''


class KubeflowMembershipTest(unittest.TestCase):

    def test_lookup_uses_authoritative_admins_and_members_case_insensitively(
            self):
        cases = {
            'OrgAdmin': True,
            'orgadmin': True,
            'JerT33': True,
            'jErT33': True,
            'AnotherMember': True,
            'JerT3': False,
            'external': False,
            'TeamOnlyUser': False,
            'OtherOrgAdmin': False,
            'OtherOrgMember': False,
        }
        for username, expected in cases.items():
            with self.subTest(username=username), mock.patch.object(
                    membership.subprocess, 'run') as request:
                request.return_value.stdout = ACL
                self.assertEqual(
                    membership.is_kubeflow_member(username), expected)
                request.assert_called_once_with([
                    'gh', 'api',
                    'repos/kubeflow/internal-acls/contents/github-orgs/kubeflow/org.yaml?ref=master',
                    '--header', 'Accept: application/vnd.github.raw+json'
                ],
                                                check=True,
                                                capture_output=True,
                                                text=True,
                                                timeout=30)

    def test_lookup_accepts_equivalent_yaml_representations(self):
        documents = [
            ACL.split('        teams:', 1)[0],
            '''orgs:
  kubeflow:
    teams: {}
    members:
      - "JerT33" # quoted username
      - 'AnotherMember'
    admins: [OrgAdmin]
''',
            'orgs: {kubeflow: {admins: [OrgAdmin], members: [JerT33]}}',
        ]
        for document in documents:
            with self.subTest(document=document), mock.patch.object(
                    membership.subprocess, 'run') as request:
                request.return_value.stdout = document
                self.assertTrue(membership.is_kubeflow_member('jert33'))
                self.assertTrue(membership.is_kubeflow_member('orgadmin'))
                self.assertFalse(membership.is_kubeflow_member('external'))

    def test_lookup_rejects_invalid_yaml_or_membership_schema(self):
        invalid_documents = [
            '',
            'not YAML',
            'orgs: [',
            'orgs: []',
            'orgs: {kubeflow: null}',
            'orgs: {kubeflow: {admins: [], members: []}}',
            '!!python/object/apply:os.system ["echo unsafe"]',
            ACL.replace('orgs:', 'something-else:', 1),
            ACL.replace('    kubeflow:', '    wrong-org:'),
            ACL.replace('        members:\n', '', 1),
            ACL.replace('        admins:\n        - OrgAdmin',
                        '        admins: null'),
            ACL.replace('        admins:\n        - OrgAdmin',
                        '        admins: OrgAdmin'),
            ACL.replace('        - JerT33', '        - [JerT33]'),
            ACL.replace('        - JerT33', '        - {name: JerT33}'),
            ACL.replace('        - JerT33', '        - 123'),
            ACL.replace('        - JerT33', '        - true'),
            ACL.replace('        - JerT33', '        - "invalid user"'),
        ]
        for document in invalid_documents:
            with self.subTest(document=document), mock.patch.object(
                    membership.subprocess, 'run') as request:
                request.return_value.stdout = document
                with self.assertRaises(RuntimeError):
                    membership.is_kubeflow_member('JerT33')

    def test_duplicate_mapping_keys_fail_without_membership_output(self):
        documents = [
            'orgs:\n  kubeflow:\n    admins: [OrgAdmin]\n    admins: [RevokedUser]\n    members: [JerT33]\n',
            'orgs:\n  kubeflow:\n    admins: [OrgAdmin]\n    members: [JerT33]\n    members: [RevokedUser]\n',
            'orgs:\n  kubeflow:\n    admins: [OrgAdmin]\n    members: [RevokedUser]\n    members: [JerT33]\n',
            'orgs:\n  kubeflow:\n    admins: [OrgAdmin]\n    members: [JerT33]\n    "members": [JerT33]\n',
            'orgs:\n  kubeflow: {admins: [OrgAdmin], members: [JerT33]}\n  kubeflow: {admins: [OrgAdmin], members: [RevokedUser]}\n',
            'orgs: {kubeflow: {admins: [OrgAdmin], members: [JerT33]}}\norgs: {kubeflow: {admins: [OrgAdmin], members: [RevokedUser]}}\n',
            'orgs:\n  kubeflow:\n    <<: {admins: [OrgAdmin], members: [RevokedUser]}\n    members: [JerT33]\n',
        ]
        for document in documents:
            with self.subTest(document=document), tempfile.TemporaryDirectory(
            ) as directory:
                output = Path(directory) / 'output'
                with mock.patch.dict(os.environ, {
                        'PR_AUTHOR': 'RevokedUser',
                        'GITHUB_OUTPUT': str(output),
                }), mock.patch.object(membership.subprocess, 'run') as request:
                    request.return_value.stdout = document
                    with self.assertRaisesRegex(RuntimeError,
                                                'Invalid Kubeflow ACL'):
                        membership.main()
                self.assertFalse(output.exists())

    def test_cli_ignores_author_association(self):
        cases = [('JerT33', 'CONTRIBUTOR', True), ('jert33', 'NONE', True),
                 ('external', 'MEMBER', False), ('external', 'OWNER', False),
                 ('external', 'COLLABORATOR', False)]
        for author, association, expected in cases:
            with self.subTest(
                    author=author, association=association
            ), tempfile.TemporaryDirectory() as directory:
                output = Path(directory) / 'output'
                with mock.patch.dict(
                        os.environ, {
                            'PR_AUTHOR': author,
                            'AUTHOR_ASSOCIATION': association,
                            'GITHUB_OUTPUT': str(output),
                        }), mock.patch.object(
                            membership.subprocess,
                            'run') as request, mock.patch('builtins.print'):
                    request.return_value.stdout = ACL
                    self.assertEqual(membership.main(), 0)
                self.assertEqual(output.read_text(),
                                 f'is_member={str(expected).lower()}\n')

    def test_cli_emits_no_membership_result_on_lookup_failure(self):
        failures = [
            subprocess.CalledProcessError(1, ['gh'], stderr='HTTP 403'),
            subprocess.CalledProcessError(1, ['gh'], stderr='HTTP 404'),
            subprocess.TimeoutExpired(['gh'], 30),
            FileNotFoundError('gh'),
            None,  # Empty successful response is not evidence of non-membership.
        ]
        for error in failures:
            with self.subTest(
                    error=error), tempfile.TemporaryDirectory() as directory:
                output = Path(directory) / 'output'
                with mock.patch.dict(os.environ, {
                        'PR_AUTHOR': 'JerT33',
                        'GITHUB_OUTPUT': str(output),
                }), mock.patch.object(
                        membership.subprocess, 'run',
                        side_effect=error) as request:
                    request.return_value.stdout = ''
                    with self.assertRaises(RuntimeError):
                        membership.main()
                self.assertFalse(output.exists())

    def test_invalid_username_is_not_treated_as_external(self):
        for username in ('', 'invalid user', '$(echo injected)'):
            with self.subTest(username=username):
                with self.assertRaises(ValueError):
                    membership.is_kubeflow_member(username)


if __name__ == '__main__':
    unittest.main()
