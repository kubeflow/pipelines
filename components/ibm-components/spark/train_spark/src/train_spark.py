import os
import argparse
import json
from pathlib import Path


def get_secret_creds(path):
    with open(path, 'r') as f:
        cred = f.readline().strip('\'')
    f.close()
    return cred


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--bucket_name', type=str, help='Object storage bucket name', default="dummy-bucket-name")
    parser.add_argument('--data_filename', type=str, help='Name of the data binary', default="")
    parser.add_argument('--model_filename', type=str, help='Name of the training model file', default="model.py")
    parser.add_argument('--spark_entrypoint', type=str, help='Entrypoint command for training the spark model', default="python model.py")
    parser.add_argument('--output_model_file_path', type=str, help='Output path for model file path', default='/tmp/model_filepath')
    parser.add_argument('--output_train_data_file_path', type=str, help='Output path for train data file path', default='/tmp/train_data_filepath')
    args = parser.parse_args()

    cos_bucket_name = args.bucket_name
    data_filename = args.data_filename
    model_filename = args.model_filename
    spark_entrypoint = args.spark_entrypoint

    cos_endpoint = get_secret_creds("/app/secrets/cos_endpoint")
    cos_access_key = get_secret_creds("/app/secrets/cos_access_key")
    cos_secret_key = get_secret_creds("/app/secrets/cos_secret_key")
    tenant_id = get_secret_creds("/app/secrets/spark_tenant_id")
    cluster_master_url = get_secret_creds("/app/secrets/spark_cluster_master_url")
    tenant_secret = get_secret_creds("/app/secrets/spark_tenant_secret")
    instance_id = get_secret_creds("/app/secrets/spark_instance_id")

    ''' Create credentials and vcap files for spark submit'''
    creds = {
      "cos_endpoint": cos_endpoint,
      "cos_access_key": cos_access_key,
      "cos_secret_key": cos_secret_key,
      "bucket_name": cos_bucket_name,
      "data_filename": data_filename,
      "model_filename": model_filename,
      "spark_entrypoint": spark_entrypoint
    }

    with open('creds.json', 'w') as f:
        json.dump(creds, f)
    f.close()

    spark_vcap = {
        "tenant_id": tenant_id,
        "cluster_master_url": cluster_master_url,
        "tenant_secret": tenant_secret,
        "instance_id": instance_id
    }

    with open('vcap.json', 'w') as f:
        json.dump(spark_vcap, f, indent=2)
    f.close()

    os.system('chmod 777 spark-submit.sh')
    os.system('./spark-submit.sh --vcap ./vcap.json --deploy-mode cluster --conf spark.service.spark_version=2.1 --files creds.json  wrapper.py')
    os.system('cat stdout')

    Path(args.output_model_file_path).parent.mkdir(parents=True, exist_ok=True)
    Path(args.output_model_file_path).write_text("model.zip")
    Path(args.output_train_data_file_path).parent.mkdir(parents=True, exist_ok=True)
    Path(args.output_train_data_file_path).write_text("train_data.zip")
