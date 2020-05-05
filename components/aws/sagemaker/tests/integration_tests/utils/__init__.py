import os
import subprocess
import pytest
import tarfile
import yaml

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


def get_algorithm_image_registry(region, algorithm):
    return get_image_uri(region, algorithm).split(".")[0]


def run_command(cmd, *popenargs, **kwargs):
    if isinstance(cmd, str):
        cmd = cmd.split(" ")
    try:
        print("executing command: {}".format(" ".join(cmd)))
        return subprocess.check_output(
            cmd, stderr=subprocess.STDOUT, *popenargs, **kwargs
        )
    except subprocess.CalledProcessError as e:
        pytest.fail(f"Command failed. Error code: {e.returncode}, Log: {e.output}")


def extract_information(file_path, file_name):
    with tarfile.open(file_path).extractfile(file_name) as f:
        return f.read()


def replace_placeholders(file_name):
    region = get_region()
    variables_to_replace = {
        "((REGION))": region,
        "((ROLE_ARN))": get_role_arn(),
        "((DATA_BUCKET))": get_s3_data_bucket(),
        "((KMEANS_REGISTRY))": get_algorithm_image_registry(region, "kmeans"),
    }

    filedata = ""
    with open(file_name, "r") as f:
        filedata = f.read()
        for replace_key, replace_value in variables_to_replace.items():
            filedata = filedata.replace(replace_key, replace_value)
    output_filename = file_name + ".generated"
    with open(output_filename, "w") as f:
        f.write(filedata)
    return output_filename


def load_params(file_name):
    with open(file_name, "r") as f:
        return yaml.safe_load(f)
