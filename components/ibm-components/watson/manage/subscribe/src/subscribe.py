import json
import argparse
from ibm_ai_openscale import APIClient
from ibm_ai_openscale.engines import *
from ibm_ai_openscale.utils import *
from ibm_ai_openscale.supporting_classes import PayloadRecord, Feature
from ibm_ai_openscale.supporting_classes.enums import *
from watson_machine_learning_client import WatsonMachineLearningAPIClient

def get_secret_creds(path):
    with open(path, 'r') as f:
        cred = f.readline().strip('\'')
    f.close()
    return cred

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--aios_schema', type=str, help='AI OpenScale Schema Name', default="data_mart_credit_risk")
    parser.add_argument('--model_name', type=str, help='Deployed model name', default="AIOS Spark German Risk Model - Final")
    parser.add_argument('--model_uid', type=str, help='Deployed model uid', default="dummy uid")
    parser.add_argument('--label_column', type=str, help='Model label column name', default="Risk")
    args = parser.parse_args()

    aios_schema = args.aios_schema
    model_name = args.model_name
    model_uid = args.model_uid
    label_column = args.label_column

    wml_creds = get_secret_creds("/app/secrets/wml_credentials")
    aios_guid = get_secret_creds("/app/secrets/aios_guid")
    cloud_api_key = get_secret_creds("/app/secrets/cloud_api_key")
    postgres_uri = get_secret_creds("/app/secrets/postgres_uri")

    WML_CREDENTIALS = json.loads(wml_creds)

    AIOS_CREDENTIALS = {
        "instance_guid": aios_guid,
        "apikey": cloud_api_key,
        "url": "https://api.aiopenscale.cloud.ibm.com"
    }

    if postgres_uri == '':
        POSTGRES_CREDENTIALS = None
    else:
        POSTGRES_CREDENTIALS = {
            "uri": postgres_uri
        }

    wml_client = WatsonMachineLearningAPIClient(WML_CREDENTIALS)
    ai_client = APIClient(aios_credentials=AIOS_CREDENTIALS)
    print('AIOS client version:' + ai_client.version)

    ''' Setup Postgres SQL and AIOS binding '''
    SCHEMA_NAME = aios_schema
    try:
        data_mart_details = ai_client.data_mart.get_details()
        if 'internal_database' in data_mart_details['database_configuration'] and data_mart_details['database_configuration']['internal_database']:
            if POSTGRES_CREDENTIALS is None:
                print('Using existing internal datamart')
            else:
                print('Switching to external datamart')
                ai_client.data_mart.delete(force=True)
                create_postgres_schema(postgres_credentials=POSTGRES_CREDENTIALS, schema_name=SCHEMA_NAME)
                ai_client.data_mart.setup(db_credentials=POSTGRES_CREDENTIALS, schema=SCHEMA_NAME)
        else:
            print('Using existing external datamart')
    except:
        if POSTGRES_CREDENTIALS is None:
            print('Setting up internal datamart')
            ai_client.data_mart.setup(internal_db=True)
        else:
            print('Setting up external datamart')
            create_postgres_schema(postgres_credentials=POSTGRES_CREDENTIALS, schema_name=SCHEMA_NAME)
            ai_client.data_mart.setup(db_credentials=POSTGRES_CREDENTIALS, schema=SCHEMA_NAME)

    data_mart_details = ai_client.data_mart.get_details()
    print(data_mart_details)

    binding_uid = ai_client.data_mart.bindings.add('WML instance', WatsonMachineLearningInstance(WML_CREDENTIALS))
    if binding_uid is None:
        binding_uid = ai_client.data_mart.bindings.get_details()['service_bindings'][0]['metadata']['guid']
    bindings_details = ai_client.data_mart.bindings.get_details()

    print('\nWML binding ID is ' + binding_uid + '\n')

    ''' Create subscriptions '''
    subscriptions_uids = ai_client.data_mart.subscriptions.get_uids()
    for subscription in subscriptions_uids:
        sub_name = ai_client.data_mart.subscriptions.get_details(subscription)['entity']['asset']['name']
        if sub_name == model_name:
            ai_client.data_mart.subscriptions.delete(subscription)
            print('Deleted existing subscription for', model_name)

    subscription = ai_client.data_mart.subscriptions.add(WatsonMachineLearningAsset(
        model_uid,
        label_column=label_column,
        prediction_column='predictedLabel',
        probability_column='probability'
    ))
    if subscription is None:
        print('Exists already')
        # subscription already exists; get the existing one
        subscriptions_uids = ai_client.data_mart.subscriptions.get_uids()
        for sub in subscriptions_uids:
            if ai_client.data_mart.subscriptions.get_details(sub)['entity']['asset']['name'] == model_name:
                subscription = ai_client.data_mart.subscriptions.get(sub)

    subscriptions_uids = ai_client.data_mart.subscriptions.get_uids()
    print(subscription.get_details())

    ''' Scoring the model and make sure the subscriptions are setup properly '''
    credit_risk_scoring_endpoint = None
    deployment_uid = subscription.get_deployment_uids()[0]

    print('\n' + deployment_uid + '\n')

    for deployment in wml_client.deployments.get_details()['resources']:
        if deployment_uid in deployment['metadata']['guid']:
            credit_risk_scoring_endpoint = deployment['entity']['scoring_url']

    print('Scoring endpoint is: ' + credit_risk_scoring_endpoint + '\n')

    with open("/tmp/model_name.txt", "w") as report:
        report.write(model_name)
