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
        self.ci = SimpleNamespace(qualify_images=Mock(), verify_run=Mock())
        client = patch.dict(sys.modules, {'kfpr.ci_images_client': self.ci})
        client.start()
        self.addCleanup(client.stop)
        patcher = patch.object(
            q,
            'release_coverage',
            create=True,
            return_value=({('kfp-api-server', 'linux/amd64')}, {'3.9', '3.11'}),
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
            root = Path(tmp)
            context = self.context(root, backend=False, sdk=False)
            self.evidence(context)
            (root / 'source').mkdir()
            (root / 'source/VERSION').write_text('2.17.3')
            with (
                patch.object(q.tempfile, 'mkdtemp', return_value=tmp),
                patch.object(
                    q,
                    'qualify_security',
                    side_effect=RuntimeError('fixable CVE'),
                ),
            ):
                with self.assertRaisesRegex(RuntimeError, 'fixable CVE'):
                    q.qualify_release(context)
            self.assertNotIn('qualification_path', context.state.answers)
            self.assertNotIn('qualification_digest', context.state.answers)

    def test_isolated_merged_source_passes_and_evidence_can_be_reused(self):
        with (
            TemporaryDirectory() as tmp,
            patch('kfpr.steps.validate_release_pr', return_value=SHA),
        ):
            root = Path(tmp)
            source = root / 'source'
            source.mkdir()
            subprocess.run(['git', 'init', '-q', str(source)], check=True)
            (source / 'VERSION').write_text('2.17.3')
            subprocess.run(
                ['git', '-C', str(source), 'add', 'VERSION'], check=True
            )
            subprocess.run(
                [
                    'git',
                    '-C',
                    str(source),
                    '-c',
                    'user.name=Test',
                    '-c',
                    'user.email=test@example.test',
                    'commit',
                    '-qm',
                    'fixture',
                ],
                check=True,
            )
            context = self.context(root, backend=False, sdk=True)
            context.state.answers['qualification_pythons'] = (
                '/python39,/python311'
            )
            with (
                patch.object(q.tempfile, 'mkdtemp', return_value=tmp),
                patch.object(q, '_run') as run,
                patch.object(q, 'build_wheels'),
                patch.object(
                    q,
                    'qualify_wheels',
                    return_value={
                        'passed': True,
                        'python_environments': ['/venv39', '/venv311'],
                    },
                ) as wheels,
                patch.object(
                    q,
                    'qualify_security',
                    return_value=self.security(
                        context,
                        [
                            {
                                'name': 'source',
                                'target': '/source',
                                'findings': [],
                            }
                        ]
                        + self.python_scans(),
                    ),
                ) as scan,
            ):
                q.qualify_release(context)
                self.assertIn(
                    ['git', 'checkout', '--detach', SHA],
                    [call.args[0] for call in run.call_args_list],
                )
                self.assertEqual(
                    wheels.call_args.args[2], ['/python39', '/python311']
                )
                self.assertEqual(
                    scan.call_args.args[2], [Path('/venv39'), Path('/venv311')]
                )
            q.require_qualification(context)

    def inventory(self, root):
        (root / '.github/workflows').mkdir(parents=True)
        (root / 'frontend').mkdir()
        (root / 'frontend/.nvmrc').write_text('v24.1.0\n')
        (root / 'Dockerfile').write_text('FROM scratch\n')
        (
            root / '.github/workflows/image-builds-release.yml'
        ).write_text("""jobs:
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
""")

    def test_backend_uses_ci_reports_without_local_docker(self):
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            source = root / 'source'
            source.mkdir()
            (source / 'VERSION').write_text('2.17.3')
            self.inventory(source)
            context = self.context(root, sdk=False)
            self.ci.qualify_images.return_value = {
                'images': [{'image_id': 'sha256:' + 'b' * 64}],
                'reports': [
                    {
                        'image_id': 'sha256:' + 'b' * 64,
                        'path': '/download/image.json',
                    }
                ],
                'run': {'id': 123},
            }
            with (
                patch('kfpr.steps.validate_release_pr', return_value=SHA),
                patch.object(q.tempfile, 'mkdtemp', return_value=tmp),
                patch.object(q, '_run') as run,
                patch.object(q.subprocess, 'check_output', return_value=''),
                patch.object(
                    q, 'qualify_security', return_value=self.security(context)
                ) as scan,
            ):
                q.qualify_release(context)
            self.assertFalse(
                any(call.args[0][0] == 'docker' for call in run.call_args_list)
            )
            self.assertEqual(context.state.answers['release_sha'], SHA)
            self.assertEqual(
                self.ci.qualify_images.call_args.args[1], q.image_matrix(source)
            )
            self.assertEqual(
                scan.call_args.kwargs['image_reports'],
                [('sha256:' + 'b' * 64, Path('/download/image.json'))],
            )
            report = json.loads(
                Path(context.state.answers['qualification_path']).read_text()
            )
            self.assertEqual(report['image_ci'], {'id': 123})

    def test_publication_rechecks_ci_run(self):
        with (
            TemporaryDirectory() as tmp,
            patch('kfpr.steps.validate_release_pr', return_value=SHA),
        ):
            context = self.context(Path(tmp), sdk=False)
            self.evidence(context, image_ci={'id': 123})
            self.ci.verify_run.side_effect = ValueError('CI attempt changed')
            with self.assertRaisesRegex(ValueError, 'CI attempt changed'):
                q.require_qualification(context)
            self.ci.verify_run.assert_called_once_with(context, {'id': 123})

    def test_local_scanner_failure_closes_owned_builder_and_retains_evidence(
        self,
    ):
        with TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / 'source'
            source.mkdir()
            (source / 'VERSION').write_text('2.17.3')
            self.inventory(source)
            context = self.context(root, sdk=False)
            context.state.answers['image_qualification'] = 'local'
            closed = []

            def images(*args):
                try:
                    yield {
                        'name': 'kfp-api-server',
                        'platform': 'linux/amd64',
                        'image_id': 'sha256:' + 'b' * 64,
                    }
                finally:
                    closed.append(True)

            def fail_scan(source, output, environments, image_ids, **kwargs):
                self.assertEqual(next(image_ids), 'sha256:' + 'b' * 64)
                output.mkdir()
                (output / 'image-0.json').write_text('{"raw": "retained"}')
                raise RuntimeError('scanner failed')

            with (
                patch('kfpr.steps.validate_release_pr', return_value=SHA),
                patch.object(q.tempfile, 'mkdtemp', return_value=temporary),
                patch.object(q, '_run'),
                patch(
                    'kfpr.local_images.build_images', side_effect=images
                ) as builds,
                patch.object(q, 'qualify_security', side_effect=fail_scan),
            ):
                with self.assertRaisesRegex(RuntimeError, 'scanner failed'):
                    q.qualify_release(context)
            self.assertEqual(closed, [True])
            self.assertEqual(builds.call_args.args[3], q.image_matrix(source))
            self.assertTrue((root / 'security/image-0.json').is_file())
            self.assertTrue((root / 'images.json').is_file())
            self.assertNotIn('qualification_path', context.state.answers)
            self.ci.qualify_images.assert_not_called()

    def test_explicit_local_evidence_skips_image_ci_but_mode_changes_invalidate(
        self,
    ):
        with (
            TemporaryDirectory() as temporary,
            patch('kfpr.steps.validate_release_pr', return_value=SHA),
        ):
            context = self.context(Path(temporary), sdk=False)
            context.state.answers['image_qualification'] = 'local'
            self.evidence(context)
            q.require_qualification(context)
            self.ci.verify_run.assert_not_called()
            context.state.answers['image_qualification'] = 'ci'
            with self.assertRaisesRegex(ValueError, 'scope'):
                q.require_qualification(context)

    def test_unknown_image_matrix_cannot_silently_reduce_coverage(self):
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.inventory(root)
            path = root / '.github/workflows/image-builds-release.yml'
            path.write_text(path.read_text() + '        include: []\n')
            with self.assertRaisesRegex(ValueError, 'Unsupported'):
                q.image_matrix(root)

    def test_dry_run_has_no_build_scan_or_file_side_effects(self):
        with (
            TemporaryDirectory() as tmp,
            patch.object(q, '_run') as run,
            patch.object(q, 'qualify_security') as scan,
        ):
            context = self.context(Path(tmp))
            context.runner.dry_run = True
            q.qualify_release(context)
            q.require_qualification(context)
            run.assert_not_called()
            scan.assert_not_called()
            self.assertEqual(list(Path(tmp).iterdir()), [])


if __name__ == '__main__':
    unittest.main()
