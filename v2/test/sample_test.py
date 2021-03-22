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

# TODO(Bobgy): switch to load from URL.
repo_root = '../..'
download_gcs_blob = kfp.components.load_component_from_file(
    repo_root + '/components/google-cloud/storage/download_blob/component.yaml'
)

# TODO(Bobgy): clean up unused lines
# download_gcs_tgz = kfp.components.load_component_from_text(
#     '''
# name: Download Folder from Tarball on GCS
# inputs:
# - {name: GCS Path, type: URI}
# outputs:
# - {name: Folder}
# implementation:
#   container:
#     image: gcr.io/google.com/cloudsdktool/cloud-sdk:latest
#     command:
#     - sh
#     - -exc
#     - |
#         uri="$0"
#         output_path="$1"
#         gsutil cp "$uri" artifact.tar.gz
#         mkdir -p "$output_path"
#         tar xvf artifact.tar.gz -C "$output_path"
#     - inputValue: GCS Path
#     - outputPath: Folder
# '''
# )

run_sample = kfp.components.load_component_from_text(
    '''
name: Run KFP Sample
inputs:
- {name: Name, type: String}
- {name: Sample Path, type: Path}
- {name: GCS Root, type: URI}
- {name: Host, type: URI, default: "http://ml-pipeline:8888"}
implementation:
  container:
    image: python:3.7-alpine
    command:
    - sh
    - -exc
    - |
        sample_path="$0"
        name="$1"
        root="$2"
        host="$3"

        python3 "$sample_path" --pipeline_root "$root/$name" --host "$host"

    - inputValue: Sample Path
    - inputValue: Name
    - inputValue: GCS Root
    - inputValue: Host
'''
)

kaniko = kfp.components.load_component_from_text(
    '''
name: Kaniko
inputs:
- {name: dockerfile, type: String}
- {name: context, type: URI}
- {name: context-sub-path, type: Path, default: ""}
- {name: destination, type: URI, description: destination should not contain tag}
outputs:
- {name: digest, type: URI, description: Image URI with full digest}
metadata:
  annotations:
    author: Yuan Gong <gongyuan94@gmail.com>
implementation:
  container:
    # The debug image contains busybox, while the normal images do not.
    image: gcr.io/kaniko-project/executor:debug
    command:
    - sh
    - -exc
    - |
        dockerfile="$0"
        context="$1"
        context_sub_path="$2"
        destination="$3"
        digest_output_path="$4"
        /kaniko/executor \
            --dockerfile "${dockerfile}" \
            --context "${context}" \
            ${context_sub_path:+ --context-sub-path "${context_sub_path}"} \
            --destination "${destination}" \
            --digest-file "digest"
        mkdir -p "$(dirname "${digest_output_path}")";
        echo -n "${destination}@$(cat digest)" > "${digest_output_path}"
    - inputValue: dockerfile
    - inputValue: context
    - inputValue: context-sub-path
    - inputValue: destination
    - outputPath: digest
'''
)


@kfp.dsl.pipeline(name='v2 sample test')
def v2_sample_test(
    context: 'URI' = CONTEXT,
    launcher_destination: 'URI' = 'gcr.io/gongyuan-pipeline-test/kfp-launcher',
    gcs_root: 'URI' = 'gs://gongyuan-test/v2',
    samples_destination: 'URI' = 'gcr.io/gongyuan-pipeline-test/v2-sample-test',
    kfp_host: 'URI' = 'http://ml-pipeline:8888',
    samples_config: 'JSONArray' = json.dumps([
        {'sample_name': 'fail', 'path': 'v2/test/samples/fail.py'},
        {'sample_name': 'two_step', 'path': 'v2/test/samples/two_step.py'}]),
):
    build_kfp_launcher_op = kaniko(
        context=context,
        context_sub_path='v2',
        destination=launcher_destination,
        dockerfile='launcher_container/Dockerfile',
    )
    build_kfp_launcher_op.set_display_name('build_kfp_launcher')
    build_samples_image_op = kaniko(
        context=context,
        destination=samples_destination,
        dockerfile='v2/test/Dockerfile',
    )
    build_samples_image_op.execution_options.caching_strategy.max_cache_staleness = "P0D"
    build_samples_image_op.set_display_name('build_samples_image_op')
    # TODO(Bobgy): clean up unused lines
    # download_context_op = download_gcs_tgz(gcs_path=context)
    with kfp.dsl.ParallelFor(samples_config) as sample:
        run_sample_op = run_sample(
            name=sample.sample_name,
            sample_path=sample.path,
            gcs_root=gcs_root,
            host=kfp_host,
        )
        run_sample_op.container.image = build_samples_image_op.outputs['digest']
        run_sample_op.set_display_name(sample.sample_name)
        


def main(context: str = CONTEXT, host: str = HOST):
    client = kfp.Client(host=host)
    client.create_run_from_pipeline_func(
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


# main()

# %%
if __name__ == "__main__":
    import fire
    fire.Fire(main)

# %%
