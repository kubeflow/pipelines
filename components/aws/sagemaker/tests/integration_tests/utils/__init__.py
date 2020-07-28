import os
import subprocess
import pytest
import tarfile
from ruamel.yaml import YAML
import random
import string
import shutil

from sagemaker.amazon.amazon_estimator import get_image_uri


def get_region():
    return os.environ.get("AWS_REGION")


def get_role_arn():
    return os.environ.get("ROLE_ARN")


def get_s3_data_bucket():
    return os.environ.get("S3_DATA_BUCKET")


def get_minio_service_port():
    return os.environ.get("MINIO_SERVICE_PORT")


def get_kfp_namespace():
    return os.environ.get("NAMESPACE")


def get_fsx_subnet():
    return os.environ.get("FSX_SUBNET")


def get_fsx_security_group():
    return os.environ.get("FSX_SECURITY_GROUP")


def get_fsx_id():
    return os.environ.get("FSX_ID")


def get_algorithm_image_registry(region, algorithm, repo_version=1):
    return get_image_uri(region, algorithm, repo_version).split(".")[0]


def run_command(cmd, *popenargs, **kwargs):
    if isinstance(cmd, str):
        cmd = cmd.split(" ")
    try:
        print("executing command: {}".format(" ".join(cmd)))
        return subprocess.check_output(
            cmd, *popenargs, stderr=subprocess.STDOUT, **kwargs
        )
    except subprocess.CalledProcessError as e:
        pytest.fail(f"Command failed. Error code: {e.returncode}, Log: {e.output}")


def read_from_file_in_tar(file_path, file_name="data", decode=True):
    """Opens a local tarball and reads the contents of the file as specified.
    Arguments:
    - file_path: The local path of the tarball file.
    - file_name: The name of the file inside the tarball to be read. (Default `"data"`)
    - decode: Ensures the contents of the file is decoded to type `str`. (Default `True`)

    See: https://github.com/kubeflow/pipelines/blob/2e14fe732b3f878a710b16d1a63beece6c19330a/sdk/python/kfp/components/_components.py#L182
    """
    with tarfile.open(file_path).extractfile(file_name) as f:
        if decode:
            return f.read().decode()
        else:
            return f.read()


def replace_placeholders(input_filename, output_filename):
    region = get_region()
    variables_to_replace = {
        "((REGION))": region,
        "((ROLE_ARN))": get_role_arn(),
        "((DATA_BUCKET))": get_s3_data_bucket(),
        "((KMEANS_REGISTRY))": get_algorithm_image_registry(region, "kmeans"),
        "((XGBOOST_REGISTRY))": get_algorithm_image_registry(region, "xgboost", "1.0-1"),
        "((BUILTIN_RULE_REGISTRY))": get_rule_image_registry(region, "built-in"),
        "((FSX_ID))": get_fsx_id(),
        "((FSX_SUBNET))": get_fsx_subnet(),
        "((FSX_SECURITY_GROUP))": get_fsx_security_group(),
    }

    filedata = ""
    with open(input_filename, "r") as f:
        filedata = f.read()
        for replace_key, replace_value in variables_to_replace.items():
            if replace_value is None:
                continue
            filedata = filedata.replace(replace_key, replace_value)

    with open(output_filename, "w") as f:
        f.write(filedata)
    return output_filename


def get_rule_image_registry(region, rule_type):
    region_to_account = {}
    if rule_type == "built-in":
        region_to_account = {
            "ap-east-1": "199566480951",
            "ap-northeast-1": "430734990657",
            "ap-northeast-2": "578805364391",
            "ap-south-1": "904829902805",
            "ap-southeast-1": "972752614525",
            "ap-southeast-2": "184798709955",
            "ca-central-1": "519511493484",
            "cn-north-1": "618459771430",
            "cn-northwest-1": "658757709296",
            "eu-central-1": "482524230118",
            "eu-north-1": "314864569078",
            "eu-west-1": "929884845733",
            "eu-west-2": "250201462417",
            "eu-west-3": "447278800020",
            "me-south-1": "986000313247",
            "sa-east-1": "818342061345",
            "us-east-1": "503895931360",
            "us-east-2": "915447279597",
            "us-west-1": "685455198987",
            "us-west-2": "895741380848",
            "us-gov-west-1": "515509971035"
        }
    elif rule_type == "custom":
        region_to_account = {
            "ap-east-1": "645844755771",
            "ap-northeast-1": "670969264625",
            "ap-northeast-2": "326368420253",
            "ap-south-1": "552407032007",
            "ap-southeast-1": "631532610101",
            "ap-southeast-2": "445670767460",
            "ca-central-1": "105842248657",
            "cn-north-1": "617202126805",
            "cn-northwest-1": "658559488188",
            "eu-central-1": "691764027602",
            "eu-north-1": "091235270104",
            "eu-west-1": "606966180310",
            "eu-west-2": "074613877050",
            "eu-west-3": "224335253976",
            "me-south-1": "050406412588",
            "sa-east-1": "466516958431",
            "us-east-1": "864354269164",
            "us-east-2": "840043622174",
            "us-west-1": "952348334681",
            "us-west-2": "759209512951",
            "us-gov-west-1": "515361955729"
        }
    else:
        raise ValueError(
            "Rule type: {} does not have mapping to account_id with images".format(rule_type)
        )

    if region in region_to_account:
        return region_to_account[region]

    raise ValueError(
        "Rule type ({rule_type}) is unsupported for region ({region}).".format(
            rule_type=rule_type, region=region
        )
    )
    

def load_params(file_name):
    with open(file_name, "r") as f:
        yaml = YAML(typ="safe")
        return yaml.load(f)


def generate_random_string(length):
    """Generate a random string with twice the length of input parameter"""
    assert isinstance(length, int)
    return "".join(
        [random.choice(string.ascii_lowercase) for n in range(length)]
        + [random.choice(string.digits) for n in range(length)]
    )


def mkdir(directory_path):
    if not os.path.exists(directory_path):
        os.makedirs(directory_path)
    return directory_path


def remove_dir(dir_path):
    shutil.rmtree(dir_path)
