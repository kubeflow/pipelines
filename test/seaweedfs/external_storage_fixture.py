# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Exercise external object storage with real, expiring brokered STS
credentials."""

import argparse
import base64
import hashlib
import json
import os
import secrets
import subprocess
import tempfile
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.error import URLError
from urllib.request import urlopen

import boto3
from botocore.config import Config
from botocore.exceptions import ClientError

BUCKET = 'external-artifacts'
REGION = 'us-east-1'
TENANTS = {'team-a': 'customer-17', 'team-b': 'customer-23'}
SESSION_SECONDS = 12


def encoded(value):
    return base64.urlsafe_b64encode(value).decode().rstrip('=')


def run(*args, **kwargs):
    return subprocess.run(
        args, check=True, text=True, capture_output=True, **kwargs
    ).stdout.strip()


class ExternalStorage:
    """Disposable OIDC issuer, storage gateway, and shared-file credential
    broker."""

    def __init__(self, directory, image):
        self.directory = directory
        self.config_directory = directory / 'storage-config'
        self.image = image
        self.container = 'kfp-external-artifact-' + secrets.token_hex(6)
        self.control_token = secrets.token_urlsafe(32)
        self.lock = threading.Lock()
        self.generations = {tenant: 0 for tenant in TENANTS}
        self.server = None
        self.endpoint = None
        self.started = False
        self.evidence = {
            'status': 'failed',
            'sessions': [],
            'checks': [],
            'cleanupErrors': [],
        }

    def start(self):
        run(
            'openssl', 'genrsa', '-out', str(self.directory / 'key.pem'), '2048'
        )
        (self.directory / 'key.pem').chmod(0o600)
        modulus = run(
            'openssl',
            'rsa',
            '-in',
            str(self.directory / 'key.pem'),
            '-noout',
            '-modulus',
        ).split('=')[1]
        jwks = {
            'keys': [
                {
                    'kty': 'RSA',
                    'use': 'sig',
                    'alg': 'RS256',
                    'kid': 'fixture',
                    'n': encoded(bytes.fromhex(modulus)),
                    'e': 'AQAB',
                }
            ]
        }
        fixture = self

        class Handler(BaseHTTPRequestHandler):
            def respond(self, status, body):
                payload = json.dumps(body).encode()
                self.send_response(status)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(payload)

            def do_GET(self):
                if self.path == '/jwks':
                    self.respond(200, jwks)
                elif self.path == '/.well-known/openid-configuration':
                    self.respond(
                        200,
                        {
                            'issuer': fixture.issuer,
                            'jwks_uri': fixture.issuer + '/jwks',
                            'authorization_endpoint': fixture.issuer
                            + '/authorize',
                            'token_endpoint': fixture.issuer + '/token',
                            'response_types_supported': ['id_token'],
                            'subject_types_supported': ['public'],
                            'id_token_signing_alg_values_supported': ['RS256'],
                        },
                    )
                else:
                    self.respond(404, {})

            def do_POST(self):
                tenant = self.path.removeprefix('/refresh/')
                if (
                    self.headers.get('X-Fixture-Control')
                    != fixture.control_token
                    or tenant not in TENANTS
                    or self.path != '/refresh/' + tenant
                ):
                    self.respond(403, {})
                    return
                try:
                    with fixture.lock:
                        self.respond(200, fixture.refresh(tenant))
                except Exception:
                    self.respond(
                        500, {'error': 'STS credential refresh failed'}
                    )

            def log_message(self, *args):
                pass

        self.server = ThreadingHTTPServer(('0.0.0.0', 0), Handler)
        self.issuer = 'http://host.docker.internal:' + str(
            self.server.server_port
        )
        self.control_url = 'http://127.0.0.1:' + str(self.server.server_port)
        threading.Thread(target=self.server.serve_forever, daemon=True).start()
        self.configure()
        run(
            'docker',
            'run',
            '--detach',
            '--name',
            self.container,
            '--add-host',
            'host.docker.internal:host-gateway',
            '--publish',
            '127.0.0.1::8333',
            '--volume',
            str(self.config_directory) + ':/fixture:ro',
            self.image,
            'server',
            '-dir=/tmp',
            '-s3',
            '-iam',
            '-filer',
            '-master.volumePreallocate=false',
            '-s3.config=/fixture/s3.json',
            '-s3.iam.config=/fixture/iam.json',
        )
        self.started = True
        port = run(
            'docker',
            'inspect',
            '--format',
            '{{(index (index .NetworkSettings.Ports "8333/tcp") 0).HostPort}}',
            self.container,
        )
        self.endpoint = 'http://127.0.0.1:' + port
        self.evidence['image'] = self.image
        self.evidence['imageID'] = run(
            'docker', 'inspect', '--format', '{{.Image}}', self.container
        )
        self.wait_until_ready()
        self.admin = self.client(
            {
                'AccessKeyId': 'fixture-admin',
                'SecretAccessKey': 'fixture-admin-secret',
            }
        )
        self.admin.create_bucket(Bucket=BUCKET)
        for tenant, customer in TENANTS.items():
            for suffix in ['model.txt', 'directory/part.txt']:
                self.admin.put_object(
                    Bucket=BUCKET,
                    Key='model/' + customer + '-training/' + suffix,
                    Body=(tenant + ' historical artifact').encode(),
                )
        self.evidence['checks'].append(
            'seeded pre-existing non-namespace roots'
        )

    def container_state(self):
        state = json.loads(
            run(
                'docker',
                'inspect',
                '--format',
                '{{json .State}}',
                self.container,
                timeout=10,
            )
        )
        return {
            key: state.get(key) for key in ['Status', 'ExitCode', 'OOMKilled']
        }

    def startup_failure(self, state):
        # Never publish raw gateway logs: a provider may log credential material.
        logs = subprocess.run(
            ['docker', 'logs', '--tail', '100', self.container],
            text=True,
            capture_output=True,
            check=False,
            timeout=10,
        )
        output = (logs.stdout + logs.stderr).lower()
        errors = [
            message
            for message in [
                'permission denied',
                'no such file or directory',
                'address already in use',
                'no space left on device',
                'connection refused',
                'invalid character',
                'context deadline exceeded',
            ]
            if message in output
        ]
        diagnostic = {'containerState': state, 'detectedErrors': errors}
        self.evidence['startupFailure'] = diagnostic
        return RuntimeError(
            'External storage startup failed: ' + json.dumps(diagnostic)
        )

    def wait_until_ready(self):
        for _ in range(60):
            try:
                with urlopen(self.endpoint + '/status', timeout=1):
                    return
            except (URLError, TimeoutError, ConnectionError):
                state = self.container_state()
                if state['Status'] != 'running':
                    raise self.startup_failure(state) from None
                time.sleep(1)
        raise self.startup_failure(state)

    def configure(self):
        # The image drops to its own UID. Mount only readable gateway config,
        # keeping the signing key and refreshed session files in the private parent.
        self.config_directory.mkdir(mode=0o755)
        self.config_directory.chmod(0o755)
        policies, roles = [], []
        for tenant, customer in TENANTS.items():
            root = 'model/' + customer + '-training/'
            policies.append(
                {
                    'name': tenant,
                    'document': {
                        'Version': '2012-10-17',
                        'Statement': [
                            {
                                'Effect': 'Allow',
                                'Action': ['s3:GetObject'],
                                'Resource': [
                                    'arn:aws:s3:::' + BUCKET + '/' + root + '*'
                                ],
                            },
                            {
                                'Effect': 'Allow',
                                'Action': ['s3:ListBucket'],
                                'Resource': ['arn:aws:s3:::' + BUCKET],
                                'Condition': {
                                    'StringLike': {'s3:prefix': root + '*'}
                                },
                            },
                        ],
                    },
                }
            )
            roles.append(
                {
                    'roleName': tenant,
                    'roleArn': 'arn:aws:iam::role/' + tenant,
                    'attachedPolicies': [tenant],
                    'trustPolicy': {
                        'Version': '2012-10-17',
                        'Statement': [
                            {
                                'Effect': 'Allow',
                                'Principal': {'Federated': 'fixture'},
                                'Action': ['sts:AssumeRoleWithWebIdentity'],
                                'Condition': {
                                    'StringEquals': {
                                        'oidc:sub': 'system:serviceaccount:'
                                        + tenant
                                        + ':artifact-reader'
                                    }
                                },
                            }
                        ],
                    },
                }
            )
        self.write_storage_config(
            'iam.json',
            {
                'sts': {
                    'tokenDuration': str(SESSION_SECONDS) + 's',
                    'maxSessionLength': '1h',
                    'issuer': 'kfp-fixture',
                    'signingKey': base64.b64encode(
                        secrets.token_bytes(48)
                    ).decode(),
                },
                'providers': [
                    {
                        'name': 'fixture',
                        'type': 'oidc',
                        'enabled': True,
                        'config': {
                            'issuer': self.issuer,
                            'clientId': 'artifact-storage',
                            'jwksUri': self.issuer + '/jwks',
                        },
                    }
                ],
                'policy': {'defaultEffect': 'Deny'},
                'policies': policies,
                'roles': roles,
            },
        )
        self.write_storage_config(
            's3.json',
            {
                'identities': [
                    {
                        'name': 'fixture-admin',
                        'credentials': [
                            {
                                'accessKey': 'fixture-admin',
                                'secretKey': 'fixture-admin-secret',
                            }
                        ],
                        'actions': ['Admin'],
                    }
                ]
            },
        )

    def write_storage_config(self, name, value):
        path = self.config_directory / name
        path.write_text(json.dumps(value))
        path.chmod(0o644)

    def write_json(self, name, value):
        (self.directory / name).write_text(json.dumps(value))

    def assume(self, tenant, role=None):
        claims = {
            'iss': self.issuer,
            'aud': 'artifact-storage',
            'sub': 'system:serviceaccount:' + tenant + ':artifact-reader',
            'iat': int(time.time()),
            'exp': int(time.time()) + 600,
        }
        payload = (
            encoded(json.dumps({'alg': 'RS256', 'kid': 'fixture'}).encode())
            + '.'
            + encoded(json.dumps(claims).encode())
        ).encode()
        signature = subprocess.run(
            [
                'openssl',
                'dgst',
                '-sha256',
                '-sign',
                str(self.directory / 'key.pem'),
            ],
            input=payload,
            capture_output=True,
            check=True,
        ).stdout
        sts = boto3.client(
            'sts',
            endpoint_url=self.endpoint,
            region_name=REGION,
            aws_access_key_id='not-used',
            aws_secret_access_key='not-used',
        )
        # Omit DurationSeconds: SeaweedFS's configured short default makes actual
        # session expiry testable without weakening the AWS 900s request minimum.
        return sts.assume_role_with_web_identity(
            RoleArn='arn:aws:iam::role/' + (role or tenant),
            RoleSessionName='artifact-fixture-' + str(self.generations[tenant]),
            WebIdentityToken=payload.decode() + '.' + encoded(signature),
        )['Credentials']

    def client(self, credentials):
        return boto3.client(
            's3',
            endpoint_url=self.endpoint,
            region_name=REGION,
            aws_access_key_id=credentials['AccessKeyId'],
            aws_secret_access_key=credentials['SecretAccessKey'],
            aws_session_token=credentials.get('SessionToken'),
            config=Config(
                s3={'addressing_style': 'path'}, retries={'max_attempts': 0}
            ),
        )

    def refresh(self, tenant):
        credentials = self.assume(tenant)
        path = self.directory / (tenant + '.credentials')
        temporary = path.with_suffix('.tmp')
        temporary.write_text(
            '[artifact-reader]\naws_access_key_id = '
            + credentials['AccessKeyId']
            + '\naws_secret_access_key = '
            + credentials['SecretAccessKey']
            + '\naws_session_token = '
            + credentials['SessionToken']
            + '\n'
        )
        temporary.chmod(0o600)
        temporary.replace(path)
        self.generations[tenant] += 1
        evidence = {
            'tenant': tenant,
            'generation': self.generations[tenant],
            'expiration': credentials['Expiration'].isoformat(),
            'credentialFingerprint': hashlib.sha256(
                credentials['AccessKeyId'].encode()
            ).hexdigest(),
        }
        self.evidence['sessions'].append(evidence)
        return evidence

    def assert_denied(self, operation, **arguments):
        try:
            operation(Bucket=BUCKET, **arguments)
        except ClientError as error:
            if error.response['Error']['Code'] not in (
                'AccessDenied',
                'InvalidAccessKeyId',
                'ExpiredToken',
            ):
                raise
        else:
            raise AssertionError('Storage allowed a forbidden request')

    def verify_storage(self):
        for tenant, customer in TENANTS.items():
            other = 'customer-23' if tenant == 'team-a' else 'customer-17'
            credentials = self.assume(tenant)
            client = self.client(credentials)
            own_root = 'model/' + customer + '-training/'
            assert (
                client.get_object(Bucket=BUCKET, Key=own_root + 'model.txt')[
                    'Body'
                ].read()
                == (tenant + ' historical artifact').encode()
            )
            assert (
                len(
                    client.list_objects_v2(Bucket=BUCKET, Prefix=own_root)[
                        'Contents'
                    ]
                )
                == 2
            )
            self.assert_denied(
                client.get_object, Key='model/' + other + '-training/model.txt'
            )
            self.assert_denied(
                client.list_objects_v2, Prefix='model/' + other + '-training/'
            )
            try:
                self.assume(
                    tenant, 'team-b' if tenant == 'team-a' else 'team-a'
                )
            except ClientError as error:
                assert error.response['Error']['Code'] == 'AccessDenied'
            else:
                raise AssertionError(
                    'OIDC subject could assume the other tenant role'
                )
        self.evidence['checks'].append(
            'real STS role trust and cross-tenant read/list isolation'
        )
        expires = credentials['Expiration'].timestamp()
        time.sleep(max(0, expires - time.time()) + 1)
        self.assert_denied(client.get_object, Key=own_root + 'model.txt')
        self.evidence['checks'].append('real expired session denied by storage')

    def manifest(self):
        namespaces = {}
        for tenant in TENANTS:
            self.refresh(tenant)
            namespaces[tenant] = {
                'namespaceUID': 'fixture-' + tenant + '-uid',
                'configurationRevision': hashlib.sha256(
                    b'fixture receiver template v1'
                ).hexdigest(),
                'storage': {
                    'source': 's3',
                    'bucket': BUCKET,
                    'endpoint': self.endpoint,
                    'region': REGION,
                },
                'identity': {
                    'kind': 'shared-credentials-file',
                    'principal': 'arn:aws:iam::role/' + tenant,
                    'credentialFile': str(
                        self.directory / (tenant + '.credentials')
                    ),
                    'profile': 'artifact-reader',
                },
            }
        value = {
            'namespaces': namespaces,
            'controlURL': self.control_url,
            'controlToken': self.control_token,
        }
        self.write_json('fixture.json', value)
        return str(self.directory / 'fixture.json')

    def close(self):
        if self.started:
            try:
                run('docker', 'rm', '--force', '--volumes', self.container)
            except subprocess.CalledProcessError:
                self.evidence['cleanupErrors'].append(
                    'could not remove disposable storage'
                )
        if self.server:
            self.server.shutdown()
            self.server.server_close()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--image', default='ghcr.io/chrislusf/seaweedfs:4.48')
    parser.add_argument('--storage-only', action='store_true')
    parser.add_argument('--evidence', type=Path)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[2]
    with tempfile.TemporaryDirectory(
        prefix='kfp-external-artifact-'
    ) as directory:
        fixture = ExternalStorage(Path(directory), args.image)
        try:
            fixture.start()
            fixture.verify_storage()
            manifest = fixture.manifest()
            if not args.storage_only:
                subprocess.run(
                    [
                        'npm',
                        'test',
                        '--',
                        'integration-tests/artifact-external-storage-live.test.ts',
                    ],
                    cwd=root / 'frontend/server',
                    env={
                        **os.environ,
                        'KFP_EXTERNAL_ARTIFACT_FIXTURE': manifest,
                    },
                    check=True,
                )
                fixture.evidence['checks'].append(
                    'KFP external-storage integration suite passed'
                )
            fixture.evidence['status'] = 'passed'
        finally:
            fixture.close()
            if args.evidence:
                args.evidence.write_text(json.dumps(fixture.evidence, indent=2))
            if fixture.evidence['cleanupErrors']:
                raise RuntimeError('External-storage fixture cleanup failed')
        print(
            'External storage: real OIDC/STS, existing roots, and tenant isolation passed.'
        )


if __name__ == '__main__':
    main()
