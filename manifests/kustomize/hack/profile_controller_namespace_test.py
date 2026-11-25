#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Check that a namespace override rewrites the webhook Secret reference."""

import json
import sys


def main():
    resources = []
    remaining = sys.stdin.read().strip()
    decoder = json.JSONDecoder()
    while remaining:
        resource, end = decoder.raw_decode(remaining)
        resources.append(resource)
        remaining = remaining[end:].lstrip()

    def get_resource(kind, name):
        matches = [
            resource for resource in resources
            if resource.get('kind') == kind and
            resource.get('metadata', {}).get('name') == name
        ]
        assert len(matches) == 1, f'Expected one {kind}/{name}'
        return matches[0]

    name = 'kubeflow-pipelines-profile-webhook-auth'
    secret = get_resource('Secret', name)
    deployment = get_resource('Deployment',
                              'kubeflow-pipelines-profile-controller')
    decorator = get_resource('DecoratorController',
                             'kubeflow-pipelines-profile-controller')
    namespace = secret['metadata']['namespace']
    assert namespace == 'kfp-custom'
    assert deployment['metadata']['namespace'] == namespace
    webhook = decorator['spec']['hooks']['sync']['webhook']
    assert webhook['service'] == {
        'name': 'kubeflow-pipelines-profile-controller',
        'namespace': namespace,
    }
    assert webhook['path'] == '/sync'
    assert webhook['authorization']['secretRef']['namespace'] == namespace


if __name__ == '__main__':
    main()
