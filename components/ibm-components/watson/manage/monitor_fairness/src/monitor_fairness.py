import json
import argparse
import re
from ibm_ai_openscale import APIClient
from ibm_ai_openscale.engines import *
from ibm_ai_openscale.utils import *
from ibm_ai_openscale.supporting_classes import PayloadRecord, Feature
from ibm_ai_openscale.supporting_classes.enums import *
from minio import Minio
import pandas as pd

def get_secret_creds(path):
    with open(path, 'r') as f:
        cred = f.readline().strip('\'')
    f.close()
    return cred

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--model_name', type=str, help='Deployed model name', default='AIOS Spark German Risk Model - Final')
    parser.add_argument('--fairness_threshold', type=float, help='Amount of threshold for fairness monitoring', default=0.95)
    parser.add_argument('--fairness_min_records', type=int, help='Minimum amount of records for performing a fairness monitor', default=5)
    parser.add_argument('--aios_manifest_path', type=str, help='Object storage file path for the aios manifest file', default='aios.json')
    parser.add_argument('--cos_bucket_name', type=str, help='Object storage bucket name', default='bucket-name')
    parser.add_argument('--data_filename', type=str, help='Name of the data binary', default="")
    args = parser.parse_args()

    model_name = args.model_name
    fairness_threshold = args.fairness_threshold
    fairness_min_records = args.fairness_min_records
    cos_bucket_name = args.cos_bucket_name
    aios_manifest_path = args.aios_manifest_path
    data_filename = args.data_filename

    aios_guid = get_secret_creds("/app/secrets/aios_guid")
    cloud_api_key = get_secret_creds("/app/secrets/cloud_api_key")
    cos_endpoint = get_secret_creds("/app/secrets/cos_endpoint")
    cos_access_key = get_secret_creds("/app/secrets/cos_access_key")
    cos_secret_key = get_secret_creds("/app/secrets/cos_secret_key")

    ''' Remove possible http scheme for Minio '''
    url = re.compile(r"https?://")
    cos_endpoint = url.sub('', cos_endpoint)

    ''' Upload data to Cloud object storage '''
    cos = Minio(cos_endpoint,
                access_key=cos_access_key,
                secret_key=cos_secret_key,
                secure=True)

    cos.fget_object(cos_bucket_name, aios_manifest_path, 'aios.json')
    print('Fairness definition file ' + aios_manifest_path + ' is downloaded')

    cos.fget_object(cos_bucket_name, data_filename, data_filename)
    pd_data = pd.read_csv(data_filename, sep=",", header=0, engine='python')
    print('training data ' + data_filename + ' is downloaded and loaded')

    """ Load manifest JSON file """
    with open('aios.json') as f:
        aios_manifest = json.load(f)

    """ Initiate AIOS client """

    AIOS_CREDENTIALS = {
        "instance_guid": aios_guid,
        "apikey": cloud_api_key,
        "url": "https://api.aiopenscale.cloud.ibm.com"
    }

    ai_client = APIClient(aios_credentials=AIOS_CREDENTIALS)
    print('AIOS client version:' + ai_client.version)

    ''' Setup fairness monitoring '''
    subscriptions_uids = ai_client.data_mart.subscriptions.get_uids()
    for sub in subscriptions_uids:
        if ai_client.data_mart.subscriptions.get_details(sub)['entity']['asset']['name'] == model_name:
            subscription = ai_client.data_mart.subscriptions.get(sub)

    feature_list = []
    for feature in aios_manifest['fairness_features']:
        feature_list.append(Feature(feature['feature_name'], majority=feature['majority'], minority=feature['minority'], threshold=feature['threshold']))

    subscription.fairness_monitoring.enable(
            features=feature_list,
            favourable_classes=aios_manifest['fairness_favourable_classes'],
            unfavourable_classes=aios_manifest['fairness_unfavourable_classes'],
            min_records=fairness_min_records,
            training_data=pd_data
        )

    run_details = subscription.fairness_monitoring.run()
    print('Fairness monitoring is enabled.')
