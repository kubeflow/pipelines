"""Release qualification binds scans and installation tests to reviewed
source."""

import hashlib
import json
import subprocess
import sys
import unittest
from datetime import datetime, timedelta, timezone
from pathlib import Path
from tempfile import TemporaryDirectory
from types import SimpleNamespace
from unittest.mock import Mock, patch

from kfpr import qualification as q
from kfpr.core import (
    CommandRunner,
    ReleaseContext,
    ReleaseMetadata,
    ReleaseState,
)

SHA = 'a' * 40


class QualificationTest(unittest.TestCase):
    def context(self, root, backend=True, sdk=True):
        return ReleaseContext(
            root,
            ReleaseState(root / 'state.json'),
            CommandRunner(),
            ReleaseMetadata.from_version('patch', '2.17.3'),
            'https://github.com/test/pipelines.git',
            backend,
            sdk,
        )

    def setUp(self):
        third_party = patch.object(q, '_require_third_party')
        self.third_party_check = third_party.start()
        self.addCleanup(third_party.stop)
        self.ci = SimpleNamespace(qualify_ci=Mock(), verify_run=Mock())
        client = patch.dict(
            sys.modules, {'kfpr.ci_qualification_client': self.ci}
        )
        client.start()
        self.addCleanup(client.stop)
        patcher = patch.object(
            q,
            'release_coverage',
            create=True,
            return_value=(
                {('kfp-api-server', 'linux/amd64')},
                {'3.9', '3.11'},
            ),
        )
        self.coverage_patcher = patcher
        patcher.start()
        self.addCleanup(patcher.stop)

    def python_scans(self):
        return [
            {
                'name': f'python-{i}',
                'target': path,
                'python_version': version,
                'findings': [],
            }
            for i, (path, version) in enumerate(
                [('/venv39', '3.9'), ('/venv311', '3.11')]
            )
        ]

    def security(self, context, scans=None):
        scans = (
            scans
            if scans is not None
            else [{'name': 'source', 'target': '/source', 'findings': []}]
        )
        policy = context.state.answers.get('security_policy', 'zero-fixable')
        return {
            'passed': True,
            'scanner': q.SCANNER,
            'errors': [],
            'policy': policy,
            'release_version': context.metadata.tag,
            'scans': scans,
            **q.evaluate_policy(
                scans, release_version=context.metadata.tag, policy=policy
            ),
        }

    def evidence(self, context, **overrides):
        scans = [{'name': 'source', 'target': '/source', 'findings': []}]
        images = []
        if context.include_backend:
            images = [
                {
                    'name': 'kfp-api-server',
                    'platform': 'linux/amd64',
                    'image_id': 'sha256:' + 'b' * 64,
                }
            ]
            scans.append(
                {
                    'name': 'image-0',
                    'target': images[0]['image_id'],
                    'findings': [],
                }
            )
        if context.include_sdk:
            scans.extend(self.python_scans())
        report = {
            **q._identity(context, SHA),
            'qualified_at': datetime.now(timezone.utc).isoformat(),
            'security': self.security(context, scans),
            'images': images,
            'security_ci': {'id': 123},
            'wheels': {
                'passed': True,
                'python_environments': ['/venv39', '/venv311'],
            },
        }
        report.update(overrides)
        path = context.root / 'evidence.json'
        path.write_text(json.dumps(report))
        context.state.answers['qualification_path'] = str(path)
        context.state.answers['qualification_digest'] = hashlib.sha256(
            path.read_bytes()
        ).hexdigest()
        return path

    def test_publication_rejects_absent_modified_stale_or_wrong_scope_evidence(
        self,
    ):
        with (
            TemporaryDirectory() as tmp,
            patch('kfpr.steps.validate_release_pr', return_value=SHA),
        ):
            context = self.context(Path(tmp))
            with self.assertRaisesRegex(ValueError, 'Run qualify-release'):
                q.require_qualification(context)
            path = self.evidence(context)
            q.require_qualification(context)
            path.write_text(path.read_text() + ' ')
            with self.assertRaisesRegex(ValueError, 'changed'):
                q.require_qualification(context)
            for override in (
                {'release_sha': 'b' * 40},
                {'include_backend': False},
                {'version': '2.17.4'},
                {'security': {'passed': False, 'fixable_count': 1}},
                {'wheels': {'passed': False}},
                {
                    'qualified_at': (
                        datetime.now(timezone.utc) - timedelta(days=2)
                    ).isoformat()
                },
            ):
                self.evidence(context, **override)
                with self.assertRaises(ValueError):
                    q.require_qualification(context)

    def test_publication_recomputes_findings_and_binds_approved_policy(self):
        with (
            TemporaryDirectory() as tmp,
            patch('kfpr.steps.validate_release_pr', return_value=SHA),
        ):
            context = self.context(Path(tmp), backend=False, sdk=False)
            context.state.answers['security_policy'] = (
                '2.17.3-approved-exceptions-v1'
            )
            scans = [
                {
                    'name': 'python-0',
                    'python_version': '3.9',
                    'target': '/venv',
                    'findings': [
                        {
                            'id': 'PYSEC-2026-2275',
                            'package': 'requests',
                            'installed_version': '2.32.5',
                            'fixed_versions': ['2.33.0'],
                            'fixable': True,
                        }
                    ],
                }
            ]
            scans.append(
                {'name': 'source', 'target': '/source', 'findings': []}
            )
            security = self.security(context, scans)
            self.assertEqual(security['accepted_count'], 1)
            self.evidence(context, security=security)
            q.require_qualification(context)
            context.state.answers['security_policy'] = 'zero-fixable'
            with self.assertRaises(ValueError):
                q.require_qualification(context)
            context.state.answers['security_policy'] = (
                '2.17.3-approved-exceptions-v1'
            )
            scans[0]['findings'][0]['id'] = 'CVE-new'
            self.evidence(context, security=security)
            with self.assertRaisesRegex(ValueError, 'findings'):
                q.require_qualification(context)

    def test_publication_rejects_image_scan_inventory_mismatch(self):
        with (
            TemporaryDirectory() as tmp,
            patch('kfpr.steps.validate_release_pr', return_value=SHA),
        ):
            context = self.context(Path(tmp), sdk=False)
            self.evidence(context, images=[])
            with self.assertRaisesRegex(ValueError, 'Image qualification'):
                q.require_qualification(context)
            self.evidence(context, images=[{'image_id': 'sha256:' + 'c' * 64}])
            with self.assertRaisesRegex(ValueError, 'Image scan'):
                q.require_qualification(context)

    def test_coverage_reads_metadata_from_approved_commit(self):
        self.coverage_patcher.stop()
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.inventory(root)
            workflow = (
                root / '.github/workflows/image-builds-release.yml'
            ).read_text()
            context = self.context(root)

            def output(command, **kwargs):
                self.assertEqual(kwargs['cwd'], root)
                if command[1] == 'ls-tree':
                    self.assertIn(SHA, command)
                    return 'sdk/python/setup.py\n'
                self.assertTrue(command[2].startswith(SHA + ':'))
                if command[2].endswith('setup.py'):
                    return "classifiers=['Programming Language :: Python :: 3.9', 'Programming Language :: Python :: 3.11']"
                return workflow

            with patch.object(q.subprocess, 'check_output', side_effect=output):
                images, minors = q.release_coverage(context, SHA)
            self.assertEqual(minors, {'3.9', '3.11'})
            self.assertEqual(
                images,
                {
                    (item['name'], item['platform'])
                    for item in q.image_matrix(root)
                },
            )

    def test_publication_requires_complete_scan_coverage(self):
        with (
            TemporaryDirectory() as tmp,
            patch('kfpr.steps.validate_release_pr', return_value=SHA),
        ):
            context = self.context(Path(tmp))
            path = self.evidence(context)
            complete = json.loads(path.read_text())
            for missing in ('source', 'python-0'):
                scans = [
                    scan
                    for scan in complete['security']['scans']
                    if scan['name'] != missing
                ]
                self.evidence(context, security=self.security(context, scans))
                with self.assertRaisesRegex(ValueError, 'coverage'):
                    q.require_qualification(context)
            self.evidence(context)
            with patch.object(
                q,
                'release_coverage',
                return_value=(
                    {
                        ('kfp-api-server', 'linux/amd64'),
                        ('kfp-api-server', 'linux/arm64'),
                    },
                    {'3.9', '3.11'},
                ),
            ):
                with self.assertRaisesRegex(ValueError, 'coverage'):
                    q.require_qualification(context)

    def test_live_failed_checks_block_even_fresh_evidence(self):
        with TemporaryDirectory() as tmp:
            context = self.context(Path(tmp))
            self.evidence(context)
            with patch(
                'kfpr.steps.validate_release_pr',
                side_effect=ValueError('CI failed'),
            ):
                with self.assertRaisesRegex(ValueError, 'CI failed'):
                    q.require_qualification(context)

    def test_failed_scan_invalidates_previous_success(self):
        with (
            TemporaryDirectory() as tmp,
            patch('kfpr.steps.validate_release_pr', return_value=SHA),
            patch.object(q, '_run'),
        ):
            context = self.context(Path(tmp), backend=False, sdk=False)
            self.evidence(context)
            (Path(tmp) / 'source').mkdir()
            self.ci.qualify_ci.side_effect = RuntimeError('scanner failed')
            with patch.object(q.tempfile, 'mkdtemp', return_value=tmp):
                with self.assertRaisesRegex(RuntimeError, 'scanner failed'):
                    q.qualify_release(context)
            self.assertNotIn('qualification_path', context.state.answers)
            self.assertNotIn('qualification_digest', context.state.answers)

    def test_sdk_only_dispatches_ci_and_rechecks_run_at_publication(self):
        with (
            TemporaryDirectory() as tmp,
            patch('kfpr.steps.validate_release_pr', return_value=SHA),
        ):
            root = Path(tmp)
            (root / 'source').mkdir()
            context = self.context(root, backend=False, sdk=True)
            self.ci.qualify_ci.return_value = {
                'images': [],
                'wheels': {
                    'passed': True,
                    'python_environments': ['/venv39', '/venv311'],
                },
                'security': self.security(
                    context,
                    [{'name': 'source', 'target': '/source', 'findings': []}]
                    + self.python_scans(),
                ),
                'run': {'id': 123},
            }
            with (
                patch.object(q.tempfile, 'mkdtemp', return_value=tmp),
                patch.object(q, '_run') as run,
                patch.object(q.subprocess, 'check_output', return_value=''),
                patch.object(q, 'build_wheels') as build,
            ):
                q.qualify_release(context)
                build.assert_not_called()
                self.assertIn(
                    ['git', 'checkout', '--detach', SHA],
                    [c.args[0] for c in run.call_args_list],
                )
            self.assertEqual(
                self.ci.qualify_ci.call_args.args[1:3], ([], {'3.9', '3.11'})
            )
            q.require_qualification(context)
            self.ci.verify_run.assert_called_once_with(context, {'id': 123})

    def inventory(self, root):
        (root / '.github/workflows').mkdir(parents=True)
        (root / 'frontend').mkdir()
        (root / 'frontend/.nvmrc').write_text('v24.1.0\n')
        (root / 'Dockerfile').write_text('FROM scratch\n')
        (root / '.github/workflows/image-builds-release.yml').write_text(
            """jobs:
  build-images-for-release:
    strategy:
      matrix:
        image:
          - {name: kfp-api-server, dockerfile: Dockerfile, context: .}
          - {name: kfp-writer, dockerfile: Dockerfile, context: .}
        arch:
          - {arch_name: amd64, platform: linux/amd64}
          - {arch_name: arm64, platform: linux/arm64}
        exclude:
          - image: {name: kfp-writer}
            arch: {arch_name: arm64}
"""
        )

    def test_new_approved_commit_is_available_without_moving_caller_checkout(
        self,
    ):
        self.coverage_patcher.stop()
        with TemporaryDirectory() as temporary:
            base = Path(temporary)
            upstream = base / 'upstream'
            upstream.mkdir()

            def git(root, *arguments):
                return subprocess.check_output(
                    ['git', '-c', 'core.fsmonitor=false', *arguments],
                    cwd=root,
                    text=True,
                    stderr=subprocess.PIPE,
                ).strip()

            git(upstream, 'init', '--quiet')
            self.inventory(upstream)
            (upstream / 'VERSION').write_text('2.17.2\n')
            sdk = upstream / 'sdk/python'
            sdk.mkdir(parents=True)
            metadata = sdk / 'setup.py'
            metadata.write_text(
                "classifiers=['Programming Language :: Python :: 3.9']\n"
            )
            (upstream / 'operator-notes.txt').write_text('committed notes\n')
            git(upstream, 'add', '.')
            git(
                upstream,
                '-c',
                'user.name=Qualification Test',
                '-c',
                'user.email=qualification@example.invalid',
                'commit',
                '--quiet',
                '-m',
                'Older release fixture',
            )
            caller = base / 'caller'
            git(base, 'clone', '--quiet', str(upstream), str(caller))
            caller_head = git(caller, 'rev-parse', 'HEAD')
            (caller / 'operator-notes.txt').write_text(
                'uncommitted operator notes\n'
            )
            (caller / 'scratch.txt').write_text('untracked user work\n')
            initial_status = git(caller, 'status', '--porcelain')

            (upstream / 'VERSION').write_text('2.17.3\n')
            metadata.write_text(
                "classifiers=['Programming Language :: Python :: 3.9', "
                "'Programming Language :: Python :: 3.11']\n"
            )
            git(upstream, 'add', '.')
            git(
                upstream,
                '-c',
                'user.name=Qualification Test',
                '-c',
                'user.email=qualification@example.invalid',
                'commit',
                '--quiet',
                '-m',
                'Approved release fixture',
            )
            approved = git(upstream, 'rev-parse', 'HEAD')
            with self.assertRaises(subprocess.CalledProcessError):
                git(caller, 'cat-file', '-e', approved)
            context = self.context(caller)
            context.state = ReleaseState(base / 'state.json')
            output = base / 'evidence'
            output.mkdir()
            images = [
                {**record, 'image_id': 'sha256:' + str(index + 1) * 64}
                for index, record in enumerate(q.image_matrix(upstream))
            ]
            scans = [{'name': 'source', 'target': '/source', 'findings': []}]
            scans.extend(self.python_scans())
            scans.extend(
                {
                    'name': f'image-{index}',
                    'target': record['image_id'],
                    'findings': [],
                }
                for index, record in enumerate(images)
            )
            self.ci.qualify_ci.return_value = {
                'images': images,
                'wheels': {
                    'passed': True,
                    'python_environments': ['/venv39', '/venv311'],
                },
                'security': self.security(context, scans),
                'run': {'id': 123},
            }
            original_run = q._run
            fetched = []

            def local_upstream(command, root):
                command = list(command)
                if command[:3] == [
                    'git',
                    'fetch',
                    f'https://github.com/{q.REPO}.git',
                ]:
                    fetched.append(command[-1])
                    command[2] = str(upstream)
                original_run(command, root)

            with (
                patch('kfpr.steps.validate_release_pr', return_value=approved),
                patch.object(q.tempfile, 'mkdtemp', return_value=str(output)),
                patch.object(q, '_run', side_effect=local_upstream),
            ):
                q.qualify_release(context)
                q.require_qualification(context)
            self.assertEqual(fetched, [approved])
            self.assertEqual(
                self.ci.qualify_ci.call_args.args[1], q.image_matrix(upstream)
            )
            self.assertEqual(
                self.ci.qualify_ci.call_args.args[2], {'3.9', '3.11'}
            )
            self.ci.verify_run.assert_called_once_with(context, {'id': 123})
            self.assertEqual(
                git(caller, 'show', f'{approved}:VERSION'), '2.17.3'
            )
            self.assertEqual(git(caller, 'rev-parse', 'HEAD'), caller_head)
            self.assertEqual(
                git(caller, 'status', '--porcelain'), initial_status
            )
            self.assertEqual(
                (caller / 'operator-notes.txt').read_text(),
                'uncommitted operator notes\n',
            )
            self.assertEqual(
                (caller / 'scratch.txt').read_text(), 'untracked user work\n'
            )

    def test_backend_uses_only_ci_scans_and_complete_source_inventory(self):
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            source = root / 'source'
            source.mkdir()
            (source / 'VERSION').write_text('2.17.3')
            self.inventory(source)
            context = self.context(root, sdk=False)
            self.ci.qualify_ci.return_value = {
                'images': [],
                'wheels': None,
                'security': self.security(context),
                'run': {'id': 123},
            }
            with (
                patch('kfpr.steps.validate_release_pr', return_value=SHA),
                patch.object(q.tempfile, 'mkdtemp', return_value=tmp),
                patch.object(q, '_run') as run,
                patch.object(q.subprocess, 'check_output', return_value=''),
            ):
                q.qualify_release(context)
            self.assertFalse(
                any(
                    c.args[0][0]
                    in ('docker', 'trivy', 'osv-scanner', 'pip-audit')
                    for c in run.call_args_list
                )
            )
            self.assertEqual(
                self.ci.qualify_ci.call_args.args[1], q.image_matrix(source)
            )
            report = json.loads(
                Path(context.state.answers['qualification_path']).read_text()
            )
            self.assertEqual(report['security_ci'], {'id': 123})
            self.assertEqual(report['schema'], 3)

    def test_publication_rechecks_ci_run(self):
        with (
            TemporaryDirectory() as tmp,
            patch('kfpr.steps.validate_release_pr', return_value=SHA),
        ):
            context = self.context(Path(tmp), sdk=False)
            self.evidence(context, security_ci={'id': 123})
            self.ci.verify_run.side_effect = ValueError('CI attempt changed')
            with self.assertRaisesRegex(ValueError, 'CI attempt changed'):
                q.require_qualification(context)
            self.ci.verify_run.assert_called_once_with(context, {'id': 123})

    def test_local_mode_cannot_authorize_qualification(self):
        with TemporaryDirectory() as tmp:
            context = self.context(Path(tmp), sdk=False)
            context.state.answers['image_qualification'] = 'local'
            with self.assertRaisesRegex(ValueError, 'require CI'):
                q._identity(context, SHA)
            self.ci.qualify_ci.assert_not_called()

    def test_legacy_trivy_receipt_cannot_authorize_publication(self):
        with (
            TemporaryDirectory() as tmp,
            patch('kfpr.steps.validate_release_pr', return_value=SHA),
        ):
            context = self.context(Path(tmp), sdk=False)
            self.evidence(context, schema=1)
            with self.assertRaisesRegex(
                ValueError, 'different commit/version/scope'
            ):
                q.require_qualification(context)
            self.ci.verify_run.assert_not_called()

    def test_unknown_image_matrix_cannot_silently_reduce_coverage(self):
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.inventory(root)
            path = root / '.github/workflows/image-builds-release.yml'
            path.write_text(path.read_text() + '        include: []\n')
            with self.assertRaisesRegex(ValueError, 'Unsupported'):
                q.image_matrix(root)

    def test_dry_run_has_no_build_scan_or_file_side_effects(self):
        with TemporaryDirectory() as tmp, patch.object(q, '_run') as run:
            context = self.context(Path(tmp))
            context.runner.dry_run = True
            q.qualify_release(context)
            q.require_qualification(context)
            run.assert_not_called()
            self.ci.qualify_ci.assert_not_called()
            self.assertEqual(list(Path(tmp).iterdir()), [])


class ThirdPartyPublicationTest(unittest.TestCase):
    def setUp(self):
        temporary = TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.directory = Path(temporary.name)
        manifest = self.directory / 'third-party.json'
        manifest.write_text('{"schema": 1}')
        self.scan = {
            'name': 'third-party-abc',
            'target': 'sha256:' + 'c' * 64,
            'findings': [],
            'raw_report': 'third-party/scans/abc/scan.json',
        }
        self.verified = {
            'images': [
                {'image': 'example/tool:1', 'image_id': self.scan['target']}
            ],
            'scans': [self.scan],
            'inventory': {'first_party_names': ['api']},
        }
        self.identity = {
            'source_sha': SHA,
            'workflow_sha': 'b' * 40,
            'version': '2.17.3',
            'request_id': 'c' * 32,
            'run_id': '123',
            'run_attempt': 1,
        }
        self.report = {
            'images': [{'name': 'api'}],
            'security': {'scans': [self.scan]},
            'security_ci': self.identity,
            'third_party': {
                **self.verified,
                'evidence_directory': str(self.directory),
                'manifest_sha256': hashlib.sha256(
                    manifest.read_bytes()
                ).hexdigest(),
            },
        }
        patcher = patch('kfpr.third_party.verify', return_value=self.verified)
        self.verify = patcher.start()
        self.addCleanup(patcher.stop)

    def test_publication_rechecks_raw_evidence_and_run_provenance(self):
        q._require_third_party(self.report, True, SHA)
        self.verify.assert_called_once_with(
            self.directory, {'schema': 1, **self.identity}
        )
        (self.directory / 'third-party.json').write_text('{}')
        with self.assertRaisesRegex(ValueError, 'checksum'):
            q._require_third_party(self.report, True, SHA)

    def test_missing_coverage_changed_subject_or_old_source_blocks(self):
        import copy

        for mutation in ('missing', 'subject', 'source', 'coverage'):
            report = copy.deepcopy(self.report)
            if mutation == 'missing':
                report.pop('third_party')
            elif mutation == 'subject':
                report['security']['scans'][0]['target'] = 'sha256:' + 'd' * 64
            elif mutation == 'source':
                report['security_ci']['source_sha'] = 'e' * 40
            else:
                report['security']['scans'] = []
            with (
                self.subTest(mutation=mutation),
                self.assertRaises(ValueError),
            ):
                q._require_third_party(report, True, SHA)

    def test_sdk_only_does_not_require_third_party_scans(self):
        q._require_third_party({'security': {'scans': []}}, False, SHA)
        self.verify.assert_not_called()
        with self.assertRaisesRegex(ValueError, 'Unexpected'):
            q._require_third_party(self.report, False, SHA)


if __name__ == '__main__':
    unittest.main()
