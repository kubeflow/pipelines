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

import os
from pathlib import Path
import shlex
import subprocess
import sys
import textwrap

import pytest

PRESUBMIT = Path(__file__).resolve().parents[3] / 'test/presubmit-tests-sdk.sh'


@pytest.mark.regression
def test_distributed_presubmit_reports_failed_subprocess_output(tmp_path):
    """Run the real presubmit command with a failing child process."""
    tests = tmp_path / 'sdk/python/test'
    tests.mkdir(parents=True)
    pytest_config = tmp_path / 'pytest.ini'
    pytest_config.write_text(
        '[pytest]\nmarkers = regression: SDK regression tests\n')
    diagnostic = 'child task failed before producing its outputs'
    (tests / 'fail_task.py').write_text(
        f'import sys\nprint({diagnostic!r}, file=sys.stderr)\nsys.exit(1)\n')
    (tests / 'kfp.py').write_text('VALUE = 1\n')
    (tests / 'output_test.py').write_text(
        textwrap.dedent('''\
        import os
        from pathlib import Path
        import subprocess
        import sys

        import kfp
        import pytest

        @pytest.mark.regression
        def test_failed_task():
            assert os.environ['PYTEST_XDIST_WORKER_COUNT'] == '2'
            assert kfp.VALUE == 1
            result = subprocess.run(
                [sys.executable, str(Path(__file__).with_name('fail_task.py'))],
                stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True,
            )
            print(result.stdout, end='')
            assert result.returncode == 0, 'inner task failed'

        def test_outside_regression_selection():
            Path('unexpected-test-ran').touch()
        '''))

    # Keep every pytest argument from the script, but use the existing test
    # environment instead of syncing the repository or starting the full suite.
    bin_dir = tmp_path / 'bin'
    bin_dir.mkdir()
    uv = bin_dir / 'uv'
    uv.write_text('#!/bin/sh\n'
                  'test "$1" = run && test "$2" = python || exit 2\n'
                  'shift 2\n'
                  f'exec {shlex.quote(sys.executable)} "$@"\n')
    uv.chmod(0o755)

    # Nested pytest must not inherit the outer worker's capture/coverage state.
    env = {
        key: value
        for key, value in os.environ.items()
        if not key.startswith(('PYTEST_', 'COV_CORE_', 'COVERAGE_'))
    }
    env.update({
        'PATH': f'{bin_dir}{os.pathsep}{env.get("PATH", "")}',
        'SETUP_ENV': 'false',
        'REPO_NAME': 'kubeflow/pipelines',
        'PULL_NUMBER': '1',
        'PYTEST_PARALLEL_WORKERS': '2',
        'COVERAGE_FILE': str(tmp_path / '.coverage'),
    })
    result = subprocess.run(
        ['bash', str(PRESUBMIT)],
        cwd=tmp_path,
        env=env,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        timeout=60,
    )

    assert result.returncode == 1, result.stdout
    assert not (tmp_path / 'unexpected-test-ran').exists(), result.stdout
    assert 'Captured stdout call' in result.stdout, result.stdout
    assert diagnostic in result.stdout.split('Captured stdout call', 1)[1]
