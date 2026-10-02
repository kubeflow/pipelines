#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
"""Hosted-only setup and evidence for a frontend-only upgrade/rollback
rehearsal."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import subprocess
import time

LEGACY_SHA = '02cbc725ac9ddcd950f4400d8355dd78bfcd6c57'
MANIFESTS_SHA = '88716b3f7f62b12f98d82bcfc59635bb07e7845c'
UI = 'ml-pipeline-ui'
SIGNING = 'ml-pipeline-ui-tensorboard-proxy'
MESH_DEPLOYMENTS = (
    'mysql',
    'seaweedfs',
    'ml-pipeline',
    UI,
    'ml-pipeline-persistenceagent',
    'ml-pipeline-scheduledworkflow',
    'ml-pipeline-viewer-crd',
)
PROFILES = {
    'kubeflow-user-example-com': 'user@example.com',
    'kfp-qualification-second': 'user@example.com',
    'kfp-qualification-other': 'other@example.com',
}


def require_hosted(env=None):
    env = os.environ if env is None else env
    if not (env.get('CI') == 'true' and env.get('GITHUB_ACTIONS') == 'true' and
            env.get('RUNNER_ENVIRONMENT') == 'github-hosted' and
            Path(env.get('RUNNER_TEMP', '')).is_absolute()):
        raise RuntimeError(
            'Deployment qualification requires a disposable GitHub-hosted runner'
        )


def command(*args, payload=None, timeout=300, raw=False):
    result = subprocess.run(
        args,
        input=payload,
        capture_output=True,
        text=not raw,
        timeout=timeout,
        check=False)
    if result.returncode:
        # Never echo input: kubectl apply can carry disposable authentication secrets.
        raise RuntimeError(
            f'{args[0]} {args[1]} failed ({result.returncode}): '
            f'{result.stderr[-2000:] if payload is None else "diagnostic omitted for an input-bearing operation"}'
        )
    return result.stdout


def kube(*args):
    return json.loads(command('kubectl', *args, '-o', 'json'))


def digest(value):
    return hashlib.sha256(
        json.dumps(value, sort_keys=True,
                   separators=(',', ':')).encode()).hexdigest()


def file_digest(path):
    value = hashlib.sha256()
    with open(path, 'rb') as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b''):
            value.update(chunk)
    return value.hexdigest()


def write_json(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2) + '\n')


def apply(objects):
    command(
        'kubectl',
        'apply',
        '-f',
        '-',
        payload=json.dumps({
            'apiVersion': 'v1',
            'kind': 'List',
            'items': objects,
        }))


def image_record(archive, local_tag, role, output):
    archive_sha = file_digest(archive)
    command('docker', 'load', '-i', str(archive), timeout=300)
    ref = f'localhost:5001/kfp-frontend:{role}'
    command('docker', 'tag', local_tag, ref)
    command('docker', 'push', ref, timeout=300)
    image = json.loads(command('docker', 'image', 'inspect', ref))[0]
    repo_digests = [
        value for value in image['RepoDigests']
        if value.startswith('localhost:5001/kfp-frontend@sha256:')
    ]
    if len(repo_digests) != 1:
        raise RuntimeError(
            'Expected exactly one published frontend manifest digest')
    immutable = repo_digests[0]
    if not re.fullmatch(r'localhost:5001/kfp-frontend@sha256:[0-9a-f]{64}',
                        immutable):
        raise RuntimeError('Invalid image digest')
    container = command('docker', 'create', immutable).strip()
    client = output / f'{role}-client'
    try:
        command(
            'docker', 'cp', f'{container}:/client', str(client), timeout=120)
    finally:
        command('docker', 'rm', container)
    assets = {
        path.relative_to(client).as_posix(): file_digest(path)
        for path in sorted(client.rglob('*'))
        if path.is_file()
    }
    if 'index.html' not in assets or not any(
            name.endswith('.js') for name in assets):
        raise RuntimeError('Frontend image has no production assets')
    write_json(output / f'{role}-assets.json', assets)
    return {
        'reference':
            immutable,
        'configId':
            image['Id'],
        'archiveSha256':
            archive_sha,
        'sourceSha':
            LEGACY_SHA if role == 'legacy' else os.environ['GITHUB_SHA'],
        'assetsManifestSha256':
            file_digest(output / f'{role}-assets.json')
    }


def setup_images(args):
    command('docker', 'run', '-d', '--restart=always', '-p',
            '127.0.0.1:5001:5000', '--name', 'kfp-qualification-registry',
            'registry:2.8.3')
    command('docker', 'network', 'connect', 'kind',
            'kfp-qualification-registry')
    hosts = ('server = "http://localhost:5001"\n'
             '[host."http://kfp-qualification-registry:5000"]\n'
             '  capabilities = ["pull", "resolve"]\n')
    for node in command('kind', 'get', 'nodes', '--name', 'kfp').splitlines():
        command('docker', 'exec', node, 'mkdir', '-p',
                '/etc/containerd/certs.d/localhost:5001')
        command(
            'docker',
            'exec',
            '-i',
            node,
            'tee',
            '/etc/containerd/certs.d/localhost:5001/hosts.toml',
            payload=hosts)
    output = Path(args.output)
    output.mkdir(parents=True, exist_ok=True)
    records = {
        'legacy':
            image_record(args.legacy_archive,
                         'kfp-frontend-legacy:qualification', 'legacy', output),
        'candidate':
            image_record(args.candidate_archive,
                         'kind-registry:5000/frontend:latest', 'candidate',
                         output),
    }
    write_json(output / 'images.json', records)
    return records


def render_upstream(path):
    import yaml
    source = command(
        'kubectl', 'kustomize',
        f'https://github.com/kubeflow/manifests/{path}?ref={MANIFESTS_SHA}')
    return [item for item in yaml.safe_load_all(source) if item]


def mesh_pod_evidence(pod):
    initializers = pod['spec'].get('initContainers', [])
    names = [item['name'] for item in initializers]
    if 'istio-proxy' not in names:
        raise AssertionError(
            'Qualification workload has no native mesh sidecar')
    position = names.index('istio-proxy')
    if initializers[position].get('restartPolicy') != 'Always':
        raise AssertionError(
            'Mesh sidecar is not a native restartable initializer')
    if any(index < position and name.startswith('wait-for-')
           for index, name in enumerate(names)):
        raise AssertionError(
            'Mesh sidecar must start before network wait initializers')
    statuses = pod['status'].get('initContainerStatuses', [])
    proxy = next((item for item in statuses if item['name'] == 'istio-proxy'),
                 {})
    if not proxy.get('ready') or not proxy.get('started') or not proxy.get(
            'imageID'):
        raise AssertionError('Native mesh sidecar is not started and ready')
    applications = pod['status'].get('containerStatuses', [])
    if not applications or not all(item['ready'] for item in applications):
        raise AssertionError('Mesh workload application is not ready')
    return {
        'uid': pod['metadata']['uid'],
        'serviceAccount': pod['spec']['serviceAccountName'],
        'initOrder': names,
        'proxyImageId': proxy['imageID'],
        'proxyReady': True
    }


def verify_mesh_image(pod, expected):
    evidence = mesh_pod_evidence(pod)
    if evidence['proxyImageId'] not in {
            item['proxyImageId'] for item in expected
    }:
        raise AssertionError(
            'Mesh sidecar image changed during UI qualification')
    return evidence


def ingress_ui_policy():
    return {
        'apiVersion': 'networking.k8s.io/v1',
        'kind': 'NetworkPolicy',
        'metadata': {
            'name': 'qualification-ingress-to-pipeline-ui',
            'namespace': 'kubeflow'
        },
        'spec': {
            'podSelector': {
                'matchLabels': {
                    'app': UI
                }
            },
            'policyTypes': ['Ingress'],
            'ingress': [{
                'from': [{
                    'namespaceSelector': {
                        'matchLabels': {
                            'kubernetes.io/metadata.name': 'istio-system'
                        }
                    },
                    'podSelector': {
                        'matchLabels': {
                            'app': 'istio-ingressgateway'
                        }
                    },
                }],
                'ports': [{
                    'protocol': 'TCP',
                    'port': 3000
                }]
            }],
        },
    }


def setup_mesh(args):
    # The shared test deploy intentionally omits workload injection. The real
    # ingress path requires the repository's existing ISTIO_MUTUAL destinations.
    # Include storage destinations and the API's named backend callers before
    # capturing the baseline; preserve these mesh workloads across UI changes.
    # Istio 1.30.0 reorders native sidecars before application wait initializers;
    # verify the admitted pods as well, avoiding init-container startup deadlocks.
    # Preserve the intended path even on a policy-enforcing CNI. This Kind lane
    # does not itself establish NetworkPolicy enforcement.
    apply([ingress_ui_policy()])
    patch = {
        'spec': {
            'template': {
                'metadata': {
                    'labels': {
                        'sidecar.istio.io/inject': 'true'
                    },
                    'annotations': {
                        'sidecar.istio.io/nativeSidecar': 'true'
                    },
                }
            }
        }
    }
    for deployment in MESH_DEPLOYMENTS:
        command('kubectl', '-n', 'kubeflow', 'patch', 'deployment', deployment,
                '--type=merge', '--patch', json.dumps(patch))
    evidence = {}
    for deployment in MESH_DEPLOYMENTS:
        command(
            'kubectl',
            '-n',
            'kubeflow',
            'rollout',
            'status',
            f'deployment/{deployment}',
            '--timeout=300s',
            timeout=320)
        labels = kube('-n', 'kubeflow', 'get', 'deployment',
                      deployment)['spec']['selector']['matchLabels']
        selector = ','.join(
            f'{key}={value}' for key, value in sorted(labels.items()))
        pods = kube('-n', 'kubeflow', 'get', 'pods', '-l', selector)['items']
        active = [
            pod for pod in pods if not pod['metadata'].get('deletionTimestamp')
        ]
        if not active:
            raise AssertionError(f'No ready mesh workload pods: {deployment}')
        evidence[deployment] = [mesh_pod_evidence(pod) for pod in active]
        write_json(Path(args.output) / 'mesh-readiness.json', evidence)


def pipeline_create_allowed(namespace, user):
    result = subprocess.run([
        'kubectl', 'auth', 'can-i', 'create',
        'pipelines.pipelines.kubeflow.org', '-n', namespace, '--as', user
    ],
                            capture_output=True,
                            text=True,
                            timeout=30)
    answer = result.stdout.strip()
    if (result.returncode, answer) not in ((0, 'yes'), (1, 'no')):
        raise RuntimeError(
            'Could not verify namespace pipeline creation authorization')
    return answer == 'yes'


def profile_authorization_state(output, owner_access):
    # Preserve only RBAC and synthetic profile identities, never credentials.
    roles = kube('get', 'clusterroles')['items']
    evidence = {
        'ownerCreateAllowed': owner_access,
        'roles': {
            item['metadata']['name']: {
                'labels': item['metadata'].get('labels', {}),
                'aggregationRule': item.get('aggregationRule'),
                'rules': item.get('rules', [])
            }
            for item in roles
            if item['metadata']['name'].startswith(('kubeflow-',
                                                    'aggregate-to-kubeflow-'))
        },
        'profiles': {},
    }
    for namespace in PROFILES:
        profile = kube('get', 'profile', namespace)
        bindings = kube('-n', namespace, 'get', 'rolebindings')['items']
        evidence['profiles'][namespace] = {
            'owner':
                profile['spec']['owner'],
            'namespaceOwner':
                kube('get', 'namespace', namespace)
                ['metadata'].get('annotations', {}).get('owner'),
            'status':
                profile.get('status'),
            'bindings': {
                item['metadata']['name']: {
                    'roleRef': item['roleRef'],
                    'subjects': item.get('subjects', [])
                } for item in bindings
            },
        }
    try:
        logs = command(
            'kubectl',
            '-n',
            'kubeflow',
            'logs',
            'deployment/profiles-deployment',
            '-c',
            'manager',
            '--tail=100',
            timeout=15)
        evidence['profileControllerErrors'] = [
            line[:1500]
            for line in logs.splitlines()
            if re.search(r'\b(error|failed|forbidden)\b', line, re.IGNORECASE)
        ][-20:]
    except (RuntimeError, subprocess.TimeoutExpired):
        evidence['profileControllerErrorsUnavailable'] = True
    write_json(Path(output) / 'profile-authorization-state.json', evidence)


def verify_profile_authorization(output):
    deadline = time.monotonic() + 120
    while True:
        owner_access = {
            namespace: pipeline_create_allowed(namespace, user)
            for namespace, user in PROFILES.items()
        }
        if all(owner_access.values()):
            break
        if time.monotonic() >= deadline:
            profile_authorization_state(output, owner_access)
            raise AssertionError(
                'Profile owners lack namespace pipeline creation permission')
        time.sleep(2)
    evidence = []
    for namespace, owner in PROFILES.items():
        binding = kube('-n', namespace, 'get', 'rolebinding', 'namespaceAdmin')
        if binding['roleRef'] != {
                'apiGroup': 'rbac.authorization.k8s.io',
                'kind': 'ClusterRole',
                'name': 'kubeflow-admin'
        } or not any(
                subject.get('kind') == 'User' and subject.get('name') == owner
                for subject in binding['subjects']):
            raise AssertionError(
                'Profile owner does not have its canonical namespace role binding'
            )
        for user in sorted(set(PROFILES.values())):
            allowed = pipeline_create_allowed(namespace, user)
            if allowed != (user == owner):
                raise AssertionError(
                    'Pipeline creation authorization crossed a profile boundary'
                )
            evidence.append({
                'namespace': namespace,
                'user': user,
                'allowed': allowed
            })
    profile_authorization_state(output, owner_access)
    write_json(Path(output) / 'profile-authorization.json', evidence)


def setup_auth(args):
    import bcrypt
    import yaml
    password = secrets.token_urlsafe(24)
    print(f'::add-mask::{password}', flush=True)
    auth_file = Path(
        os.environ['RUNNER_TEMP']) / 'kfp-qualification-credentials.json'
    auth_file.write_text(
        json.dumps({
            'password': password,
            'users': list(set(PROFILES.values()))
        }))
    auth_file.chmod(0o600)
    # Use upstream ExtAuthz, OIDC and Dashboard, not an identity-header fixture.
    # Profile owners bind kubeflow-admin; install its canonical aggregation chain
    # rather than the shared test deployment's edit-only role fixture.
    apply(render_upstream('common/kubeflow-roles/base'))
    apply(render_upstream('common/istio/istio-install/overlays/oauth2-proxy'))
    apply(render_upstream('common/istio/kubeflow-istio-resources/base'))
    oauth = render_upstream('common/oauth2-proxy/overlays/m2m-dex-only')
    dex = render_upstream('common/dex/overlays/oauth2-proxy')
    client_secret = secrets.token_urlsafe(32)
    for obj in oauth:
        if obj['kind'] == 'Secret' and obj['metadata']['name'].startswith(
                'oauth2-proxy'):
            import base64
            for key, value in {
                    'client-secret': client_secret,
                    'cookie-secret': secrets.token_urlsafe(24)
            }.items():
                obj.setdefault('data', {})[key] = base64.b64encode(
                    value.encode()).decode()
    for obj in dex:
        if obj['kind'] == 'ConfigMap' and obj['metadata']['name'] == 'dex':
            config = yaml.safe_load(obj['data']['config.yaml'])
            config['staticPasswords'] = [{
                'email':
                    email,
                'username':
                    email.split('@')[0],
                'userID':
                    str(index + 1000),
                'hash':
                    bcrypt.hashpw(password.encode(), bcrypt.gensalt()).decode()
            } for index, email in enumerate(sorted(set(PROFILES.values())))]
            for client in config['staticClients']:
                client.pop('secretEnv', None)
                client['secret'] = client_secret
            obj['data']['config.yaml'] = yaml.safe_dump(config)
    apply(dex)
    apply(oauth)
    apply(
        render_upstream(
            'applications/dashboard/upstream/centraldashboard/overlays/istio'))
    apply([{
        'apiVersion': 'kubeflow.org/v1beta1',
        'kind': 'Profile',
        'metadata': {
            'name': namespace
        },
        'spec': {
            'owner': {
                'kind': 'User',
                'name': email
            }
        }
    } for namespace, email in PROFILES.items()])
    for namespace, deployment in [('istio-system', 'istiod'), ('auth', 'dex'),
                                  ('oauth2-proxy', 'oauth2-proxy'),
                                  ('kubeflow', 'dashboard')]:
        command(
            'kubectl',
            '-n',
            namespace,
            'rollout',
            'status',
            f'deployment/{deployment}',
            '--timeout=300s',
            timeout=320)
    deadline = time.monotonic() + 300
    while True:
        ready = True
        for namespace in PROFILES:
            result = subprocess.run([
                'kubectl', '-n', namespace, 'get', 'secret',
                'mlpipeline-minio-artifact'
            ],
                                    capture_output=True)
            ready = ready and result.returncode == 0
        if ready:
            break
        if time.monotonic() > deadline:
            raise RuntimeError('Qualification profiles were not reconciled')
        time.sleep(3)
    for namespace in PROFILES:
        command(
            'kubectl',
            '-n',
            namespace,
            'rollout',
            'status',
            'deployment/ml-pipeline-ui-artifact',
            '--timeout=300s',
            timeout=320)
    verify_profile_authorization(args.output)
    setup_mesh(args)
    write_json(
        Path(args.output) / 'authentication.json', {
            'manifestsSourceSha': MANIFESTS_SHA,
            'provider': 'Dex and OAuth2 Proxy',
            'dashboard': 'ghcr.io/kubeflow/dashboard/dashboard:v2.0.0',
            'profiles': PROFILES,
            'credentialFileUploaded': False,
        })


def normalize_resources(items):
    result = {}
    for item in items:
        kind = item['kind']
        metadata = item['metadata']
        name = metadata['name']
        if name == UI and kind == 'Deployment':
            continue
        key = '/'.join([kind, metadata.get('namespace', ''), name])
        if kind in ('Deployment', 'StatefulSet'):
            value = item['spec']['template']
        elif kind == 'ConfigMap':
            # Kubernetes automatically manages this cluster CA projection.
            if name == 'kube-root-ca.crt':
                continue
            value = {
                'data': item.get('data', {}),
                'binaryData': item.get('binaryData', {})
            }
        elif kind in ('Role', 'ClusterRole'):
            value = item.get('rules', [])
        elif kind in ('RoleBinding', 'ClusterRoleBinding'):
            value = {
                'roleRef': item['roleRef'],
                'subjects': item.get('subjects', [])
            }
        elif kind == 'ServiceAccount':
            value = {
                key: item[key]
                for key in ('automountServiceAccountToken', 'imagePullSecrets')
                if key in item
            }
        else:
            value = item.get('spec', {})
        result[key] = {'uid': metadata['uid'], 'contentSha256': digest(value)}
    return result


def snapshot(args):
    namespaces = ['kubeflow'] + (
        list(PROFILES) + ['auth', 'oauth2-proxy', 'istio-system']
        if args.mode == 'multiuser' else [])
    resources = []
    pods = {}
    mesh = {}
    mesh_seed = json.loads((
        Path(args.output) /
        'mesh-readiness.json').read_text()) if args.mode == 'multiuser' else {}
    for namespace in namespaces:
        resources += kube(
            '-n', namespace, 'get',
            'deployments,statefulsets,configmaps,serviceaccounts,roles,rolebindings,networkpolicies'
        )['items']
        for pod in kube('-n', namespace, 'get', 'pods')['items']:
            meta = pod['metadata']
            # Workflow/Job pods are expected to come and go. Existing long-running
            # controllers and services must not be replaced by a UI-only operation.
            owners = meta.get('ownerReferences', [])
            if meta.get('deletionTimestamp'):
                continue
            app = meta.get('labels', {}).get('app')
            if namespace == 'kubeflow' and app in mesh_seed:
                mesh[f'{namespace}/{meta["name"]}'] = verify_mesh_image(
                    pod, mesh_seed[app])
            if app == UI:
                continue
            if not any(owner['kind'] in ('ReplicaSet', 'StatefulSet')
                       for owner in owners):
                continue
            if meta.get('deletionTimestamp'):
                continue
            pods[f'{namespace}/{meta["name"]}'] = meta['uid']
    resources += [
        item
        for item in kube('get', 'clusterroles,clusterrolebindings')['items']
        if any(token in item['metadata']['name'].lower()
               for token in ('pipeline', 'kubeflow', 'profile', 'dashboard',
                             'istio', 'dex', 'oauth'))
    ]
    if args.mode == 'multiuser':
        resources += kube(
            'get',
            'authorizationpolicies,requestauthentications,virtualservices,destinationrules',
            '-A')['items']
    signing = kube('-n', 'kubeflow', 'get', 'secret', SIGNING)
    # This private file is outside uploaded reports. Only compare outcomes leave the runner.
    authentication_secrets = {}
    if args.mode == 'multiuser':
        for namespace in ['auth', 'oauth2-proxy']:
            for secret in kube('-n', namespace, 'get', 'secrets')['items']:
                if secret.get('type') == 'kubernetes.io/service-account-token':
                    continue
                authentication_secrets[
                    f'{namespace}/{secret["metadata"]["name"]}'] = {
                        'uid': secret['metadata']['uid'],
                        'dataHash': digest(secret.get('data', {}))
                    }
    state = {
        'resources': normalize_resources(resources),
        'pods': pods,
        'authenticationSecrets': authentication_secrets,
        'signingUid': signing['metadata']['uid'],
        'signingHash': digest(signing.get('data', {}))
    }
    if not signing.get('data', {}).get('signing-secret'):
        raise RuntimeError('TensorBoard signing Secret is empty')
    target = Path(
        os.environ['RUNNER_TEMP']) / f'kfp-deployment-{args.phase}-state.json'
    write_json(target, state)
    target.chmod(0o600)
    if args.phase != 'baseline':
        baseline = json.loads(
            (target.parent / 'kfp-deployment-baseline-state.json').read_text())
        assert_preserved(baseline, state)
    write_json(
        Path(args.output) / f'{args.phase}-invariants.json', {
            'phase':
                args.phase,
            'resources':
                state['resources'],
            'backendPodUids':
                pods,
            'readyMeshPods':
                mesh,
            'signingSecretPreserved':
                args.phase != 'baseline',
            'signingSecretPresent':
                True,
            'preserved':
                True,
            'authenticationSecretsPreserved':
                args.phase != 'baseline' if args.mode == 'multiuser' else None,
        })


def assert_preserved(before, after):
    for name in ('resources', 'pods', 'signingUid', 'signingHash',
                 'authenticationSecrets'):
        if before[name] != after[name]:
            raise AssertionError(f'UI-only rollback invariant changed: {name}')


def runtime_manifest(node, image_digest, output):
    if not re.fullmatch(r'sha256:[0-9a-f]{64}', image_digest):
        raise AssertionError('Runtime manifest address must be a SHA256 digest')
    raw = command(
        'docker',
        'exec',
        node,
        'ctr',
        '-n',
        'k8s.io',
        'content',
        'get',
        image_digest,
        raw=True)
    directory = Path(output) / 'runtime-manifests'
    directory.mkdir(parents=True, exist_ok=True)
    # Persist the exact returned bytes before validation, including failed proofs.
    (directory /
     f'{image_digest.removeprefix("sha256:")}.json').write_bytes(raw)
    return raw


def verify_image_identity(status, expected, read_manifest):
    if not status.get('ready'):
        raise AssertionError('Qualified UI container is not ready')
    reported = status['imageID']
    manifest = expected['reference'].split('@')[1]
    if reported.endswith(manifest):
        return {'method': 'manifest-digest', 'reportedImageId': reported}
    if reported == expected[
            'configId'] or reported == 'containerd://' + expected['configId']:
        return {'method': 'config-digest', 'reportedImageId': reported}
    # containerd may retain an import alias after CRI removes that image name.
    # Read the actual reported digest from the node's content store; never infer
    # identity from the requested tag or from another registered image alias.
    image = reported.removeprefix('docker-pullable://').removeprefix(
        'containerd://')
    reported_digest = image.rsplit('@', 1)[-1]
    if not re.fullmatch(r'sha256:[0-9a-f]{64}', reported_digest):
        raise AssertionError(
            'Running UI image has no resolvable manifest digest')

    def verified_content(address):
        if not re.fullmatch(r'sha256:[0-9a-f]{64}', address):
            raise AssertionError(
                'Runtime content address is not a SHA256 digest')
        raw = read_manifest(address)
        if not isinstance(raw, bytes) or 'sha256:' + hashlib.sha256(
                raw).hexdigest() != address:
            raise AssertionError(
                'Runtime manifest content does not match its digest')
        content = json.loads(raw)
        if content.get('schemaVersion') != 2:
            raise AssertionError('Runtime image content is not schema 2')
        return raw, content

    raw, content = verified_content(reported_digest)
    resolved_digest = reported_digest
    image_types = ('application/vnd.docker.distribution.manifest.v2+json',
                   'application/vnd.oci.image.manifest.v1+json')
    if content.get('mediaType') in (
            'application/vnd.oci.image.index.v1+json',
            'application/vnd.docker.distribution.manifest.list.v2+json'):
        # Docker save/load may register an index with latest/ci descriptors for
        # the same image. Only one distinct child is unambiguous; never guess a
        # platform or accept an index itself as proof of a running image config.
        descriptors = content.get('manifests', [])
        identities = {(item.get('digest'), item.get('size'),
                       item.get('mediaType')) for item in descriptors}
        if len(identities) != 1:
            raise AssertionError(
                'Runtime image index does not identify one unique image')
        child_digest, child_size, child_type = identities.pop()
        if child_type not in image_types or not isinstance(
                child_size, int) or child_size <= 0:
            raise AssertionError('Runtime image index descriptor is invalid')
        raw, content = verified_content(child_digest)
        if len(raw) != child_size or content.get('mediaType') != child_type:
            raise AssertionError(
                'Runtime child manifest does not match its index descriptor')
        resolved_digest = child_digest
    if content.get('mediaType') not in image_types:
        raise AssertionError(
            'Runtime image content is not a schema-2 image manifest')
    resolved = content.get('config', {}).get('digest')
    if resolved != expected['configId']:
        raise AssertionError(
            'Running UI image identity differs from the qualified immutable image'
        )
    return {
        'method': 'resolved-runtime-manifest-config-digest',
        'reportedImageId': reported,
        'verifiedManifestDigest': reported_digest,
        'resolvedManifestDigest': resolved_digest,
        'resolvedConfigId': resolved
    }


def swap(args):
    output = Path(args.output)
    images = json.loads((output / 'images.json').read_text())
    phase = args.phase
    template_path = Path(
        os.environ['RUNNER_TEMP']) / 'kfp-legacy-ui-template.json'
    deployment = kube('-n', 'kubeflow', 'get', 'deployment', UI)
    if phase == 'rollback':
        template = json.loads(template_path.read_text())
    else:
        template = deployment['spec']['template']
        container = next(item for item in template['spec']['containers']
                         if item['name'] == UI)
        container['image'] = images['legacy' if phase ==
                                    'baseline' else 'candidate']['reference']
        if phase == 'baseline':
            write_json(template_path, template)
    started = time.monotonic()
    command('kubectl', '-n', 'kubeflow', 'patch', 'deployment', UI,
            '--type=merge', '--patch',
            json.dumps({'spec': {
                'template': template
            }}))
    command(
        'kubectl',
        '-n',
        'kubeflow',
        'rollout',
        'status',
        f'deployment/{UI}',
        '--timeout=300s',
        timeout=320)
    current = kube('-n', 'kubeflow', 'get', 'deployment', UI)
    if current['spec']['template'] != template:
        raise AssertionError(
            'Restored pod template does not match the captured template')
    pods = kube('-n', 'kubeflow', 'get', 'pods', '-l', f'app={UI}')['items']
    running = [
        pod for pod in pods if not pod['metadata'].get('deletionTimestamp')
    ]
    if not running or not all(pod['status'].get('containerStatuses')
                              for pod in running):
        raise AssertionError('UI has no running image identity')
    expected = images['candidate' if phase == 'candidate' else 'legacy']
    identity_evidence = {
        'expectedReference': expected['reference'],
        'expectedConfigId': expected['configId'],
        'pods': [],
    }
    identity_path = output / f'{phase}-image-identity.json'
    for pod in running:
        status = next(item for item in pod['status']['containerStatuses']
                      if item['name'] == UI)
        evidence = {
            'uid': pod['metadata']['uid'],
            'node': pod['spec']['nodeName'],
            'imageId': status['imageID'],
            'ready': status['ready']
        }
        identity_evidence['pods'].append(evidence)
        write_json(identity_path, identity_evidence)
        if args.mode == 'multiuser':
            baseline_mesh = json.loads(
                (output / 'mesh-readiness.json').read_text())
            evidence['mesh'] = verify_mesh_image(pod, baseline_mesh[UI])
        evidence['verification'] = verify_image_identity(
            status, expected, lambda image: runtime_manifest(
                pod['spec']['nodeName'], image, output))
        write_json(identity_path, identity_evidence)
    write_json(
        output / f'{phase}-rollout.json', {
            'phase':
                phase,
            'strategy':
                current['spec']['strategy'],
            'elapsedMs': (time.monotonic() - started) * 1000,
            'podTemplateSha256':
                digest(template),
            'image':
                next(item['image']
                     for item in template['spec']['containers']
                     if item['name'] == UI),
            'pods': [{
                'uid':
                    pod['metadata']['uid'],
                'containers': [{
                    'name': item['name'],
                    'imageID': item['imageID'],
                    'ready': item['ready']
                } for item in pod['status']['containerStatuses']]
            } for pod in running],
        })


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        'operation', choices=['images', 'auth', 'snapshot', 'swap'])
    parser.add_argument('--output', required=True)
    parser.add_argument(
        '--mode', choices=['standalone', 'multiuser'], default='standalone')
    parser.add_argument(
        '--phase',
        choices=['baseline', 'candidate', 'rollback'],
        default='baseline')
    parser.add_argument('--legacy-archive')
    parser.add_argument('--candidate-archive')
    args = parser.parse_args()
    require_hosted()
    {
        'images': setup_images,
        'auth': setup_auth,
        'snapshot': snapshot,
        'swap': swap
    }[args.operation](
        args)


if __name__ == '__main__':
    main()
