# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Provision disposable storage using the real profile controller policies."""

import base64
from contextlib import redirect_stdout
import copy
import hashlib
import importlib.util
from io import StringIO
import json
import os
from pathlib import Path
import sys
from types import SimpleNamespace
from unittest.mock import patch

import boto3

endpoint = os.environ['S3_ENDPOINT_URL']
os.environ.setdefault('KFP_VERSION', 'storage-authorization-fixture')
repo = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location(
    'profile_sync', repo / 'manifests/kustomize/base/installs/multi-user/'
    'pipelines-profile-controller/sync.py')
producer = importlib.util.module_from_spec(spec)
sys.dont_write_bytecode = True
spec.loader.exec_module(producer)
try:
    producer.s3.create_bucket(Bucket='mlpipeline')
except producer.s3.exceptions.BucketAlreadyOwnedByYou:
    pass

# The fixture runs the real reconciliation code without starting its HTTP server.
with patch.object(
        producer,
        'ThreadingHTTPServer',
        side_effect=lambda address, handler: SimpleNamespace(
            RequestHandlerClass=handler)):
    handler_type = producer.server_factory(**producer.get_settings_from_env(
        artifacts_proxy_enabled="true")).RequestHandlerClass
handler = handler_type.__new__(handler_type)
identities = {}
parents = {}
observations = {}
for namespace, legacy_root in [('team-a', 'model/customer-17-training/'),
                               ('team-b', 'model/customer-23-training/')]:
    parents[namespace] = {
        'metadata': {
            'name': namespace,
            'uid': f'fixture-{namespace}-uid',
            'labels': {
                'pipelines.kubeflow.org/enabled': 'true'
            }
        }
    }
    with redirect_stdout(StringIO()):
        resources = handler.sync(
            parents[namespace], {
                'Secret.v1': {},
                'ConfigMap.v1': {},
                'Deployment.apps/v1': {},
                'Service.v1': {}
            })['attachments']
    observations[namespace] = resources
    secret = next(
        resource for resource in resources
        if resource.get('kind') == 'Secret' and
        resource['metadata']['name'] == 'mlpipeline-minio-artifact')
    identities[namespace] = {
        key: base64.b64decode(value).decode()
        for key, value in secret['data'].items()
    }
    # An administrator can authorize an existing arbitrary prefix without moving
    # objects. Preserve the stock own-private and explicitly shared permissions.
    producer.iam.put_user_policy(
        UserName=namespace,
        PolicyName='ExistingArtifactRoot',
        PolicyDocument=json.dumps({
            'Version':
                '2012-10-17',
            'Statement': [{
                'Effect': 'Allow',
                'Action': ['s3:Get*', 's3:List*'],
                'Resource': [f'arn:aws:s3:::mlpipeline/{legacy_root}*']
            }]
        }))

policy = json.loads(
    producer.s3.get_bucket_policy(Bucket='mlpipeline')['Policy'])
for namespace, legacy_root in [('team-a', 'model/customer-17-training/'),
                               ('team-b', 'model/customer-23-training/')]:
    policy['Statement'].append({
        'Sid': f'ExistingRoot{namespace.replace("-", "")}',
        'Effect': 'Allow',
        # SeaweedFS 4.34 accepts exact ARN strings, while object-shaped AWS
        # principals require a newer release. Both forms express the same scope.
        'Principal': producer.iam.get_user(UserName=namespace)['User']['Arn'],
        'Action': 's3:ListBucket',
        'Resource': 'arn:aws:s3:::mlpipeline',
        'Condition': {
            'StringLike': {
                's3:prefix': f'{legacy_root}*'
            }
        }
    })
producer.s3.put_bucket_policy(Bucket='mlpipeline', Policy=json.dumps(policy))

objects = {
    'private-artifacts/team-a/model.txt':
        'team-a current artifact',
    'private-artifacts/team-b/model.txt':
        'team-b secret artifact',
    'private/team-a/model.txt':
        'team-a existing private artifact',
    'private/team-b/model.txt':
        'team-b existing private artifact',
    'model/customer-17-training/model.txt':
        'team-a historical artifact',
    'model/customer-23-training/model.txt':
        'team-b historical secret',
    'model/customer-17-training/directory/part.txt':
        'team-a historical directory',
    'model/customer-23-training/directory/part.txt':
        'team-b historical directory',
    'shared/model.txt':
        'administrator-shared artifact',
}
for key, value in objects.items():
    producer.s3.put_object(Bucket='mlpipeline', Key=key, Body=value.encode())

# Assert the storage policy independently before testing the KFP serving path.
for namespace, credentials in identities.items():
    tenant = boto3.client(
        's3',
        endpoint_url=endpoint,
        region_name='us-east-1',
        aws_access_key_id=credentials['accesskey'],
        aws_secret_access_key=credentials['secretkey'])
    own = 'customer-17' if namespace == 'team-a' else 'customer-23'
    other = 'customer-23' if namespace == 'team-a' else 'customer-17'
    assert tenant.get_object(
        Bucket='mlpipeline',
        Key=f'model/{own}-training/model.txt')['Body'].read()
    try:
        tenant.get_object(
            Bucket='mlpipeline', Key=f'model/{other}-training/model.txt')
    except tenant.exceptions.ClientError as error:
        assert error.response['Error']['Code'] == 'AccessDenied', error
    else:
        raise AssertionError(
            f'{namespace} could read the other tenant artifact')

# Exercise controller reconciliation against the real IAM API for pre-existing
# credentials and administrator-granted historical roots. Kubernetes observation
# and readiness are simulated here; this fixture does not replace cluster rollout
# acceptance. No credential or capability value is printed.
authorizations = {}
original_bucket_policy = producer.s3.get_bucket_policy(
    Bucket='mlpipeline')['Policy']
for namespace, credentials in identities.items():
    keys = producer.iam.list_access_keys(
        UserName=namespace)['AccessKeyMetadata']
    assert any(key['AccessKeyId'] == credentials['accesskey'] and
               key.get('Status') == 'Active' for key in keys)
    states = []
    for attempt in range(4):
        attachments = {
            'Secret.v1': {},
            'ConfigMap.v1': {},
            'Deployment.apps/v1': {},
            'Service.v1': {},
            'NetworkPolicy.networking.k8s.io/v1': {},
            'Role.rbac.authorization.k8s.io/v1': {},
            'RoleBinding.rbac.authorization.k8s.io/v1': {}
        }
        for original in observations[namespace]:
            resource = copy.deepcopy(original)
            if resource['kind'] == 'Deployment':
                resource['metadata']['generation'] = 1
                resource['status'] = {
                    'observedGeneration': 1,
                    'replicas': 1,
                    'updatedReplicas': 1,
                    'readyReplicas': 1,
                    'availableReplicas': 1,
                    'unavailableReplicas': 0
                }
            resource_type = f"{resource['kind']}.{resource['apiVersion']}"
            attachments.setdefault(
                resource_type, {}
            )[f"{resource['metadata']['namespace']}/{resource['metadata']['name']}"] = resource
        with redirect_stdout(StringIO()):
            observations[namespace] = handler.sync(parents[namespace],
                                                   attachments)['attachments']
        capability = next(
            resource for resource in observations[namespace]
            if resource['kind'] == 'Secret' and
            resource['metadata']['name'] == f'kfp-artifact-storage-{namespace}')
        authorization = json.loads(
            base64.b64decode(capability['data']['authorization.json']))
        states.append(authorization['state'])
        if authorization['state'] == 'ready':
            break
    assert authorization[
        'state'] == 'ready', f'{namespace}: readiness states {states}'
    # The Node fixture reaches this store through Docker's random localhost port,
    # not Kubernetes DNS. Rebind only that transport coordinate; IAM verification,
    # namespace UID and credentials above come from real controller reconciliation.
    from urllib.parse import urlparse
    parsed_endpoint = urlparse(endpoint)
    authorization['credentialBinding'] = hashlib.sha256('\0'.join([
        namespace, parents[namespace]['metadata']['uid'],
        f'{parsed_endpoint.scheme}://{parsed_endpoint.hostname}:{parsed_endpoint.port}',
        credentials['accesskey'], credentials['secretkey']
    ]).encode()).hexdigest()
    authorizations[namespace] = authorization
assert producer.s3.get_bucket_policy(
    Bucket='mlpipeline')['Policy'] == original_bucket_policy

Path(sys.argv[1]).write_text(
    json.dumps({
        'endpoint': endpoint,
        'identities': identities,
        'authorizations': authorizations
    }))
print(
    'Provisioned two isolated identities with stock and historical artifact roots.'
)
