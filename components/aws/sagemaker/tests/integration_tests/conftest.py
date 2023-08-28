import pytest
import boto3
import kfp
import os
from utils import sagemaker_utils
import utils

from datetime import datetime
from filelock import FileLock
from sagemaker import image_uris
from botocore.config import Config


def pytest_addoption(parser):
    parser.addoption(
        "--region",
        default="us-west-2",
        required=False,
        help="AWS region where test will run",
    )
    parser.addoption(
        "--sagemaker-role-arn",
        required=True,
        help="SageMaker execution IAM role ARN",
    )
    parser.addoption(
        "--robomaker-role-arn",
        required=True,
        help="RoboMaker execution IAM role ARN",
    )
    parser.addoption(
        "--assume-role-arn",
        required=True,
        help="The ARN of a role which the assume role tests will assume to access SageMaker.",
    )
    parser.addoption(
        "--s3-data-bucket",
        required=True,
        help="Regional S3 bucket name in which test data is hosted",
    )
    parser.addoption(
        "--minio-service-port",
        default="9000",
        required=False,
        help="Localhost port to which minio service is mapped to",
    )
    parser.addoption(
        "--kfp-namespace",
        default="kubeflow",
        required=False,
        help="Cluster namespace where kubeflow pipelines is installed",
    )
    parser.addoption(
        "--fsx-subnet",
        required=False,
        help="The subnet in which FSx is installed",
        default="",
    )
    parser.addoption(
        "--fsx-security-group",
        required=False,
        help="The security group SageMaker should use when running the FSx test",
        default="",
    )
    parser.addoption(
        "--fsx-id",
        required=False,
        help="The file system ID of the FSx instance",
        default="",
    )


@pytest.fixture(scope="session", autouse=True)
def region(request):
    os.environ["AWS_REGION"] = request.config.getoption("--region")
    return request.config.getoption("--region")


@pytest.fixture(scope="session", autouse=True)
def assume_role_arn(request):
    os.environ["ASSUME_ROLE_ARN"] = request.config.getoption("--assume-role-arn")
    return request.config.getoption("--assume-role-arn")


@pytest.fixture(scope="session", autouse=True)
def sagemaker_role_arn(request):
    os.environ["SAGEMAKER_ROLE_ARN"] = request.config.getoption("--sagemaker-role-arn")
    return request.config.getoption("--sagemaker-role-arn")


@pytest.fixture(scope="session", autouse=True)
def robomaker_role_arn(request):
    os.environ["ROBOMAKER_ROLE_ARN"] = request.config.getoption("--robomaker-role-arn")
    return request.config.getoption("--robomaker-role-arn")


@pytest.fixture(scope="session", autouse=True)
def s3_data_bucket(request):
    os.environ["S3_DATA_BUCKET"] = request.config.getoption("--s3-data-bucket")
    return request.config.getoption("--s3-data-bucket")


@pytest.fixture(scope="session", autouse=True)
def minio_service_port(request):
    os.environ["MINIO_SERVICE_PORT"] = request.config.getoption("--minio-service-port")
    return request.config.getoption("--minio-service-port")


@pytest.fixture(scope="session", autouse=True)
def kfp_namespace(request):
    os.environ["NAMESPACE"] = request.config.getoption("--kfp-namespace")
    return request.config.getoption("--kfp-namespace")


@pytest.fixture(scope="session", autouse=True)
def fsx_subnet(request):
    os.environ["FSX_SUBNET"] = request.config.getoption("--fsx-subnet")
    return request.config.getoption("--fsx-subnet")


@pytest.fixture(scope="session", autouse=True)
def fsx_security_group(request):
    os.environ["FSX_SECURITY_GROUP"] = request.config.getoption("--fsx-security-group")
    return request.config.getoption("--fsx-security-group")


@pytest.fixture(scope="session", autouse=True)
def fsx_id(request):
    os.environ["FSX_ID"] = request.config.getoption("--fsx-id")
    return request.config.getoption("--fsx-id")


@pytest.fixture(scope="session")
def boto3_session(region):
    return boto3.Session(region_name=region)


@pytest.fixture(scope="session")
def sagemaker_client(boto3_session):
    return boto3_session.client(service_name="sagemaker")


@pytest.fixture(scope="session")
def robomaker_client(boto3_session):
    return boto3_session.client(service_name="robomaker")


@pytest.fixture(scope="session")
def s3_client(boto3_session):
    return boto3_session.client(service_name="s3")


@pytest.fixture(scope="session")
def kfp_client():
    kfp_installed_namespace = utils.get_kfp_namespace()
    return kfp.Client(namespace=kfp_installed_namespace)


def get_experiment_id(kfp_client):
    exp_name = datetime.now().strftime("%Y-%m-%d-%H-%M")
    try:
        experiment = kfp_client.get_experiment(experiment_name=exp_name)
    except ValueError:
        experiment = kfp_client.create_experiment(name=exp_name)
    return experiment.id


@pytest.fixture(scope="session")
def experiment_id(kfp_client, tmp_path_factory, worker_id):
    if worker_id == "master":
        return get_experiment_id(kfp_client)

    # Locking taking as an example from
    # https://pytest-xdist.readthedocs.io/en/latest/how-to.html#making-session-scoped-fixtures-execute-only-once
    # get the temp directory shared by all workers
    root_tmp_dir = tmp_path_factory.getbasetemp().parent

    fn = root_tmp_dir / "experiment_id"
    with FileLock(str(fn) + ".lock"):
        if fn.is_file():
            data = fn.read_text()
        else:
            data = get_experiment_id(kfp_client)
            fn.write_text(data)
    return data


@pytest.fixture(scope="session")
# Deploy endpoint for testing model monitoring
def deploy_endpoint(sagemaker_client, s3_data_bucket, sagemaker_role_arn, region):
    model_name = "model-monitor-" + utils.generate_random_string(5) + "-model"
    endpoint_name = "model-monitor-" + utils.generate_random_string(5) + "-endpoint-v2"
    endpoint_config_name = (
        "model-monitor-" + utils.generate_random_string(5) + "-endpoint-config"
    )

    # create sagemaker model
    # TODO upgrade to a newer xgboost version
    image_uri = image_uris.retrieve("xgboost", region, "0.90-1")
    create_model_api_response = sagemaker_client.create_model(
        ModelName=model_name,
        PrimaryContainer={
            "Image": image_uri,
            "ModelDataUrl": f"s3://{s3_data_bucket}/model-monitor/xgb-churn-prediction-model.tar.gz",
            "Environment": {},
        },
        ExecutionRoleArn=sagemaker_role_arn,
    )

    # create sagemaker endpoint config
    create_endpoint_config_api_response = sagemaker_client.create_endpoint_config(
        EndpointConfigName=endpoint_config_name,
        ProductionVariants=[
            {
                "VariantName": "variant-1",
                "ModelName": model_name,
                "InitialInstanceCount": 1,
                "InstanceType": "ml.m5.large",
            },
        ],
        DataCaptureConfig={
            "EnableCapture": True,
            "CaptureOptions": [{"CaptureMode": "Input"}, {"CaptureMode": "Output"}],
            "InitialSamplingPercentage": 100,
            "DestinationS3Uri": f"s3://{s3_data_bucket}/model-monitor/datacapture",
        },
    )

    # create sagemaker endpoint
    create_endpoint_api_response = sagemaker_client.create_endpoint(
        EndpointName=endpoint_name,
        EndpointConfigName=endpoint_config_name,
    )

    try:
        sagemaker_client.get_waiter("endpoint_in_service").wait(
            EndpointName=endpoint_name
        )
    finally:
        resp = sagemaker_client.describe_endpoint(EndpointName=endpoint_name)
        endpoint_status = resp["EndpointStatus"]
        endpoint_arn = resp["EndpointArn"]
        print(f"Deployed endpoint {endpoint_arn}, ended with status {endpoint_status}")

        if endpoint_status != "InService":
            message = sagemaker_client.describe_endpoint(EndpointName=endpoint_name)[
                "FailureReason"
            ]
            print(
                "Endpoint deployment failed with the following error: {}".format(
                    message
                )
            )
            raise Exception("Endpoint deployment failed")

    yield endpoint_name

    # delete model and endpoint config
    print("deleting endpoint.................")
    sagemaker_utils.delete_endpoint(sagemaker_client, endpoint_name)
    sagemaker_client.delete_endpoint_config(EndpointConfigName=endpoint_config_name)
    sagemaker_client.delete_model(ModelName=model_name)
