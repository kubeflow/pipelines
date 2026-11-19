# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Validate Kubernetes modifiers at the native template boundary, not retired IR snapshots."""

import pytest
import yaml

from kfp import compiler, dsl, kubernetes


@pytest.mark.parametrize(
    'method,kwargs,field,expected',
    [
        (
            'add_node_selector',
            {'label_key': 'pool', 'label_value': 'training'},
            'nodeSelector',
            {'pool': 'training'},
        ),
        (
            'add_pod_label',
            {'label_key': 'team', 'label_value': 'ml'},
            'metadata',
            {'labels': {'team': 'ml'}},
        ),
        (
            'add_pod_annotation',
            {'annotation_key': 'note', 'annotation_value': 'native'},
            'metadata',
            {'annotations': {'note': 'native'}},
        ),
        (
            'add_toleration',
            {'key': 'pool', 'operator': 'Exists'},
            'tolerations',
            [{'key': 'pool', 'operator': 'Exists'}],
        ),
        ('set_timeout', {'seconds': 60}, 'activeDeadlineSeconds', 60),
    ],
)
def test_template_fields(tmp_path, method, kwargs, field, expected):
    @dsl.container_component
    def step():
        return dsl.ContainerSpec(image='busybox', command=['true'])

    @dsl.pipeline
    def pipeline():
        getattr(kubernetes, method)(step(), **kwargs)

    path = tmp_path / 'pipeline.yaml'
    compiler.Compiler(disable_caching=True).compile(pipeline, str(path))
    template = next(
        t
        for t in yaml.safe_load(path.read_text())['spec']['templates']
        if 'container' in t
    )
    assert template[field] == expected


def test_image_env_and_volume_bindings(tmp_path):
    @dsl.container_component
    def step():
        return dsl.ContainerSpec(image='busybox', command=['true'])

    @dsl.pipeline
    def pipeline():
        task = step()
        kubernetes.set_image_pull_policy(task, 'IfNotPresent')
        kubernetes.use_secret_as_env(task, 'secret', {'key': 'SECRET_VALUE'})
        kubernetes.use_config_map_as_env(
            task, 'config', {'key': 'CONFIG_VALUE'}
        )
        kubernetes.mount_pvc(task, 'data', '/data')

    path = tmp_path / 'pipeline.yaml'
    compiler.Compiler(disable_caching=True).compile(pipeline, str(path))
    template = next(
        t
        for t in yaml.safe_load(path.read_text())['spec']['templates']
        if 'container' in t
    )
    assert template['container']['imagePullPolicy'] == 'IfNotPresent'
    env = {
        entry['name']: entry['valueFrom']
        for entry in template['container']['env']
    }
    assert env['SECRET_VALUE']['secretKeyRef'] == {
        'name': 'secret',
        'key': 'key',
        'optional': False,
    }
    assert env['CONFIG_VALUE']['configMapKeyRef'] == {
        'name': 'config',
        'key': 'key',
        'optional': False,
    }
    assert (
        template['volumes'][0]['persistentVolumeClaim']['claimName'] == 'data'
    )
    assert template['container']['volumeMounts'][0]['mountPath'] == '/data'


def test_unmapped_configuration_is_rejected(tmp_path):
    @dsl.container_component
    def step():
        return dsl.ContainerSpec(image='busybox', command=['true'])

    @dsl.pipeline
    def pipeline():
        step().platform_config['kubernetes'] = {'unsupportedSetting': True}

    with pytest.raises(ValueError, match='no native lowering'):
        compiler.Compiler(disable_caching=True).compile(
            pipeline, str(tmp_path / 'pipeline.yaml')
        )
