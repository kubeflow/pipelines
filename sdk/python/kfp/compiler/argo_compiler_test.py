# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0

from unittest.mock import patch

import pytest
import yaml

from kfp import compiler, dsl
from kfp.dsl.types.type_utils import InconsistentTypeException


@dsl.component(base_image='python:3.11', install_kfp_package=False)
def identity(value: int) -> int:
    return value


def test_decorator_captures_without_building_ir(tmp_path):
    with patch(
        'kfp.compiler.pipeline_spec_builder.create_pipeline_spec',
        side_effect=AssertionError('IR builder called'),
    ):

        @dsl.pipeline(name='native-test')
        def pipeline(value: int = 3):
            identity(value=value).set_caching_options(False)

        compiler.Compiler().compile(pipeline, str(tmp_path / 'pipeline.yaml'))
    manifest = yaml.safe_load((tmp_path / 'pipeline.yaml').read_text())
    assert manifest['kind'] == 'WorkflowTemplate'
    assert manifest['apiVersion'] == 'argoproj.io/v1alpha1'
    assert 'pipelineSpec' not in str(manifest)
    task = next(t for t in manifest['spec']['templates'] if 'container' in t)
    assert '--argo_input' in task['container']['args']
    assert '--executor_input' not in task['container']['args']


def test_type_check_is_preserved():
    with pytest.raises(InconsistentTypeException, match='Incompatible'):

        @dsl.pipeline
        def invalid():
            identity(value='wrong')


def test_workflow_and_cron_are_explicit(tmp_path):
    @dsl.pipeline(name='native-kinds')
    def pipeline(value: int = 1):
        identity(value=value).set_caching_options(False)

    for kind in ('Workflow', 'CronWorkflow'):
        path = tmp_path / f'{kind}.yaml'
        compiler.Compiler().compile(
            pipeline,
            str(path),
            resource_kind=kind,
            schedule='0 9 * * *' if kind == 'CronWorkflow' else None,
        )
        manifest = yaml.safe_load(path.read_text())
        assert manifest['kind'] == kind
        if kind == 'CronWorkflow':
            assert manifest['spec']['schedules'] == ['0 9 * * *']
            assert manifest['spec']['suspend'] is True


def test_conditions_loops_and_collection(tmp_path):
    @dsl.component(base_image='python:3.11', install_kfp_package=False)
    def total(values: list) -> int:
        return sum(values)

    @dsl.pipeline(name='native-control-flow')
    def pipeline(value: int = 2):
        with dsl.If(value > 0):
            identity(value=value).set_caching_options(False)
        with dsl.Else():
            identity(value=0).set_caching_options(False)
        with dsl.ParallelFor([1, 2, 3], parallelism=2) as item:
            result = identity(value=item).set_caching_options(False)
        total(values=dsl.Collected(result.output)).set_caching_options(False)

    path = tmp_path / 'control.yaml'
    compiler.Compiler().compile(pipeline, str(path), resource_kind='Workflow')
    manifest = yaml.safe_load(path.read_text())
    templates = manifest['spec']['templates']
    assert any(t.get('parallelism') == 2 for t in templates)
    assert any(
        'withParam' in task
        for t in templates
        for task in t.get('dag', {}).get('tasks', [])
    )
    assert (
        sum(
            'when' in task
            for t in templates
            for task in t.get('dag', {}).get('tasks', [])
        )
        == 2
    )


def test_caching_requires_digest_and_keys_runtime_values(tmp_path):
    @dsl.pipeline(name='cache-test')
    def pipeline(value: int = 1):
        identity(value=value)

    path = tmp_path / 'cache.yaml'
    with pytest.raises(ValueError, match='immutable image'):
        compiler.Compiler().compile(pipeline, str(path))
    compiler.Compiler(
        image_digests={'python:3.11': 'sha256:' + 'a' * 64}
    ).compile(pipeline, str(path))
    template = next(
        t
        for t in yaml.safe_load(path.read_text())['spec']['templates']
        if 'container' in t
    )
    assert 'toJson(inputs.parameters)' in template['memoize']['key']
    assert template['container']['image'] == 'python:kfp-' + 'a' * 64


@pytest.mark.parametrize(
    'image,repository',
    [
        ('repo/image:dev', 'repo/image'),
        ('localhost:5000/image:dev', 'localhost:5000/image'),
        ('image', 'image'),
    ],
)
def test_local_image_snapshot_reference(image, repository):
    from kfp.compiler.argo_compiler import local_image_reference

    assert (
        local_image_reference(image, 'sha256:' + 'a' * 64)
        == repository + ':kfp-' + 'a' * 64
    )


def test_cache_rejects_runtime_package_installation(tmp_path):
    @dsl.component(base_image='python@sha256:' + 'a' * 64)
    def installed(value: int) -> int:
        return value

    @dsl.pipeline(name='installed-cache')
    def pipeline():
        installed(value=1)

    with pytest.raises(ValueError, match='installs packages at execution time'):
        compiler.Compiler().compile(pipeline, str(tmp_path / 'pipeline.yaml'))


def test_cache_rejects_mutable_external_configuration(tmp_path):
    from kfp import kubernetes

    @dsl.pipeline(name='mutable-cache')
    def pipeline():
        task = identity(value=1)
        kubernetes.use_secret_as_env(task, 'mutable-secret', {'key': 'VALUE'})

    with pytest.raises(ValueError, match='mutable external'):
        compiler.Compiler(
            image_digests={'python:3.11': 'sha256:' + 'a' * 64}
        ).compile(pipeline, str(tmp_path / 'pipeline.yaml'))


def test_retry_normalizes_integral_float_factor_and_duration_aliases(tmp_path):
    @dsl.pipeline(name='retry-normalization')
    def pipeline():
        identity(value=1).set_retry(
            1, backoff_factor=2.0, backoff_duration='2 minutes'
        ).set_caching_options(False)

    path = tmp_path / 'retry.yaml'
    compiler.Compiler().compile(pipeline, str(path))
    template = next(
        t
        for t in yaml.safe_load(path.read_text())['spec']['templates']
        if 'container' in t
    )
    assert template['retryStrategy']['backoff']['factor'] == '2'
    assert template['retryStrategy']['backoff']['duration'] == '120s'


def test_retry_rejects_fractional_factor(tmp_path):
    @dsl.pipeline(name='fractional-retry')
    def pipeline():
        identity(value=1).set_retry(1, backoff_factor=2.5).set_caching_options(
            False
        )

    with pytest.raises(ValueError, match='integer backoff_factor'):
        compiler.Compiler().compile(pipeline, str(tmp_path / 'retry.yaml'))


def test_dynamic_resources_fail_before_manifest_publication(tmp_path):
    @dsl.pipeline(name='dynamic-resource')
    def pipeline(cpu: str = '100m'):
        identity(value=1).set_cpu_request(cpu).set_caching_options(False)

    with pytest.raises(ValueError, match='dynamic execution configuration'):
        compiler.Compiler().compile(pipeline, str(tmp_path / 'pipeline.yaml'))
    assert not (tmp_path / 'pipeline.yaml').exists()


def test_resources_and_kubernetes_configuration(tmp_path):
    from kfp import kubernetes

    @dsl.pipeline(name='configuration')
    def pipeline():
        task = identity(value=1).set_caching_options(False)
        task.set_cpu_request('100m').set_memory_limit('128Mi').set_retry(
            2, policy='OnFailure'
        )
        kubernetes.add_node_selector(task, 'pool', 'cpu')
        kubernetes.use_secret_as_env(task, 'my-secret', {'key': 'TOKEN'})
        kubernetes.add_toleration(task, key='pool', operator='Exists')

    path = tmp_path / 'config.yaml'
    compiler.Compiler().compile(pipeline, str(path))
    template = next(
        t
        for t in yaml.safe_load(path.read_text())['spec']['templates']
        if 'container' in t
    )
    assert template['container']['resources'] == {
        'requests': {'cpu': '100m'},
        'limits': {'memory': '128Mi'},
    }
    assert template['retryStrategy']['retryPolicy'] == 'OnFailure'
    assert template['nodeSelector'] == {'pool': 'cpu'}
    assert (
        template['container']['env'][-1]['valueFrom']['secretKeyRef']['name']
        == 'my-secret'
    )


def test_exit_handler_is_failure_aware_native_dag(tmp_path):
    @dsl.component(base_image='python:3.11', install_kfp_package=False)
    def report(status: dsl.PipelineTaskFinalStatus):
        print(status.state)

    @dsl.pipeline(name='exit-test')
    def pipeline():
        handler = report().set_caching_options(False)
        with dsl.ExitHandler(handler):
            identity(value=1).set_caching_options(False)

    path = tmp_path / 'exit.yaml'
    compiler.Compiler().compile(pipeline, str(path))
    manifest = yaml.safe_load(path.read_text())
    wrapper = next(
        t
        for t in manifest['spec']['templates']
        if t.get('dag', {}).get('failFast') is False
    )
    assert wrapper['dag']['target'] == 'body exit'
    cleanup = wrapper['dag']['tasks'][1]
    assert cleanup['depends'] == 'body.Succeeded || body.Failed || body.Errored'
    assert 'SUCCEEDED' in cleanup['arguments']['parameters'][0]['value']
    assert 'FAILED' in cleanup['arguments']['parameters'][0]['value']
    target = next(
        t
        for t in manifest['spec']['templates']
        if t['name'] == cleanup['template']
    )
    assert {'name': 'status'} in target['inputs']['parameters']
    args = target['container']['args']
    import json

    assert (
        'status' in json.loads(args[args.index('--argo_input') + 1])['inputs']
    )


def test_unvalidated_container_scalar_outputs_fail_compilation(tmp_path):
    @dsl.container_component
    def integer_output(value: dsl.OutputPath(int)):
        return dsl.ContainerSpec(
            image='busybox:1.36.1',
            command=['sh', '-c'],
            args=['echo not-an-integer > "$1"', 'component', value],
        )

    @dsl.pipeline(name='typed-output')
    def pipeline():
        integer_output().set_caching_options(False)

    with pytest.raises(ValueError, match='use a Python component bridge'):
        compiler.Compiler().compile(pipeline, str(tmp_path / 'pipeline.yaml'))


def test_metadata_port_paths_and_storage_keys_do_not_collide(tmp_path):
    @dsl.component(base_image='python:3.11', install_kfp_package=False)
    def make(metadata: dsl.Output[dsl.Artifact]):
        with open(metadata.path, 'w') as stream:
            stream.write('payload')

    @dsl.pipeline(name='metadata-paths')
    def pipeline():
        make().set_caching_options(False)

    path = tmp_path / 'pipeline.yaml'
    compiler.Compiler().compile(pipeline, str(path))
    template = next(
        t
        for t in yaml.safe_load(path.read_text())['spec']['templates']
        if 'container' in t
    )
    artifacts = template['outputs']['artifacts']
    assert len({artifact['path'] for artifact in artifacts}) == len(artifacts)
    assert len({artifact['s3']['key'] for artifact in artifacts}) == len(
        artifacts
    )
    identity = next(
        parameter
        for parameter in template['outputs']['parameters']
        if parameter['name'] == 'kfp-meta-metadata'
    )
    assert identity['valueFrom']['path'] == '/tmp/kfp/identities/metadata.json'


def test_nested_pipeline_outputs(tmp_path):
    @dsl.pipeline(name='nested-inner')
    def inner(value: int) -> int:
        return identity(value=value).set_caching_options(False).output

    @dsl.pipeline(name='nested-outer')
    def outer(value: int = 3):
        first = inner(value=value)
        identity(value=first.output).set_caching_options(False)

    path = tmp_path / 'nested.yaml'
    compiler.Compiler().compile(outer, str(path))
    manifest = yaml.safe_load(path.read_text())
    assert (
        sum('dag' in template for template in manifest['spec']['templates'])
        == 2
    )


def test_versions_are_content_addressed(tmp_path):
    @dsl.pipeline(name='native-version')
    def pipeline(value: int = 1):
        identity(value=value).set_caching_options(False)

    names = []
    for value in (1, 1, 2):
        path = tmp_path / 'version.yaml'
        compiler.Compiler().compile(
            pipeline, str(path), pipeline_parameters={'value': value}
        )
        names.append(yaml.safe_load(path.read_text())['metadata']['name'])
    assert names[0] == names[1]
    assert names[1] != names[2]
