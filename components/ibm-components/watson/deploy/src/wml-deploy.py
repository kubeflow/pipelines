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

    client.deployments.list()
    client.deployments.delete("ad489b8c-f821-47fc-a69c-da5d6bb8479d")
    client.deployments.delete("04ed6bdb-9e88-4868-af7e-4feb57014596")
    client.deployments.delete("a774a043-b91b-485e-8eab-4c5cb6970b50")
    client.deployments.delete("d4e6d1a4-b3e7-44e0-a4ab-a260228a0f17")
    client.deployments.delete("e06477d4-13c4-428d-b226-61498bd87c2e")

    # deploy the model
    #deployment_desc  = "deployment of %s" % wml_model_name
    meta_props = {
        client.deployments.ConfigurationMetaNames.NAME: deployment_name,
        client.deployments.ConfigurationMetaNames.ONLINE: {}
    }
    deployment_details = client.deployments.create(model_uid, meta_props)
    #deployment       = client.deployments.create(model_uid, deployment_name, deployment_desc)
    scoring_endpoint = client.deployments.get_scoring_href(deployment_details)
    deployment_uid = client.deployments.get_uid(deployment_details)
    print("deployment_uid: ", deployment_uid)

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
        payload = {client.deployments.ScoringMetaNames.INPUT_DATA: [test_data['payload']]}
        data_file.close()

        print("Scoring result: ")
        result = client.deployments.score(deployment_uid, payload)
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
