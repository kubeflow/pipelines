"""Prepare committed Go publication sources without copying checkout
secrets."""

import argparse
from pathlib import Path

from kfpr.go_build_provenance import prepare_context

GO_IMAGES = {
    'kfp-api-server': 'backend/Dockerfile',
    'kfp-driver': 'backend/Dockerfile.driver',
    'kfp-launcher': 'backend/Dockerfile.launcher',
    'kfp-persistence-agent': 'backend/Dockerfile.persistenceagent',
    'kfp-scheduled-workflow-controller': 'backend/Dockerfile.scheduledworkflow',
    'kfp-viewer-crd-controller': 'backend/Dockerfile.viewercontroller',
}


def prepare(
    image: str, context: str, dockerfile: str, sha: str, destination: Path
) -> dict[str, str]:
    if image not in GO_IMAGES:
        if Path(dockerfile).resolve() in {
            Path(recipe).resolve() for recipe in GO_IMAGES.values()
        }:
            raise ValueError(
                'Go Dockerfile requires its registered publication image'
            )
        return {'context': context, 'dockerfile': dockerfile, 'source_sha': ''}
    if (
        Path(context).resolve() != Path.cwd().resolve()
        or dockerfile != GO_IMAGES[image]
    ):
        raise ValueError(
            'Go publication requires the repository root and registered Dockerfile'
        )
    prepared = prepare_context(Path.cwd(), sha, dockerfile, destination)
    return {
        'context': str(prepared['context']),
        'dockerfile': str(prepared['dockerfile']),
        'source_sha': sha,
    }


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    for name in (
        'image',
        'context',
        'dockerfile',
        'sha',
        'destination',
        'output',
    ):
        parser.add_argument('--' + name, required=True)
    args = parser.parse_args()
    result = prepare(
        args.image,
        args.context,
        args.dockerfile,
        args.sha,
        Path(args.destination),
    )
    if any('\n' in value or '\r' in value for value in result.values()):
        raise ValueError('Build context output must be single-line')
    with Path(args.output).open('a') as output:
        for key, value in result.items():
            output.write(f'{key}={value}\n')


if __name__ == '__main__':
    main()
