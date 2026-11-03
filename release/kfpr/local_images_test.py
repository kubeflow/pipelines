"""Local qualification owns and cleans only its image tags and builders."""

import subprocess
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest.mock import patch

from kfpr.local_images import build_images


class LocalImagesTest(unittest.TestCase):
    def fixture(self, root):
        (root / 'frontend').mkdir()
        (root / 'frontend/.nvmrc').write_text('v24.1.0')
        return [
            {
                'name': 'api',
                'platform': 'linux/amd64',
                'dockerfile': 'Dockerfile',
                'context': '.',
            },
            {
                'name': 'api',
                'platform': 'linux/arm64',
                'dockerfile': 'Dockerfile',
                'context': '.',
            },
        ]

    def test_scan_finishes_before_cleanup_and_next_build(self):
        with TemporaryDirectory() as temporary:
            root = Path(temporary)
            records = self.fixture(root)
            commands = []

            def run(command, **kwargs):
                commands.append(command)
                return subprocess.CompletedProcess(command, 0)

            with (
                patch('kfpr.local_images.subprocess.run', side_effect=run),
                patch(
                    'kfpr.local_images.subprocess.check_output',
                    return_value='sha256:' + 'a' * 64,
                ),
            ):
                images = build_images(root, 'b' * 40, '2.17.3', records)
                first = next(images)
                self.assertFalse(any('rm' in command for command in commands))
                second = next(images)
                self.assertNotEqual(first['tag'], second['tag'])
                first_build = next(
                    command
                    for command in commands
                    if command[:3] == ['docker', 'buildx', 'build']
                )
                first_builder = first_build[first_build.index('--builder') + 1]
                self.assertIn(['docker', 'image', 'rm', first['tag']], commands)
                self.assertIn(
                    ['docker', 'buildx', 'rm', '--force', first_builder],
                    commands,
                )
                images.close()
            self.assertEqual(
                sum(
                    command[:3] == ['docker', 'buildx', 'create']
                    for command in commands
                ),
                2,
            )
            self.assertEqual(
                sum(
                    command[:3] == ['docker', 'buildx', 'rm']
                    for command in commands
                ),
                2,
            )
            self.assertFalse(
                any(
                    'prune' in command or '--use' in command
                    for command in commands
                )
            )
            self.assertFalse(
                any(
                    command[:3] == ['docker', 'image', 'rm']
                    and command[-1].startswith('sha256:')
                    for command in commands
                )
            )

    def test_build_failure_cleans_partial_image_and_private_builder(self):
        with TemporaryDirectory() as temporary:
            root = Path(temporary)
            records = self.fixture(root)
            commands = []

            def run(command, **kwargs):
                commands.append(command)
                if command[:3] == ['docker', 'buildx', 'build']:
                    raise subprocess.CalledProcessError(1, command)
                return subprocess.CompletedProcess(command, 0)

            with (
                patch('kfpr.local_images.subprocess.run', side_effect=run),
                patch(
                    'kfpr.local_images.subprocess.check_output',
                    return_value='sha256:' + 'a' * 64,
                ),
                self.assertRaises(subprocess.CalledProcessError),
            ):
                next(build_images(root, 'b' * 40, '2.17.3', records))
            self.assertTrue(
                any(
                    command[:3] == ['docker', 'image', 'rm']
                    for command in commands
                )
            )
            self.assertEqual(
                commands[-1][:4], ['docker', 'buildx', 'rm', '--force']
            )

    def test_invalid_image_id_still_removes_tag_and_builder(self):
        with TemporaryDirectory() as temporary:
            root = Path(temporary)
            records = self.fixture(root)
            with (
                patch('kfpr.local_images.subprocess.run') as run,
                patch(
                    'kfpr.local_images.subprocess.check_output',
                    return_value='invalid-image',
                ),
                self.assertRaisesRegex(
                    ValueError, 'Invalid locally built image'
                ),
            ):
                next(build_images(root, 'b' * 40, '2.17.3', records))
            commands = [call.args[0] for call in run.call_args_list]
            self.assertTrue(
                any(
                    command[:3] == ['docker', 'image', 'rm']
                    for command in commands
                )
            )
            self.assertEqual(
                commands[-1][:4], ['docker', 'buildx', 'rm', '--force']
            )

    def test_image_cleanup_failure_still_removes_private_builder(self):
        with TemporaryDirectory() as temporary:
            root = Path(temporary)
            records = self.fixture(root)
            commands = []

            def run(command, **kwargs):
                commands.append(command)
                if command[:3] == ['docker', 'image', 'rm']:
                    raise subprocess.CalledProcessError(1, command)

            with (
                patch('kfpr.local_images.subprocess.run', side_effect=run),
                patch(
                    'kfpr.local_images.subprocess.check_output',
                    return_value='sha256:' + 'a' * 64,
                ),
            ):
                images = build_images(root, 'b' * 40, '2.17.3', records)
                next(images)
                with self.assertRaises(subprocess.CalledProcessError):
                    images.close()
            self.assertEqual(
                commands[-1][:4], ['docker', 'buildx', 'rm', '--force']
            )

    def test_builder_create_failure_does_not_remove_existing_builder(self):
        with TemporaryDirectory() as temporary:
            root = Path(temporary)
            records = self.fixture(root)
            with (
                patch(
                    'kfpr.local_images.subprocess.run',
                    side_effect=subprocess.CalledProcessError(1, 'create'),
                ) as run,
                self.assertRaises(subprocess.CalledProcessError),
            ):
                next(build_images(root, 'b' * 40, '2.17.3', records))
            self.assertEqual(run.call_count, 1)


if __name__ == '__main__':
    unittest.main()
