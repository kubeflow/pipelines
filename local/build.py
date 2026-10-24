"""Build a native proof wheel plus its matching SDK wheel from this
checkout."""

import argparse
import importlib.metadata
import json
import os
import platform
import shutil
import subprocess
import sys
import sysconfig
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
PROJECT = Path(__file__).resolve().parent


def run(args: list[str], *, cwd: Path = ROOT) -> None:
    """Run an existing build tool and propagate its exact failure."""
    print('+ ' + ' '.join(args), flush=True)
    subprocess.run(args, cwd=cwd, check=True)


def prepare_assets(stage: Path) -> None:
    """Compile native/API and UI assets and install locked production Node
    deps."""
    if platform.system() != 'Linux' or platform.machine() not in (
        'x86_64',
        'aarch64',
    ):
        raise RuntimeError(
            'This proof currently builds only native Linux x86_64/aarch64 '
            'wheels. Cross-compilation and macOS release wheels are follow-up.'
        )
    for command in ('go', 'node', 'npm'):
        if shutil.which(command) is None:
            raise RuntimeError(f'Missing build tool: {command}')
    node_version = (
        (ROOT / 'frontend' / '.nvmrc').read_text().strip().lstrip('v')
    )
    actual_node = (
        subprocess.check_output(
            ['node', '--version'],
            text=True,
        )
        .strip()
        .lstrip('v')
    )
    if actual_node != node_version:
        raise RuntimeError(
            f'Build with Node.js {node_version}, matching frontend/.nvmrc.'
        )
    if not (ROOT / 'frontend/node_modules/.bin/vite').is_file():
        raise RuntimeError(
            'Run "npm ci" in frontend before building this proof.'
        )
    assets = stage / 'src/kfp_local/_assets'
    (assets / 'bin').mkdir(parents=True)
    run(
        [
            'go',
            'build',
            '-trimpath',
            '-ldflags=-s -w',
            '-o',
            str(assets / 'bin/apiserver'),
            './backend/src/apiserver',
        ]
    )
    run(['npm', '--prefix', 'frontend', 'run', 'build'])
    run(['npm', '--prefix', 'frontend/server', 'run', 'build'])
    ui = assets / 'ui'
    shutil.copytree(ROOT / 'frontend/build', ui / 'build')
    server = ui / 'server'
    shutil.copytree(ROOT / 'frontend/server/dist', server / 'dist')
    for name in ('package.json', 'package-lock.json'):
        shutil.copy2(ROOT / 'frontend/server' / name, server / name)
    run(
        [
            'npm',
            'ci',
            '--prefix',
            str(server),
            '--omit=dev',
            '--ignore-scripts',
        ]
    )
    (assets / 'config').mkdir()
    shutil.copy2(
        ROOT / 'backend/src/apiserver/config/config.json',
        assets / 'config/config.json',
    )
    libc, libc_version = platform.libc_ver()
    if not libc or not libc_version:
        raise RuntimeError(
            'Cannot determine the native runtime libc requirement.'
        )
    platform_tag = sysconfig.get_platform().replace('-', '_').replace('.', '_')
    (assets / 'manifest.json').write_text(
        json.dumps(
            {
                'version': 1,
                'system': platform.system(),
                'machine': platform.machine(),
                'libc': libc,
                'libc_version': libc_version,
                'node_version': node_version,
                'wheel_tag': f'py3-none-{platform_tag}',
            },
            indent=2,
        )
        + '\n'
    )


def main() -> None:
    """Produce installable artifacts without writing generated assets into
    Git."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output-dir', type=Path, default=ROOT / 'dist/local')
    args = parser.parse_args()
    try:
        importlib.metadata.version('build')
    except importlib.metadata.PackageNotFoundError as exc:
        raise RuntimeError(
            'Install the Python build frontend, for example with '
            '"uv sync --frozen --extra ci", before building native assets.'
        ) from exc
    output = args.output_dir.expanduser().resolve()
    output.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='kfp-local-wheel-') as temporary:
        stage = Path(temporary)
        shutil.copytree(
            PROJECT / 'src',
            stage / 'src',
            ignore=shutil.ignore_patterns('__pycache__', '_assets'),
        )
        for name in ('pyproject.toml', 'hatch_build.py', 'README.md'):
            shutil.copy2(PROJECT / name, stage / name)
        shutil.copy2(ROOT / 'LICENSE', stage / 'LICENSE')
        prepare_assets(stage)
        run(
            [
                sys.executable,
                '-m',
                'build',
                '--wheel',
                '--outdir',
                str(output),
                str(stage),
            ]
        )
    run(
        [
            sys.executable,
            '-m',
            'pip',
            'wheel',
            '--no-deps',
            '--wheel-dir',
            str(output),
            str(ROOT / 'sdk/python'),
        ]
    )
    print(f'Native proof and matching SDK wheels: {output}', flush=True)


if __name__ == '__main__':
    os.environ.setdefault('GOMAXPROCS', '4')
    main()
