#!/usr/bin/env python3
"""Disposable provisioning candidate. No action occurs on import or --help."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tarfile
import zipfile
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

SOURCE = '6bea51e2003f9072a55068fb85e03cdb9391435d'
HEAD = 'f6c500fa138bfc506856006ded58a211254531d5'
UPSTREAM = '88716b3f7f62b12f98d82bcfc59635bb07e7845c'
COMPONENTS = ('apiserver', 'persistenceagent', 'scheduledworkflow', 'frontend', 'viewer-crd-controller', 'driver', 'launcher')
FIXTURES = ('flip_coin', 'collected_parameters', 'if_else_with_oneof_artifacts')



class CommandFailure(RuntimeError):
    """Only allowlisted command/error categories, never child output."""
    def __init__(self, args, status, stderr=b''):
        executable = Path(args[0]).name
        if executable not in ('kubectl', 'docker', 'kind', 'git', 'tar'):
            executable = 'subprocess'
        operation = next((a for a in args[1:] if a in ('kustomize', 'apply', 'wait', 'rollout', 'exec', 'load', 'inspect', 'create', 'get', 'archive')), 'command')
        category = 'unclassified'
        for pattern, label in ((b'not in or below', 'kustomize-load-restriction'),
                               (b'no matches for kind', 'resource-kind-unavailable'),
                               (b'Forbidden', 'authorization-denied'),
                               (b'connection refused', 'connection-refused'),
                               (b'no such file', 'missing-file'),
                               (b'timed out', 'timeout')):
            if pattern in stderr:
                category = label
                break
        super().__init__(f'{executable} {operation}: exit={status}, category={category}')


def call(args, data=None, timeout=300):
    # Never propagate child output/error into exception messages or console logs.
    try:
        result = subprocess.run(args, input=data, stdout=subprocess.PIPE,
                                stderr=subprocess.PIPE, timeout=timeout, check=False)
    except subprocess.TimeoutExpired:
        raise CommandFailure(args, 'timeout') from None
    if result.returncode:
        raise CommandFailure(args, result.returncode, result.stderr)
    return result.stdout


def save(path, value):
    path = Path(path)
    tmp = path.with_suffix(path.suffix + '.tmp')
    tmp.write_text(json.dumps(value, indent=2) + '\n')
    tmp.chmod(0o600)
    tmp.replace(path)


def sha(path):
    with open(path, 'rb') as src:
        return hashlib.file_digest(src, 'sha256').hexdigest()


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise urllib.error.URLError('redirect refused')


def verify_image(image):
    """Bind saved GitHub producer metadata, ZIP, Docker tar and config digest.

    Metadata files must be fetched through authenticated gh API by the operator;
    offline validation cannot authenticate a fabricated metadata file.
    """
    artifact = json.loads(Path(image['artifact_metadata']).read_text())
    run = json.loads(Path(image['producer_run_metadata']).read_text())
    job = json.loads(Path(image['producer_job_metadata']).read_text())
    log = Path(image['producer_job_log']).read_text()
    commit = json.loads(Path(image['source_commit_metadata']).read_text())
    steps = {step['name']: step.get('conclusion') for step in job['steps']}
    if (run.get('head_sha') != HEAD
            or not re.fullmatch(r'\.github/workflows/[a-zA-Z0-9_-]+\.ya?ml', run.get('path', ''))
            or run.get('repository', {}).get('full_name') != 'kubeflow/pipelines'
            or artifact.get('workflow_run', {}).get('id') != run.get('id')
            or artifact.get('workflow_run', {}).get('head_sha') != HEAD
            or artifact.get('expired') is not False
            or job.get('run_id') != run.get('id') or job.get('conclusion') != 'success'
            or f"image-build ({image['component']}," not in job.get('name', '')
            or artifact.get('name') not in (image['component'], f"ci-image-artifact-v2-{image['component']}-amd64-{commit.get('tree', {}).get('sha')}")
            or commit.get('sha') != SOURCE
            or HEAD not in [parent['sha'] for parent in commit.get('parents', [])]
            or f'image-builds.yml@refs/pull/14818/merge ({SOURCE})' not in log
            or not re.search(r'Z ' + SOURCE + r'\r?$', log, re.MULTILINE)
            or f"actions/runs/{run['id']}/artifacts/{artifact['id']}" not in log
            or steps.get('Upload artifact') != 'success'):
        raise RuntimeError('verified exact-source producer metadata required')
    # Current comparison's seven producer jobs built from this exact merge tree.
    # A retained-cache consumer requires tracing its source artifact separately;
    # never substitute its current run head for its actual producer provenance.
    if (steps.get('Validate retained image') == 'success'
            or not any(steps.get(name) == 'success' for name in
                       ('Build and save Docker image', 'Rebuild Images in case of failure'))):
        raise RuntimeError('fresh build producer required; trace retained cache to its producer')
    if artifact.get('digest') != 'sha256:' + sha(image['artifact_zip']):
        raise RuntimeError('GitHub artifact ZIP digest mismatch or unavailable')
    with zipfile.ZipFile(image['artifact_zip']) as archive:
        matches = [m for m in archive.infolist() if m.filename == image['archive_member']]
        if len(matches) != 1:
            raise RuntimeError('unique artifact Docker tar member required')
        with archive.open(matches[0]) as member:
            if hashlib.file_digest(member, 'sha256').hexdigest() != image['archive_sha256']:
                raise RuntimeError('artifact ZIP does not contain supplied Docker tar')
    with zipfile.ZipFile(image['artifact_zip']) as zipped, zipped.open(image['archive_member']) as saved, tarfile.open(fileobj=saved) as archive:
        def member(name):
            matches = [m for m in archive.getmembers() if m.name == name]
            if len(matches) != 1 or not matches[0].isfile() or matches[0].size > 4 * 1024 * 1024:
                raise RuntimeError('unique bounded regular Docker metadata member required')
            return archive.extractfile(matches[0]).read()
        manifest = json.loads(member('manifest.json'))
        if len(manifest) != 1:
            raise RuntimeError('single-image Docker save archive required')
        config = member(manifest[0]['Config'])
        if 'sha256:' + hashlib.sha256(config).hexdigest() != image['image_id']:
            raise RuntimeError('tar config does not match expected image ID; existing Docker images cannot satisfy this check')
        if json.loads(config).get('architecture') != 'amd64':
            raise RuntimeError('native amd64 experiment image required')



def validate(c):
    if c['source_sha'] != SOURCE or not re.fullmatch(r'kfp-apf-representative-[a-z0-9-]{1,24}', c['cluster']):
        raise RuntimeError('wrong source or non-disposable cluster name')
    if not re.fullmatch(r'kindest/node@sha256:[0-9a-f]{64}', c['node_image']):
        raise RuntimeError('node image must have explicit immutable digest')
    if set(c['images']) != set(COMPONENTS) or set(c['fixtures']) != set(FIXTURES):
        raise RuntimeError('all seven images and three fixture identities required')
    for image in c['images'].values():
        if image['source_sha'] != SOURCE or not re.fullmatch(r'sha256:[0-9a-f]{64}', image['image_id']):
            raise RuntimeError('image source/image ID invalid')
        if sha(image['archive']) != image['archive_sha256']:
            raise RuntimeError('image archive digest mismatch')
        verify_image(image)
    for fixture in c['fixtures'].values():
        if fixture['source_sha'] != SOURCE or sha(fixture['path']) != fixture['sha256']:
            raise RuntimeError('fixture source/hash mismatch')
    if c.get('qps') not in (20, -1):
        raise RuntimeError('paired experiment requires QPS 20 or -1')


class Provision:
    def __init__(self, config, state):
        self.c, self.d = config, Path(state).resolve()
        self.context = 'kind-' + config['cluster']
        self.kubeconfig = self.d / 'kubeconfig'
        self.marker = self.d / 'owner.json'
        self.snapshot = self.d / 'source'

    def k(self, *args, data=None, timeout=300):
        return call(['kubectl', '--kubeconfig', str(self.kubeconfig), '--context', self.context,
                     '--request-timeout=30s', *args], data, timeout)

    def guard(self):
        marker = json.loads(self.marker.read_text())
        if marker['cluster'] != self.c['cluster'] or marker['source_sha'] != SOURCE:
            raise RuntimeError('ownership marker mismatch')
        actual = self.k('get', 'namespace', 'kube-system', '-o', 'jsonpath={.metadata.uid}').decode()
        if not marker.get('namespace_uid') or actual != marker['namespace_uid']:
            raise RuntimeError('cluster UID mismatch; refusing existing/unowned cluster')

    def apply(self, objects):
        for obj in objects:
            self.k('apply', '-f', '-', data=json.dumps(obj).encode())

    def create(self):
        validate(self.c)
        if self.d.exists():
            raise RuntimeError('state directory must be new; no implicit resume')
        if self.c['cluster'] in call(['kind', 'get', 'clusters']).decode().split():
            raise RuntimeError('cluster already exists; never adopt')
        self.d.mkdir(mode=0o700, parents=True)
        save(self.marker, {'cluster': self.c['cluster'], 'source_sha': SOURCE, 'creation_started': True})
        self.snapshot.mkdir()
        archive = self.d / 'source.tar'
        # Stream the monorepo snapshot to disk; capturing it in call() doubles
        # archive memory on a small hosted runner before Kind starts.
        with archive.open('wb') as out:
            result = subprocess.run(['git', '-C', self.c['source_worktree'], 'archive', SOURCE],
                                    stdout=out, stderr=subprocess.DEVNULL, timeout=300, check=False)
        if result.returncode:
            raise RuntimeError('source archive failed; pinned merge commit must exist locally')
        call(['tar', '-xf', str(archive), '-C', str(self.snapshot)])
        archive.unlink()
        # Separate constrained scenario only: preserve all bootstrap APF policies.
        if self.c.get('scenario_id')!='constrained-stock-apf-bulk-list-v2':
            raise RuntimeError('explicit constrained scenario identity required')
        kind_config=self.d/'kind-config.json'
        save(kind_config, {'kind':'Cluster','apiVersion':'kind.x-k8s.io/v1alpha4','nodes':[{'role':'control-plane','kubeadmConfigPatches':['kind: ClusterConfiguration\napiServer:\n  extraArgs:\n    - name: max-requests-inflight\n      value: "10"\n    - name: max-mutating-requests-inflight\n      value: "10"\n']}]})
        call(['kind', 'create', 'cluster', '--name', self.c['cluster'], '--image', self.c['node_image'],
              '--config',str(kind_config),'--kubeconfig', str(self.kubeconfig), '--wait', '180s'], timeout=300)
        uid = self.k('get', 'namespace', 'kube-system', '-o', 'jsonpath={.metadata.uid}').decode()
        save(self.marker, {'cluster': self.c['cluster'], 'source_sha': SOURCE, 'namespace_uid': uid})

    def deploy(self):
        import yaml
        validate(self.c)
        self.guard()
        for name, image in self.c['images'].items():
            call(['docker', 'load', '-i', image['archive']])
            found = json.loads(call(['docker', 'image', 'inspect', image['image_id']]))[0]['Id']
            if found != image['image_id']:
                raise RuntimeError('loaded image identity mismatch')
            call(['docker', 'tag', found, self.ref(name)])
            call(['kind', 'load', 'docker-image', '--name', self.c['cluster'], self.ref(name)])
        self.k('apply', '-k', str(self.snapshot / 'manifests/kustomize/cluster-scoped-resources'))
        self.apply([{'apiVersion': 'v1', 'kind': 'Namespace', 'metadata': {'name': 'kubeflow'}}])
        for path in ('common/istio/istio-crds/base', 'common/istio/istio-namespace/base', 'common/istio/istio-install/base'):
            self.k('apply', '-k', f'https://github.com/kubeflow/manifests/{path}?ref={UPSTREAM}')
        self.k('-n', 'istio-system', 'wait', 'pods', '--all', '--for=condition=Ready', '--timeout=300s', timeout=330)
        self.k('apply', '-f', str(self.snapshot / 'manifests/kustomize/third-party/metacontroller/base/crd.yaml'))
        self.k('wait', 'crd/compositecontrollers.metacontroller.k8s.io', '--for=condition=Established', '--timeout=60s')
        self.k('apply', '-k', f'https://github.com/kubeflow/manifests/applications/dashboard/upstream/profile-controller/overlays/kubeflow?ref={UPSTREAM}')
        rendered = self.k('kustomize', '--load-restrictor', 'LoadRestrictionsNone', str(self.snapshot / '.github/resources/manifests/multiuser/default'))
        objects = list(yaml.safe_load_all(rendered))
        replacements = {f'kind-registry:5000/{name}:latest': self.ref(name) for name in COMPONENTS}
        for obj in objects:
            template = obj.get('spec', {}).get('template', {}).get('spec', {})
            for container in template.get('containers', []) + template.get('initContainers', []):
                if container.get('image') in replacements:
                    container['image'] = replacements[container['image']]
                    container['imagePullPolicy'] = 'IfNotPresent'
                if obj.get('kind') == 'Deployment' and obj['metadata']['name'] == 'ml-pipeline' and container['name'] == 'ml-pipeline-api-server':
                    env = {e['name']: e for e in container.get('env', [])}
                    for key, val in {'CLIENTQPS': str(self.c['qps']), 'CLIENTBURST': '40', 'KFP_RUNTIME_PROFILE_TIMINGS': 'true',
                                     'V2_DRIVER_IMAGE': self.ref('driver'), 'V2_LAUNCHER_IMAGE': self.ref('launcher')}.items():
                        env[key] = {'name': key, 'value': val}
                    container['env'] = list(env.values())
                if obj.get('kind') == 'Deployment' and obj['metadata']['name'] == 'kubeflow-pipelines-profile-controller' and container['name'] == 'profile-controller':
                    env = {e['name']: e for e in container.get('env', [])}
                    for key, value in {'FRONTEND_IMAGE': 'kfp-apf-local/frontend', 'FRONTEND_TAG': SOURCE}.items():
                        env[key] = {'name': key, 'value': value}
                    container['env'] = list(env.values())
        self.apply(objects)
        # Every rendered controller is required, including DB/object store/Argo,
        # metacontroller and the KFP namespace profile hook.
        for obj in objects:
            if obj.get('kind') in ('Deployment', 'StatefulSet', 'DaemonSet'):
                meta = obj['metadata']
                self.k('-n', meta.get('namespace', 'kubeflow'), 'rollout', 'status',
                       obj['kind'].lower() + '/' + meta['name'], '--timeout=600s', timeout=630)
        for deployment in ('ml-pipeline', 'ml-pipeline-persistenceagent', 'ml-pipeline-scheduledworkflow'):
            self.k('-n', 'kubeflow', 'rollout', 'status', 'deployment/' + deployment, '--timeout=600s', timeout=630)
        self.k('-n', 'kubeflow', 'wait', 'pods', '-l', 'app.kubernetes.io/name=profile-controller', '--for=condition=Ready', '--timeout=300s', timeout=330)
        self.k('wait', 'crd/profiles.kubeflow.org', '--for=condition=Established', '--timeout=60s')
        self.wait_iam()
        save(self.d / 'deployment-ready.json', {'source_sha': SOURCE, 'iam_streak': 3})
        # Policies only, never Secrets or raw server logs.
        for resource in ('flowschemas', 'prioritylevelconfigurations'):
            save(self.d / (resource + '.json'), json.loads(self.k('get', resource, '-o', 'json')))
        save(self.d / 'image-identities.json', {'source_sha': SOURCE, 'images': {n: {'image_id': v['image_id'], 'ref': self.ref(n), 'archive_sha256': v['archive_sha256']} for n, v in self.c['images'].items()}})

    def wait_iam(self):
        # Same source repo contract: probe IAM DNS:8111 from the KFP profile
        # hook, requiring 3 consecutive successes before creating Profiles.
        # Do not invoke the helper's failure path, which collects raw logs.
        deadline, successes = time.monotonic() + 120, 0
        while time.monotonic() < deadline:
            try:
                self.k('-n', 'kubeflow', 'exec', 'deployment/kubeflow-pipelines-profile-controller',
                       '-c', 'profile-controller', '--', 'python', '-c',
                       'import socket; socket.create_connection(("seaweedfs.kubeflow", 8111), timeout=5).close()',
                       timeout=min(10, max(0.1, deadline - time.monotonic())))
                successes += 1
                if successes == 3:
                    return
            except RuntimeError:
                successes = 0
            time.sleep(min(5, max(0, deadline - time.monotonic())))
        raise RuntimeError('IAM service DNS not stable from KFP profile hook within 120s')

    def ref(self, name):
        return f'kfp-apf-local/{name}:{SOURCE}'

    def tenants(self):
        self.guard()
        if json.loads((self.d / 'deployment-ready.json').read_text()).get('source_sha') != SOURCE:
            raise RuntimeError('completed deployment readiness required')
        self.wait_iam()
        self.k('apply', '-f', str(self.snapshot / 'test_data/kubernetes/seaweedfs/kubeflow-edit-clusterrole.yaml'))
        names = [self.c['cluster'] + '-' + x for x in ('noisy', 'quiet1', 'quiet2', 'quiet3')]
        for ns in names:
            self.apply([{'apiVersion': 'kubeflow.org/v1beta1', 'kind': 'Profile', 'metadata': {'name': ns}, 'spec': {'owner': {'kind': 'User', 'name': ns + '@example.invalid'}}}])
            end = time.monotonic() + 300
            while True:
                try:
                    self.k('-n', ns, 'get', 'secret/mlpipeline-minio-artifact', '-o', 'name')
                    self.k('-n', ns, 'get', 'serviceaccount/default-editor', '-o', 'name')
                    break
                except RuntimeError:
                    if time.monotonic() >= end:
                        raise RuntimeError('Profile artifact secret/SA not ready; diagnose IAM without logging secrets') from None
                    time.sleep(5)
        self.apply([{'apiVersion': 'networking.k8s.io/v1', 'kind': 'NetworkPolicy', 'metadata': {'name': 'apf-tenant-access', 'namespace': 'kubeflow'},
                     'spec': {'podSelector': {}, 'policyTypes': ['Ingress'], 'ingress': [{'from': [{'namespaceSelector': {'matchExpressions': [{'key': 'kubernetes.io/metadata.name', 'operator': 'In', 'values': names}]}}]}, {'from': [{'podSelector': {}}]}]}}])
        # Namespaced uploads need no shared-pipeline RoleBinding in kubeflow.
        save(self.d / 'tenants.json', names)

    def upload(self):
        validate(self.c)
        self.guard()
        if (self.d / 'upload-started.json').exists():
            raise RuntimeError('upload previously attempted; reconcile IDs/ambiguous creates, never blind retry')
        save(self.d / 'upload-started.json', {'source_sha': SOURCE, 'started': True})
        # Port-forward PID stays owned by this invocation; endpoint never accepts arbitrary input.
        port = int(self.c.get('port', 18888))
        if not 1024 <= port <= 65535:
            raise RuntimeError('invalid local port')
        forward = subprocess.Popen(['kubectl', '--kubeconfig', str(self.kubeconfig), '--context', self.context, '-n', 'kubeflow', 'port-forward', '--address=127.0.0.1', 'svc/ml-pipeline', f'{port}:8888'], stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True)
        try:
            import select
            deadline = time.monotonic() + 30
            while True:
                if forward.poll() is not None:
                    raise RuntimeError('owned port-forward failed')
                remaining = deadline - time.monotonic()
                if remaining <= 0:
                    raise RuntimeError('owned endpoint readiness deadline')
                ready, _, _ = select.select([forward.stdout], [], [], min(remaining, 1))
                if ready and forward.stdout.readline().startswith(f'Forwarding from 127.0.0.1:{port} ->'):
                    break
            endpoint = f'http://127.0.0.1:{port}'
            tenants = []
            for index, ns in enumerate(json.loads((self.d / 'tenants.json').read_text())):
                token = self.k('-n', ns, 'create', 'token', 'default-editor', '--audience=pipelines.kubeflow.org', '--duration=1h').decode().strip()
                token_file = self.d / (ns + '.token')
                token_file.write_text(token)
                token_file.chmod(0o600)
                experiment = self.post(endpoint + '/apis/v2beta1/experiments', token, json.dumps({'display_name': ns, 'namespace': ns}).encode(), 'application/json')
                row = {'name': 'noisy' if index == 0 else f'quiet{index}', 'namespace': ns, 'service_account': 'default-editor', 'experiment_id': experiment['experiment_id'], 'token_file': str(token_file), 'fixtures': []}
                tenants.append(row)
                save(self.d / 'created-ids.json', tenants)
                for fixture in FIXTURES:
                    boundary = 'apf-' + uuid.uuid4().hex
                    data = Path(self.c['fixtures'][fixture]['path']).read_bytes()
                    body = (f'--{boundary}\r\nContent-Disposition: form-data; name="uploadfile"; filename="{fixture}.yaml"\r\nContent-Type: application/yaml\r\n\r\n'.encode() + data + f'\r\n--{boundary}--\r\n'.encode())
                    url = endpoint + '/apis/v2beta1/pipelines/upload?' + urllib.parse.urlencode({'name': ns + '-' + fixture, 'namespace': ns})
                    pipeline = self.post(url, token, body, 'multipart/form-data; boundary=' + boundary)
                    row['fixtures'].append({'name': fixture, 'pipeline_id': pipeline['pipeline_id']})
                    save(self.d / 'created-ids.json', tenants)
                    version = self.post(endpoint + '/apis/v2beta1/pipelines/upload_version?' + urllib.parse.urlencode({'pipelineid': pipeline['pipeline_id'], 'name': 'exact-source'}), token, body, 'multipart/form-data; boundary=' + boundary)
                    row['fixtures'][-1]['version_id'] = version['pipeline_version_id']
                    save(self.d / 'created-ids.json', tenants)
            save(self.d / 'workload-config-skeleton.json', {'endpoint': endpoint, 'label': 'UNMEASURED', 'request_seconds': 10, 'drain_seconds': 600, 'max_inflight_per_tenant': 16, 'tenants': tenants, 'phases': []})
        finally:
            forward.terminate()
            try:
                forward.wait(timeout=10)
            except subprocess.TimeoutExpired:
                forward.kill()
                forward.wait(timeout=5)

    @staticmethod
    def post(url, token, body, content_type):
        request = urllib.request.Request(url, data=body, headers={'Authorization': 'Bearer ' + token, 'Content-Type': content_type}, method='POST')
        try:
            with urllib.request.build_opener(NoRedirect()).open(request, timeout=30) as response:
                return json.load(response)
        except (urllib.error.URLError, ValueError):
            raise RuntimeError('KFP provisioning POST failed or ambiguous; no retry; reconcile created-ids and cluster') from None

    def cleanup(self):
        self.guard()
        if not (self.d / 'evidence-captured.json').exists():
            raise RuntimeError('capture evidence first; cleanup requires evidence-captured.json')
        call(['kind', 'delete', 'cluster', '--name', self.c['cluster']], timeout=300)
        for token in self.d.glob('*.token'):
            token.unlink()
        save(self.d / 'deleted.json', {'cluster': self.c['cluster'], 'deleted': True})


def main():
    os.umask(0o077)
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('action', choices=['validate', 'create', 'deploy', 'tenants', 'upload', 'cleanup'])
    p.add_argument('--config', required=True)
    p.add_argument('--state', required=True)
    a = p.parse_args()
    c = json.loads(Path(a.config).read_text())
    if a.action == 'validate':
        validate(c)
    else:
        getattr(Provision(c, a.state), a.action)()
    print(a.action + ' completed; this is not an overload acceptance result')


if __name__ == '__main__':
    try:
        main()
    except CommandFailure as error:
        print('Provisioning command failed: ' + str(error), file=sys.stderr)
        sys.exit(1)
    except Exception:
        print('Provisioning stopped; raw exception/child output suppressed to protect credentials. See stage ownership/created-ID records.', file=sys.stderr)
        sys.exit(1)
