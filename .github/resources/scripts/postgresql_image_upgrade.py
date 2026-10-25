#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Qualify same-major PostgreSQL image patches, not PostgreSQL KFP support."""

import argparse
import copy
import json
from pathlib import Path
import re
import socket
import subprocess
import sys
import time

import yaml

ROOT = Path(__file__).resolve().parents[3]
FIXTURES = ROOT / '.github/resources/postgresql-upgrade'
PROFILES = {
    '18': ('third-party/postgresql/base', 'postgres', 'postgres-pvc', 'user',
           'docker.io/postgres:18.3-alpine3.22'),
    '14': ('base/metadata/overlays/postgres', 'metadata-postgres-db',
           'metadata-postgres', 'root', 'docker.io/postgres:14.7-alpine3.17'),
}


def run(args, **kwargs):
    """Fail at the actual command, including commands whose output is
    captured."""
    return subprocess.run(args, check=True, text=True, **kwargs)


def select_resources(documents, lane, namespace):
    """Keep database dependencies and the actual legacy MLMD deployment."""
    _, deployment, claim, _, _ = PROFILES[lane]
    selected = []
    for original in documents:
        if not original:
            continue
        kind, name = original['kind'], original['metadata']['name']
        keep = (
            kind in ('Secret', 'ConfigMap', 'ServiceAccount') or
            (kind == 'PersistentVolumeClaim' and name == claim) or
            (kind == 'Service' and name in (deployment, 'postgres-service')) or
            (kind == 'Deployment' and
             name in (deployment, 'metadata-grpc-deployment')))
        if keep:
            resource = copy.deepcopy(original)
            resource['metadata']['namespace'] = namespace
            selected.append(resource)
    database = next(
        r for r in selected
        if r['kind'] == 'Deployment' and r['metadata']['name'] == deployment)
    image = database['spec']['template']['spec']['containers'][0]['image']
    # Require an immutable same-major candidate. A major change requires a
    # migration test, not reuse of this retained-directory acceptance lane.
    match = re.search(r':(\d+)\.(\d+)[^@]*@sha256:[0-9a-f]{64}$', image)
    if not match or match[1] != lane:
        raise ValueError('Candidate must be a digest-pinned same-major '
                         'PostgreSQL image; update the manifest or design a '
                         'major-version migration test.')
    return selected, image, int(match[1]) * 10000 + int(match[2])


class Acceptance:
    """Run on a disposable Kind cluster with its real provisioned volumes."""

    def __init__(self, lane, output, schema):
        self.lane, self.output, self.schema = lane, output, schema
        self.namespace = 'postgres-upgrade-' + lane
        _, self.deployment, self.claim, self.user, self.old = PROFILES[lane]
        self.forwards = []
        self.receipt = {'lane': lane, 'old_image': self.old, 'phases': []}

    def kubectl(self, *args, **kwargs):
        return run(['kubectl', '-n', self.namespace, *args], **kwargs)

    def sql(self, statement, database='postgres'):
        return self.kubectl(
            'exec',
            '-i',
            'deployment/' + self.deployment,
            '--',
            'psql',
            '-v',
            'ON_ERROR_STOP=1',
            '-U',
            self.user,
            '-d',
            database,
            '-At',
            input=statement,
            capture_output=True).stdout.strip()

    def wait_database(self):
        self.kubectl('rollout', 'status', 'deployment/' + self.deployment,
                     '--timeout=240s')
        deadline = time.monotonic() + 120
        while time.monotonic() < deadline:
            try:
                if self.sql('SELECT 1') == '1':
                    return
            except subprocess.CalledProcessError:
                pass
            time.sleep(2)
        raise RuntimeError('Database failed readiness; inspect saved pod logs '
                           'and volume events before changing the fixture.')

    def forward(self, deployment, local, remote):
        log = (self.output / f'forward-{local}.log').open('a')
        process = subprocess.Popen([
            'kubectl', '-n', self.namespace, 'port-forward',
            'deployment/' + deployment, f'{local}:{remote}'
        ],
                                   stdout=log,
                                   stderr=subprocess.STDOUT)
        self.forwards.append((process, log))
        for _ in range(60):
            if process.poll() is not None:
                break
            try:
                with socket.create_connection(('127.0.0.1', local), timeout=1):
                    return
            except OSError:
                time.sleep(1)
        raise RuntimeError('Port forwarding failed; inspect the saved '
                           f'forward-{local}.log and deployment readiness.')

    def stop_forwards(self):
        for process, log in self.forwards:
            process.terminate()
            process.wait(timeout=15)
            log.close()
        self.forwards.clear()

    def application(self, phase):
        if self.lane == '18':
            self.forward(self.deployment, 15432, 5432)
            run([str(self.schema), 'seed' if phase == 'old' else 'verify'])
        else:
            self.kubectl('apply', '-f', str(self.output / 'mlmd.yaml'))
            self.kubectl('rollout', 'status',
                         'deployment/metadata-grpc-deployment',
                         '--timeout=240s')
            self.forward('metadata-grpc-deployment', 15433, 8080)
            run([
                sys.executable,
                str(FIXTURES / 'mlmd-client.py'),
                'seed' if phase == 'old' else phase
            ])
        self.stop_forwards()
        self.receipt['phases'].append(phase)
        (self.output /
         'receipt.json').write_text(json.dumps(self.receipt, indent=2) + '\n')

    def stop_mlmd(self):
        if self.lane == '14':
            self.kubectl('scale', 'deployment/metadata-grpc-deployment',
                         '--replicas=0')
            self.kubectl('wait', '--for=delete', 'pod', '-l',
                         'component=metadata-grpc-server', '--timeout=120s')

    def diagnostic(self):
        self.stop_forwards()
        for resource in ('pods', 'pvc', 'events', 'deployments'):
            with (self.output / f'{resource}.yaml').open('w') as output:
                subprocess.run([
                    'kubectl', '-n', self.namespace, 'get', resource, '-o',
                    'yaml'
                ],
                               stdout=output,
                               stderr=subprocess.STDOUT,
                               check=False)
        for deployment in (self.deployment, 'metadata-grpc-deployment'):
            with (self.output / f'{deployment}.log').open('w') as output:
                subprocess.run([
                    'kubectl', '-n', self.namespace, 'logs',
                    'deployment/' + deployment, '--all-containers'
                ],
                               stdout=output,
                               stderr=subprocess.STDOUT,
                               check=False)

    def execute(self):
        source = ROOT / 'manifests/kustomize' / PROFILES[self.lane][0]
        rendered = run(
            ['kubectl', 'kustomize', str(source)], capture_output=True).stdout
        resources, candidate, expected_version = select_resources(
            yaml.safe_load_all(rendered), self.lane, self.namespace)
        self.receipt['candidate_image'] = candidate
        (self.output / 'rendered.yaml').write_text(rendered)
        mlmd = [
            r for r in resources if r['kind'] == 'Deployment' and
            r['metadata']['name'] == 'metadata-grpc-deployment'
        ]
        (self.output / 'mlmd.yaml').write_text(yaml.safe_dump_all(mlmd))
        resources = [r for r in resources if r not in mlmd]
        database = next(r for r in resources if r['kind'] == 'Deployment')
        database['spec']['template']['spec']['containers'][0][
            'image'] = self.old
        run(['kubectl', 'create', 'namespace', self.namespace])
        self.kubectl('apply', '-f', '-', input=yaml.safe_dump_all(resources))
        self.wait_database()
        claim = json.loads(
            self.kubectl(
                'get', 'pvc', self.claim, '-o', 'json',
                capture_output=True).stdout)
        self.receipt['pvc_uid'] = claim['metadata']['uid']
        self.receipt['pv_name'] = claim['spec']['volumeName']
        self.sql('''CREATE SCHEMA patch_acceptance;
CREATE TABLE patch_acceptance.records (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 name text UNIQUE NOT NULL, payload jsonb NOT NULL, created timestamptz NOT NULL);
CREATE INDEX records_payload ON patch_acceptance.records USING gin(payload);
INSERT INTO patch_acceptance.records(name,payload,created)
 SELECT 'record-'||i,jsonb_build_object('value',i),
 TIMESTAMPTZ '2026-01-01 12:00:00+00' FROM generate_series(1,1000) i;''')
        if self.lane == '14':
            self.sql('CREATE DATABASE mlmdpostgres')
        self.application('old')
        digest_query = """SELECT md5(string_agg(id||name||payload::text||
created::text,',' ORDER BY id)) FROM patch_acceptance.records"""
        before = self.sql(digest_query)
        system_id = self.sql(
            'SELECT system_identifier FROM pg_control_system()')
        self.stop_mlmd()
        container = database['spec']['template']['spec']['containers'][0][
            'name']
        self.kubectl('set', 'image', 'deployment/' + self.deployment,
                     container + '=' + candidate)
        self.wait_database()
        if self.sql('SHOW server_version_num') != str(expected_version):
            raise RuntimeError(
                'Wrong PostgreSQL version; inspect rendered image '
                'and runtime image ID before accepting the pin.')
        if self.sql(digest_query) != before or self.sql(
                'SELECT system_identifier FROM pg_control_system()'
        ) != system_id:
            raise RuntimeError('Upgrade changed retained data or database '
                               'identity; investigate the mounted PVC.')
        self.sql('''DO $$ BEGIN
 IF (SELECT count(*) FROM patch_acceptance.records
     WHERE payload @> '{"value":500}'::jsonb) <> 1
 THEN RAISE EXCEPTION 'JSONB lookup failed'; END IF;
 IF (SELECT count(*) FROM pg_index
     WHERE indrelid='patch_acceptance.records'::regclass AND indisvalid) <> 3
 THEN RAISE EXCEPTION 'index lost'; END IF;
END $$;
INSERT INTO patch_acceptance.records(name,payload,created)
 VALUES ('after-upgrade','{"value":1001}',now());''')
        self.application('upgraded')
        self.stop_mlmd()
        self.kubectl('rollout', 'restart', 'deployment/' + self.deployment)
        self.wait_database()
        if self.sql('SELECT count(*) FROM patch_acceptance.records') != '1001':
            raise RuntimeError('Post-upgrade write was not retained; inspect '
                               'database and volume logs.')
        self.application('restarted')
        after = json.loads(
            self.kubectl(
                'get', 'pvc', self.claim, '-o', 'json',
                capture_output=True).stdout)
        if (after['metadata']['uid'] != self.receipt['pvc_uid'] or
                after['spec']['volumeName'] != self.receipt['pv_name']):
            raise RuntimeError('PVC changed during acceptance; repeat with the '
                               'original volume intact.')
        self.receipt['status'] = 'passed'
        (self.output /
         'receipt.json').write_text(json.dumps(self.receipt, indent=2) + '\n')
        print(
            f'PASS PostgreSQL {self.lane} retained-PVC image upgrade',
            flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--lane', required=True, choices=PROFILES)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument(
        '--schema-helper', type=Path, default=Path('/tmp/kfp-schema'))
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    acceptance = Acceptance(args.lane, args.output, args.schema_helper)
    try:
        acceptance.execute()
    finally:
        acceptance.diagnostic()


if __name__ == '__main__':
    main()
