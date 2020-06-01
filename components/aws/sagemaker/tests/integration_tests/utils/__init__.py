import os
import subprocess
import pytest
import tarfile
import yaml
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


def get_algorithm_image_registry(region, algorithm):
    return get_image_uri(region, algorithm).split(".")[0]


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


def read_from_file_in_tar(file_path, file_name, decode=True):
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


def load_params(file_name):
    with open(file_name, "r") as f:
        return yaml.safe_load(f)


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
