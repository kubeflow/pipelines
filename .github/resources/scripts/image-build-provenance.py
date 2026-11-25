"""Reuse qualification provenance for ordinary image builds and publication."""

import argparse
import json
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[3] / 'release'))
from kfpr import go_build_provenance as provenance

BINARIES = {
    'backend/Dockerfile': '/bin/apiserver',
    'backend/Dockerfile.cacheserver': '/bin/cache_server',
    'backend/Dockerfile.driver': '/bin/driver',
    'backend/Dockerfile.launcher': '/bin/launcher-v2',
    'backend/Dockerfile.persistenceagent': '/bin/persistence_agent',
    'backend/Dockerfile.scheduledworkflow': '/bin/controller',
    'backend/Dockerfile.viewercontroller': '/bin/controller',
}


def recipe_path(value: str | Path) -> str:
    """Validate recipe spelling before deciding whether verification
    applies."""
    path = Path(value)
    if path.is_absolute() or '..' in path.parts:
        raise ValueError('Dockerfile must be relative without parent traversal')
    return path.as_posix()


def prepare(
    source: str | Path,
    sha: str,
    dockerfile: str | Path,
    destination: Path,
    context: str = '.',
) -> dict[str, str | Path]:
    """Isolate committed source while retaining non-Go build ignore rules."""
    source = Path(source).resolve()
    dockerfile = recipe_path(dockerfile)
    context = recipe_path(context)
    if dockerfile in BINARIES and context != '.':
        raise ValueError('KFP Go images require the repository-root context')
    prepared = provenance.prepare_context(source, sha, dockerfile, destination)
    if dockerfile not in BINARIES:
        # Keep the in-tree recipe so Docker retains its original ignore rules.
        # The snapshot excludes the untracked tooling checkout without moving
        # local actions that GitHub still needs during post-job cleanup.
        snapshot = prepared['context']
        build_context = snapshot / context
        if (
            not build_context.is_dir()
            or not build_context.resolve().is_relative_to(snapshot)
        ):
            raise ValueError(
                'Build context must remain inside committed source'
            )
        prepared['context'] = build_context
        prepared['dockerfile'] = snapshot / dockerfile
    return prepared


def verify(
    source: str | Path,
    sha: str,
    dockerfile: str | Path,
    image: str,
    platform: str,
    output: str | Path,
) -> dict[str, object]:
    """Validate immutable Go binaries without executing candidate code."""
    output = Path(output)
    dockerfile = recipe_path(dockerfile)
    if dockerfile not in BINARIES:
        output.mkdir(parents=True, exist_ok=True)
        evidence = {
            'schema': 1,
            'status': 'not-a-kfp-go-image',
            'dockerfile': dockerfile,
        }
        (output / 'go-provenance.json').write_text(json.dumps(evidence) + '\n')
        return evidence
    image_id = subprocess.check_output(
        ['docker', 'image', 'inspect', '--format', '{{.Id}}', image], text=True
    ).strip()
    identity = provenance.source_identity(Path(source), sha)
    return provenance.collect(
        image_id,
        platform,
        sha,
        identity['source_time'],
        [BINARIES[dockerfile]],
        output,
    )


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest='command', required=True)
    for command in ('prepare', 'verify'):
        subparser = commands.add_parser(command)
        subparser.add_argument('--source', required=True, type=Path)
        subparser.add_argument('--sha', required=True)
        subparser.add_argument('--dockerfile', required=True)
        if command == 'prepare':
            subparser.add_argument('--destination', required=True, type=Path)
            subparser.add_argument('--context', default='.')
        else:
            subparser.add_argument('--image', required=True)
            subparser.add_argument('--platform', required=True)
            subparser.add_argument('--output', required=True, type=Path)
    args = vars(parser.parse_args())
    command = args.pop('command')
    result = prepare(**args) if command == 'prepare' else verify(**args)
    print(json.dumps(result, default=str))


if __name__ == '__main__':
    main()
