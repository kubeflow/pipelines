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

# TODO(Bobgy): switch to load from URL.
repo_root = '../..'
download_gcs_blob = kfp.components.load_component_from_file(
    repo_root + '/components/google-cloud/storage/download_blob/component.yaml'
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

run_sample = kfp.components.load_component_from_text(
    '''
name: Run KFP Sample
inputs:
- {name: Name, type: String}
- {name: Sample Path, type: Path}
- {name: GCS Root, type: URI}
metadata:
  annotations:
    author: Yuan Gong <gongyuan94@gmail.com>
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

        python3 "$sample_path" --pipeline_root "$root/$name"

    - inputValue: Sample Path
    - inputValue: Name
    - inputValue: GCS Root
'''
)

kaniko = kfp.components.load_component_from_text(
    '''
name: Kaniko
inputs:
- {name: dockerfile, type: String}
- {name: context, type: URI}
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
        /kaniko/executor \
            --dockerfile "$0" \
            --context "$1" \
            --destination "$2" \
            --digest-file "digest"
        mkdir -p "$(dirname "$3")";
        echo -n "$2@$(cat digest)" > "$3"
    - inputValue: dockerfile
    - inputValue: context
    - inputValue: destination
    - outputPath: digest
'''
)


@kfp.dsl.pipeline(name='v2 sample test')
def v2_sample_test(
    context: 'URI',
    launcher_destination: 'URI',
    launcher_dockerfile: str,
    gcs_root: 'URI',
    samples_destination: 'URI',
    samples_dockerfile: str,
):
    build_kfp_launcher_op = kaniko(
        context=context,
        destination=launcher_destination,
        dockerfile=launcher_dockerfile
    )
    build_samples_image_op = kaniko(
        context=context,
        destination=samples_destination,
        dockerfile=samples_dockerfile
    )
    download_context_op = download_gcs_tgz(gcs_path=context)
    run_sample_op = run_sample(
        name='two-step',
        sample_path='v2/samples/two_step.py',
        gcs_root=gcs_root
    )
    run_sample_op.container.image = build_samples_image_op.outputs['digest']


def main(context: str = CONTEXT):
    client = kfp.Client(host=HOST)
    client.create_run_from_pipeline_func(
        v2_sample_test, {
            'context':
                context,
            'launcher_destination':
                'gcr.io/gongyuan-pipeline-test/kfp-launcher:latest',
            'launcher_dockerfile':
                'launcher_container/Dockerfile',
            'gcs_root':
                'gs://gongyuan-test/v2'
        }
    )


# main()

# %%
if __name__ == "__main__":
    import fire
    fire.Fire(main)

# %%
