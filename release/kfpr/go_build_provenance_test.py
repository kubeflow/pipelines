"""Actual source and binary provenance must be complete and mutually bound."""

import copy
import json
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from kfpr import go_build_provenance as provenance

SHA = 'a' * 40
TIME = '2026-11-14T12:34:56Z'
IMAGE = 'sha256:' + 'b' * 64


def build_info():
    return {
        'GoVersion': 'go1.27.4',
        'Path': provenance.MODULE + '/backend/src/apiserver',
        'Main': {
            'Path': provenance.MODULE,
            'Version': provenance.pseudo_version(SHA, TIME),
        },
        'ELFMachine': 'EM_AARCH64',
        'Settings': [
            {'Key': k, 'Value': v}
            for k, v in {
                'vcs': 'git',
                'vcs.revision': SHA,
                'vcs.time': TIME,
                'vcs.modified': 'false',
                'GOOS': 'linux',
                'GOARCH': 'arm64',
            }.items()
        ],
    }


class GoBuildProvenanceTest(unittest.TestCase):
    def test_native_pseudo_version_binds_revision_and_time(self):
        self.assertEqual(
            provenance.pseudo_version(SHA, TIME),
            'v0.0.0-20261114123456-aaaaaaaaaaaa',
        )
        info = build_info()
        info['Settings'].append({'Key': 'CGO_CFLAGS'})
        provenance.validate_binary(info, SHA, TIME, 'linux/arm64')

    def test_wrong_dirty_or_missing_metadata_is_rejected(self):
        for key, value in [
            ('vcs', 'svn'),
            ('vcs.revision', 'c' * 40),
            ('vcs.time', '2025-01-01T00:00:00Z'),
            ('vcs.modified', 'true'),
            ('GOARCH', 'amd64'),
            ('GOOS', 'darwin'),
        ]:
            info = build_info()
            for setting in info['Settings']:
                if setting['Key'] == key:
                    setting['Value'] = value
            with self.subTest(key=key), self.assertRaises(ValueError):
                provenance.validate_binary(info, SHA, TIME, 'linux/arm64')
        for value in (
            '(devel)',
            'v2.18.1',
            'v0.0.0-20261114123456-bbbbbbbbbbbb',
        ):
            info = build_info()
            info['Main']['Version'] = value
            with self.subTest(version=value), self.assertRaises(ValueError):
                provenance.validate_binary(info, SHA, TIME, 'linux/arm64')
        for mutation in (
            lambda i: i.update(ELFMachine='EM_X86_64'),
            lambda i: i.update(Path='command-line-arguments'),
            lambda i: i['Main'].update(Replace={'Path': 'other'}),
            lambda i: i['Settings'].pop(),
            lambda i: i['Settings'].append(i['Settings'][0]),
        ):
            info = build_info()
            mutation(info)
            with self.assertRaises(ValueError):
                provenance.validate_binary(info, SHA, TIME, 'linux/arm64')

    def init_source(self, root):
        source = root / 'source'
        source.mkdir()
        (source / 'Dockerfile').write_text('FROM scratch\nCOPY . /source\n')
        (source / '.dockerignore').write_text('.git\nDockerfile\n')
        (source / 'tracked').write_text('committed\n')
        for args in (
            ['init', '--template='],
            ['add', '.'],
            [
                '-c',
                'user.name=Fixture',
                '-c',
                'user.email=fixture@example.invalid',
                'commit',
                '-m',
                'source',
            ],
        ):
            provenance._git(args, source)
        return source, provenance._git(['rev-parse', 'HEAD'], source)

    def test_context_does_not_copy_caller_secrets_hooks_or_untracked_files(
        self,
    ):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            source, sha = self.init_source(root)
            provenance._git(
                ['config', 'http.extraHeader', 'AUTHORIZATION: fixture-secret'],
                source,
            )
            (source / '.git/hooks').mkdir()
            (source / '.git/hooks/post-checkout').write_text('secret hook')
            (source / 'untracked-secret').write_text('not source')
            result = provenance.prepare_context(
                source, sha, 'Dockerfile', root / 'prepared'
            )
            context = result['context']
            self.assertEqual(
                provenance._git(['rev-parse', 'HEAD'], context), sha
            )
            self.assertEqual(
                provenance._git(['status', '--porcelain'], context), ''
            )
            self.assertFalse((context / 'untracked-secret').exists())
            self.assertFalse((context / '.git/hooks/post-checkout').exists())
            self.assertNotIn(
                'fixture-secret', (context / '.git/config').read_text()
            )
            self.assertEqual(
                (context / '.dockerignore').read_bytes(),
                (source / '.dockerignore').read_bytes(),
            )
            self.assertEqual(
                result['dockerfile'].read_bytes(),
                (source / 'Dockerfile').read_bytes(),
            )
            self.assertEqual(
                result['dockerfile']
                .with_name('build.Dockerfile.dockerignore')
                .read_text(),
                '',
            )
            self.assertNotIn(result['dockerfile'], context.rglob('*'))

    def test_source_identity_works_in_bare_repository(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            source, sha = self.init_source(root)
            bare = root / 'source.git'
            bare.mkdir()
            provenance._git(['init', '--bare', '--template='], bare)
            provenance._git(
                ['-c', 'protocol.file.allow=always', 'fetch', str(source), sha],
                bare,
            )
            self.assertEqual(
                provenance.source_identity(bare, sha),
                provenance.source_identity(source, sha),
            )

    def test_raw_scan_version_must_match_binary(self):
        report = {
            'results': [
                {
                    'source': {'path': '/bin/app', 'type': 'artifact'},
                    'packages': [
                        {
                            'package': {
                                'name': provenance.MODULE,
                                'ecosystem': 'Go',
                                'version': provenance.pseudo_version(SHA, TIME),
                            },
                            'vulnerabilities': [],
                        }
                    ],
                }
            ]
        }
        self.assertEqual(
            provenance.scan_binaries(report),
            {'/bin/app': provenance.pseudo_version(SHA, TIME)},
        )
        duplicate = copy.deepcopy(report)
        duplicate['results'].append(duplicate['results'][0])
        with self.assertRaises(ValueError):
            provenance.scan_binaries(duplicate)
        with patch.object(provenance, 'verify'):
            evidence = {
                'binaries': [
                    {
                        'path': '/bin/app',
                        'version': provenance.pseudo_version(SHA, TIME),
                    }
                ]
            }
            provenance.verify_scan(
                evidence, report, IMAGE, 'linux/arm64', SHA, TIME
            )
            report['results'][0]['packages'][0]['package']['version'] = (
                '(devel)'
            )
            with self.assertRaises(ValueError):
                provenance.verify_scan(
                    evidence, report, IMAGE, 'linux/arm64', SHA, TIME
                )

    def test_context_supports_worktree_git_file(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            source, sha = self.init_source(root)
            worktree = root / 'worktree'
            provenance._git(
                ['worktree', 'add', '--detach', str(worktree), sha], source
            )
            self.assertTrue((worktree / '.git').is_file())
            prepared = provenance.prepare_context(
                worktree, sha, 'Dockerfile', root / 'prepared'
            )
            self.assertTrue((prepared['context'] / '.git').is_dir())

    def test_committed_symlink_ancestor_cannot_supply_external_recipe(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            source, _ = self.init_source(root)
            outside = root / 'outside'
            outside.mkdir()
            (outside / 'Dockerfile').write_text('FROM external-secret\n')
            (source / 'linked').symlink_to(outside, target_is_directory=True)
            provenance._git(['add', 'linked'], source)
            provenance._git(
                [
                    '-c',
                    'user.name=Fixture',
                    '-c',
                    'user.email=fixture@example.invalid',
                    'commit',
                    '-m',
                    'symlink',
                ],
                source,
            )
            sha = provenance._git(['rev-parse', 'HEAD'], source)
            with self.assertRaisesRegex(ValueError, 'tracked regular file'):
                provenance.prepare_context(
                    source, sha, 'linked/Dockerfile', root / 'prepared'
                )
            self.assertFalse((root / 'prepared/build.Dockerfile').exists())

    def test_dirty_source_and_escaping_recipe_fail_closed(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            source, sha = self.init_source(root)
            with self.assertRaises(ValueError):
                provenance.prepare_context(
                    source, sha, '../Dockerfile', root / 'prepared'
                )
            (source / 'tracked').write_text('modified\n')
            with self.assertRaisesRegex(ValueError, 'tracked modifications'):
                provenance.prepare_context(
                    source, sha, 'Dockerfile', root / 'prepared'
                )
            provenance._git(['add', 'tracked'], source)
            with self.assertRaisesRegex(ValueError, 'tracked modifications'):
                provenance.prepare_context(
                    source, sha, 'Dockerfile', root / 'prepared'
                )

    def test_collect_never_runs_candidate_and_offline_verify_checks_summary(
        self,
    ):
        info = build_info()
        commands = []

        def capture(command, **kwargs):
            commands.append(command)
            if command[:3] == ['docker', 'image', 'inspect']:
                return json.dumps(
                    {'Id': IMAGE, 'Os': 'linux', 'Architecture': 'arm64'}
                )
            if command[:2] == ['docker', 'create']:
                return 'container-id\n'
            self.assertEqual(command[:2], ['go', 'run'])
            self.assertEqual(kwargs['env']['GOPROXY'], 'off')
            return json.dumps(info)

        def run(command, **kwargs):
            commands.append(command)
            if command[:2] == ['docker', 'cp']:
                self.assertEqual(command[2], '-L')
                Path(command[-1]).write_bytes(b'fixture-binary')

        with (
            tempfile.TemporaryDirectory() as temp,
            patch.object(subprocess, 'check_output', side_effect=capture),
            patch.object(subprocess, 'run', side_effect=run),
        ):
            result = provenance.collect(
                IMAGE,
                'linux/arm64',
                SHA,
                TIME,
                ['/usr/bin/apiserver'],
                Path(temp),
            )
            provenance.verify(
                result, IMAGE, 'linux/arm64', SHA, TIME, ['/usr/bin/apiserver']
            )
            self.assertTrue((Path(temp) / 'go-provenance.json').is_file())
            bad = copy.deepcopy(result)
            bad['binaries'][0]['revision'] = 'c' * 40
            with self.assertRaises(ValueError):
                provenance.verify(
                    bad, IMAGE, 'linux/arm64', SHA, TIME, ['/usr/bin/apiserver']
                )
            with self.assertRaises(ValueError):
                provenance.verify(
                    result, IMAGE, 'linux/arm64', SHA, TIME, ['/bin/other']
                )
            self.assertFalse(
                any(
                    c[:2]
                    in (
                        ['docker', 'run'],
                        ['docker', 'start'],
                        ['docker', 'exec'],
                    )
                    for c in commands
                )
            )
            self.assertEqual(commands[-1], ['docker', 'rm', 'container-id'])

            def symlink_copy(command, **kwargs):
                if command[:2] == ['docker', 'cp']:
                    Path(command[-1]).symlink_to(Path(temp) / 'outside-binary')

            with (
                patch.object(subprocess, 'run', side_effect=symlink_copy),
                self.assertRaisesRegex(ValueError, 'regular scratch file'),
            ):
                provenance.collect(
                    IMAGE,
                    'linux/arm64',
                    SHA,
                    TIME,
                    ['/usr/bin/apiserver'],
                    Path(temp),
                )


if __name__ == '__main__':
    unittest.main()
