# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Small executable corpus for native control-flow semantics."""

from kfp import dsl


@dsl.component(base_image='kfp-argo-demo:dev', install_kfp_package=False)
def number(value: int, fail: bool = False) -> int:
    if fail:
        raise ValueError('deliberate compatibility failure')
    return value


@dsl.component(base_image='kfp-argo-demo:dev', install_kfp_package=False)
def total(values: list) -> int:
    assert all(isinstance(value, int) for value in values), values
    return sum(values)


@dsl.container_component
def container_dataset(dataset: dsl.Output[dsl.Dataset]):
    return dsl.ContainerSpec(
        image='busybox:1.36.1',
        command=['sh', '-c'],
        args=['printf "[1,2,3]" > "$1"', 'component', dataset.path],
    )


@dsl.component(base_image='kfp-argo-demo:dev', install_kfp_package=False)
def read_dataset(dataset: dsl.Input[dsl.Dataset]) -> int:
    import json

    assert dataset.uri.startswith('s3://argo-artifacts/'), dataset.uri
    with open(dataset.path) as stream:
        return sum(json.load(stream))


@dsl.pipeline(name='native-container-source')
def container_source() -> dsl.Dataset:
    return container_dataset().set_caching_options(False).outputs['dataset']


@dsl.pipeline(name='native-container-artifacts')
def container_artifacts() -> int:
    source = container_source()
    return read_dataset(dataset=source.output).set_caching_options(False).output


@dsl.component(base_image='kfp-argo-demo:dev', install_kfp_package=False)
def large_dataset(dataset: dsl.Output[dsl.Dataset]):
    with open(dataset.path, 'w') as stream:
        stream.write('payload')
    dataset.metadata['large'] = 'x' * 100_000


@dsl.component(base_image='kfp-argo-demo:dev', install_kfp_package=False)
def metadata_size(dataset: dsl.Input[dsl.Dataset]) -> int:
    return len(dataset.metadata['large'])


@dsl.pipeline(name='native-large-metadata')
def large_metadata() -> int:
    source = large_dataset().set_caching_options(False)
    return (
        metadata_size(dataset=source.outputs['dataset'])
        .set_caching_options(False)
        .output
    )


@dsl.pipeline(name='native-retry-failure')
def retry_failure():
    number(value=1, fail=True).set_cpu_request('50m').set_memory_limit(
        '64Mi'
    ).set_retry(
        1, backoff_duration='0s', backoff_factor=2.0, policy='OnFailure'
    ).set_caching_options(False)


@dsl.pipeline(name='native-nested')
def nested(value: int) -> int:
    return number(value=value).set_caching_options(False).output


@dsl.component(base_image='kfp-argo-demo:dev', install_kfp_package=False)
def report_status(
    status: dsl.PipelineTaskFinalStatus, fail_cleanup: bool = False
) -> str:
    if fail_cleanup:
        raise ValueError('deliberate cleanup failure')
    assert status.state in ('SUCCEEDED', 'FAILED'), status
    assert status.pipeline_task_name, status
    return status.state


@dsl.pipeline(name='native-exit')
def exit_flow(fail: bool = False, fail_cleanup: bool = False):
    cleanup = report_status(fail_cleanup=fail_cleanup).set_caching_options(
        False
    )
    with dsl.ExitHandler(cleanup):
        number(value=42, fail=fail).set_caching_options(False)


@dsl.pipeline(name='native-control-flow')
def control_flow(value: int = 2, fail: bool = False) -> int:
    upstream = nested(value=value)
    with dsl.If(upstream.output > 0):
        number(value=10).set_caching_options(False)
    with dsl.Else():
        number(value=-10).set_caching_options(False)
    with dsl.ParallelFor([1, 2, 3], parallelism=2) as item:
        result = number(value=item, fail=fail).set_caching_options(False)
    return (
        total(values=dsl.Collected(result.output))
        .set_caching_options(False)
        .output
    )
