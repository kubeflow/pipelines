from typing import NamedTuple
from collections import namedtuple
import kfp
from kfp import dsl
from kfp import components

def get_current_namespace():
    """Returns current namespace if available, else kubeflow"""
    try:
        current_namespace = open(
            "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
        ).read()
    except:
        current_namespace = "kubeflow"
    return current_namespace

@dsl.component(base_image="python:slim")
def create_worker_spec(
    worker_num: int = 0
) -> NamedTuple(
    "CreatWorkerSpec", [("worker_spec", dict)]
): # type: ignore
    """
    Creates pytorch-job worker spec
    """
    worker = {}
    if worker_num > 0:
        worker = {
            "replicas": worker_num,
            "restartPolicy": "OnFailure",
            "template": {
                "metadata": {
                    "annotations": {
                        "sidecar.istio.io/inject": "false"
                    }
                },
                "spec": {
                    "containers": [
                        {
                            "args": [
                                "--backend",
                                "gloo",
                            ],
                            "image": "public.ecr.aws/pytorch-samples/pytorch_dist_mnist:latest",
                            "name": "pytorch",
                            "resources": {
                                "requests": {
                                    "memory": "4Gi",
                                    "cpu": "2000m",
                                    # Uncomment for GPU
                                    # "nvidia.com/gpu": 1,
                                },
                                "limits": {
                                    "memory": "4Gi",
                                    "cpu": "2000m",
                                    # Uncomment for GPU
                                    # "nvidia.com/gpu": 1,
                                },
                            },
                        }
                    ]
                },
            },
        }

    worker_spec_output = namedtuple(
        "MyWorkerOutput", ["worker_spec"]
    )
    return worker_spec_output(worker)

# container component description setting inputs and implementation
@dsl.container_component
def pytorch_job_launcher(
    name: str,
    namespace: str = 'kubeflow',
    version: str = 'v1',
    master_spec: dict = {},
    worker_spec: dict = {},
    job_timeout_minutes: int = 1440,
    delete_after_done: bool = True,
    clean_pod_policy: str = 'Running',
    active_deadline_seconds: int = None,
    backoff_limit: int = None,
    ttl_seconds_after_finished: int = None,
):
    return dsl.ContainerSpec(
        image='quay.io/rh_ee_fwaters/kubeflow-pytorchjob-launcher:v2',
        command=['python', '/ml/launch_pytorchjob.py'],
        args=[
            '--name', name,
            '--namespace', namespace,
            '--version', version,
            '--masterSpec', master_spec,
            '--workerSpec', worker_spec,
            '--jobTimeoutMinutes', job_timeout_minutes,
            '--deleteAfterDone', delete_after_done,
            '--cleanPodPolicy', clean_pod_policy,
            dsl.IfPresentPlaceholder(input_name='active_deadline_seconds', then=['--activeDeadlineSeconds', active_deadline_seconds]),
            dsl.IfPresentPlaceholder(input_name='backoff_limit', then=['--backoffLimit', backoff_limit]),
            dsl.IfPresentPlaceholder(input_name='ttl_seconds_after_finished', then=['--ttlSecondsAfterFinished', ttl_seconds_after_finished])
        ]
    )

@dsl.pipeline(
    name="launch-kubeflow-pytorchjob",
    description="An example to launch pytorch.",
)
def pytorch_job_pipeline():
    pytorch_job = pytorch_job_launcher(
        name="sample-pytorch-job",
        namespace="kubeflow",
        version="v1",
        master_spec={},
        worker_spec={},
        job_timeout_minutes=1440,
        delete_after_done=True,
        clean_pod_policy="Running"
    )

@dsl.component()
def mnist_train(
    namespace: str = get_current_namespace(),
    worker_replicas: int = 1,
    ttl_seconds_after_finished: int = -1,
    job_timeout_minutes: int = 600,
    delete_after_done: bool = False,
):
    pytorchjob_launcher_op = pytorch_job_pipeline

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
                            "--backend",
                            "gloo",
                        ],
                        # Or, create your own image from
                        # https://github.com/kubeflow/pytorch-operator/tree/master/examples/mnist
                        "image": "public.ecr.aws/pytorch-samples/pytorch_dist_mnist:latest",
                        "name": "pytorch",
                        "resources": {
                            "requests": {
                                "memory": "4Gi",
                                "cpu": "2000m",
                                # Uncomment for GPU
                                # "nvidia.com/gpu": 1,
                            },
                            "limits": {
                                "memory": "4Gi",
                                "cpu": "2000m",
                                # Uncomment for GPU
                                # "nvidia.com/gpu": 1,
                            },
                        },
                    }
                ],
                # If imagePullSecrets required
                # "imagePullSecrets": [
                #     {"name": "image-pull-secret"},
                # ],
            },
        },
    }
    create_worker_spec(worker_replicas)

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
        worker_spec=create_worker_spec.outputs[
            "worker_spec"
        ],
        ttl_seconds_after_finished=ttl_seconds_after_finished,
        job_timeout_minutes=job_timeout_minutes,
        delete_after_done=delete_after_done,
    )


if __name__ == "__main__":
    import kfp.compiler as compiler

    pipeline_file = "test.tar.gz"
    print(
        f"Compiling pipeline as {pipeline_file}"
    )
    compiler.Compiler().compile(
        mnist_train, pipeline_file
    )

#     # To run:
#     client = kfp.Client()
#     run = client.create_run_from_pipeline_package(
#         pipeline_file,
#         arguments={},
#         run_name="test pytorchjob run"
#     )
#     print(f"Created run {run}")
