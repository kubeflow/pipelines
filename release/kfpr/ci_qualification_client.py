"""Verify complete OSV qualification evidence from upstream read-only CI."""

import hashlib
import json
import re
import tempfile
import uuid
from pathlib import Path

from . import go_build_provenance
from .ci_images_client import REPO, WORKFLOW, _api, verify_run
from .ci_images_runner import KFP_GO_IMAGES
from .ci_security_runner import verify_security
from .github_api import default_branch
from .osv_image_scan import SCANNER, validate_report
from .osv_reports import findings
from .security import evaluate_policy
from .security_exceptions import exception_digest, load_exceptions
from .third_party import verify as verify_third_party
from .workflow_runs import dispatch_and_watch, validate_run


def read_artifact(
    directory, expected_images, expected_minors, identity, *, source_repo=None
):
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
        paths = go_build_provenance.scan_binaries(envelope['report'])
        if name in KFP_GO_IMAGES and not paths:
            raise ValueError('OSV omitted the expected KFP Go binary')
        if paths:
            if source_repo is None:
                raise ValueError(
                    'KFP binary provenance requires the source Git repository'
                )
            proof_name = f'provenance/{name}-{platform.split("/")[1]}.json'
            if record.get('go_provenance_file') != proof_name:
                raise ValueError('Missing KFP binary provenance')
            proof_bytes = (directory / proof_name).read_bytes()
            if hashlib.sha256(proof_bytes).hexdigest() != record.get(
                'go_provenance_sha256'
            ):
                raise ValueError('KFP binary provenance checksum mismatch')
            source_info = go_build_provenance.source_identity(
                source_repo, identity['source_sha']
            )
            go_build_provenance.verify_scan(
                json.loads(proof_bytes),
                envelope['report'],
                image_id,
                platform,
                identity['source_sha'],
                source_info['source_time'],
            )
        elif any(key.startswith('go_provenance_') for key in record):
            raise ValueError('Unexpected KFP binary provenance')
        image_scans.append(
            {
                'name': f'image-{name}-{platform.split("/")[1]}',
                'target': image_id,
                'raw_report': filename,
                'findings': findings(envelope['report']),
            }
        )
    third_party = None
    third_party_scans = []
    if expected_images:
        third_party_directory = directory / 'third-party'
        manifest_path = third_party_directory / 'third-party.json'
        checksum = hashlib.sha256(manifest_path.read_bytes()).hexdigest()
        if checksum != manifest.get('third_party_manifest_sha256'):
            raise ValueError(
                'Third-party qualification manifest checksum mismatch'
            )
        third_party = verify_third_party(third_party_directory, expected)
        if third_party['inventory']['first_party_names'] != sorted(
            {record['name'] for record in expected_images}
        ):
            raise ValueError(
                'Third-party ownership does not match the release image inventory'
            )
        third_party_scans = third_party['scans']
        third_party = {
            **third_party,
            'evidence_directory': str(third_party_directory.resolve()),
            'manifest_sha256': checksum,
        }
    elif manifest.get('third_party_manifest_sha256') is not None:
        raise ValueError(
            'SDK-only qualification contains unexpected third-party evidence'
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
        'third_party': third_party,
        'scans': security['scans'] + image_scans + third_party_scans,
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
    result = collect_ci(
        context, source_sha, expected_images, expected_minors, output_dir
    )
    verify_run(context, result['run'])
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
    (Path(output_dir) / 'security.json').write_text(
        json.dumps(security, indent=2) + '\n'
    )
    if not security['passed']:
        raise ValueError(
            f'Security qualification failed: {security["blocking_count"]} fixable blocking findings; see {Path(output_dir) / "security.json"}'
        )
    result['security'] = security
    return result


def collect_ci(
    context,
    source_sha,
    expected_images,
    expected_minors,
    output_dir,
    *,
    resume=False,
):
    """Collect immutable source evidence without reading or writing release
    state."""
    if not re.fullmatch(r'[0-9a-f]{40}', source_sha):
        raise ValueError('An exact source SHA is required for CI collection')
    output_dir = Path(output_dir).resolve()
    output_dir.mkdir(parents=True, exist_ok=True)
    identity_path = output_dir / 'request.json'
    if resume:
        identity = json.loads(identity_path.read_text())
        expected = {
            'repository': REPO,
            'workflow': WORKFLOW,
            'source_sha': source_sha,
            'version': context.metadata.tag,
            'include_backend': context.include_backend,
            'include_sdk': context.include_sdk,
        }
        if not isinstance(identity, dict) or any(
            identity.get(k) != v for k, v in expected.items()
        ):
            raise ValueError(
                'Candidate request belongs to a different source or scope'
            )
        if (
            not re.fullmatch(r'[0-9a-f]{40}', identity.get('workflow_sha', ''))
            or not re.fullmatch(r'[0-9a-f]{32}', identity.get('request_id', ''))
            or identity.get('workflow_ref') not in ('master', 'main')
        ):
            raise ValueError('Invalid candidate workflow identity')
        expected_title = f'KFP image qualification {context.metadata.tag} {source_sha} {identity["request_id"]}'
        if identity.get('title') != expected_title:
            raise ValueError(
                'Candidate workflow title does not match its source'
            )
    else:
        if identity_path.exists():
            raise ValueError(
                'CI request already exists; resume it or choose a new output directory'
            )
        workflow_ref = default_branch(lambda endpoint: _api(context, endpoint))
        workflow_sha = _api(context, f'commits/{workflow_ref}')['sha']
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
            'workflow_ref': workflow_ref,
            'source_sha': source_sha,
            'version': context.metadata.tag,
            'request_id': request_id,
            'title': f'KFP image qualification {context.metadata.tag} {source_sha} {request_id}',
        }
        identity.update(
            include_backend=context.include_backend,
            include_sdk=context.include_sdk,
        )
        identity_path.write_text(json.dumps(identity, indent=2) + '\n')
    request_id = identity['request_id']
    inputs = {
        key: identity[key] for key in ('source_sha', 'version', 'request_id')
    }
    inputs.update(
        include_backend=str(context.include_backend).lower(),
        include_sdk=str(context.include_sdk).lower(),
    )
    if resume:
        if not re.fullmatch(r'[1-9][0-9]*', str(identity.get('run_id', ''))):
            raise ValueError(
                'Request has no recorded run; inspect upstream Actions before retrying'
            )
        if (
            type(identity.get('run_attempt')) is not int
            or identity['run_attempt'] < 1
        ):
            raise ValueError('Invalid candidate workflow attempt')
        current = _api(context, f'actions/runs/{identity["run_id"]}')
        validate_run(current, identity, completed=False)
        if current.get('status') != 'completed':
            context.runner.run(
                [
                    'gh',
                    'run',
                    'watch',
                    identity['run_id'],
                    '--repo',
                    REPO,
                    '--exit-status',
                ],
                cwd=context.root,
                check=False,
            )
        final = _api(context, f'actions/runs/{identity["run_id"]}')
        validate_run(final, identity, completed=False)
    else:
        try:
            final = dispatch_and_watch(
                lambda endpoint: _api(context, endpoint),
                lambda command, **kwargs: context.runner.run(
                    command, cwd=context.root, **kwargs
                ),
                identity,
                inputs,
                on_discovered=lambda discovered: identity_path.write_text(
                    json.dumps(discovered, indent=2) + '\n'
                ),
            )
        finally:
            identity_path.write_text(json.dumps(identity, indent=2) + '\n')
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
    directory = Path(tempfile.mkdtemp(prefix=artifact + '-', dir=output_dir))
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
        directory,
        expected_images,
        expected_minors,
        identity,
        source_repo=context.root,
    )
    validate_run(_api(context, f'actions/runs/{identity["run_id"]}'), identity)
    result['run'] = identity
    return result
