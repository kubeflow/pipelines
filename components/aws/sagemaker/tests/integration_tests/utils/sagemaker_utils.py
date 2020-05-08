import pytest
from botocore.exceptions import WaiterError


def describe_training_job(client, training_job_name):
    return client.describe_training_job(TrainingJobName=training_job_name)


def describe_model(client, model_name):
    return client.describe_model(ModelName=model_name)


def describe_endpoint(client, endpoint_name):
    return client.describe_endpoint(EndpointName=endpoint_name)


def delete_endpoint(client, endpoint_name):
    client.delete_endpoint(EndpointName=endpoint_name)
    waiter = client.get_waiter("endpoint_deleted")
    try:
        waiter.wait(EndpointName=endpoint_name)
    except WaiterError as e:
        pytest.fail(f"Endpoint - {endpoint_name} deletion failed. Error: {e.__dict__}")


def describe_hpo_job(client, job_name):
    return client.describe_hyper_parameter_tuning_job(
        HyperParameterTuningJobName=job_name
    )


def describe_transform_job(client, job_name):
    return client.describe_transform_job(TransformJobName=job_name)
