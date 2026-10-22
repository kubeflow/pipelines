"""Build local qualification images with disposable, privately owned caches."""

import re
import subprocess
import uuid
from collections.abc import Iterator
from pathlib import Path


def build_images(
    root: Path, sha: str, version: str, records: list[dict]
) -> Iterator[dict]:
    """Yield each image until scanned, then remove its tag and private
    builder."""
    node = (root / 'frontend/.nvmrc').read_text().strip().removeprefix('v')
    for record in records:
        identifier = uuid.uuid4().hex
        builder = f'kfpr-qualification-{identifier}'
        tag = f'kfpr-qualification/{record["name"]}:{sha}-{record["platform"].split("/")[1]}-{identifier}'
        # Never select this builder globally. A failed create does not establish
        # ownership, so it must not trigger deletion of an existing builder.
        subprocess.run(
            [
                'docker',
                'buildx',
                'create',
                '--name',
                builder,
                '--driver',
                'docker-container',
            ],
            cwd=root,
            check=True,
        )
        try:
            command = [
                'docker',
                'buildx',
                'build',
                '--builder',
                builder,
                '--load',
                '--pull',
                '--platform',
                record['platform'],
                '--file',
                record['dockerfile'],
                '--tag',
                tag,
            ]
            for key, value in {
                'NODE_VERSION': node,
                'COMMIT_SHA': sha,
                'COMMIT_HASH': sha,
                'TAG_NAME': version,
            }.items():
                command.extend(['--build-arg', f'{key}={value}'])
            try:
                subprocess.run(
                    command + [record['context']], cwd=root, check=True
                )
                image_id = subprocess.check_output(
                    ['docker', 'image', 'inspect', '--format', '{{.Id}}', tag],
                    cwd=root,
                    text=True,
                ).strip()
                if not re.fullmatch(r'sha256:[0-9a-f]{64}', image_id):
                    raise ValueError(f'Invalid locally built image ID: {tag}')
                yield {**record, 'image_id': image_id, 'tag': tag}
            finally:
                # Partial exports may leave a tag even when the build fails.
                if subprocess.check_output(
                    ['docker', 'image', 'ls', '--quiet', '--no-trunc', tag],
                    cwd=root,
                    text=True,
                ).strip():
                    subprocess.run(
                        ['docker', 'image', 'rm', tag], cwd=root, check=True
                    )
        finally:
            # docker-container removal also removes its private state/cache.
            # No shared builder, image ID, cache prune or user volume is touched.
            subprocess.run(
                ['docker', 'buildx', 'rm', '--force', builder],
                cwd=root,
                check=True,
            )
