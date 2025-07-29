import argparse
import json
import os
import re
from pyspark.sql import SparkSession
from pyspark.ml.pipeline import PipelineModel
from pyspark import SparkConf, SparkContext
from pyspark.ml import Pipeline, Model
from watson_machine_learning_client import WatsonMachineLearningAPIClient
from minio import Minio
from pathlib import Path


def get_secret_creds(path):
    with open(path, 'r') as f:
        cred = f.readline().strip('\'')
    f.close()
    return cred


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--bucket_name', type=str, help='Object storage bucket name', default="dummy-bucket-name")
    parser.add_argument('--model_filepath', type=str, help='Name of the trained spark model packaged as zip', default="model.zip")
    parser.add_argument('--train_data_filepath', type=str, help='Name of the train_data zip', default="train_data.zip")
    parser.add_argument('--aios_manifest_path', type=str, help='Object storage file path for the aios manifest file', default="")
    parser.add_argument('--problem_type', type=str, help='Model problem type', default="BINARY_CLASSIFICATION")
    parser.add_argument('--model_name', type=str, help='model name for the trained model', default="Spark German Risk Model - Final")
    parser.add_argument('--deployment_name', type=str, help='deployment name for the trained model', default="Spark German Risk Deployment - Final")
    parser.add_argument('--output_model_uid_path', type=str, help='Output path for model uid', default='/tmp/model_uid')
    args = parser.parse_args()

    cos_bucket_name = args.bucket_name
    model_filepath = args.model_filepath
    aios_manifest_path = args.aios_manifest_path
    train_data_filepath = args.train_data_filepath
    problem_type = args.problem_type
    MODEL_NAME = args.model_name
    DEPLOYMENT_NAME = args.deployment_name

    wml_url = get_secret_creds("/app/secrets/wml_url")
    wml_instance_id = get_secret_creds("/app/secrets/wml_instance_id")
    wml_apikey = get_secret_creds("/app/secrets/wml_apikey")
    cos_endpoint = get_secret_creds("/app/secrets/cos_endpoint")
    cos_access_key = get_secret_creds("/app/secrets/cos_access_key")
    cos_secret_key = get_secret_creds("/app/secrets/cos_secret_key")

    ''' Remove possible http scheme for Minio '''
    url = re.compile(r"https?://")
    cos_endpoint = url.sub('', cos_endpoint)

    WML_CREDENTIALS = {
                       "url": wml_url,
                       "instance_id": wml_instance_id,
                       "apikey": wml_apikey
                      }
    ''' Load Spark model '''
    cos = Minio(cos_endpoint,
                access_key=cos_access_key,
                secret_key=cos_secret_key,
                secure=True)

    cos.fget_object(cos_bucket_name, model_filepath, model_filepath)
    cos.fget_object(cos_bucket_name, train_data_filepath, train_data_filepath)
    cos.fget_object(cos_bucket_name, 'evaluation.json', 'evaluation.json')
    if aios_manifest_path:
        cos.fget_object(cos_bucket_name, aios_manifest_path, aios_manifest_path)

    os.system('unzip %s' % model_filepath)
    print('model ' + model_filepath + ' is downloaded')
    os.system('unzip %s' % train_data_filepath)
    print('train_data ' + train_data_filepath + ' is downloaded')

    sc = SparkContext()
    model = PipelineModel.load(model_filepath.split('.')[0])
    pipeline = Pipeline(stages=model.stages)
    spark = SparkSession.builder.getOrCreate()
    train_data = spark.read.csv(path=train_data_filepath.split('.')[0], sep=",", header=True, inferSchema=True)

    ''' Remove previous deployed model '''
    wml_client = WatsonMachineLearningAPIClient(WML_CREDENTIALS)
    model_deployment_ids = wml_client.deployments.get_uids()
    deleted_model_id = None
    for deployment_id in model_deployment_ids:
        deployment = wml_client.deployments.get_details(deployment_id)
        model_id = deployment['entity']['deployable_asset']['guid']
        if deployment['entity']['name'] == DEPLOYMENT_NAME:
            print('Deleting deployment id', deployment_id)
            wml_client.deployments.delete(deployment_id)
            print('Deleting model id', model_id)
            wml_client.repository.delete(model_id)
            deleted_model_id = model_id
    wml_client.repository.list_models()

    ''' Save and Deploy model '''
    if aios_manifest_path:
        with open(aios_manifest_path) as f:
            aios_manifest = json.load(f)
            OUTPUT_DATA_SCHEMA = {'fields': aios_manifest['model_schema'], 'type': 'struct'}
        f.close()
    else:
        OUTPUT_DATA_SCHEMA = None

    with open('evaluation.json') as f:
        evaluation = json.load(f)
    f.close()

    if problem_type == 'BINARY_CLASSIFICATION':
        EVALUATION_METHOD = 'binary'
    else:
        EVALUATION_METHOD = 'multiclass'

    ''' Define evaluation threshold '''
    model_props = {
        wml_client.repository.ModelMetaNames.NAME: "{}".format(MODEL_NAME),
        wml_client.repository.ModelMetaNames.EVALUATION_METHOD: EVALUATION_METHOD,
        wml_client.repository.ModelMetaNames.EVALUATION_METRICS: evaluation['metrics']
    }
    if aios_manifest_path:
        model_props[wml_client.repository.ModelMetaNames.OUTPUT_DATA_SCHEMA] = OUTPUT_DATA_SCHEMA

    wml_models = wml_client.repository.get_details()
    model_uid = None
    for model_in in wml_models['models']['resources']:
        if MODEL_NAME == model_in['entity']['name']:
            model_uid = model_in['metadata']['guid']
            break

    if model_uid is None:
        print("Storing model ...")

        published_model_details = wml_client.repository.store_model(model=model, meta_props=model_props, training_data=train_data, pipeline=pipeline)
        model_uid = wml_client.repository.get_model_uid(published_model_details)
        print("Done")
    else:
        print("Model already exist")

    Path(args.output_model_uid_path).parent.mkdir(parents=True, exist_ok=True)
    Path(args.output_model_uid_path).write_text(model_uid)
