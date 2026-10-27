"""Qualify installed release wheels in isolated supported Python
environments."""

import hashlib
import json
import os
import re
import shutil
import subprocess
import tempfile
from pathlib import Path

from .packages import LEGACY_PACKAGES


def default_interpreters(root: Path) -> list[str]:
    """Use the branch's declared Python minors without dropping older
    support."""
    sdk = root / 'sdk/python'
    source = '\n'.join(
        path.read_text()
        for path in (sdk / 'pyproject.toml', sdk / 'setup.py')
        if path.is_file()
    )
    return ['python' + version for version in supported_python_versions(source)]


def supported_python_versions(source: str) -> list[str]:
    """Parse advertised Python minors from SDK packaging metadata."""
    versions = set(
        re.findall(r'Programming Language :: Python :: (3\.\d+)', source)
    )
    if not versions:
        minimum = re.search(
            r'(?:requires-python|python_requires)\s*=\s*[\'\"]>=\s*(3\.\d+)',
            source,
        )
        if minimum:
            versions.add(minimum.group(1))
    if not versions:
        raise ValueError(
            'Cannot determine supported Python versions; add Python minor classifiers or a requires-python minimum to SDK package metadata'
        )
    return sorted(versions, key=lambda v: tuple(map(int, v.split('.'))))


_SMOKE = """import importlib
import importlib.metadata
import json
from pathlib import Path
import sys
from kfp import compiler, dsl, kubernetes
from kfp.pipeline_spec import pipeline_spec_pb2
from kfp.kubernetes import kubernetes_executor_config_pb2

expected = json.loads(sys.argv[1])
for name, version in expected.items():
    actual = importlib.metadata.version(name)
    if actual != version:
        raise RuntimeError(f"{name}: expected {version}, installed {actual}")
importlib.import_module(sys.argv[2])

@dsl.container_component
def echo():
    return dsl.ContainerSpec(image='busybox:1.36', command=['echo', 'release-smoke'])

@dsl.pipeline(name='release-smoke')
def pipeline():
    kubernetes.add_pod_label(echo(), 'release-smoke', 'true')

compiler.Compiler().compile(pipeline, package_path='pipeline.yaml')
compiled = Path('pipeline.yaml').read_text()
if 'release-smoke' not in compiled or 'platforms' not in compiled:
    raise RuntimeError('Compiled pipeline is missing Kubernetes platform configuration')
print(json.dumps({'python': sys.version, 'distributions': expected}))
"""


_BOOTSTRAP = """import importlib.metadata
import json
import sys
from pip._vendor.packaging.requirements import Requirement
from pip._vendor.packaging.utils import canonicalize_name

installed = {canonicalize_name(d.metadata['Name']): d for d in importlib.metadata.distributions()}
pending = list(json.loads(sys.argv[1]))
required = set()
try:
    while pending:
        name = canonicalize_name(pending.pop())
        if name in required:
            continue
        required.add(name)
        distribution = installed[name]
        # Include inactive extras conservatively: a declared installer dependency
        # must never be discarded because qualification did not enable its extra.
        for requirement in distribution.requires or []:
            dependency = canonicalize_name(Requirement(requirement).name)
            if dependency in installed:
                pending.append(dependency)
except (KeyError, ValueError):
    required.update(('pip', 'setuptools'))
print(json.dumps(sorted({'pip', 'setuptools'}.intersection(installed) - required)))
"""

_FREEZE = """import importlib.metadata
print('\\n'.join(sorted(f"{d.metadata['Name']}=={d.version}" for d in importlib.metadata.distributions())))
"""


def qualify_wheels(
    root: Path, version: str, interpreters: list[str], output_dir: Path
) -> dict:
    """Install and exercise local artifacts; retain evidence and fail on any
    lane."""
    if not interpreters:
        raise ValueError(
            'Specify at least one Python interpreter for wheel qualification'
        )
    resolved = []
    actual_versions = set()
    for interpreter in interpreters:
        executable = shutil.which(interpreter)
        if executable is None:
            raise ValueError(
                f'Missing Python interpreter {interpreter}; install it before qualification'
            )
        resolved.append(str(Path(executable).absolute()))
        actual_versions.add(
            subprocess.check_output(
                [
                    executable,
                    '-I',
                    '-c',
                    'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")',
                ],
                text=True,
            ).strip()
        )
    required_versions = {
        name.removeprefix('python') for name in default_interpreters(root)
    }
    missing = required_versions - actual_versions
    if missing:
        raise ValueError(
            f'Missing supported Python versions: {", ".join(sorted(missing))}; supply an interpreter for every declared minor'
        )

    root = root.resolve()
    output_dir = output_dir.resolve()
    output_dir.mkdir(parents=True, exist_ok=True)
    consolidated = (root / 'sdk/python/kfp/server_api/__init__.py').is_file()
    packages = (('sdk/python', 'kfp', ()),) if consolidated else LEGACY_PACKAGES
    wheels = []
    for directory, name, _ in packages:
        matches = list(
            (root / directory / 'dist').glob(
                f'{name.replace("-", "_")}-{version}-*.whl'
            )
        )
        if len(matches) != 1:
            raise ValueError(
                f'Expected one {name} {version} wheel; found {len(matches)}. Build release wheels first.'
            )
        wheels.append(matches[0])
    expected = {name: version for _, name, _ in packages}
    report = {
        'version': version,
        'wheels': {
            str(path): hashlib.sha256(path.read_bytes()).hexdigest()
            for path in wheels
        },
        'environments': [],
        'python_environments': [],
        'passed': False,
    }
    environment = os.environ.copy()
    for key in ('PYTHONPATH', 'PYTHONHOME', 'VIRTUAL_ENV'):
        environment.pop(key, None)
    environment['PYTHONNOUSERSITE'] = '1'

    try:
        for index, executable in enumerate(resolved):
            lane = output_dir / f'python-{index + 1}'
            lane.mkdir(exist_ok=True)
            result = {
                'interpreter': executable,
                'version': version,
                'distributions': expected,
                'passed': False,
                'freeze': str(lane / 'requirements.txt'),
            }
            report['environments'].append(result)
            working = Path(
                tempfile.mkdtemp(prefix='kfpr-smoke-', dir=output_dir)
            )
            report['python_environments'].append(str(working / 'venv'))
            with (lane / 'commands.log').open('w') as log:

                def run(command):
                    completed = subprocess.run(
                        command,
                        cwd=working,
                        env=environment,
                        text=True,
                        stdout=subprocess.PIPE,
                        stderr=subprocess.STDOUT,
                    )
                    log.write(f'$ {command!r}\n{completed.stdout}\n')
                    log.flush()
                    if completed.returncode:
                        raise RuntimeError(
                            f'Wheel qualification failed for {executable}; inspect {lane / "commands.log"}'
                        )
                    return completed.stdout

                run([executable, '-m', 'venv', str(working / 'venv')])
                python = str(
                    working
                    / 'venv'
                    / (
                        'Scripts/python.exe'
                        if os.name == 'nt'
                        else 'bin/python'
                    )
                )
                run(
                    [
                        python,
                        '-m',
                        'pip',
                        'install',
                        '--disable-pip-version-check',
                        *map(str, wheels),
                    ]
                )
                run([python, '-m', 'pip', 'check'])
                result['removed_bootstrap'] = json.loads(
                    run([python, '-c', _BOOTSTRAP, json.dumps(list(expected))])
                )
                if result['removed_bootstrap']:
                    run(
                        [
                            python,
                            '-m',
                            'pip',
                            'uninstall',
                            '--yes',
                            *result['removed_bootstrap'],
                        ]
                    )
                (lane / 'requirements.txt').write_text(
                    run([python, '-c', _FREEZE])
                )
                script = working / 'smoke.py'
                script.write_text(_SMOKE)
                result['output'] = run(
                    [
                        python,
                        str(script),
                        json.dumps(expected),
                        'kfp.server_api' if consolidated else 'kfp_server_api',
                    ]
                )
                shutil.copyfile(
                    working / 'pipeline.yaml', lane / 'pipeline.yaml'
                )
                result['passed'] = True
        report['passed'] = True
    finally:
        (output_dir / 'qualification.json').write_text(
            json.dumps(report, indent=2) + '\n'
        )
    return report
