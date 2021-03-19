# %%
HOST = 'https://71a74112589c16e8-dot-asia-east1.pipelines.googleusercontent.com'
CONTEXT = 'gs://gongyuan-pipeline-test/sample-test/src/context.tar.gz'

# %%
import kfp
import kfp.components as comp


def fail():
    import sys
    println("failed")
    sys.exit(1)


def echo():
    println("hello world")


fail_op = comp.func_to_container_op(fail)
echo_op = comp.func_to_container_op(echo)

# steps
# 1. build kfp-launcher container
# 2. run v2 experimental pipelines using newly built kfp-launcher
# 3. verify pipeline results in MLMD and GCS

kaniko = kfp.components.load_component_from_text(
    '''
name: Kaniko
inputs:
- {name: dockerfile, type: String}
- {name: context, type: String}
- {name: destination, type: String}
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
def v2_sample_test(context: str, destination: str, dockerfile: str):
    kaniko(context=context, destination=destination, dockerfile=dockerfile)


def main(context: str = CONTEXT):
    client = kfp.Client(host=HOST)
    client.create_run_from_pipeline_func(
        v2_sample_test, {
            'context': context,
            'destination': 'gcr.io/gongyuan-pipeline-test/kfp-launcher:latest',
            'dockerfile': 'launcher_container/Dockerfile'
        }
    )


# main()

# %%
if __name__ == "__main__":
    import fire
    fire.Fire(main)

# %%
