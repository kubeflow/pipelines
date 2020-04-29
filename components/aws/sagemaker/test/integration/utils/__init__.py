import os
import subprocess
import pytest
import tarfile


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
