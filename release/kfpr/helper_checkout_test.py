"""Exercise source-launched SDK preparation after checkout removes helpers."""

import os
import shutil
import subprocess
import sys
import unittest
import zipfile
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock


def _prepare_after_checkout(root: Path, missing_module: bool) -> None:
    # Import before simulating checkout, as the source/editable CLI does.
    from kfpr import core, packages, steps

    Path(packages.__file__).unlink()
    stale = []
    for relative, name, _ in packages.LEGACY_PACKAGES:
        dist = root / relative / 'dist'
        dist.mkdir(parents=True)
        artifact = dist / f'{name.replace("-", "_")}-2.17.2-py3-none-any.whl'
        artifact.write_bytes(b'old release')
        stale.append(artifact)
    preserved = root / 'sdk/python/dist/kfp_other-2.17.2-py3-none-any.whl'
    preserved.write_bytes(b'unrelated package')

    class BuildRunner:
        dry_run = False

        def __init__(self):
            self.built = []
            self.generated = False

        def run(self, command, cwd=None):
            if len(command) > 1 and command[1].endswith('packages.py'):
                # Execute the real helper command so a missing source file
                # fails exactly as it would after a maintenance checkout.
                return subprocess.run(command, cwd=cwd, check=True)
            if command[:2] == ['make', 'python']:
                assert all(not artifact.exists() for artifact in stale)
                assert preserved.read_bytes() == b'unrelated package'
                self.generated = True
            if command[1:3] == ['-m', 'build']:
                assert self.generated, (
                    'bindings must be generated before builds'
                )
                relative = Path(cwd).relative_to(root).as_posix()
                _, name, modules = next(
                    package
                    for package in packages.LEGACY_PACKAGES
                    if package[0] == relative
                )
                normalized = name.replace('-', '_')
                wheel = (
                    Path(cwd) / 'dist' / f'{normalized}-2.17.3-py3-none-any.whl'
                )
                with zipfile.ZipFile(wheel, 'w') as archive:
                    archive.writestr(
                        f'{normalized}-2.17.3.dist-info/METADATA',
                        f'Name: {name}\nVersion: 2.17.3\n',
                    )
                    for module in modules:
                        if missing_module and name == 'kfp-kubernetes':
                            continue
                        archive.writestr(module, '# generated source\n')
                self.built.append(name)
            return subprocess.CompletedProcess(command, 0)

    runner = BuildRunner()
    context = core.ReleaseContext(
        root=root,
        state=core.ReleaseState(root / 'state.json'),
        runner=runner,
        metadata=core.ReleaseMetadata.from_version('patch', '2.17.3'),
        fork_remote='origin',
        include_backend=False,
        include_sdk=True,
    )
    with mock.patch.object(
        steps, '_generate_sdk_release_notes', return_value=''
    ):
        if missing_module:
            try:
                steps._update_sdk_version_files(context)
            except ValueError as error:
                assert 'kubernetes_executor_config_pb2.py' in str(error)
                assert 'regenerate bindings and rebuild' in str(error)
            else:
                raise AssertionError('missing generated modules were accepted')
        else:
            steps._update_sdk_version_files(context)
    assert runner.built == [name for _, name, _ in packages.LEGACY_PACKAGES]


class HelperCheckoutTest(unittest.TestCase):
    def test_dry_run_does_not_clean_or_validate_artifacts(self):
        from kfpr import core, steps

        with TemporaryDirectory() as directory:
            root = Path(directory)
            runner = mock.Mock(spec=core.CommandRunner)
            runner.dry_run = True
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=runner,
                metadata=core.ReleaseMetadata.from_version('patch', '2.17.3'),
                fork_remote='origin',
                include_backend=False,
                include_sdk=True,
            )
            with (
                mock.patch('kfpr.steps.clean_legacy_distributions') as clean,
                mock.patch('kfpr.steps.validate_legacy_wheels') as validate,
            ):
                steps._update_sdk_version_files(context)
            clean.assert_not_called()
            validate.assert_not_called()
            self.assertEqual(list(root.iterdir()), [])

    def run_checkout(self, scenario: str) -> None:
        with TemporaryDirectory() as directory:
            temporary = Path(directory)
            tooling = temporary / 'tooling'
            copied = tooling / 'kfpr'
            shutil.copytree(
                Path(__file__).parent,
                copied,
                ignore=shutil.ignore_patterns('__pycache__'),
            )
            root = temporary / 'checkout'
            root.mkdir()
            result = subprocess.run(
                [
                    sys.executable,
                    str(copied / Path(__file__).name),
                    '--scenario',
                    scenario,
                    str(root),
                ],
                cwd=temporary,
                env={**os.environ, 'PYTHONPATH': str(tooling)},
                capture_output=True,
                text=True,
            )
            self.assertEqual(
                result.returncode, 0, result.stdout + result.stderr
            )

    def test_cleanup_and_validation_survive_source_checkout(self):
        self.run_checkout('valid')

    def test_missing_generated_module_rejected_after_source_checkout(self):
        self.run_checkout('missing')


if __name__ == '__main__':
    if len(sys.argv) > 1 and sys.argv[1] == '--scenario':
        _prepare_after_checkout(Path(sys.argv[3]), sys.argv[2] == 'missing')
    else:
        unittest.main()
