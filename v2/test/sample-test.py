# %%
HOST = 'https://71a74112589c16e8-dot-asia-east1.pipelines.googleusercontent.com'
CONTEXT = 'gs://gongyuan-pipeline-test/sample-test/src/context.tar.gz'
SDK = 'gs://gongyuan-pipeline-test/sample-test/src/kfp-sdk.tar.gz'

# %%
import kfp
import kfp.components as comp


def echo():
    println("hello world")


echo_op = comp.func_to_container_op(echo)

download_gcs_blob = kfp.components.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/1.4.1/components/google-cloud/storage/download_blob/component.yaml'
)

# steps
# 1. build kfp-launcher container
# 2. run v2 experimental pipelines using newly built kfp-launcher
# 3. verify pipeline results in MLMD and GCS
download_gcs_tgz = kfp.components.load_component_from_text(
    '''
name: Download Folder from Tarball on GCS
inputs:
- {name: GCS Path, type: URI}
outputs:
- {name: Folder}
metadata:
  annotations:
    author: Yuan Gong <gongyuan94@gmail.com>
implementation:
  container:
    image: gcr.io/google.com/cloudsdktool/cloud-sdk:latest
    command:
    - sh
    - -exc
    - |
        uri="$0"
        output_path="$1"
        gsutil cp "$uri" artifact.tar.gz
        mkdir -p "$output_path"
        tar xvf artifact.tar.gz -C "$output_path"
    - inputValue: GCS Path
    - outputPath: Folder
'''
)

# run_sample = kfp.components.load_component_from_text(
#     '''
# name: Run KFP Sample
# inputs:
# - {name: sdk-uri, type: String}
# - {name: context-uri, type: String}
# - {name: sample-path, type: String}
# metadata:
#   annotations:
#     author: Yuan Gong <gongyuan94@gmail.com>
# implementation:
#   container:
#     image: gcr.io/google.com/cloudsdktool/cloud-sdk:latest
#     command:
#     - sh
#     - -exc
#     - |
#         mkdir -p "$(dirname "$3")";
#         /kaniko/executor \
#             --dockerfile "$0" \
#             --context "$1" \
#             --destination "$2" \
#             --digest-file "$3"
#     - inputValue: sdk-uri
#     - inputValue: context-uri
#     - inputValue: sample-path
#     - outputPath: digest
# '''
# )

kaniko = kfp.components.load_component_from_text(
    '''
name: Kaniko
inputs:
- {name: dockerfile, type: String}
- {name: context, type: URI}
- {name: destination, type: URI}
outputs:
- {name: digest, type: String}
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
        mkdir -p "$(dirname "$3")";
        /kaniko/executor \
            --dockerfile "$0" \
            --context "$1" \
            --destination "$2" \
            --digest-file "$3"
    - inputValue: dockerfile
    - inputValue: context
    - inputValue: destination
    - outputPath: digest
'''
)


@kfp.dsl.pipeline(name='v2 sample test')
def v2_sample_test(
    context: 'URI', destination: 'URI', dockerfile: str, sdk: 'URI'
):
    kaniko(context=context, destination=destination, dockerfile=dockerfile)
    download_gcs_tgz(gcs_path=context)
    download_gcs_blob(gcs_path=sdk)


def main(context: str = CONTEXT, sdk=SDK):
    client = kfp.Client(host=HOST)
    client.create_run_from_pipeline_func(
        v2_sample_test, {
            'context': context,
            'destination': 'gcr.io/gongyuan-pipeline-test/kfp-launcher:latest',
            'dockerfile': 'launcher_container/Dockerfile',
            'sdk': sdk,
        }
    )


# main()

# %%
if __name__ == "__main__":
    import fire
    fire.Fire(main)

# %%
