import utils
import os

from minio import Minio


def get_artifact_in_minio(workflow_json, step_name, artifact_name, output_dir):
    """Minio is the S3 style object storage server for K8s. This method parses
    a pipeline run's workflow json to fetch the output artifact location in
    Minio server for given step in the pipeline and downloads it.

    There are two types of nodes in the workflow_json: DAG and pod. DAG
    corresonds to the whole pipeline and pod corresponds to a step in
    the DAG. Check `node["type"] != "DAG"` deals with case where name of
    component is part of the pipeline name
    """

    s3_data = {}
    minio_access_key = "minio"
    minio_secret_key = "minio123"
    minio_port = utils.get_minio_service_port()
    for node in workflow_json["status"]["nodes"].values():
        if step_name in node["name"] and node["type"] != "DAG":
            for artifact in node["outputs"]["artifacts"]:
                if artifact["name"] == artifact_name:
                    s3_data = artifact["s3"]
    minio_client = Minio(
        "localhost:{}".format(minio_port),
        access_key=minio_access_key,
        secret_key=minio_secret_key,
        secure=False,
    )
    output_file = os.path.join(output_dir, artifact_name + ".tgz")
    minio_client.fget_object(s3_data["bucket"], s3_data["key"], output_file)
    # https://docs.min.io/docs/python-client-api-reference.html#fget_object

    return output_file


def artifact_download_iterator(workflow_json, outputs_dict, output_dir):
    output_files = {}
    for step_name, artifacts in outputs_dict.items():
        output_files[step_name] = {}
        for artifact in artifacts:
            output_files[step_name][artifact] = get_artifact_in_minio(
                workflow_json, step_name, step_name + "-" + artifact, output_dir
            )

    return output_files
