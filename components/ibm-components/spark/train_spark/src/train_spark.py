import os
import argparse
import json
import subprocess
import time


def get_secret(path):
    with open(path, 'r') as f:
        cred = f.readline().strip('\'')
    f.close()
    return cred


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--bucket_name', type=str, help='Object storage bucket name', default="dummy-bucket-name")
    parser.add_argument('--data_filename', type=str, help='Name of the data binary', default="credit_risk_training.csv")
    parser.add_argument('--model_filename', type=str, help='Name of the training model file', default="model.py")
    parser.add_argument('--spark_entrypoint', type=str, help='Entrypoint command for training the spark model', default="python model.py")
    args = parser.parse_args()

    cos_bucket_name = args.bucket_name
    data_filename = args.data_filename
    model_filename = args.model_filename
    spark_entrypoint = args.spark_entrypoint

    cos_url = get_secret("/app/secrets/cos_url")
    cos_apikey = get_secret("/app/secrets/cos_apikey")
    cos_resource_instance_id = get_secret("/app/secrets/cos_resource_id")
    tenant_id = get_secret("/app/secrets/spark_tenant_id")
    cluster_master_url = get_secret("/app/secrets/spark_cluster_master_url")
    tenant_secret = get_secret("/app/secrets/spark_tenant_secret")
    instance_id = get_secret("/app/secrets/spark_instance_id")

    ''' Create credentials and vcap files for spark submit'''
    creds = {
      "cos_url": cos_url,
      "cos_apikey": cos_apikey,
      "cos_resource_id": cos_resource_instance_id,
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

    with open("/tmp/spark_model.txt", "w") as report:
        report.write(model_filename)
