import json
import argparse
from ibm_ai_openscale import APIClient
from ibm_ai_openscale.engines import *
from ibm_ai_openscale.utils import *
from ibm_ai_openscale.supporting_classes import PayloadRecord, Feature
from ibm_ai_openscale.supporting_classes.enums import *

def get_secret_creds(path):
    with open(path, 'r') as f:
        cred = f.readline().strip('\'')
    f.close()
    return cred

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--model_name', type=str, help='Deployed model name', default="AIOS Spark German Risk Model - Final")
    parser.add_argument('--quality_threshold', type=float, help='Amount of threshold for quality monitoring', default=0.7)
    parser.add_argument('--quality_min_records', type=int, help='Minimum amount of records for performing a quality monitor', default=5)
    args = parser.parse_args()

    model_name = args.model_name
    quality_threshold = args.quality_threshold
    quality_min_records = args.quality_min_records

    aios_guid = get_secret_creds("/app/secrets/aios_guid")
    cloud_api_key = get_secret_creds("/app/secrets/cloud_api_key")

    AIOS_CREDENTIALS = {
        "instance_guid": aios_guid,
        "apikey": cloud_api_key,
        "url": "https://api.aiopenscale.cloud.ibm.com"
    }

    ai_client = APIClient(aios_credentials=AIOS_CREDENTIALS)
    print('AIOS client version:' + ai_client.version)

    ''' Setup quality monitoring '''
    subscriptions_uids = ai_client.data_mart.subscriptions.get_uids()
    for sub in subscriptions_uids:
        if ai_client.data_mart.subscriptions.get_details(sub)['entity']['asset']['name'] == model_name:
            subscription = ai_client.data_mart.subscriptions.get(sub)

    subscription.quality_monitoring.enable(threshold=quality_threshold, min_records=quality_min_records)
    # Runs need to post the minial payload records in order to trigger the monitoring run.
    # run_details = subscription.quality_monitoring.run()

    print('Quality monitoring is enabled.')
