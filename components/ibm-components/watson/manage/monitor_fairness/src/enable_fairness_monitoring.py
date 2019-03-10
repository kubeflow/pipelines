import json
import argparse
import ibm_boto3
from ibm_botocore.client import Config
from ibm_ai_openscale import APIClient
from ibm_ai_openscale.engines import *
from ibm_ai_openscale.utils import *
from ibm_ai_openscale.supporting_classes import PayloadRecord, Feature
from ibm_ai_openscale.supporting_classes.enums import *

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--model_name', type=str, help='Deployed model name', default='AIOS Spark German Risk Model - Final')
    parser.add_argument('--fairness_threshold', type=float, help='Amount of threshold for fairness monitoring', default=0.95)
    parser.add_argument('--fairness_min_records', type=int, help='Minimum amount of records for performing a fairness monitor', default=5)
    parser.add_argument('--aios_manifest_path', type=str, help='Object storage file path for the aios manifest file', default='aios.json')
    parser.add_argument('--cos_bucket_name', type=str, help='Object storage bucket name', default='bucket-name')
    args = parser.parse_args()

    model_name = args.model_name
    fairness_threshold = args.fairness_threshold
    fairness_min_records = args.fairness_min_records
    cos_bucket_name = args.cos_bucket_name
    aios_manifest_path = args.aios_manifest_path

    with open("/app/secrets/aios_guid", 'r') as f:
        aios_guid = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/cloud_api_key", 'r') as f:
        cloud_api_key = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/cos_url", 'r') as f:
        cos_url = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/cos_apikey", 'r') as f:
        cos_apikey = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/cos_resource_id", 'r') as f:
        cos_resource_instance_id = f.readline().strip('\'')
    f.close()

    ''' Upload data to IBM Cloud object storage '''
    cos = ibm_boto3.resource('s3',
                             ibm_api_key_id=cos_apikey,
                             ibm_service_instance_id=cos_resource_instance_id,
                             ibm_auth_endpoint='https://iam.bluemix.net/oidc/token',
                             config=Config(signature_version='oauth'),
                             endpoint_url=cos_url)

    cos.Bucket(cos_bucket_name).download_file(aios_manifest_path, 'aios.json')

    print('Fairness definition file ' + aios_manifest_path + ' is downloaded')

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
            prediction_column='predictedLabel',
            favourable_classes=aios_manifest['fairness_favourable_classes'],
            unfavourable_classes=aios_manifest['fairness_unfavourable_classes'],
            min_records=fairness_min_records
        )

    run_details = subscription.fairness_monitoring.run()
    print('Fairness monitoring is enabled.')
