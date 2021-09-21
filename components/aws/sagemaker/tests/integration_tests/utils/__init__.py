import json
import os
import subprocess
import pytest
import tarfile
from ruamel.yaml import YAML
import random
import string
import shutil

from sagemaker.image_uris import retrieve


def get_region():
    return os.environ.get("AWS_REGION")


def get_sagemaker_role_arn():
    return os.environ.get("SAGEMAKER_ROLE_ARN")


def get_robomaker_role_arn():
    return os.environ.get("ROBOMAKER_ROLE_ARN")


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


def get_algorithm_image_registry(framework, region, version=None):
    return retrieve(framework, region, version).split(".")[0]


def get_assume_role_arn():
    return os.environ.get("ASSUME_ROLE_ARN")


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
        "((SAGEMAKER_ROLE_ARN))": get_sagemaker_role_arn(),
        "((DATA_BUCKET))": get_s3_data_bucket(),
        "((KMEANS_REGISTRY))": get_algorithm_image_registry("kmeans", region, "1"),
        "((XGBOOST_REGISTRY))": get_algorithm_image_registry(
            "xgboost", region, "1.0-1"
        ),
        "((BUILTIN_RULE_IMAGE))": get_algorithm_image_registry("debugger", region),
        "((FSX_ID))": get_fsx_id(),
        "((FSX_SUBNET))": get_fsx_subnet(),
        "((FSX_SECURITY_GROUP))": get_fsx_security_group(),
        "((ASSUME_ROLE_ARN))": get_assume_role_arn(),
        "((ROBOMAKER_ROLE_ARN))": get_robomaker_role_arn(),
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


def load_params(file_name):
    with open(file_name, "r") as f:
        yaml = YAML(typ="safe")
        return yaml.load(f)


def generate_random_string(length):
    """Generate a random string with twice the length of input parameter."""
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
