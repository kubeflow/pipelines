import kfp
from kfp import dsl
from typing import Optional
import uuid


def get_current_namespace():
    """Returns current namespace if available, else kubeflow"""
    try:
        current_namespace = open(
            "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
        ).read()
    except:
        current_namespace = "kubeflow"
    return current_namespace


@dsl.component()
def create_master_spec() -> dict:
    # Define master spec
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
                            },
                            "limits": {
                                "memory": "4Gi",
                                "cpu": "2000m",
                            },
                        },
                    }
                ],
            },
        },
    }

    return master


@dsl.component
def create_worker_spec(
    worker_num: int = 0
) -> dict:
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

    return worker

# container component description setting inputs and implementation
@dsl.container_component
def pytorch_job_launcher(
    name: str,
    kind: str = "PyTorchJob",
    namespace: str = "kubeflow",
    version: str = 'v2',
    master_spec: dict = {},
    worker_spec: dict = {},
    job_timeout_minutes: int = 1440,
    delete_after_done: bool = True,
    clean_pod_policy: str = 'Running',
    active_deadline_seconds: Optional[int] = None,
    backoff_limit: Optional[int] = None,
    ttl_seconds_after_finished: Optional[int] = None,
):
    command_args = [
        '--name', name,
        '--kind', kind,
        '--namespace', namespace,
        '--version', version,
        '--masterSpec', master_spec,
        '--workerSpec', worker_spec,
        '--jobTimeoutMinutes', job_timeout_minutes,
        '--deleteAfterDone', delete_after_done,
        '--cleanPodPolicy', clean_pod_policy,]
    if active_deadline_seconds is not None and isinstance(active_deadline_seconds, int):
        command_args.append(['--activeDeadlineSeconds', str(active_deadline_seconds)])
    if backoff_limit is not None and isinstance(backoff_limit, int):
        command_args.append(['--backoffLimit', str(backoff_limit)])
    if ttl_seconds_after_finished is not None and isinstance(ttl_seconds_after_finished, int):
        command_args.append(['--ttlSecondsAfterFinished', str(ttl_seconds_after_finished)])
    
    return dsl.ContainerSpec(
        image='quay.io/rh_ee_fwaters/kubeflow-pytorchjob-launcher:v2',
        command=['python', '/ml/src/launch_pytorchjob.py'],
        args=command_args
    )


@dsl.pipeline(
    name="launch-kubeflow-pytorchjob",
    description="An example to launch pytorch.",
)
def pytorch_job_pipeline(
    kind: str = "PyTorchJob",
    worker_replicas: int = 1,
    ttl_seconds_after_finished: int = 3600,
    job_timeout_minutes: int = 1440,
    delete_after_done: bool = True,
    clean_pod_policy: str ="Running"
):
    
    namespace = get_current_namespace()
    worker_spec = create_worker_spec(worker_num=worker_replicas)
    master_spec = create_master_spec()

    result = pytorch_job_launcher(
        name=f"mnist-train-{uuid.uuid4().hex[:8]}",
        kind=kind,
        namespace=namespace,
        version="v2",
        worker_spec=worker_spec.output,
        master_spec=master_spec.output,
        ttl_seconds_after_finished=ttl_seconds_after_finished,
        job_timeout_minutes=job_timeout_minutes,
        delete_after_done=delete_after_done,
        clean_pod_policy=clean_pod_policy,
    )


if __name__ == "__main__":
    import kfp.compiler as compiler

    pipeline_file = "test.yaml"
    print(
        f"Compiling pipeline as {pipeline_file}"
    )
    compiler.Compiler().compile(
        pytorch_job_pipeline, pipeline_file
    )

    # To run:
    host="http://localhost:8080"
    client = kfp.Client(host=host)
    run = client.create_run_from_pipeline_package(
        pipeline_file,
        arguments={},
        run_name="test pytorchjob run"
    )
    print(f"Created run {run}")
