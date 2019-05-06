import os
from shutil import copyfile
import sys
import json
import re

# Install ibm cos package if the spark kernel is not running on IBM service.
# call_command('pip install ibm-cos-sdk --user')

import ibm_boto3
from ibm_botocore.client import Config


# Load Credential file
copyfile('../tmp/creds.json', './creds.json')
with open('creds.json') as f:
    creds = json.load(f)
f.close()


# Download the data and model file from the object storage.
cos = ibm_boto3.resource('s3',
                         ibm_api_key_id=creds['cos_apikey'],
                         ibm_service_instance_id=creds['cos_resource_id'],
                         ibm_auth_endpoint='https://iam.bluemix.net/oidc/token',
                         config=Config(signature_version='oauth'),
                         endpoint_url=creds['cos_url'])

cos.Bucket(creds['bucket_name']).download_file(creds['data_filename'], creds['data_filename'])
cos.Bucket(creds['bucket_name']).download_file(creds['model_filename'], creds['model_filename'])

os.system('chmod 755 %s' % creds['model_filename'])
os.system(creds['spark_entrypoint'])
os.system('zip -r model.zip model')
os.system('zip -r pipeline.zip pipeline')
os.system('zip -r train_data.zip train_data')

cos.Bucket(creds['bucket_name']).upload_file('model.zip', 'model.zip')
cos.Bucket(creds['bucket_name']).upload_file('pipeline.zip', 'pipeline.zip')
cos.Bucket(creds['bucket_name']).upload_file('train_data.zip', 'train_data.zip')
cos.Bucket(creds['bucket_name']).upload_file('evaluation.json', 'evaluation.json')

print('Trained model, pipeline, and train_data are uploaded.')
