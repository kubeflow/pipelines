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
# 
# define the function to deploy the model

def getSecret(secret):
    with open(secret, 'r') as f:
        res = f.readline().strip('\'')
    f.close()
    return res

def deploy(args):
    from watson_machine_learning_client import WatsonMachineLearningAPIClient
    from minio import Minio
    import os
    
    wml_model_name = args.model_name
    wml_scoring_payload = args.scoring_payload
    model_uid = args.model_uid

    # retrieve credentials
    wml_url = getSecret("/app/secrets/wml_url")
    wml_username = getSecret("/app/secrets/wml_username")
    wml_password = getSecret("/app/secrets/wml_password")
    wml_instance_id = getSecret("/app/secrets/wml_instance_id")

    cos_endpoint = getSecret("/app/secrets/cos_endpoint")
    cos_access_key = getSecret("/app/secrets/cos_access_key")
    cos_secret_key = getSecret("/app/secrets/cos_secret_key")

    cos_input_bucket = getSecret("/app/secrets/cos_input_bucket")

    # set up the WML client
    wml_credentials = {
                       "url": wml_url,
                       "username": wml_username,
                       "password": wml_password,
                       "instance_id": wml_instance_id
                      }
    client = WatsonMachineLearningAPIClient( wml_credentials )
        
    # deploy the model
    deployment_name    = wml_model_name
    deployment_desc    = "deployment of %s" %wml_model_name
    deployment       = client.deployments.create( model_uid, deployment_name, deployment_desc )
    scoring_endpoint = client.deployments.get_scoring_url( deployment )
    print( "scoring_endpoint: ", scoring_endpoint )

    # download scoring payload
    payload_file = os.path.join('/app', wml_scoring_payload)
    
    cos = Minio(cos_endpoint,
               access_key = cos_access_key,
               secret_key = cos_secret_key)
    cos.fget_object(cos_input_bucket, wml_scoring_payload, payload_file)

    # scoring the deployment
    import json
    with open( payload_file ) as data_file: 
        test_data = json.load( data_file )
    payload = test_data[ 'payload' ]
    data_file.close()

    print("Scoring result: ")
    result = client.deployments.score( scoring_endpoint, payload )
    print(result)

    with open("/tmp/output", "w") as f:
        print(result, file=f)
    f.close()

if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument('--model-name', type=str, required=True)
    parser.add_argument('--scoring-payload', type=str, required=True)
    parser.add_argument('--model-uid', type=str, required=True)
    args = parser.parse_args()
    deploy(args)
