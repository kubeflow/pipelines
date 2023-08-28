import logging
import re
from datetime import datetime
from time import sleep
import os
import pickle
import gzip
import io
import numpy
import json

from utils import get_s3_data_bucket


def describe_training_job(client, training_job_name):
    return client.describe_training_job(TrainingJobName=training_job_name)


def describe_model(client, model_name):
    return client.describe_model(ModelName=model_name)


def describe_endpoint(client, endpoint_name):
    return client.describe_endpoint(EndpointName=endpoint_name)


def list_endpoints(client, name_contains):
    return client.list_endpoints(NameContains=name_contains)


def describe_endpoint_config(client, endpoint_config_name):
    return client.describe_endpoint_config(EndpointConfigName=endpoint_config_name)


def delete_endpoint(client, endpoint_name):
    client.delete_endpoint(EndpointName=endpoint_name)
    waiter = client.get_waiter("endpoint_deleted")
    waiter.wait(EndpointName=endpoint_name)


def describe_monitoring_schedule(client, monitoring_schedule_name):
    return client.describe_monitoring_schedule(
        MonitoringScheduleName=monitoring_schedule_name
    )


def describe_data_quality_job_definition(client, job_definition_name):
    return client.describe_data_quality_job_definition(
        JobDefinitionName=job_definition_name
    )


def describe_hpo_job(client, job_name):
    return client.describe_hyper_parameter_tuning_job(
        HyperParameterTuningJobName=job_name
    )


def describe_transform_job(client, job_name):
    return client.describe_transform_job(TransformJobName=job_name)


def describe_workteam(client, workteam_name):
    return client.describe_workteam(WorkteamName=workteam_name)


def list_workteams(client):
    return client.list_workteams()


def get_cognito_member_definitions(client):
    # This is one way to get the user_pool and client_id for the SageMaker Workforce.
    # An alternative would be to take these values as user input via params or a config file.
    # The current mechanism expects that there exists atleast one private workteam in the region.
    default_workteam = list_workteams(client)["Workteams"][0]["MemberDefinitions"][0][
        "CognitoMemberDefinition"
    ]
    return (
        default_workteam["UserPool"],
        default_workteam["ClientId"],
        default_workteam["UserGroup"],
    )


def list_labeling_jobs_for_workteam(client, workteam_arn):
    return client.list_labeling_jobs_for_workteam(WorkteamArn=workteam_arn)


def describe_labeling_job(client, labeling_job_name):
    return client.describe_labeling_job(LabelingJobName=labeling_job_name)


def get_workteam_arn(client, workteam_name):
    response = describe_workteam(client, workteam_name)
    return response["Workteam"]["WorkteamArn"]


def delete_workteam(client, workteam_name):
    client.delete_workteam(WorkteamName=workteam_name)


def stop_labeling_job(client, labeling_job_name):
    client.stop_labeling_job(LabelingJobName=labeling_job_name)


def describe_processing_job(client, processing_job_name):
    return client.describe_processing_job(ProcessingJobName=processing_job_name)


def run_predict_mnist(boto3_session, endpoint_name, download_dir):
    """https://github.com/awslabs/amazon-sagemaker-
    examples/blob/a8c20eeb72dc7d3e94aaaf28be5bf7d7cd5695cb.

    /sagemaker-python-sdk/1P_kmeans_lowlevel/kmeans_mnist_lowlevel.ipynb
    """
    # Download and load dataset
    region = boto3_session.region_name
    download_path = os.path.join(download_dir, "mnist.pkl.gz")
    boto3_session.resource("s3", region_name=region).Bucket(
        get_s3_data_bucket()
    ).download_file("algorithms/mnist.pkl.gz", download_path)
    with gzip.open(download_path, "rb") as f:
        train_set, valid_set, test_set = pickle.load(f, encoding="latin1")

    # Function to create a csv from numpy array
    def np2csv(arr):
        csv = io.BytesIO()
        numpy.savetxt(csv, arr, delimiter=",", fmt="%g")
        return csv.getvalue().decode().rstrip()

    # Run prediction on an image
    runtime = boto3_session.client("sagemaker-runtime")
    payload = np2csv(train_set[0][30:31])

    response = runtime.invoke_endpoint(
        EndpointName=endpoint_name,
        ContentType="text/csv",
        Body=payload,
    )
    return json.loads(response["Body"].read().decode())
