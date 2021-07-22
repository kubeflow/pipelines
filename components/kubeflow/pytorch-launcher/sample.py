import copy
import json
import kfp
from kfp import components
import kfp.dsl as dsl


def get_current_namespace():
    """Returns current namespace if available, else kubeflow"""
    try:
        current_namespace = open("/var/run/secrets/kubernetes.io/serviceaccount/namespace").read()
    except:
        current_namespace = "kubeflow"
    return current_namespace


pytorchjob_launcher_op = components.load_component_from_file("./component.yaml")
# pytorchjob_launcher_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/master/components/kubeflow/pytorch-launcher/component.yaml')


@dsl.pipeline(
    name="Launch kubeflow pytorchjob",
    description="An example to launch pytorch."
)
def mnist_train(
        name: str="mnist",
        namespace: str=get_current_namespace(),
        worker_replicas: int=3,
        ttl_seconds_after_finished: int=-1,
        job_timeout_minutes: int=600,
        delete_after_done: bool=False):
    # Define the master and worker definitions by dict.
    # These can also be defined using the kubeflow-pytorchjob API
    master = {
        "replicas": 1,
        "restartPolicy": "OnFailure",
        "template": {
            "metadata": {
                "annotations": {
                    # See https://github.com/kubeflow/website/issues/2011
                    "sidecar.istio.io/inject": "false"
                }
            },
            "spec": {
                "containers": [
                    {
                    # To override default command
                    # "command": [
                    #   "python",
                    #   "/opt/mnist/src/mnist.py"
                    # ],
                    "args": [
                        "--backend", "gloo",
                    ],
                    # Or, create your own image from 
                    # https://github.com/kubeflow/pytorch-operator/tree/master/examples/mnist
                    "image": "gcr.io/kubeflow-examples/pytorch-dist-mnist:v20180702-a57993c",
                    "name": "pytorch",
                    "resources": {
                        "requests": {
                            "memory": "4Gi",
                            "cpu": "2000m",
                        },
                        "limits": {
                            "memory": "4Gi",
                            "cpu": "2000m",
                        },
                    },
                    # If using GPUs
                    # "resources": {
                    #   "limits": {
                    #     "nvidia.com/gpu": 1,
                    #   }
                    # }
                    }
                ],
                # If imagePullSecrets required
                # "imagePullSecrets": [
                #     {"name": "image-pull-secret"},
                # ],
            },
        },
    }

    # You could have a worker spec that is different from master
    # (maybe master needs a PVC for saving results)
    # This does not handle replicas==0.  Would need if statement at kfp
    # runtime to handle that (doing 'if worker_replicas > 0' here does
    # not work)
    worker = copy.deepcopy(master)
    worker['replicas'] = worker_replicas

    # Launch and monitor the job with the launcher
    pytorchjob_launcher_op(
        # Note: name needs to be a unique pytorchjob name in the namespace.
        # Using RUN_ID_PLACEHOLDER is one way of getting something unique.
        name=f"name-{kfp.dsl.RUN_ID_PLACEHOLDER}",
        namespace=namespace,
        master_spec=master,
        # pass worker_spec as a string because the JSON serializer will convert
        # the placeholder for worker_replicas (which it sees as a string) into
        # a quoted variable (eg a string) instead of an unquoted variable
        # (number).  If worker_replicas is quoted in the spec, it will break in
        # k8s.  See https://github.com/kubeflow/pipelines/issues/4776
        worker_spec=str(worker),
        ttl_seconds_after_finished=ttl_seconds_after_finished,
        job_timeout_minutes=job_timeout_minutes,
        delete_after_done=delete_after_done
    )


if __name__ == "__main__":
    import kfp.compiler as compiler
    pipeline_file = __file__ + ".tar.gz"
    print(f"Compiling pipeline as {pipeline_file}")
    compiler.Compiler().compile(mnist_train, pipeline_file)

#     # To run:
#     client = kfp.Client()
#     run = client.create_run_from_pipeline_package(
#         pipeline_file,
#         arguments={},
#         run_name="test pytorchjob run"
#     )
#     print(f"Created run {run}")
