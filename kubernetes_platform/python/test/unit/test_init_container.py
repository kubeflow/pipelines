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

from google.protobuf import json_format
from kfp import dsl
from kfp import kubernetes
import pytest


class TestAddInitContainer:

    def test_add_init_container_all_fields(self):

        @dsl.pipeline
        def my_pipeline():
            task = print_greeting()
            kubernetes.add_init_container(
                task,
                name='fetch-config',
                image='busybox:1.36',
                command=['sh', '-c'],
                args=['wget -O /config/settings.json $CONFIG_URL'],
                env={'CONFIG_URL': 'http://config-server/settings.json'},
                volume_mounts={'config-volume': '/config'},
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-print-greeting': {
                                'initContainers': [{
                                    'name':
                                        'fetch-config',
                                    'image':
                                        'busybox:1.36',
                                    'command': ['sh', '-c'],
                                    'args': [
                                        'wget -O /config/settings.json $CONFIG_URL'
                                    ],
                                    'env': [{
                                        'name':
                                            'CONFIG_URL',
                                        'value':
                                            'http://config-server/settings.json'
                                    }],
                                    'volumeMounts': [{
                                        'volumeName': 'config-volume',
                                        'mountPath': '/config'
                                    }],
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_add_init_container_minimal(self):

        @dsl.pipeline
        def my_pipeline():
            task = print_greeting()
            kubernetes.add_init_container(
                task,
                name='wait-for-database',
                image='busybox:1.36',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-print-greeting': {
                                'initContainers': [{
                                    'name': 'wait-for-database',
                                    'image': 'busybox:1.36',
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_add_init_container_multiple_preserves_order(self):

        @dsl.pipeline
        def my_pipeline():
            task = print_greeting()
            kubernetes.add_init_container(
                task,
                name='first-init',
                image='busybox:1.36',
            )
            kubernetes.add_init_container(
                task,
                name='second-init',
                image='busybox:1.36',
            )

        init_containers = json_format.MessageToDict(
            my_pipeline.platform_spec
        )['platforms']['kubernetes']['deploymentSpec']['executors'][
            'exec-print-greeting']['initContainers']
        assert [init_container['name'] for init_container in init_containers
               ] == ['first-init', 'second-init']

    def test_add_init_container_reserved_name(self):
        with pytest.raises(
                ValueError,
                match=r'Init container name "kfp-launcher" is reserved by Kubeflow Pipelines.',
        ):

            @dsl.pipeline
            def my_pipeline():
                task = print_greeting()
                kubernetes.add_init_container(
                    task,
                    name='kfp-launcher',
                    image='busybox:1.36',
                )

    def test_add_init_container_empty_name(self):
        with pytest.raises(
                ValueError,
                match=r'Argument for "name" must be a non-empty string.',
        ):

            @dsl.pipeline
            def my_pipeline():
                task = print_greeting()
                kubernetes.add_init_container(
                    task,
                    name='',
                    image='busybox:1.36',
                )

    def test_add_init_container_empty_image(self):
        with pytest.raises(
                ValueError,
                match=r'Argument for "image" must be a non-empty string.',
        ):

            @dsl.pipeline
            def my_pipeline():
                task = print_greeting()
                kubernetes.add_init_container(
                    task,
                    name='fetch-config',
                    image='',
                )

    def test_add_init_container_duplicate_name(self):
        with pytest.raises(
                ValueError,
                match=r'An init container named "fetch-config" was already added to this task.',
        ):

            @dsl.pipeline
            def my_pipeline():
                task = print_greeting()
                kubernetes.add_init_container(
                    task,
                    name='fetch-config',
                    image='busybox:1.36',
                )
                kubernetes.add_init_container(
                    task,
                    name='fetch-config',
                    image='busybox:1.36',
                )

    def test_add_init_container_native_sidecar(self):

        @dsl.pipeline
        def my_pipeline():
            task = print_greeting()
            kubernetes.add_init_container(
                task,
                name='log-forwarder',
                image='busybox:1.36',
                restart_policy='Always',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-print-greeting': {
                                'initContainers': [{
                                    'name': 'log-forwarder',
                                    'image': 'busybox:1.36',
                                    'restartPolicy': 'Always',
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_add_init_container_invalid_restart_policy(self):
        with pytest.raises(
                ValueError,
                match=r'Argument for "restart_policy" must be "Always" if provided. Got: Never.',
        ):

            @dsl.pipeline
            def my_pipeline():
                task = print_greeting()
                kubernetes.add_init_container(
                    task,
                    name='log-forwarder',
                    image='busybox:1.36',
                    restart_policy='Never',
                )

    def test_add_init_container_empty_env_name(self):
        with pytest.raises(
                ValueError,
                match=r'Environment variable names in "env" must be non-empty strings.',
        ):

            @dsl.pipeline
            def my_pipeline():
                task = print_greeting()
                kubernetes.add_init_container(
                    task,
                    name='fetch-config',
                    image='busybox:1.36',
                    env={'': 'value'},
                )

    def test_add_init_container_relative_mount_path(self):
        with pytest.raises(
                ValueError,
                match=r'Mount path for volume "config-volume" must be an absolute path. Got: config.',
        ):

            @dsl.pipeline
            def my_pipeline():
                task = print_greeting()
                kubernetes.add_init_container(
                    task,
                    name='fetch-config',
                    image='busybox:1.36',
                    volume_mounts={'config-volume': 'config'},
                )


@dsl.component
def print_greeting():
    print('hello world')
