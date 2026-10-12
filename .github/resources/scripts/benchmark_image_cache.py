#!/usr/bin/env python3
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
"""Measure the API server builder's layer and persisted Go caches."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import shutil
import subprocess
import tempfile
import time
import uuid

ROOT = Path(__file__).resolve().parents[3]
MARKER = 'kfp-cache-benchmark-source-change'
SCENARIOS = ('cold', 'warm', 'source-edit-warm', 'source-edit-cold')


def command(args, log, env, cwd=None):
    started = time.monotonic()
    print(f'Running {log.name}', flush=True)
    with log.open('w') as stream:
        subprocess.run(
            args,
            cwd=cwd,
            env=env,
            stdout=stream,
            stderr=subprocess.STDOUT,
            check=True,
            timeout=900)
    return round(time.monotonic() - started, 3)


def archive_source(repository, context, log, env):
    archive = context.parent / 'source.tar'
    command(
        ['git', 'archive', '--format=tar', '--output',
         str(archive), 'HEAD'],
        log,
        env,
        cwd=repository)
    context.mkdir()
    command(['tar', '-xf', str(archive), '-C',
             str(context)], log.with_suffix('.extract.log'), env)
    archive.unlink()


def log_counts(log):
    return {
        'cached_steps':
            len(re.findall(r'^#\d+ CACHED$', log, re.MULTILINE)),
        'compiler_invocations':
            len(
                re.findall(r'^#\d+ [\d.]+ \S*/pkg/tool/[^/\s]+/compile(?: |$)',
                           log, re.MULTILINE)),
    }


def validate_results(results):
    cold, warm, edited_warm, edited_cold = results
    if cold['sha256'] != warm['sha256']:
        raise RuntimeError('An unchanged warm build changed the API binary')
    if edited_warm['sha256'] != edited_cold['sha256']:
        raise RuntimeError('The source-edited warm and cold binaries differ')
    if cold['sha256'] == edited_warm['sha256']:
        raise RuntimeError('The source edit did not change the API binary')
    if not cold['compiler_invocations'] or not edited_cold[
            'compiler_invocations']:
        raise RuntimeError('Cold builds did not show Go compiler invocations')
    if warm['compiler_invocations'] or not warm['cached_steps']:
        raise RuntimeError('The unchanged build did not reuse Docker layers')
    if not 0 < edited_warm['compiler_invocations'] < edited_cold[
            'compiler_invocations']:
        raise RuntimeError('The source edit did not demonstrate Go cache reuse')


def write_summary(output, summary):
    (output / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
    lines = [
        '## API server compilation cache benchmark',
        '',
        'Four fresh BuildKit builders use the actual API server Go stage, '
        'instrumented with `go build -x`. This measures compilation, not the '
        'full image or CI. Layer and Go caches are transferred locally; GitHub '
        'cache upload/download latency is not measured. The cold build includes '
        'the initial layer-cache export.',
        '',
        '| Scenario | Setup (s) | Go restore (s) | Build (s) | Go save (s) | '
        'Cleanup (s) | Cached steps | Go compile commands |',
        '| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |',
    ]
    for result in summary['results']:
        seconds = result['seconds']
        values = [result['scenario']] + [
            str(seconds.get(key, 0))
            for key in ('setup', 'restore', 'build', 'save', 'cleanup')
        ] + [
            str(result.get(key, ''))
            for key in ('cached_steps', 'compiler_invocations')
        ]
        lines.append('| ' + ' | '.join(values) + ' |')
    lines.extend(['', f"Result: {summary['status']}"])
    if summary.get('error'):
        lines.append(summary['error'])
    markdown = '\n'.join(lines) + '\n'
    (output / 'summary.md').write_text(markdown)
    if os.environ.get('GITHUB_STEP_SUMMARY'):
        with open(os.environ['GITHUB_STEP_SUMMARY'], 'a') as stream:
            stream.write(markdown)


def benchmark(repository, output, runner_temp):
    output.mkdir(parents=True, exist_ok=False)
    summary = {
        'status': 'failed',
        'results': [],
        'initial_free_bytes': shutil.disk_usage(runner_temp).free
    }
    try:
        with tempfile.TemporaryDirectory(
                prefix='kfp-cache-benchmark-', dir=runner_temp) as temporary:
            workspace = Path(temporary)
            context = workspace / 'source'
            env = {
                **os.environ, 'TARGET_ARCH': 'amd64',
                'RUNNER_TEMP': str(workspace)
            }
            env.pop('GITHUB_OUTPUT', None)
            archive_source(repository, context, output / 'source.log', env)
            dockerfile = context / 'backend/Dockerfile'
            original = dockerfile.read_text()
            compile_command = 'go build -o /bin/apiserver '
            if original.count(compile_command) != 1:
                raise RuntimeError('Expected exactly one API server Go build')
            dockerfile.write_text(
                original.replace(compile_command,
                                 'go build -x -o /bin/apiserver ') +
                '\nFROM scratch AS cache-benchmark-output\n'
                'COPY --from=builder /bin/apiserver /apiserver\n')
            scripts = repository / '.github/resources/scripts'
            layers = workspace / 'layers'
            for scenario in SCENARIOS:
                if scenario == 'source-edit-warm':
                    with (context /
                          'backend/src/apiserver/main.go').open('a') as stream:
                        stream.write(
                            f'\nfunc init() {{ println("{MARKER}") }}\n')
                builder = f'kfp-cache-benchmark-{uuid.uuid4().hex[:12]}'
                result = {
                    'scenario': scenario,
                    'builder': builder,
                    'seconds': {}
                }
                summary['results'].append(result)
                seconds = result['seconds']
                binary_output = workspace / scenario
                try:
                    seconds['setup'] = command([
                        'bash',
                        str(scripts / 'setup-buildx-with-retry.sh'), builder
                    ], output / f'{scenario}.setup.log', env)
                    if scenario in ('warm', 'source-edit-warm'):
                        seconds['restore'] = command([
                            'bash',
                            str(scripts / 'buildkit-go-cache.sh'), 'restore',
                            str(dockerfile), builder
                        ], output / f'{scenario}.restore.log', env)
                    build = [
                        'docker', 'buildx', 'build', '--builder', builder,
                        '--platform', 'linux/amd64', '--progress', 'plain',
                        '--file',
                        str(dockerfile), '--target', 'cache-benchmark-output',
                        '--output', f'type=local,dest={binary_output}'
                    ]
                    if scenario == 'cold':
                        build.extend([
                            '--cache-to', f'type=local,dest={layers},mode=max'
                        ])
                    elif scenario in ('warm', 'source-edit-warm'):
                        build.extend(
                            ['--cache-from', f'type=local,src={layers}'])
                    build.append(str(context))
                    build_log = output / f'{scenario}.build.log'
                    seconds['build'] = command(build, build_log, env)
                    result.update(log_counts(build_log.read_text()))
                    with (binary_output / 'apiserver').open('rb') as stream:
                        result['sha256'] = hashlib.file_digest(
                            stream, 'sha256').hexdigest()
                    shutil.rmtree(binary_output)
                    if scenario == 'cold':
                        seconds['save'] = command([
                            'bash',
                            str(scripts / 'buildkit-go-cache.sh'), 'save',
                            str(dockerfile), builder
                        ], output / f'{scenario}.save.log', env)
                        cache = workspace / 'go-build-cache'
                        summary['go_cache_bytes'] = sum(
                            path.stat().st_size
                            for path in cache.rglob('*')
                            if path.is_file())
                        if not summary['go_cache_bytes']:
                            raise RuntimeError('The Go cache export was empty')
                finally:
                    # Remove only the uniquely named builder owned by this case.
                    # Stop on cleanup failure rather than accumulating builders.
                    seconds['cleanup'] = command(
                        ['docker', 'buildx', 'rm', '--force', builder],
                        output / f'{scenario}.cleanup.log', env)
                    result['free_bytes_after_cleanup'] = shutil.disk_usage(
                        workspace).free
            validate_results(summary['results'])
            summary['status'] = 'passed'
    except Exception as error:
        summary['error'] = str(error)
        raise
    finally:
        write_summary(output, summary)
    return summary


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=Path)
    args = parser.parse_args()
    runner_temp = Path(os.environ['RUNNER_TEMP'])
    if not runner_temp.is_absolute() or runner_temp == Path('/'):
        parser.error('RUNNER_TEMP must be an absolute temporary directory')
    if os.environ.get(
            'TARGET_ARCH') != 'amd64' or platform.machine() != 'x86_64':
        parser.error('Run this benchmark on a native AMD64 runner')
    benchmark(ROOT, args.output.resolve(), runner_temp)


if __name__ == '__main__':
    main()
