from time import sleep
from kubernetes import client, config
import os


def k8s_client():
    return config.new_client_from_config()


def _get_resource(k8s_client, job_name, plural):
    """Get the custom resource detail similar to: kubectl describe <resource> JOB_NAME -n NAMESPACE.
    Returns:
        None or object: None if the resource doesnt exist in server, otherwise the
            custom object.
    """
    _api = client.CustomObjectsApi(k8s_client)
    namespace = os.environ.get("NAMESPACE")
    job_description = _api.get_namespaced_custom_object(
        "sagemaker.services.k8s.aws",
        "v1alpha1",
        namespace.lower(),
        plural,
        job_name.lower(),
    )
    return job_description


def _delete_resource(k8s_client, job_name, plural):
    """Delete the custom resource
    Returns:
        None or object: None if the resource doesnt exist in server, otherwise the
            custom object.
    """
    _api = client.CustomObjectsApi(k8s_client)
    namespace = os.environ.get("NAMESPACE")
    try:
        _api.delete_namespaced_custom_object(
            "sagemaker.services.k8s.aws",
            "v1alpha1",
            namespace.lower(),
            plural,
            job_name.lower(),
        )
    except:
        return False
    return True


# TODO: Make this a generalized function for non-job resources.
def wait_for_trainingjob_status(
    k8s_client, training_job_name, desiredStatuses, wait_periods, period_length
):
    for _ in range(wait_periods):
        response = _get_resource(k8s_client, training_job_name, "trainingjobs")
        if response["status"]["trainingJobStatus"] in desiredStatuses:
            return True
        sleep(period_length)
    return False


def wait_for_condition(k8s_client, resource_name, validator_function, wait_periods=10, period_length=8):
    for _ in range(wait_periods):
        if not validator_function(k8s_client, resource_name):
            sleep(period_length)
        else:
            return True
    return False

def does_endpoint_exist(
    k8s_client, endpoint_name
):
    try:
        response = _get_resource(k8s_client, endpoint_name, "endpoints")
        if response:
            return True
    except:
        return False
def is_endpoint_deleted(
    k8s_client, endpoint_name
):
    try:
        response = _get_resource(k8s_client, endpoint_name, "endpoints")
        if response:
            return False
    except:
        return True
