#!/usr/bin/env python3
"""Validate unified SDK artifacts and install them outside the source tree."""

import argparse
from email.parser import Parser
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import zipfile

REQUIRED_MODULES = {
    'kfp/pipeline_spec/pipeline_spec_pb2.py',
    'kfp/kubernetes/kubernetes_executor_config_pb2.py',
    'kfp/kubernetes/__init__.py',
    'kfp/server_api/__init__.py',
    'kfp/server_api/api/run_service_api.py',
}
RETIRED_DISTRIBUTIONS = {
    'kfp-pipeline-spec', 'kfp-server-api', 'kfp-kubernetes'
}
IMPORT_CHECK = '''
from importlib import metadata
import kfp
from kfp import kubernetes, server_api
from kfp.pipeline_spec import pipeline_spec_pb2
from kfp.kubernetes import kubernetes_executor_config_pb2
from kfp.server_api.api.run_service_api import RunServiceApi

assert kfp.__version__ == metadata.version("kfp")
assert kubernetes.__version__ == kfp.__version__ == server_api.__version__
assert pipeline_spec_pb2.PipelineSpec().SerializeToString() == b""
assert kubernetes_executor_config_pb2.KubernetesExecutorConfig().SerializeToString() == b""
for name in ("kfp-pipeline-spec", "kfp-server-api", "kfp-kubernetes"):
    try:
        metadata.distribution(name)
    except metadata.PackageNotFoundError:
        continue
    raise AssertionError(f"Retired distribution is still installed: {name}")
'''


def check_wheel(path: Path) -> None:
    """Check module ownership, dependency metadata, and test exclusions."""
    with zipfile.ZipFile(path) as wheel:
        names = set(wheel.namelist())
        missing = REQUIRED_MODULES - names
        if missing:
            raise AssertionError(f'{path}: missing modules: {sorted(missing)}')
        metadata_files = [
            name for name in names if name.endswith('.dist-info/METADATA')
        ]
        if len(metadata_files) != 1:
            raise AssertionError(f'{path}: expected one distribution')
        metadata = Parser().parsestr(
            wheel.read(metadata_files[0]).decode('utf-8'))
        if metadata['Name'] != 'kfp':
            raise AssertionError(f'{path}: not a kfp distribution')
        for dependency in metadata.get_all('Requires-Dist', []):
            if any(
                    dependency.startswith(name)
                    for name in RETIRED_DISTRIBUTIONS):
                raise AssertionError(
                    f'{path}: retired dependency: {dependency}')
        for name in names:
            filename = Path(name).name
            if (filename.startswith('test_') or filename.endswith(
                ('_test.py', '_tests.py')) or '/testdata/' in name or
                    '/test_data/' in name):
                raise AssertionError(f'{path}: contains test payload: {name}')


def run(command: list[str], cwd: Path) -> None:
    """Execute validation with no imports inherited from the checkout."""
    environment = {
        key: value for key, value in os.environ.items() if key != 'PYTHONPATH'
    }
    subprocess.run(command, cwd=cwd, env=environment, check=True)


def main() -> None:
    """Check wheel and sdist builds using a clean, consumer-like
    environment."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('dist', type=Path)
    parser.add_argument('--python', default=sys.executable)
    args = parser.parse_args()
    distributions = args.dist.resolve()
    wheels = list(distributions.glob('kfp-*.whl'))
    sdists = list(distributions.glob('kfp-*.tar.gz'))
    if len(wheels) != 1 or len(sdists) != 1:
        raise AssertionError('Expected exactly one kfp wheel and one sdist.')
    check_wheel(wheels[0])
    with tempfile.TemporaryDirectory(prefix='kfp-distribution-') as directory:
        root = Path(directory)
        run([args.python, '-m', 'venv', str(root / 'venv')], root)
        python = str(root / 'venv/bin/python')
        run([python, '-m', 'pip', 'install', str(wheels[0])], root)
        run([python, '-I', '-c', IMPORT_CHECK], root)
        run([python, '-m', 'pip', 'check'], root)
        # pip builds in isolation from the archive, without the monorepo/protoc.
        run([
            python, '-m', 'pip', 'wheel', '--no-deps',
            str(sdists[0]), '--wheel-dir',
            str(root / 'rebuilt')
        ], root)
        rebuilt = next((root / 'rebuilt').glob('kfp-*.whl'))
        check_wheel(rebuilt)
        run([
            python, '-m', 'pip', 'install', '--force-reinstall', '--no-deps',
            str(rebuilt)
        ], root)
        run([python, '-I', '-c', IMPORT_CHECK], root)


if __name__ == '__main__':
    main()
