"""Bind Go main-module provenance to a pristine source commit and image."""

import hashlib
import json
import os
import re
import subprocess
import tempfile
from datetime import datetime, timezone
from pathlib import Path, PurePosixPath

MODULE = 'github.com/kubeflow/pipelines'


def _git(args, cwd):
    environment = {
        k: v for k, v in os.environ.items() if not k.startswith('GIT_')
    }
    environment.update(
        GIT_CONFIG_NOSYSTEM='1',
        GIT_CONFIG_GLOBAL=os.devnull,
        GIT_NO_REPLACE_OBJECTS='1',
        GIT_TERMINAL_PROMPT='0',
    )
    return subprocess.check_output(
        [
            'git',
            '-c',
            'core.hooksPath=' + os.devnull,
            '-c',
            'core.fsmonitor=false',
            *args,
        ],
        cwd=cwd,
        env=environment,
        text=True,
    ).strip()


def source_identity(source_repo: Path, source_sha: str) -> dict:
    """Read immutable commit metadata, including from a bare evidence repo."""
    if not re.fullmatch(r'[0-9a-f]{40}', source_sha):
        raise ValueError('Require an exact source commit SHA')
    if _git(['cat-file', '-t', source_sha], source_repo) != 'commit':
        raise ValueError('Source identity must name a commit')
    timestamp = int(
        _git(['show', '-s', '--format=%ct', source_sha], source_repo)
    )
    return {
        'source_tree': _git(['rev-parse', source_sha + '^{tree}'], source_repo),
        'source_time': datetime.fromtimestamp(timestamp, timezone.utc).strftime(
            '%Y-%m-%dT%H:%M:%SZ'
        ),
    }


def prepare_context(
    source: Path, source_sha: str, dockerfile: str, destination: Path
) -> dict:
    """Copy committed source with sanitized Git metadata, never caller secrets.

    An external Dockerfile-specific ignore file retains the complete
    tracked tree and .git for native VCS stamping without changing
    normal build ignores.
    """
    source = Path(source).resolve()
    destination = Path(destination).resolve()
    if not re.fullmatch(r'[0-9a-f]{40}', source_sha):
        raise ValueError('Require an exact source SHA for build provenance')
    relative = PurePosixPath(dockerfile)
    if (
        relative.is_absolute()
        or '..' in relative.parts
        or str(relative) != dockerfile
    ):
        raise ValueError('Dockerfile must be a source-relative tracked path')
    if _git(['rev-parse', 'HEAD'], source) != source_sha:
        raise ValueError('Build source HEAD differs from requested commit')
    if _git(['status', '--porcelain', '--untracked-files=no'], source):
        raise ValueError('Build source has tracked modifications')
    entries = _git(['ls-files', '--stage'], source)
    if entries.startswith('160000 ') or '\n160000 ' in entries:
        raise ValueError(
            'Submodule build contexts need explicit provenance support'
        )
    destination.mkdir(parents=True, exist_ok=False)
    context = destination / 'source'
    context.mkdir()
    _git(['init', '--template='], context)
    _git(
        [
            '-c',
            'protocol.file.allow=always',
            'fetch',
            '--no-tags',
            '--depth=1',
            '--no-recurse-submodules',
            str(source),
            source_sha,
        ],
        context,
    )
    _git(['checkout', '--detach', 'FETCH_HEAD'], context)
    if _git(['rev-parse', 'HEAD'], context) != source_sha or _git(
        ['status', '--porcelain', '--untracked-files=all'], context
    ):
        raise ValueError(
            'Prepared build context is not the clean requested commit'
        )
    tracked = _git(['ls-tree', '-z', source_sha, '--', dockerfile], context)
    entries = tracked.split('\0')
    if len(entries) != 2 or entries[1] or '\t' not in entries[0]:
        raise ValueError('Dockerfile must be an exact tracked regular file')
    metadata, tracked_path = entries[0].split('\t', 1)
    fields = metadata.split()
    if (
        len(fields) != 3
        or fields[0] not in ('100644', '100755')
        or fields[1] != 'blob'
        or tracked_path != dockerfile
    ):
        raise ValueError('Dockerfile must be an exact tracked regular file')
    recipe = context
    for part in relative.parts:
        recipe = recipe / part
        if recipe.is_symlink():
            raise ValueError('Dockerfile path must not contain symlinks')
    if not recipe.is_file() or not recipe.resolve().is_relative_to(context):
        raise ValueError('Dockerfile must remain inside the source context')
    if _git(['hash-object', '--', str(recipe)], context) != fields[2]:
        raise ValueError('Dockerfile bytes differ from the source commit')
    # A caller worktree's .git can contain credentials, hooks or external paths.
    # The fresh repository above contains only objects fetched by their SHA.
    external = destination / 'build.Dockerfile'
    external.write_bytes(recipe.read_bytes())
    external.with_name(external.name + '.dockerignore').write_text('')
    return {
        'context': context,
        'dockerfile': external,
        **source_identity(context, source_sha),
    }


def pseudo_version(source_sha: str, source_time: str) -> str:
    """Expected native Go version for a clean, untagged source snapshot."""
    if not re.fullmatch(r'[0-9a-f]{40}', source_sha) or not re.fullmatch(
        r'\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z', source_time
    ):
        raise ValueError('Invalid source commit identity for Go pseudo-version')
    timestamp = datetime.strptime(source_time, '%Y-%m-%dT%H:%M:%SZ')
    return (
        'v0.0.0-' + timestamp.strftime('%Y%m%d%H%M%S') + '-' + source_sha[:12]
    )


def validate_binary(
    info: dict, source_sha: str, source_time: str, platform: str
) -> None:
    """Reject absent, dirty, replaced, wrong-source or wrong-platform
    binaries."""
    if platform not in ('linux/amd64', 'linux/arm64'):
        raise ValueError('Unsupported Go binary platform')
    if not isinstance(info, dict) or not isinstance(info.get('Main'), dict):
        raise ValueError('Missing Go main-module build information')
    main = info['Main']
    if (
        main.get('Path') != MODULE
        or main.get('Version') != pseudo_version(source_sha, source_time)
        or main.get('Replace')
    ):
        raise ValueError(
            'Go binary must identify KFP at its actual source pseudo-version'
        )
    if not isinstance(info.get('Path'), str) or not info['Path'].startswith(
        MODULE + '/'
    ):
        raise ValueError('Go binary requires its real KFP package path')
    settings = info.get('Settings')
    if not isinstance(settings, list) or any(
        not isinstance(s, dict)
        or not set(s) <= {'Key', 'Value'}
        or not isinstance(s.get('Key'), str)
        or not s['Key']
        or not isinstance(s.get('Value', ''), str)
        for s in settings
    ):
        raise ValueError('Invalid Go binary build settings')
    values = {s['Key']: s.get('Value', '') for s in settings}
    if len(values) != len(settings):
        raise ValueError('Duplicate Go binary build settings')
    expected = {
        'vcs': 'git',
        'vcs.revision': source_sha,
        'vcs.time': source_time,
        'vcs.modified': 'false',
        'GOOS': 'linux',
        'GOARCH': platform.split('/')[1],
    }
    if any(values.get(k) != v for k, v in expected.items()):
        raise ValueError(
            'Go binary provenance does not match clean source and platform'
        )
    if (
        info.get('ELFMachine')
        != {'linux/amd64': 'EM_X86_64', 'linux/arm64': 'EM_AARCH64'}[platform]
    ):
        raise ValueError(
            'Go binary ELF architecture does not match image platform'
        )


def collect(
    image_id: str,
    platform: str,
    source_sha: str,
    source_time: str,
    binary_paths: list[str],
    output: Path,
) -> dict:
    """Read candidate binaries without running the image or its executables."""
    if not re.fullmatch(r'sha256:[0-9a-f]{64}', image_id):
        raise ValueError('Require an immutable image ID for binary provenance')
    if not binary_paths or len(set(binary_paths)) != len(binary_paths):
        raise ValueError('Require distinct KFP binary paths from the raw scan')
    for value in binary_paths:
        path = PurePosixPath(value)
        if not path.is_absolute() or '..' in path.parts or str(path) != value:
            raise ValueError('Invalid scanned binary path')
    metadata = json.loads(
        subprocess.check_output(
            ['docker', 'image', 'inspect', '--format', '{{json .}}', image_id],
            text=True,
        )
    )
    if (
        metadata.get('Id') != image_id
        or str(metadata.get('Os')) + '/' + str(metadata.get('Architecture'))
        != platform
    ):
        raise ValueError(
            'Binary extraction image identity or platform mismatch'
        )
    output = Path(output)
    output.mkdir(parents=True, exist_ok=True)
    evidence = {
        'schema': 1,
        'image_id': image_id,
        'platform': platform,
        'source_sha': source_sha,
        'source_time': source_time,
        'binaries': [],
    }
    container = subprocess.check_output(
        [
            'docker',
            'create',
            '--entrypoint',
            '/nonexecuted-provenance-reader',
            image_id,
        ],
        text=True,
    ).strip()
    try:
        with tempfile.TemporaryDirectory(prefix='kfpr-binaries-') as temporary:
            temporary = Path(temporary).resolve()
            for index, path in enumerate(sorted(binary_paths)):
                binary = temporary / f'binary-{index}'
                subprocess.run(
                    ['docker', 'cp', '-L', container + ':' + path, str(binary)],
                    check=True,
                )
                if (
                    binary.is_symlink()
                    or not binary.is_file()
                    or not binary.resolve().is_relative_to(temporary)
                ):
                    raise ValueError(
                        'Extracted Go binary must be a regular scratch file'
                    )
                environment = dict(
                    os.environ,
                    GO111MODULE='off',
                    GOTOOLCHAIN='local',
                    GOPROXY='off',
                    GOSUMDB='off',
                )
                info = json.loads(
                    subprocess.check_output(
                        [
                            'go',
                            'run',
                            str(
                                Path(__file__).with_name('read_go_buildinfo.go')
                            ),
                            str(binary),
                        ],
                        cwd=temporary,
                        env=environment,
                        text=True,
                    )
                )
                validate_binary(info, source_sha, source_time, platform)
                settings = {
                    item['Key']: item.get('Value', '')
                    for item in info['Settings']
                }
                evidence['binaries'].append(
                    {
                        'path': path,
                        'sha256': hashlib.sha256(
                            binary.read_bytes()
                        ).hexdigest(),
                        'build_info': info,
                        'module': info['Main']['Path'],
                        'version': info['Main']['Version'],
                        'vcs': settings['vcs'],
                        'revision': settings['vcs.revision'],
                        'time': settings['vcs.time'],
                        'modified': settings['vcs.modified'] != 'false',
                    }
                )
    finally:
        subprocess.run(
            ['docker', 'rm', container], check=True, stdout=subprocess.DEVNULL
        )
    (output / 'go-provenance.json').write_text(
        json.dumps(evidence, indent=2) + '\n'
    )
    return evidence


def verify(
    evidence: dict,
    image_id: str,
    platform: str,
    source_sha: str,
    source_time: str,
    binary_paths: list[str],
) -> None:
    """Revalidate retained provenance independently of the collection
    process."""
    if not isinstance(evidence, dict) or any(
        evidence.get(k) != v
        for k, v in {
            'schema': 1,
            'image_id': image_id,
            'platform': platform,
            'source_sha': source_sha,
            'source_time': source_time,
        }.items()
    ):
        raise ValueError('Go provenance identity mismatch')
    binaries = evidence.get('binaries')
    if (
        not isinstance(binaries, list)
        or not binary_paths
        or any(not isinstance(b, dict) for b in binaries)
    ):
        raise ValueError('Missing Go binary provenance coverage')
    if sorted(b.get('path', '') for b in binaries) != sorted(
        binary_paths
    ) or len(set(binary_paths)) != len(binary_paths):
        raise ValueError('Go binary provenance coverage mismatch')
    for binary in binaries:
        if not isinstance(binary.get('sha256'), str) or not re.fullmatch(
            r'[0-9a-f]{64}', binary['sha256']
        ):
            raise ValueError('Invalid Go binary checksum')
        info = binary.get('build_info')
        validate_binary(info, source_sha, source_time, platform)
        settings = {
            item['Key']: item.get('Value', '') for item in info['Settings']
        }
        expected = {
            'module': info['Main']['Path'],
            'version': info['Main']['Version'],
            'vcs': settings['vcs'],
            'revision': settings['vcs.revision'],
            'time': settings['vcs.time'],
            'modified': False,
        }
        if (
            any(binary.get(k) != v for k, v in expected.items())
            or binary.get('modified') is not False
        ):
            raise ValueError(
                'Go provenance summary differs from binary metadata'
            )


def scan_binaries(report: dict) -> dict[str, str]:
    """Find every KFP binary in the untouched full OSV dependency inventory."""
    from .osv_reports import inventory

    result = {}
    for path, ecosystem, module, version in inventory(report):
        if ecosystem == 'Go' and module == MODULE:
            if path in result:
                raise ValueError('Duplicate KFP binary in OSV inventory')
            result[path] = version
    return result


def verify_scan(
    evidence: dict,
    report: dict,
    image_id: str,
    platform: str,
    source_sha: str,
    source_time: str,
) -> None:
    """Bind scanner package identity to the actual extracted binary
    metadata."""
    binaries = scan_binaries(report)
    verify(
        evidence, image_id, platform, source_sha, source_time, list(binaries)
    )
    for binary in evidence['binaries']:
        if binaries[binary['path']].removeprefix('v') != binary[
            'version'
        ].removeprefix('v'):
            raise ValueError(
                'OSV KFP package version differs from binary provenance'
            )
