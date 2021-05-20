# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import json
import boto3
import botocore
import requests
import argparse
import time
from ruamel.yaml import YAML
import subprocess
import os
from pathlib import Path

''' global initialization '''
yaml = YAML(typ='safe')


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--model_def_file_path', type=str, help='Object storage bucket file path for the training model definition')
    parser.add_argument('--manifest_file_path', type=str, help='Object storage bucket file path for the FfDL manifest')
    parser.add_argument('--output_training_id_path', type=str, help='Output path for training id', default='/tmp/training_id.txt')
    args = parser.parse_args()

    with open("/app/secrets/s3_url", 'r') as f:
        s3_url = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/training_bucket", 'r') as f:
        data_bucket_name = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/result_bucket", 'r') as f:
        result_bucket_name = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/s3_access_key_id", 'r') as f:
        s3_access_key_id = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/s3_secret_access_key", 'r') as f:
        s3_secret_access_key = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/ffdl_rest", 'r') as f:
        ffdl_rest = f.readline().strip('\'')
    f.close()

    model_def_file_path = args.model_def_file_path
    manifest_file_path = args.manifest_file_path

    ''' Download FfDL CLI for log streaming '''
    res = requests.get('https://github.com/IBM/FfDL/raw/master/cli/bin/ffdl-linux', allow_redirects=True)
    open('ffdl', 'wb').write(res.content)
    subprocess.call(['chmod', '755', 'ffdl'])

    ''' Download the training model definition and FfDL manifest '''

    client = boto3.resource(
        's3',
        endpoint_url=s3_url,
        aws_access_key_id=s3_access_key_id,
        aws_secret_access_key=s3_secret_access_key,
    )

    try:
        client.Bucket(data_bucket_name).download_file(model_def_file_path, 'model.zip')
        client.Bucket(data_bucket_name).download_file(manifest_file_path, 'manifest.yml')
    except botocore.exceptions.ClientError as e:
        if e.response['Error']['Code'] == "404":
            print("The object does not exist.")
        else:
            raise

    ''' Update FfDL manifest with the corresponding object storage credentials '''

    f = open('manifest.yml', 'r')
    manifest = yaml.safe_load(f.read())
    f.close()

    manifest['data_stores'][0]['connection']['auth_url'] = s3_url
    manifest['data_stores'][0]['connection']['user_name'] = s3_access_key_id
    manifest['data_stores'][0]['connection']['password'] = s3_secret_access_key
    manifest['data_stores'][0]['training_data']['container'] = data_bucket_name
    manifest['data_stores'][0]['training_results']['container'] = result_bucket_name

    f = open('manifest.yml', 'w')
    yaml.default_flow_style = False
    yaml.dump(manifest, f)
    f.close()

    ''' Submit Training job to FfDL and monitor its status '''

    files = {
        'manifest': open('manifest.yml', 'rb'),
        'model_definition': open('model.zip', 'rb')
    }

    headers = {
        'accept': 'application/json',
        'Authorization': 'test',
        'X-Watson-Userinfo': 'bluemix-instance-id=test-user'
    }

    response = requests.post(ffdl_rest + '/v1/models?version=2017-02-13', files=files, headers=headers)
    print(response.json())
    id = response.json()['model_id']

    print('Training job has started, please visit the FfDL UI for more details')

    training_status = 'PENDING'
    os.environ['DLAAS_PASSWORD'] = 'test'
    os.environ['DLAAS_USERNAME'] = 'test-user'
    os.environ['DLAAS_URL'] = ffdl_rest

    while training_status != 'COMPLETED':
        response = requests.get(ffdl_rest + '/v1/models/' + id + '?version=2017-02-13', headers=headers)
        training_status = response.json()['training']['training_status']['status']
        print('Training Status: ' + training_status)
        if training_status == 'COMPLETED':
            Path(args.output_training_id_path).parent.mkdir(parents=True, exist_ok=True)
            Path(args.output_training_id_path).write_text(json.dumps(id))
            exit(0)
        if training_status == 'FAILED':
            print('Training failed. Exiting...')
            exit(1)
        if training_status == 'PROCESSING':
            counter = 0
            process = subprocess.Popen(['./ffdl', 'logs', id, '--follow'], stdout=subprocess.PIPE)
            while True:
                output = process.stdout.readline()
                if output:
                    print(output.strip())
                elif process.poll() is not None:
                    break
                else:
                    counter += 1
                    time.sleep(5)
                    if counter > 5:
                        break
        time.sleep(10)
