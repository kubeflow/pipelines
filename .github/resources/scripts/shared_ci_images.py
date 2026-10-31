#!/usr/bin/env python3
"""Publish and resolve immutable CI image sets using read-only GitHub APIs."""

import argparse
import hashlib
import io
import json
import os
import re
import subprocess
import sys
import time
import zipfile
from pathlib import Path

WORKFLOW = '.github/workflows/ci-images.yml'
IMAGES = (
    'apiserver',
    'scheduledworkflow',
    'persistenceagent',
    'frontend',
    'viewer-crd-controller',
    'driver',
    'launcher',
)
ARCHITECTURES = ('amd64', 'arm64')


class InvalidEvidence(ValueError):
    pass


class NotReady(Exception):
    completed_producer = None


class APIError(Exception):
    pass


def require(condition, message):
    if not condition:
        raise InvalidEvidence(message)


def positive(value):
    return type(value) is int and value > 0


def matches(pattern, value):
    return isinstance(value, str) and re.fullmatch(pattern, value) is not None


def validate_identity(value):
    fields = {
        'version',
        'repository',
        'repository_id',
        'event',
        'ref',
        'source_sha',
        'run_id',
        'run_attempt',
        'pr',
    }
    require(
        isinstance(value, dict) and set(value) == fields, 'Malformed identity'
    )
    require(value['version'] == 1, 'Unsupported identity version')
    require(
        matches(r'[\w.-]+/[\w.-]+', value['repository']), 'Invalid repository'
    )
    for key in ('repository_id', 'run_id', 'run_attempt'):
        require(positive(value[key]), f'Invalid {key}')
    require(
        matches('[0-9a-f]{40}', value['source_sha']), 'Invalid checkout SHA'
    )
    require(
        value['event'] in ('pull_request', 'push', 'workflow_dispatch'),
        'Unsupported event',
    )
    require(matches('refs/.+', value['ref']), 'Invalid ref')
    pr = value['pr']
    if value['event'] != 'pull_request':
        require(pr is None, 'Unexpected PR identity')
        return value
    fields = {
        'number',
        'head_repository',
        'head_repository_id',
        'head_sha',
        'base_repository',
        'base_repository_id',
        'base_ref',
        'base_sha',
    }
    require(isinstance(pr, dict) and set(pr) == fields, 'Malformed PR identity')
    for key in ('number', 'head_repository_id', 'base_repository_id'):
        require(positive(pr[key]), f'Invalid PR {key}')
    for key in ('head_repository', 'base_repository'):
        require(matches(r'[\w.-]+/[\w.-]+', pr[key]), f'Invalid PR {key}')
    for key in ('head_sha', 'base_sha'):
        require(matches('[0-9a-f]{40}', pr[key]), f'Invalid PR {key}')
    require(
        isinstance(pr['base_ref'], str) and bool(pr['base_ref']),
        'Invalid base ref',
    )
    require(
        pr['base_repository'] == value['repository']
        and pr['base_repository_id'] == value['repository_id'],
        'PR base repository mismatch',
    )
    require(
        value['ref'] == f'refs/pull/{pr["number"]}/merge', 'Expected merge ref'
    )
    return value


def environment_identity(environ=None, checkout=True):
    env = os.environ if environ is None else environ
    event = json.loads(Path(env['GITHUB_EVENT_PATH']).read_text())
    value = {
        'version': 1,
        'repository': env['GITHUB_REPOSITORY'],
        'repository_id': int(env['GITHUB_REPOSITORY_ID']),
        'event': env['GITHUB_EVENT_NAME'],
        'ref': env['GITHUB_REF'],
        'source_sha': env['GITHUB_SHA'],
        'run_id': int(env['GITHUB_RUN_ID']),
        'run_attempt': int(env['GITHUB_RUN_ATTEMPT']),
        'pr': None,
    }
    if value['event'] == 'pull_request':
        pr = event['pull_request']
        value['pr'] = {'number': pr['number']}
        for side in ('head', 'base'):
            value['pr'][side + '_repository'] = pr[side]['repo']['full_name']
            value['pr'][side + '_repository_id'] = pr[side]['repo']['id']
            value['pr'][side + '_sha'] = pr[side]['sha']
        value['pr']['base_ref'] = pr['base']['ref']
    validate_identity(value)
    if checkout:
        actual = subprocess.check_output(
            ['git', 'rev-parse', 'HEAD'], text=True
        ).strip()
        require(
            actual == value['source_sha'], 'Checkout does not equal GITHUB_SHA'
        )
    return value


def same_source(producer, consumer):
    validate_identity(producer)
    validate_identity(consumer)
    for field in ('repository', 'repository_id', 'source_sha', 'ref', 'pr'):
        require(
            producer[field] == consumer[field], f'Producer {field} mismatch'
        )
    events = (
        ('push', 'workflow_dispatch')
        if consumer['event'] == 'workflow_dispatch'
        else (consumer['event'],)
    )
    require(producer['event'] in events, 'Producer event mismatch')


def validate_run(run, identity):
    validate_identity(identity)
    require(isinstance(run, dict), 'Malformed run')
    for field, expected in (
        ('id', identity['run_id']),
        ('run_attempt', identity['run_attempt']),
        ('event', identity['event']),
        ('path', WORKFLOW),
    ):
        require(run.get(field) == expected, f'Producer run {field} mismatch')
    repo, head, pr = (
        run.get('repository') or {},
        run.get('head_repository') or {},
        identity['pr'],
    )
    require(
        repo.get('id') == identity['repository_id']
        and repo.get('full_name') == identity['repository'],
        'Producer repository mismatch',
    )
    require(
        run.get('head_sha')
        == (pr['head_sha'] if pr else identity['source_sha']),
        'Producer head SHA mismatch',
    )
    require(
        head.get('id')
        == (pr['head_repository_id'] if pr else identity['repository_id'])
        and head.get('full_name')
        == (pr['head_repository'] if pr else identity['repository']),
        'Producer head repository mismatch',
    )
    # Real fork runs can have pull_requests: []; never require a populated array.
    require(
        run.get('status')
        in (
            'queued',
            'in_progress',
            'completed',
            'waiting',
            'pending',
            'requested',
        ),
        'Unknown producer status',
    )
    if run['status'] == 'completed':
        require(
            run.get('conclusion') == 'success',
            f'Producer failed ({run.get("conclusion")}); rerun the complete producer, then consumers',
        )


class GitHub:
    def __init__(self, repository):
        require(
            matches(r'[\w.-]+/[\w.-]+', repository), 'Invalid API repository'
        )
        self.prefix = f'repos/{repository}'
        self.deadline = None

    def request(self, suffix, pages=False, binary=False):
        command = ['gh', 'api', f'{self.prefix}/{suffix}']
        if pages:
            command += ['--paginate', '--slurp']
        if binary:
            command += ['--allow-escape-sequences']
        remaining = (
            90 if self.deadline is None else self.deadline - time.monotonic()
        )
        if remaining <= 0:
            raise APIError('GitHub API wait deadline reached')
        try:
            result = subprocess.run(
                command,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                timeout=min(90, remaining),
                check=False,
            )
        except (OSError, subprocess.TimeoutExpired) as error:
            raise APIError('GitHub API unavailable') from error
        if result.returncode:
            raise APIError(f'GitHub API request failed: {suffix.split("?")[0]}')
        if binary:
            return result.stdout
        try:
            return json.loads(result.stdout)
        except (ValueError, UnicodeError) as error:
            raise InvalidEvidence('Malformed API JSON') from error

    def listing(self, suffix, key):
        pages = self.request(suffix, pages=True)
        require(isinstance(pages, list), 'Malformed API pages')
        result = []
        for page in pages:
            require(
                isinstance(page, dict) and isinstance(page.get(key), list),
                'Malformed API list',
            )
            require(
                all(isinstance(item, dict) for item in page[key]),
                'Malformed API entry',
            )
            result.extend(page[key])
        return result

    def runs(self, sha):
        require(matches('[0-9a-f]{40}', sha), 'Invalid lookup SHA')
        return self.listing(
            f'actions/workflows/ci-images.yml/runs?head_sha={sha}&per_page=100',
            'workflow_runs',
        )

    def run(self, run_id):
        require(positive(run_id), 'Invalid run ID')
        return self.request(f'actions/runs/{run_id}')

    def artifacts(self, run_id):
        require(positive(run_id), 'Invalid run ID')
        return self.listing(
            f'actions/runs/{run_id}/artifacts?per_page=100', 'artifacts'
        )

    def artifact(self, artifact_id):
        require(positive(artifact_id), 'Invalid artifact ID')
        return self.request(f'actions/artifacts/{artifact_id}')

    def document(self, artifact, filename):
        data = self.request(
            f'actions/artifacts/{artifact["id"]}/zip', binary=True
        )
        try:
            with zipfile.ZipFile(io.BytesIO(data)) as archive:
                require(
                    archive.namelist() == [filename],
                    'Unexpected provenance archive files',
                )
                require(
                    archive.getinfo(filename).file_size <= 1024 * 1024,
                    'Oversized provenance document',
                )
                return json.loads(archive.read(filename))
        except (
            zipfile.BadZipFile,
            ValueError,
            KeyError,
            UnicodeError,
        ) as error:
            raise InvalidEvidence('Invalid provenance archive') from error


def artifact_name(image, architecture, attempt):
    suffix = '-arm64' if architecture == 'arm64' else ''
    return f'{image}{suffix}-attempt-{attempt}'


def filenames(image, architecture):
    require(architecture in ARCHITECTURES, 'Unsupported architecture')
    require(image in IMAGES or image == 'runtime-base-images', 'Unknown image')
    if image == 'runtime-base-images':
        require(architecture == 'amd64', 'Runtime artifact is AMD64 only')
        return ('runtime-base-images.tar', 'modelcar.tar')
    base = image + ('-arm64' if architecture == 'arm64' else '')
    return (
        (base + '.tar', base + '.json')
        if architecture == 'arm64'
        else (base + '.tar',)
    )


def validate_artifact(artifact, identity, name, artifact_id=None):
    require(
        isinstance(artifact, dict) and positive(artifact.get('id')),
        'Invalid artifact ID',
    )
    require(
        artifact_id is None or artifact['id'] == artifact_id,
        'Artifact ID mismatch',
    )
    require(artifact.get('name') == name, 'Artifact name/attempt mismatch')
    require(
        artifact.get('expired') is False,
        'Artifact expired; rerun complete producer',
    )
    require(
        type(artifact.get('size_in_bytes')) is int
        and artifact['size_in_bytes'] > 0,
        'Empty artifact',
    )
    run, pr = artifact.get('workflow_run') or {}, identity['pr']
    expected = {
        'id': identity['run_id'],
        'repository_id': identity['repository_id'],
        'head_repository_id': pr['head_repository_id']
        if pr
        else identity['repository_id'],
        'head_sha': pr['head_sha'] if pr else identity['source_sha'],
    }
    for field, value in expected.items():
        require(run.get(field) == value, f'Artifact {field} mismatch')
    return artifact


def named_artifact(artifacts, identity, name):
    found = [a for a in artifacts if a.get('name') == name]
    require(len(found) <= 1, f'Duplicate artifact {name}')
    if not found:
        raise NotReady(f'Missing {name}')
    return validate_artifact(found[0], identity, name)


def checksum(path):
    require(
        path.is_file() and not path.is_symlink() and path.stat().st_size > 0,
        f'Missing, empty or unsafe file {path.name}',
    )
    digest = hashlib.sha256()
    with path.open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest()


def make_record(identity, architecture, image, artifact_id, directory, api):
    validate_run(api.run(identity['run_id']), identity)
    artifact = validate_artifact(
        api.artifact(artifact_id),
        identity,
        artifact_name(image, architecture, identity['run_attempt']),
        artifact_id,
    )
    return {
        'identity': identity,
        'architecture': architecture,
        'image': image,
        'artifact_id': artifact['id'],
        'artifact_name': artifact['name'],
        'files': {
            name: checksum(directory / name)
            for name in filenames(image, architecture)
        },
    }


def artifact_index(artifacts):
    """Index one fresh run snapshot; never retain this map between phases."""
    require(isinstance(artifacts, list), 'Malformed artifact snapshot')
    indexed = {}
    for artifact in artifacts:
        require(
            isinstance(artifact, dict) and positive(artifact.get('id')),
            'Malformed artifact snapshot entry',
        )
        require(
            artifact['id'] not in indexed, 'Duplicate artifact ID in snapshot'
        )
        indexed[artifact['id']] = artifact
    return indexed


def validate_record(record, identity, architecture, image, artifacts=None):
    fields = {
        'identity',
        'architecture',
        'image',
        'artifact_id',
        'artifact_name',
        'files',
    }
    require(
        isinstance(record, dict) and set(record) == fields,
        'Malformed image record',
    )
    require(record['identity'] == identity, 'Mixed source or producer attempt')
    require(
        record['architecture'] == architecture and record['image'] == image,
        'Image/architecture mismatch',
    )
    name = artifact_name(image, architecture, identity['run_attempt'])
    require(record['artifact_name'] == name, 'Record artifact name mismatch')
    files = record['files']
    require(
        isinstance(files, dict)
        and set(files) == set(filenames(image, architecture)),
        'File inventory mismatch',
    )
    require(
        all(matches('[0-9a-f]{64}', digest) for digest in files.values()),
        'Invalid checksum',
    )
    require(positive(record['artifact_id']), 'Invalid record artifact ID')
    if artifacts is not None:
        validate_artifact(
            artifacts.get(record['artifact_id']),
            identity,
            name,
            record['artifact_id'],
        )
    return record


def make_manifest(identity, architecture, records, api, artifacts=None):
    require(
        isinstance(records, list) and len(records) == len(IMAGES),
        'Incomplete image record set',
    )
    by_image = {
        record.get('image'): record
        for record in records
        if isinstance(record, dict)
    }
    require(set(by_image) == set(IMAGES), 'Duplicate or missing image record')
    if artifacts is None:
        validate_run(api.run(identity['run_id']), identity)
        artifacts = artifact_index(api.artifacts(identity['run_id']))
    ordered = [
        validate_record(
            by_image[image], identity, architecture, image, artifacts
        )
        for image in IMAGES
    ]
    require(
        len({record['artifact_id'] for record in ordered}) == len(IMAGES),
        'Repeated artifact ID',
    )
    return {
        'version': 1,
        'identity': identity,
        'architecture': architecture,
        'records': ordered,
    }


def write_json(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(path.name + '.tmp')
    temporary.write_text(json.dumps(value, indent=2) + '\n')
    temporary.replace(path)


def candidate_runs(runs, consumer):
    result = []
    pr = consumer['pr']
    events = (
        ('push', 'workflow_dispatch')
        if consumer['event'] == 'workflow_dispatch'
        else (consumer['event'],)
    )
    for run in runs:
        require(isinstance(run, dict), 'Malformed producer run')
        if run.get('event') not in events or run.get('head_sha') != (
            pr['head_sha'] if pr else consumer['source_sha']
        ):
            continue
        repo, head = (
            run.get('repository') or {},
            run.get('head_repository') or {},
        )
        if (
            repo.get('id') != consumer['repository_id']
            or repo.get('full_name') != consumer['repository']
        ):
            continue
        if head.get('id') != (
            pr['head_repository_id'] if pr else consumer['repository_id']
        ):
            continue
        require(
            positive(run.get('id')) and positive(run.get('run_attempt')),
            'Malformed matching run ID/attempt',
        )
        require(run.get('path') == WORKFLOW, 'Unexpected producer workflow')
        result.append(run)
    return sorted(result, key=lambda run: run['id'], reverse=True)


def load_state(path, consumer, architecture):
    if not path.exists():
        return None
    require(not path.is_symlink(), 'Unsafe state path')
    state = json.loads(path.read_text())
    require(
        isinstance(state, dict)
        and set(state)
        == {'consumer', 'identity', 'architecture', 'sets', 'last_kind'},
        'Malformed state',
    )
    require(
        state['consumer'] == consumer and state['architecture'] == architecture,
        'State consumer/architecture mismatch',
    )
    same_source(state['identity'], consumer)
    require(isinstance(state['sets'], dict), 'Malformed state sets')
    return state


def resolve_once(api, consumer, architecture, kind, state, pin=None):
    run = None
    artifacts = None
    try:
        if state:
            identity = state['identity']
            run = api.run(identity['run_id'])
            validate_run(run, identity)
        else:
            sha = (
                consumer['pr']['head_sha']
                if consumer['pr']
                else consumer['source_sha']
            )
            runs = candidate_runs(api.runs(sha), consumer)
            if not runs:
                raise NotReady(
                    'No matching producer; run ci-images.yml for this revision'
                )
            run = runs[0]
            identity = dict(
                consumer,
                run_id=run['id'],
                run_attempt=run['run_attempt'],
                event=run['event'],
            )
            validate_run(run, identity)
            artifacts = api.artifacts(run['id'])
            source = named_artifact(
                artifacts,
                identity,
                f'ci-image-source-attempt-{run["run_attempt"]}',
            )
            identity = api.document(source, 'source.json')
            same_source(identity, consumer)
            validate_run(run, identity)
            state = {
                'consumer': consumer,
                'identity': identity,
                'architecture': architecture,
                'sets': {},
                'last_kind': kind,
            }
            # Persist only validated identity, before waiting for any image set.
            # Future polls must not rediscover/reselect another run or attempt.
            if pin is not None:
                pin(state)
        if artifacts is None:
            artifacts = api.artifacts(identity['run_id'])
        attempt = identity['run_attempt']
        indexed = artifact_index(artifacts)
        if kind == 'runtime':
            require(architecture == 'amd64', 'Runtime images require AMD64')
            metadata = named_artifact(
                artifacts,
                identity,
                f'runtime-base-images-record-amd64-attempt-{attempt}',
            )
            record = api.document(metadata, 'runtime-base-images.json')
            records = [
                validate_record(
                    record,
                    identity,
                    architecture,
                    'runtime-base-images',
                    indexed,
                )
            ]
        else:
            metadata = named_artifact(
                artifacts,
                identity,
                f'ci-image-set-{architecture}-attempt-{attempt}',
            )
            manifest = api.document(metadata, 'manifest.json')
            require(
                isinstance(manifest, dict)
                and set(manifest)
                == {'version', 'identity', 'architecture', 'records'},
                'Malformed ready manifest',
            )
            require(
                manifest['version'] == 1
                and manifest['identity'] == identity
                and manifest['architecture'] == architecture,
                'Ready manifest identity mismatch',
            )
            records = make_manifest(
                identity,
                architecture,
                manifest['records'],
                api,
                artifacts=indexed,
            )['records']
        validate_run(api.run(identity['run_id']), identity)
        state['sets'][kind] = records
        state['last_kind'] = kind
        return state
    except NotReady as error:
        if run is not None and run.get('status') == 'completed':
            error.completed_producer = (run['id'], run['run_attempt'])
        raise


def resolve(
    api,
    consumer,
    architecture,
    kind,
    path,
    timeout=2400,
    poll=30,
    clock=time.monotonic,
    sleep=time.sleep,
):
    require(timeout > 0 and poll > 0, 'Wait bounds must be positive')
    state = load_state(path, consumer, architecture)
    deadline = clock() + timeout
    publication_deadline = None
    completed_producer = None
    delay = min(poll, 120)
    previous_deadline = getattr(api, 'deadline', None)
    try:
        while True:
            api.deadline = min(deadline, publication_deadline or deadline)
            try:
                state = resolve_once(
                    api,
                    consumer,
                    architecture,
                    kind,
                    state,
                    pin=lambda pinned: write_json(path, pinned),
                )
                write_json(path, state)
                return state
            except (NotReady, APIError) as error:
                state = load_state(path, consumer, architecture)
                completed = getattr(error, 'completed_producer', None)
                if completed is not None and completed != completed_producer:
                    completed_producer = completed
                    publication_deadline = clock() + 60
                effective_deadline = min(
                    deadline, publication_deadline or deadline
                )
                remaining = effective_deadline - clock()
                if remaining <= 0:
                    if (
                        publication_deadline is not None
                        and clock() >= publication_deadline
                    ):
                        raise InvalidEvidence(
                            'Completed producer is missing current-attempt artifacts after '
                            '60 seconds; rerun the complete producer, then consumers'
                        ) from error
                    raise InvalidEvidence(
                        f'Timed out waiting for shared CI images: {error}'
                    ) from error
                print(f'Waiting for shared CI images: {error}', file=sys.stderr)
                sleep(min(delay, remaining))
                delay = min(delay * 2, 120)
    finally:
        api.deadline = previous_deadline


def emit_outputs(path, state, kind):
    records = state['sets'][kind]
    names = [name for record in records for name in record['files']]
    with path.open('a') as stream:
        stream.write(
            f'run-id={state["identity"]["run_id"]}\nrepository={state["identity"]["repository"]}\n'
        )
        stream.write(
            'artifact-ids='
            + ','.join(str(record['artifact_id']) for record in records)
            + '\n'
        )
        stream.write(
            'required-files<<SHARED_CI_FILES\n'
            + '\n'.join(names)
            + '\nSHARED_CI_FILES\n'
        )


def verify(api, state, consumer, directory, layout, kind=None):
    kind = kind or state['last_kind']
    identity, architecture = state['identity'], state['architecture']
    require(state['consumer'] == consumer, 'State consumer changed')
    same_source(identity, consumer)
    require(
        kind in ('runtime', 'images') and kind in state['sets'],
        'Missing selected set',
    )
    records = state['sets'][kind]
    require(isinstance(records, list), 'Malformed selected records')
    for record in records:
        require(isinstance(record, dict), 'Malformed selected record')
        image = record.get('image')
        require(
            image in (IMAGES if kind == 'images' else ('runtime-base-images',)),
            'Unexpected selected image',
        )
        validate_record(record, identity, architecture, image)
    require(
        directory.is_dir() and not directory.is_symlink(),
        'Unsafe download directory',
    )
    for record in records:
        for name, expected in record['files'].items():
            require(
                checksum(directory / name) == expected,
                f'Checksum mismatch: {name}',
            )
    # The resolver already bound immutable IDs. Check remote availability once,
    # after hashing, then recheck the attempt immediately before accepting files.
    indexed = artifact_index(api.artifacts(identity['run_id']))
    if kind == 'images':
        make_manifest(identity, architecture, records, api, artifacts=indexed)
    else:
        require(len(records) == 1, 'Invalid runtime set')
        validate_record(
            records[0], identity, architecture, 'runtime-base-images', indexed
        )
    validate_run(api.run(identity['run_id']), identity)
    if layout == 'nested' and kind == 'images':
        for record in records:
            require(
                not (directory / record['image']).exists(),
                'Destination already exists',
            )
        for record in records:
            target = directory / record['image']
            target.mkdir()
            for name in record['files']:
                (directory / name).replace(target / name)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest='command', required=True)
    source = commands.add_parser('source')
    source.add_argument('--output', required=True, type=Path)
    record = commands.add_parser('record')
    record.add_argument('--architecture', required=True, choices=ARCHITECTURES)
    record.add_argument(
        '--image', required=True, choices=(*IMAGES, 'runtime-base-images')
    )
    record.add_argument('--artifact-id', required=True, type=int)
    record.add_argument('--directory', required=True, type=Path)
    record.add_argument('--output', required=True, type=Path)
    publish = commands.add_parser('publish')
    publish.add_argument('--architecture', required=True, choices=ARCHITECTURES)
    publish.add_argument('--records-dir', required=True, type=Path)
    publish.add_argument('--output', required=True, type=Path)
    resolver = commands.add_parser('resolve')
    resolver.add_argument(
        '--architecture', required=True, choices=ARCHITECTURES
    )
    resolver.add_argument(
        '--kind', required=True, choices=('runtime', 'images')
    )
    resolver.add_argument('--state', required=True, type=Path)
    resolver.add_argument('--output', required=True, type=Path)
    resolver.add_argument('--timeout-seconds', type=int, default=2400)
    resolver.add_argument('--poll-seconds', type=int, default=30)
    verifier = commands.add_parser('verify')
    verifier.add_argument('--state', required=True, type=Path)
    verifier.add_argument('--directory', required=True, type=Path)
    verifier.add_argument('--layout', required=True, choices=('nested', 'flat'))
    verifier.add_argument('--kind', choices=('runtime', 'images'))
    args = parser.parse_args()
    try:
        identity = environment_identity()
        api = GitHub(identity['repository'])
        if args.command == 'source':
            write_json(args.output, identity)
        elif args.command == 'record':
            write_json(
                args.output,
                make_record(
                    identity,
                    args.architecture,
                    args.image,
                    args.artifact_id,
                    args.directory,
                    api,
                ),
            )
        elif args.command == 'publish':
            require(
                {p.name for p in args.records_dir.iterdir()}
                == {f'{image}.json' for image in IMAGES},
                'Expected exactly seven IMAGE.json records',
            )
            records = [
                json.loads((args.records_dir / f'{image}.json').read_text())
                for image in IMAGES
            ]
            write_json(
                args.output,
                make_manifest(identity, args.architecture, records, api),
            )
        elif args.command == 'resolve':
            state = resolve(
                api,
                identity,
                args.architecture,
                args.kind,
                args.state,
                args.timeout_seconds,
                args.poll_seconds,
            )
            emit_outputs(args.output, state, args.kind)
        else:
            architecture = json.loads(args.state.read_text())['architecture']
            state = load_state(args.state, identity, architecture)
            require(state is not None, 'Missing resolver state')
            verify(api, state, identity, args.directory, args.layout, args.kind)
    except (
        InvalidEvidence,
        APIError,
        KeyError,
        OSError,
        ValueError,
        subprocess.SubprocessError,
    ) as error:
        print(f'::error::{error}', file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
