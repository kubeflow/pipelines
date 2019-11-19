import os
from shutil import copyfile
import sys
import json
import re


os.system('pip install Minio --user')

from minio import Minio


# Load Credential file
copyfile('../tmp/creds.json', './creds.json')
with open('creds.json') as f:
    creds = json.load(f)
f.close()

# Remove possible http scheme for Minio
url = re.compile(r"https?://")
cos_endpoint = url.sub('', creds['cos_endpoint'])

# Download the data and model file from the object storage.
cos = Minio(cos_endpoint,
            access_key=creds['cos_access_key'],
            secret_key=creds['cos_secret_key'],
            secure=True)

cos.fget_object(creds['bucket_name'], creds['data_filename'], creds['data_filename'])
cos.fget_object(creds['bucket_name'], creds['model_filename'], creds['model_filename'])

os.system('chmod 755 %s' % creds['model_filename'])
os.system(creds['spark_entrypoint'])
os.system('zip -r model.zip model')
os.system('zip -r train_data.zip train_data')

cos.fput_object(creds['bucket_name'], 'model.zip', 'model.zip')
cos.fput_object(creds['bucket_name'], 'train_data.zip', 'train_data.zip')
cos.fput_object(creds['bucket_name'], 'evaluation.json', 'evaluation.json')

print('Trained model and train_data are uploaded.')
