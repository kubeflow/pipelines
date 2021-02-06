import pytest
import boto3
import kfp
import os
import utils

from datetime import datetime
from filelock import FileLock


def pytest_addoption(parser):
    parser.addoption(
        "--region",
        default="us-west-2",
        required=False,
        help="AWS region where test will run",
    )
    parser.addoption(
        "--sagemaker-role-arn", required=True, help="SageMaker execution IAM role ARN",
    )
    parser.addoption(
        "--robomaker-role-arn", required=True, help="RoboMaker execution IAM role ARN",
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
    # https://github.com/pytest-dev/pytest-xdist#making-session-scoped-fixtures-execute-only-once
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
