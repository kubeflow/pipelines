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
    import re

    wml_model_name = args.model_name
    model_uid = args.model_uid
    wml_scoring_payload = args.scoring_payload if args.scoring_payload else ''
    deployment_name = args.deployment_name if args.deployment_name else wml_model_name

    # retrieve credentials
    wml_url = getSecret("/app/secrets/wml_url")
    wml_instance_id = getSecret("/app/secrets/wml_instance_id")
    wml_apikey = getSecret("/app/secrets/wml_apikey")

    # set up the WML client
    wml_credentials = {
                       "url": wml_url,
                       "instance_id": wml_instance_id,
                       "apikey": wml_apikey
                      }
    client = WatsonMachineLearningAPIClient(wml_credentials)

    # deploy the model
    deployment_desc  = "deployment of %s" % wml_model_name
    deployment       = client.deployments.create(model_uid, deployment_name, deployment_desc)
    scoring_endpoint = client.deployments.get_scoring_url(deployment)
    print("scoring_endpoint: ", scoring_endpoint)

    if wml_scoring_payload:
        # download scoring payload if exist
        cos_endpoint = getSecret("/app/secrets/cos_endpoint")
        cos_access_key = getSecret("/app/secrets/cos_access_key")
        cos_secret_key = getSecret("/app/secrets/cos_secret_key")
        cos_input_bucket = getSecret("/app/secrets/cos_input_bucket")

        # Make sure http scheme is not exist for Minio
        url = re.compile(r"https?://")
        cos_endpoint = url.sub('', cos_endpoint)

        payload_file = os.path.join('/app', wml_scoring_payload)

        cos = Minio(cos_endpoint,
                    access_key=cos_access_key,
                    secret_key=cos_secret_key)
        cos.fget_object(cos_input_bucket, wml_scoring_payload, payload_file)

        # scoring the deployment
        import json
        with open(payload_file) as data_file:
            test_data = json.load(data_file)
        payload = test_data['payload']
        data_file.close()

        print("Scoring result: ")
        result = client.deployments.score(scoring_endpoint, payload)
    else:
        result = 'Scoring payload is not provided'

    print(result)
    with open("/tmp/scoring_endpoint", "w") as f:
        print(scoring_endpoint, file=f)
    f.close()
    with open("/tmp/model_uid", "w") as f:
        print(model_uid, file=f)
    f.close()


if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument('--model-name', type=str, required=True)
    parser.add_argument('--model-uid', type=str, required=True)
    parser.add_argument('--deployment-name', type=str)
    parser.add_argument('--scoring-payload', type=str)
    args = parser.parse_args()
    deploy(args)
