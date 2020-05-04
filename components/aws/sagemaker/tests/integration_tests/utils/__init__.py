import os
import subprocess
import pytest
import tarfile
import yaml

# https://github.com/aws/sagemaker-python-sdk/blob/fbebd805e6efc212211b87eafbff3dc84c1062b9/src/sagemaker/amazon/amazon_estimator.py
KMEANS_REGISTRY = {
    "us-east-1": "382416733822",
    "us-east-2": "404615174143",
    "us-west-2": "174872318107",
    "eu-west-1": "438346466558",
    "eu-central-1": "664544806723",
    "ap-northeast-1": "351501993468",
    "ap-northeast-2": "835164637446",
    "ap-southeast-2": "712309505854",
    "us-gov-west-1": "226302683700",
    "ap-southeast-1": "475088953585",
    "ap-south-1": "991648021394",
    "ca-central-1": "469771592824",
    "eu-west-2": "644912444149",
    "us-west-1": "632365934929",
    "us-iso-east-1": "490574956308",
    "ap-east-1": "286214385809",
    "eu-north-1": "669576153137",
    "eu-west-3": "749696950732",
    "sa-east-1": "855470959533",
    "me-south-1": "249704162688",
    "cn-north-1": "390948362332",
    "cn-northwest-1": "387376663083",
}


XGBOOST_REGISTRY = {
    "us-east-1": "811284229777",
    "us-east-2": "825641698319",
    "us-west-2": "433757028032",
    "eu-west-1": "685385470294",
    "eu-central-1": "813361260812",
    "ap-northeast-1": "501404015308",
    "ap-northeast-2": "306986355934",
    "ap-southeast-2": "544295431143",
    "us-gov-west-1": "226302683700",
    "ap-southeast-1": "475088953585",
    "ap-south-1": "991648021394",
    "ca-central-1": "469771592824",
    "eu-west-2": "644912444149",
    "us-west-1": "632365934929",
    "us-iso-east-1": "490574956308",
    "ap-east-1": "286214385809",
    "eu-north-1": "669576153137",
    "eu-west-3": "749696950732",
    "sa-east-1": "855470959533",
    "me-south-1": "249704162688",
    "cn-north-1": "390948362332",
    "cn-northwest-1": "387376663083",
}


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


def replace_placeholders(file_name):
    region = get_region()
    variables_to_replace = {
        "((REGION))": region,
        "((ROLE_ARN))": get_role_arn(),
        "((DATA_BUCKET))": get_s3_data_bucket(),
        "((KMEANS_REGISTRY))": KMEANS_REGISTRY[region],
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
