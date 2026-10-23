"""Verify complete OSV qualification evidence from upstream read-only CI."""

import hashlib
import json
import re
import uuid
from pathlib import Path

from .ci_images_client import REPO, WORKFLOW, _api, verify_run
from .ci_security_runner import verify_security
from .osv_image_scan import SCANNER, validate_report
from .osv_reports import findings
from .security import evaluate_policy
from .security_exceptions import exception_digest, load_exceptions
from .workflow_runs import dispatch_and_watch


def read_artifact(directory, expected_images, expected_minors, identity):
    directory = Path(directory)
    for path in directory.rglob('*'):
        if path.is_symlink() or not path.resolve().is_relative_to(
            directory.resolve()
        ):
            raise ValueError('Qualification artifact contains an escaping path')
    manifest = json.loads((directory / 'manifest.json').read_text())
    expected = {
        'schema': 1,
        **{
            key: identity[key]
            for key in (
                'version',
                'source_sha',
                'workflow_sha',
                'request_id',
                'run_id',
                'run_attempt',
            )
        },
    }
    if any(manifest.get(k) != v for k, v in expected.items()):
        raise ValueError(
            'Qualification manifest provenance does not match the run'
        )
    if (
        manifest.get('scanner') != SCANNER
        or manifest.get('report_format') != 'json'
        or manifest.get('include_backend') is not bool(expected_images)
        or manifest.get('include_sdk') is not bool(expected_minors)
    ):
        raise ValueError(
            'Qualification scanner or scope does not match the request'
        )
    inventory = {(r['name'], r['platform']) for r in expected_images}
    records = manifest.get('images')
    if (
        not isinstance(records, list)
        or len(records) != len(inventory)
        or {(r.get('name'), r.get('platform')) for r in records} != inventory
    ):
        raise ValueError(
            'Qualification image inventory is incomplete or duplicated'
        )
    image_scans = []
    for record in records:
        name, platform = record['name'], record['platform']
        if not re.fullmatch(r'[a-z0-9-]+', name) or platform not in (
            'linux/amd64',
            'linux/arm64',
        ):
            raise ValueError('Invalid image qualification inventory')
        image_id = record.get('image_id', '')
        filename = f'scans/{name}-{platform.split("/")[1]}.json'
        if (
            record.get('scan_file') != filename
            or not re.fullmatch(r'sha256:[0-9a-f]{64}', image_id)
            or record.get('scanner') != SCANNER
            or record.get('report_format') != 'json'
        ):
            raise ValueError('Invalid image qualification scan identity')
        payload = (directory / filename).read_bytes()
        if hashlib.sha256(payload).hexdigest() != record.get('scan_sha256'):
            raise ValueError('Image qualification checksum mismatch')
        envelope = json.loads(payload)
        validate_report(envelope, image_id, report_format='json')
        image_scans.append(
            {
                'name': f'image-{name}-{platform.split("/")[1]}',
                'target': image_id,
                'raw_report': filename,
                'findings': findings(envelope['report']),
            }
        )
    security_manifest = directory / 'security/security.json'
    if hashlib.sha256(
        security_manifest.read_bytes()
    ).hexdigest() != manifest.get('security_manifest_sha256'):
        raise ValueError(
            'Source/Python qualification manifest checksum mismatch'
        )
    security = verify_security(
        directory / 'security', expected, set(expected_minors)
    )
    return {
        'images': records,
        'wheels': security['wheels'],
        'scans': security['scans'] + image_scans,
        'run': identity,
    }


def qualify_ci(context, expected_images, expected_minors, output_dir):
    """Run every scanner remotely; apply the selected policy to verified
    data."""
    source_sha = context.state.answers.get('release_sha', '')
    if not re.fullmatch(r'[0-9a-f]{40}', source_sha):
        raise ValueError('Resolve the merged release SHA before qualification')
    policy = context.state.answers.get('security_policy', 'zero-fixable')
    exceptions = None
    if policy == 'approved-exceptions':
        path = context.state.answers.get('security_exceptions')
        if not path:
            raise ValueError('Approved exceptions require security_exceptions')
        exceptions = load_exceptions(Path(path))
    evaluate_policy(
        [],
        release_version=context.metadata.tag,
        policy=policy,
        exceptions=exceptions,
    )
    workflow_sha = _api(context, 'commits/master')['sha']
    if not re.fullmatch(r'[0-9a-f]{40}', workflow_sha):
        raise ValueError('Invalid upstream workflow commit')
    workflow = _api(context, f'contents/{WORKFLOW}?ref={workflow_sha}')
    if workflow.get('type') != 'file' or workflow.get('path') != WORKFLOW:
        raise ValueError('Merge the qualification workflow before dispatch')
    request_id = uuid.uuid4().hex
    identity = {
        'repository': REPO,
        'workflow': WORKFLOW,
        'workflow_sha': workflow_sha,
        'source_sha': source_sha,
        'version': context.metadata.tag,
        'request_id': request_id,
        'title': f'KFP image qualification {context.metadata.tag} {source_sha} {request_id}',
    }
    inputs = {
        key: identity[key] for key in ('source_sha', 'version', 'request_id')
    }
    inputs.update(
        include_backend=str(context.include_backend).lower(),
        include_sdk=str(context.include_sdk).lower(),
    )
    final = dispatch_and_watch(
        lambda endpoint: _api(context, endpoint),
        lambda command, **kwargs: context.runner.run(
            command, cwd=context.root, **kwargs
        ),
        identity,
        inputs,
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
                '--dir',
                str(output_dir / 'failed-run'),
            ],
            cwd=context.root,
            check=False,
        )
        raise RuntimeError(
            f'Qualification CI failed: https://github.com/{REPO}/actions/runs/{identity["run_id"]}'
        )
    artifact = f'kfpr-images-{request_id}-{identity["run_attempt"]}'
    directory = output_dir / artifact
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
            artifact,
            '--dir',
            str(directory),
        ],
        cwd=context.root,
    )
    result = read_artifact(
        directory, expected_images, expected_minors, identity
    )
    verify_run(context, identity)
    security = {
        'policy': policy,
        'release_version': context.metadata.tag,
        'scanner': SCANNER,
        'scans': result.pop('scans'),
        'errors': [],
        'passed': False,
    }
    if exceptions is not None:
        security.update(
            exceptions=exceptions,
            exceptions_digest=exception_digest(exceptions),
        )
    security.update(
        evaluate_policy(
            security['scans'],
            release_version=context.metadata.tag,
            policy=policy,
            exceptions=exceptions,
        )
    )
    security['passed'] = security['blocking_count'] == 0
    (output_dir / 'security.json').write_text(
        json.dumps(security, indent=2) + '\n'
    )
    if not security['passed']:
        raise ValueError(
            f'Security qualification failed: {security["blocking_count"]} fixable blocking findings; see {output_dir / "security.json"}'
        )
    result['security'] = security
    return result
