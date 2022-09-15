from time import sleep
from kubernetes import client,config
import os

def k8s_client():
    return config.new_client_from_config()

def _get_resource(k8s_client,job_name,kvars):
    """Get the custom resource detail similar to: kubectl describe <resource> JOB_NAME -n NAMESPACE.
    Returns:
        None or object: None if the resource doesnt exist in server, otherwise the
            custom object.
    """
    _api = client.CustomObjectsApi(k8s_client)
    namespace = os.environ.get("NAMESPACE")
    job_description = _api.get_namespaced_custom_object(
        kvars["group"].lower(),
        kvars["version"].lower(),
        namespace.lower(), 
        kvars["plural"].lower(),
        job_name.lower()
    )
    return job_description

def describe_training_job(k8s_client,training_job_name):
    training_vars = {
        "group":"sagemaker.services.k8s.aws",
        "version":"v1alpha1",
        "plural":"trainingjobs",
    }
    return _get_resource(k8s_client,training_job_name,training_vars)

#TODO: Make this a generalized function for non-job resources.
def wait_for_trainingjob_status(k8s_client,training_job_name,desiredStatuses,wait_periods,period_length):
    for _ in range(wait_periods):
        response = describe_training_job(k8s_client,training_job_name)
        if response["status"]["trainingJobStatus"] in desiredStatuses:return True
        sleep(period_length)
    return False
