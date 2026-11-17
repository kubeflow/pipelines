"""Exercise a pinned SeaweedFS upgrade using the unchanged release profile runtime."""

import argparse
import json
from pathlib import Path

import third_party_upgrade_compatibility as compat
from botocore.exceptions import ClientError


def denied(call):
    try:
        call()
    except ClientError as error:
        if (
            error.response['Error']['Code'] == 'AccessDenied'
            and error.response['ResponseMetadata']['HTTPStatusCode'] == 403
        ):
            return
        raise
    raise AssertionError('Cross-namespace operation unexpectedly succeeded')


def isolation(endpoint, first, second):
    users = [
        (name, compat.client(endpoint, *compat.secret_credentials(secret)))
        for name, secret in [first, second]
    ]
    for name, user in users:
        key = f'private-artifacts/{name}/isolation-check'
        user.put_object(Bucket=compat.BUCKET, Key=key, Body=compat.PAYLOAD)
        compat.verify_payload(user, key)
        listed = user.list_objects_v2(
            Bucket=compat.BUCKET, Prefix=f'private-artifacts/{name}/'
        )
        assert any(item['Key'] == key for item in listed.get('Contents', []))
    for (name, user), (other, _) in [users, users[::-1]]:
        denied(
            lambda: user.get_object(
                Bucket=compat.BUCKET,
                Key=f'private-artifacts/{other}/isolation-check',
            )
        )
        denied(
            lambda: user.put_object(
                Bucket=compat.BUCKET,
                Key=f'private-artifacts/{other}/forbidden',
                Body=b'forbidden',
            )
        )
        denied(
            lambda: user.list_objects_v2(
                Bucket=compat.BUCKET, Prefix=f'private-artifacts/{other}/'
            )
        )


def pods(evidence, stage, expected):
    document = json.loads(compat.kubectl('get', 'pods', '-o', 'json'))
    ready = [
        p
        for p in document['items']
        if not p['metadata'].get('deletionTimestamp')
        and any(
            c.get('ready')
            for c in p.get('status', {}).get('containerStatuses', [])
            if c['name'] == 'seaweedfs'
        )
    ]
    assert len(ready) == 1
    container = next(
        c for c in ready[0]['spec']['containers'] if c['name'] == 'seaweedfs'
    )
    assert container['image'] == expected
    status = next(
        c
        for c in ready[0]['status']['containerStatuses']
        if c['name'] == 'seaweedfs'
    )
    assert status['imageID']
    (evidence / f'{stage}-pods.json').write_text(
        json.dumps(document, indent=2) + '\n'
    )
    return status['imageID']


def exercise(source, config, evidence):
    compat.ROOT = source
    compat.SEAWEED = source / 'manifests/kustomize/third-party/seaweedfs/base'
    compat.PROFILE = (
        source
        / 'manifests/kustomize/base/installs/multi-user/pipelines-profile-controller'
    )
    assert (
        compat.run('git', '-C', str(source), 'rev-parse', 'HEAD').strip()
        == config['source_sha']
    )
    pair = config['pairs']['seaweedfs']
    storage = compat.rendered(compat.SEAWEED)
    assert compat.storage_image(storage) == pair['baseline']
    credentials, pvc = compat.seed_old_store(
        storage, pair['baseline'], evidence
    )
    # Create actual IAM credentials and objects BEFORE changing the store image.
    existing = compat.exercise_profile(credentials, evidence)
    with compat.forward(
        'kubeflow-pipelines-profile-controller',
        compat.PROFILE_LOCAL_PORT,
        compat.PROFILE_PORT,
        evidence,
    ) as webhook:
        other = compat.sync_profile(webhook, 'compat-other')
    with compat.forward(
        'seaweedfs', compat.S3_LOCAL_PORT, compat.S3_PORT, evidence
    ) as endpoint:
        isolation(endpoint, ('compat-user', existing), ('compat-other', other))
    old_id = pods(evidence, 'baseline', pair['baseline'])
    compat.upgrade_store(pair['candidate'])
    new_id = pods(evidence, 'upgraded', pair['candidate'])
    compat.kubectl(
        'rollout', 'restart', 'deployment/kubeflow-pipelines-profile-controller'
    )
    compat.wait_ready('kubeflow-pipelines-profile-controller')
    with compat.forward(
        'kubeflow-pipelines-profile-controller',
        compat.PROFILE_LOCAL_PORT,
        compat.PROFILE_PORT,
        evidence,
    ) as webhook:
        assert (
            compat.sync_profile(webhook, 'compat-user', existing)['data']
            == existing['data']
        )
        new_user = compat.sync_profile(webhook, 'compat-new')
    with compat.forward(
        'seaweedfs', compat.S3_LOCAL_PORT, compat.S3_PORT, evidence
    ) as endpoint:
        isolation(endpoint, ('compat-user', existing), ('compat-other', other))
        isolation(endpoint, ('compat-user', existing), ('compat-new', new_user))
    compat.verify_after_restart(credentials, existing, pvc, evidence)
    with compat.forward(
        'seaweedfs', compat.S3_LOCAL_PORT, compat.S3_PORT, evidence
    ) as endpoint:
        isolation(endpoint, ('compat-user', existing), ('compat-other', other))
    compat.write_result(evidence, pair['baseline'], pair['candidate'], pvc)
    result = json.loads((evidence / 'result.json').read_text())
    result.update(
        source_sha=config['source_sha'],
        baseline_image_id=old_id,
        candidate_image_id=new_id,
    )
    result['checks'] += [
        'pre-upgrade-user-credentials-retained',
        'new-user-after-upgrade',
        'cross-namespace-read-denied',
        'cross-namespace-write-denied',
        'cross-namespace-list-denied',
        'own-prefix-list-permitted',
        'isolation-after-restart',
    ]
    (evidence / 'result.json').write_text(json.dumps(result, indent=2) + '\n')


def main():
    parser = argparse.ArgumentParser()
    for name in ['source', 'pairs', 'evidence']:
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    source = args.source.resolve()
    evidence = args.evidence.resolve()
    evidence.mkdir(parents=True)
    config = json.loads(args.pairs.read_text())
    try:
        compat.run(
            'kind',
            'create',
            'cluster',
            '--name',
            compat.CLUSTER,
            '--image',
            'kindest/node:v1.36.1',
            '--wait',
            '120s',
        )
        exercise(source, config, evidence)
    finally:
        for name, command in {
            'pods.json': ('get', 'pods', '-o', 'json'),
            'events.txt': (
                'get',
                'events',
                '--sort-by=.metadata.creationTimestamp',
            ),
            'seaweedfs.log': (
                'logs',
                'deployment/seaweedfs',
                '--all-containers=true',
            ),
            'profile.log': (
                'logs',
                'deployment/kubeflow-pipelines-profile-controller',
            ),
        }.items():
            try:
                (evidence / name).write_text(compat.kubectl(*command))
            except Exception as error:
                (evidence / name).write_text(str(error))
        compat.delete_cluster()


if __name__ == '__main__':
    main()
