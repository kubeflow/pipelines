# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Build policy evidence for the controlled multi-user upgrade CI fixture only.

The caller asserts that source snapshots include ALL cluster and
namespace RBAC, that candidate manifests are applied without pruning
retained source objects, and that the fixture uses only the RBAC
authorizer. Those assertions justify rbac_complete/rbac_only; this
utility cannot discover or verify completeness. Never use this fixture
policy as an operator's general readiness assessment.
"""

import argparse
import copy
import json
import re
import sys

from readiness import MAX_BYTES
from readiness import MAX_ITEMS
from readiness import read_json
import schedule_policy

RBAC_KINDS = {'Role', 'RoleBinding', 'ClusterRole', 'ClusterRoleBinding'}
NAMESPACED_KINDS = {'Role', 'RoleBinding'}


def snapshot_objects(snapshot, default_namespace=None):
    """Validate identity keys before replacing source RBAC with candidate
    RBAC."""
    if not isinstance(snapshot, dict) or not isinstance(
            snapshot.get('items'), list):
        raise ValueError(
            'Supply an RBAC snapshot object containing an items list.')
    if len(snapshot['items']) > MAX_ITEMS:
        raise ValueError(
            'RBAC snapshot exceeds the record limit; reduce fixture scope.')
    objects = {}
    for original in snapshot['items']:
        if not isinstance(original,
                          dict) or original.get('kind') not in RBAC_KINDS:
            raise ValueError('Filter each snapshot to RBAC objects only.')
        item = copy.deepcopy(original)
        kind = item['kind']
        metadata = item.get('metadata')
        if not isinstance(metadata, dict) or not isinstance(
                metadata.get('name'), str) or not metadata['name'].strip():
            raise ValueError('Every RBAC object requires metadata.name.')
        namespace = metadata.get('namespace')
        if kind in NAMESPACED_KINDS:
            if namespace is None and default_namespace is not None:
                namespace = default_namespace
                metadata['namespace'] = namespace
            if not isinstance(namespace, str) or not re.fullmatch(
                    r'[a-z0-9]([-a-z0-9]*[a-z0-9])?',
                    namespace) or len(namespace) > 63:
                raise ValueError(
                    'Namespaced RBAC requires a valid explicit namespace.')
        else:
            if namespace not in (None, ''):
                raise ValueError('Cluster RBAC cannot specify a namespace.')
            namespace = ''
        key = (kind, namespace, metadata['name'])
        if key in objects:
            raise ValueError(
                'Duplicate RBAC identity within a snapshot; resolve ambiguity.')
        objects[key] = item
    return objects


def build_policy(source,
                 candidate,
                 target_revision,
                 mode,
                 candidate_namespace='kubeflow'):
    """Merge retained fixture grants with the manifests applied during
    upgrade."""
    objects = snapshot_objects(source)
    objects.update(snapshot_objects(candidate, candidate_namespace))
    bundle = dict(
        policy_contract=schedule_policy.CONTRACT,
        target_revision=target_revision,
        mode=mode,
        multi_user=True,
        rbac_complete=True,
        rbac_only=True,
        controller_user='system:serviceaccount:kubeflow:ml-pipeline-scheduledworkflow',
        default_service_account='pipeline-runner',
        allowed_service_accounts=['readiness-granted', 'readiness-denied'],
        compiled_pipeline_spec_patch={},
        recurring_runs=[],
        experiments=[],
        rbac=list(objects.values()))
    schedule_policy.validate(bundle)
    if len(json.dumps(bundle).encode('utf-8')) > MAX_BYTES:
        raise ValueError(
            'Merged policy exceeds the 16 MiB limit; reduce fixture scope.')
    return bundle


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source-rbac', required=True)
    parser.add_argument('--candidate-rbac', required=True)
    parser.add_argument('--target-revision', required=True)
    parser.add_argument('--mode', required=True, choices=['enforce', 'audit'])
    parser.add_argument('--candidate-namespace', default='kubeflow')
    args = parser.parse_args()
    try:
        bundle = build_policy(
            read_json(args.source_rbac), read_json(args.candidate_rbac),
            args.target_revision, args.mode, args.candidate_namespace)
    except (OSError, ValueError, TypeError):
        print(
            'Cannot build fixture policy: supply valid, bounded, complete RBAC snapshots and target settings.',
            file=sys.stderr)
        return 1
    print(json.dumps(bundle, indent=2))
    return 0


if __name__ == '__main__':
    sys.exit(main())
