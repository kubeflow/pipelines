import kfp
from kfp import dsl
import uuid
import datetime
import logging
from kubernetes import config
from kubeflow.training import TrainingClient
from kubeflow.training.utils import utils


def get_current_namespace():
    """Returns current namespace if available, else kubeflow"""
    try:
        current_namespace = open(
            "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
        ).read()
    except:
        current_namespace = "kubeflow"
    return current_namespace


@dsl.component(packages_to_install=['kubernetes<31,>=8.0.0', 'kubeflow-training>=1.8.0', 'retrying>=1.3.3'], base_image="python:3.11")
def pytorch_job_launcher(
    name: str,
    kind: str = "PyTorchJob",
    namespace: str = "kubeflow",
    worker_replicas: int = 1,
    job_timeout_minutes: int = 1440,
    delete_after_done: bool = True,
):
    args = ["--backend","gloo"]
    resources = {"cpu": "2000m", "memory": "4Gi"} # add "nvidia.com/gpu": 1, for GPU
    base_image="public.ecr.aws/pytorch-samples/pytorch_dist_mnist:latest"
    
    container_spec = utils.get_container_spec(base_image=base_image, name="pytorch", resources=resources, args=args)
    spec = utils.get_pod_template_spec(containers=[container_spec])
    job_template = utils.get_pytorchjob_template(name=name, namespace=namespace, num_workers=worker_replicas, worker_pod_template_spec=spec, master_pod_template_spec=spec)


    logging.getLogger(__name__).setLevel(logging.INFO)
    logging.info('Generating job template.')

    logging.info('Creating TrainingClient.')

    # remove one of these depending on where you are running this
    config.load_incluster_config()
    #config.load_kube_config()
    
    training_client = TrainingClient()

    logging.info(f"Creating PyTorchJob in namespace: {namespace}")
    training_client.create_job(job_template, namespace=namespace)

    expected_conditions = ["Succeeded", "Failed"]
    logging.info(
        f'Monitoring job until status is any of {expected_conditions}.'
    )
    training_client.wait_for_job_conditions(
        name=name,
        namespace=namespace,
        job_kind=kind,
        expected_conditions=set(expected_conditions),
        timeout=int(datetime.timedelta(minutes=job_timeout_minutes).total_seconds())
    )
    if delete_after_done:
        logging.info('Deleting job after completion.')
        training_client.delete_job(name, namespace)


@dsl.pipeline(
    name="launch-kubeflow-pytorchjob",
    description="An example to launch pytorch.",
)
def pytorch_job_pipeline(
    kind: str = "PyTorchJob",
    worker_replicas: int = 1,
):
    
    namespace = get_current_namespace()

    result = pytorch_job_launcher(
        name=f"mnist-train-{uuid.uuid4().hex[:8]}",
        kind=kind,
        namespace=namespace,
        version="v1",
        worker_replicas=worker_replicas
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
