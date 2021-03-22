# %%
HOST = 'https://71a74112589c16e8-dot-asia-east1.pipelines.googleusercontent.com'
CONTEXT = 'gs://gongyuan-pipeline-test/sample-test/src/context.tar.gz'
SDK = 'gs://gongyuan-pipeline-test/sample-test/src/kfp-sdk.tar.gz'

# %%
import kfp
import kfp.components as comp
import json


def echo():
    println("hello world")


echo_op = comp.func_to_container_op(echo)

download_gcs_tgz = kfp.components.load_component_from_file('components/download_gcs_tgz.yaml')
run_sample = kfp.components.load_component_from_file('components/run_sample.yaml')
kaniko = kfp.components.load_component_from_file('components/kaniko.yaml')

@kfp.dsl.pipeline(name='v2 sample test')
def v2_sample_test(
    context: 'URI' = CONTEXT,
    launcher_destination: 'URI' = 'gcr.io/gongyuan-pipeline-test/kfp-launcher',
    gcs_root: 'URI' = 'gs://gongyuan-test/v2',
    samples_destination: 'URI' = 'gcr.io/gongyuan-pipeline-test/v2-sample-test',
    kfp_host: 'URI' = 'http://ml-pipeline:8888',
    samples_config: 'JSONArray' = json.dumps([
        {'name': 'fail', 'path': 'v2/test/samples/fail.py'},
        {'name': 'two_step', 'path': 'v2/test/samples/two_step.py'}]),
):
    download_src_op = download_gcs_tgz(gcs_path=context)
    download_src_op.set_display_name('download_src')
    build_kfp_launcher_op = kaniko(
        context_artifact=download_src_op.outputs['folder'],
        context_sub_path='v2',
        destination=launcher_destination,
        dockerfile='launcher_container/Dockerfile',
    )
    build_kfp_launcher_op.set_display_name('build_kfp_launcher')
    build_samples_image_op = kaniko(
        context_artifact=download_src_op.outputs['folder'],
        destination=samples_destination,
        dockerfile='v2/test/Dockerfile',
    )
    build_samples_image_op.execution_options.caching_strategy.max_cache_staleness = "P0D"
    build_samples_image_op.set_display_name('build_samples_image')
    # TODO(Bobgy): clean up unused lines
    # download_context_op = download_gcs_tgz(gcs_path=context)
    with kfp.dsl.ParallelFor(samples_config) as sample:
        run_sample_op = run_sample(
            name=sample.name,
            sample_path=sample.path,
            gcs_root=gcs_root,
            host=kfp_host,
        )
        run_sample_op.container.image = build_samples_image_op.outputs['digest']
        run_sample_op.set_display_name(f'sample_{sample.name}')
        


def main(context: str = CONTEXT, host: str = HOST):
    client = kfp.Client(host=host)
    create_run_response = client.create_run_from_pipeline_func(
        v2_sample_test, {
            'context':
                context,
            'launcher_destination':
                'gcr.io/gongyuan-pipeline-test/kfp-launcher',
            'gcs_root':
                'gs://gongyuan-test/v2',
            'samples_destination':
                'gcr.io/gongyuan-pipeline-test/v2-sample-test',
            'kfp_host': host,
        }
    )
    from pprint import pprint
    pprint(create_run_response)
    print(f"{host}/#/runs/details/{create_run_response.run_id}")


# main()

# %%
if __name__ == "__main__":
    import fire
    fire.Fire(main)

# %%
