from time import sleep
from kubernetes import client, config
import os


def k8s_client():
    return config.new_client_from_config()


def _get_resource(job_name, plural):
    """Get the custom resource detail similar to: kubectl describe <resource> JOB_NAME -n NAMESPACE.
    Returns:
        None or object: None if the resource doesn't exist in server or there is an error, otherwise the
            custom object.
    """
    # Instantiate a new client every time to avoid connection issues.
    _api = client.CustomObjectsApi(k8s_client())
    namespace = os.environ.get("NAMESPACE")
    try:
        job_description = _api.get_namespaced_custom_object(
            "sagemaker.services.k8s.aws",
            "v1alpha1",
            namespace.lower(),
            plural,
            job_name.lower(),
        )
    except Exception as e:
        print(f"Exception occurred while getting resource {job_name}: {e}")
        return None
    return job_description


def _delete_resource(job_name, plural, wait_periods=10, period_length=20):
    """Delete the custom resource
    Returns:
        True or False: True if the resource is deleted, False if the resource deletion times out
    """
    _api = client.CustomObjectsApi(k8s_client())
    namespace = os.environ.get("NAMESPACE")

    try:
        _api.delete_namespaced_custom_object(
            "sagemaker.services.k8s.aws",
            "v1alpha1",
            namespace.lower(),
            plural,
            job_name.lower(),
        )
    except Exception as e:
        print(f"Exception occurred while deleting resource {job_name}: {e}")

    for _ in range(wait_periods):
        sleep(period_length)
        if _get_resource(job_name, plural) is None:
            print(f"Resource {job_name} deleted successfully.")
            return True

    print(f"Wait for resource deletion timed out, resource name: {job_name}")
    return False


# TODO: Make this a generalized function for non-job resources.
def wait_for_trainingjob_status(
    training_job_name, desiredStatuses, wait_periods, period_length
):
    for _ in range(wait_periods):
        response = _get_resource(training_job_name, "trainingjobs")
        if response["status"]["trainingJobStatus"] in desiredStatuses:
            return True
        sleep(period_length)
    return False


def wait_for_condition(
    resource_name, validator_function, wait_periods=10, period_length=8
):
    for _ in range(wait_periods):
        if not validator_function(resource_name):
            sleep(period_length)
        else:
            return True
    return False


def does_endpoint_exist(endpoint_name):
    try:
        response = _get_resource(endpoint_name, "endpoints")
        if response:
            return True
        if response is None:  # kubernetes module error
            return False
    except:
        return False


def is_endpoint_deleted(endpoint_name):
    response = _get_resource(endpoint_name, "endpoints")
    if response:
        return False
    if response is None:
        return True
