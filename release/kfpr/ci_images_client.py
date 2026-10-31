"""Dispatch and verify upstream image qualification with artifact
provenance."""

import hashlib
import json
import re
import subprocess
import uuid
from pathlib import Path

from .core import REPO, default_branch
from .security import _trivy_findings, evaluate_policy
from .security_exceptions import load_exceptions
from .workflow_runs import dispatch_and_watch, validate_run

WORKFLOW = '.github/workflows/release-qualification.yml'


def _api(context, endpoint):
    return json.loads(
        context.runner.capture(
            ['gh', 'api', f'repos/{REPO}/{endpoint}'], cwd=context.root
        )
    )


def verify_run(context, identity):
    """Reject reruns, failed runs or changed workflow provenance at
    publication."""
    if (
        identity.get('repository') != REPO
        or identity.get('workflow') != WORKFLOW
    ):
        raise ValueError(
            'Image qualification must use the upstream qualification workflow'
        )
    if not re.fullmatch(
        r'[0-9a-f]{40}', identity.get('workflow_sha', '')
    ) or not re.fullmatch(r'[0-9a-f]{32}', identity.get('request_id', '')):
        raise ValueError('Invalid image qualification workflow identity')
    if (
        identity.get('source_sha') != context.state.answers.get('release_sha')
        or identity.get('version') != context.metadata.tag
    ):
        raise ValueError('Image qualification belongs to a different release')
    title = f'KFP image qualification {identity["version"]} {identity["source_sha"]} {identity["request_id"]}'
    if (
        identity.get('title') != title
        or not re.fullmatch(r'[1-9][0-9]*', str(identity.get('run_id', '')))
        or type(identity.get('run_attempt')) is not int
        or identity['run_attempt'] < 1
    ):
        raise ValueError('Invalid image qualification run identity')
    validate_run(_api(context, f'actions/runs/{identity["run_id"]}'), identity)


def _read_artifact(
    directory,
    expected_images,
    identity,
    *,
    policy='zero-fixable',
    exceptions=None,
):
    for path in directory.rglob('*'):
        if path.is_symlink() or not path.resolve().is_relative_to(
            directory.resolve()
        ):
            raise ValueError(
                'Image qualification artifact contains a symlink or escaping path'
            )
    manifest = json.loads((directory / 'manifest.json').read_text())
    expected = {
        key: identity[key]
        for key in (
            'version',
            'source_sha',
            'workflow_sha',
            'request_id',
            'run_id',
            'run_attempt',
        )
    }
    if manifest.get('schema') != 1 or any(
        manifest.get(key) != value for key, value in expected.items()
    ):
        raise ValueError(
            'Image qualification manifest provenance does not match the run'
        )
    inventory = {
        (record['name'], record['platform']) for record in expected_images
    }
    records = manifest.get('images')
    if (
        not isinstance(records, list)
        or not inventory
        or len(records) != len(inventory)
        or {(record.get('name'), record.get('platform')) for record in records}
        != inventory
    ):
        raise ValueError(
            'Image qualification inventory is incomplete or duplicated'
        )
    reports = []
    for record in records:
        name, platform = record['name'], record['platform']
        if not re.fullmatch(r'[a-z0-9-]+', name) or platform not in (
            'linux/amd64',
            'linux/arm64',
        ):
            raise ValueError(
                'Invalid image qualification inventory name or platform'
            )
        image_id = record.get('image_id', '')
        scan_file = f'scans/{name}-{platform.split("/")[1]}.json'
        if record.get('scan_file') != scan_file or not re.fullmatch(
            r'sha256:[0-9a-f]{64}', image_id
        ):
            raise ValueError(
                'Invalid image qualification scan path or image ID'
            )
        path = directory / scan_file
        payload = path.read_bytes()
        if hashlib.sha256(payload).hexdigest() != record.get('scan_sha256'):
            raise ValueError('Image qualification scan checksum mismatch')
        data = json.loads(payload)
        if (
            data.get('ArtifactType') != 'container_image'
            or data.get('ArtifactName') != image_id
            or (data.get('Metadata') or {}).get('ImageID') != image_id
        ):
            raise ValueError(
                'Trivy report does not identify the qualified container image'
            )
        evaluated = evaluate_policy(
            [
                {
                    'name': f'image-{name}-{platform.split("/")[1]}',
                    'target': image_id,
                    'findings': _trivy_findings(data),
                }
            ],
            release_version=identity['version'],
            policy=policy,
            exceptions=exceptions,
        )
        if evaluated['blocking_count']:
            raise ValueError(
                'Qualified image contains fixable vulnerabilities; inspect raw CI scans'
            )
        reports.append({'image_id': image_id, 'path': str(path.resolve())})
    return {'images': records, 'reports': reports, 'run': identity}


def qualify_images(context, expected_images, output_dir):
    """Require successful upstream CI for the exact source and complete image
    set."""
    source_sha = context.state.answers.get('release_sha', '')
    if not re.fullmatch(r'[0-9a-f]{40}', source_sha):
        raise ValueError(
            'Resolve the merged release commit before image qualification'
        )
    policy = context.state.answers.get('security_policy', 'zero-fixable')
    exceptions = None
    if policy == 'approved-exceptions':
        exception_path = context.state.answers.get('security_exceptions')
        if not exception_path:
            raise ValueError(
                'Approved image exceptions require security_exceptions'
            )
        exceptions = load_exceptions(Path(exception_path))
    # Validate the policy before dispatch, including expiration of approvals.
    evaluate_policy(
        [],
        release_version=context.metadata.tag,
        policy=policy,
        exceptions=exceptions,
    )
    workflow_ref = default_branch(lambda endpoint: _api(context, endpoint))
    workflow_sha = _api(context, f'commits/{workflow_ref}')['sha']
    if not re.fullmatch(r'[0-9a-f]{40}', workflow_sha):
        raise ValueError('Invalid upstream workflow commit')
    try:
        workflow = _api(context, f'contents/{WORKFLOW}?ref={workflow_sha}')
    except subprocess.CalledProcessError as error:
        raise RuntimeError(
            'Cannot read upstream release-qualification.yml; verify GitHub access and merge the workflow before qualification'
        ) from error
    if workflow.get('type') != 'file' or workflow.get('path') != WORKFLOW:
        raise ValueError(
            'Upstream release-qualification.yml is unavailable; merge the workflow before qualification'
        )
    request_id = uuid.uuid4().hex
    title = f'KFP image qualification {context.metadata.tag} {source_sha} {request_id}'
    identity = {
        'repository': REPO,
        'workflow': WORKFLOW,
        'workflow_sha': workflow_sha,
        'workflow_ref': workflow_ref,
        'source_sha': source_sha,
        'version': context.metadata.tag,
        'request_id': request_id,
        'title': title,
    }
    final = dispatch_and_watch(
        lambda endpoint: _api(context, endpoint),
        lambda command, **kwargs: context.runner.run(
            command, cwd=context.root, **kwargs
        ),
        identity,
        {key: identity[key] for key in ('source_sha', 'version', 'request_id')},
    )
    output_dir = Path(output_dir).resolve()
    output_dir.mkdir(parents=True, exist_ok=True)
    if (
        final.get('status') != 'completed'
        or final.get('conclusion') != 'success'
    ):
        context.runner.run(
            [
                'gh',
                'run',
                'download',
                identity['run_id'],
                '--repo',
                REPO,
                '--pattern',
                f'kfpr-lane-{request_id}-{identity["run_attempt"]}-*',
                '--dir',
                str(output_dir / 'failed-lanes'),
            ],
            cwd=context.root,
            check=False,
        )
        raise RuntimeError(
            f'Image qualification CI failed: https://github.com/{REPO}/actions/runs/{identity["run_id"]}; diagnostics: {output_dir / "failed-lanes"}'
        )
    artifact_name = f'kfpr-images-{request_id}-{identity["run_attempt"]}'
    directory = output_dir / artifact_name
    directory.mkdir()
    context.runner.run(
        [
            'gh',
            'run',
            'download',
            identity['run_id'],
            '--repo',
            REPO,
            '--name',
            artifact_name,
            '--dir',
            str(directory),
        ],
        cwd=context.root,
    )
    result = _read_artifact(
        directory,
        expected_images,
        identity,
        policy=policy,
        exceptions=exceptions,
    )
    verify_run(context, identity)
    return result
