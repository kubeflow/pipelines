import logging
import re
from datetime import datetime
from time import sleep


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
