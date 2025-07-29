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
# define the function to store the model

def getSecret(secret):
    with open(secret, 'r') as f:
        res = f.readline().strip('\'')
    f.close()
    return res

def store(wml_model_name, run_uid, framework, framework_version, runtime_version, output_model_uid_path):
    from watson_machine_learning_client import WatsonMachineLearningAPIClient
    from pathlib import Path

    # retrieve credentials
    wml_url = getSecret("/app/secrets/wml_url")
    wml_instance_id = getSecret("/app/secrets/wml_instance_id")
    wml_apikey = getSecret("/app/secrets/wml_apikey")

    runtime_uid = framework + '_' + framework_version + '-py' + runtime_version
    runtime_type = framework + '_' + framework_version

    print("runtime_uid:", runtime_uid)
    print("runtime_type:", runtime_type)
    # set up the WML client
    wml_credentials = {
                       "url": wml_url,
                       "instance_id": wml_instance_id,
                       "apikey": wml_apikey
                      }
    client = WatsonMachineLearningAPIClient(wml_credentials)

    # store the model
    meta_props_tf = {
     client.repository.ModelMetaNames.NAME: wml_model_name,
     client.repository.ModelMetaNames.RUNTIME_UID: runtime_uid,
     client.repository.ModelMetaNames.TYPE: runtime_type
    }

    model_details = client.repository.store_model(run_uid, meta_props=meta_props_tf)

    model_uid = client.repository.get_model_uid(model_details)
    print("model_uid: ", model_uid)

    Path(output_model_uid_path).parent.mkdir(parents=True, exist_ok=True)
    Path(output_model_uid_path).write_text(model_uid)

    import time
    time.sleep(120)

if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument('--model-name', type=str, required=True)
    parser.add_argument('--run-uid', type=str, required=True)
    parser.add_argument('--framework', type=str, required=True)
    parser.add_argument('--framework-version', type=str, required=True)
    parser.add_argument('--runtime-version', type=str, required=True)
    parser.add_argument('--output-model-uid-path', type=str, default='/tmp/model_uid')
    args = parser.parse_args()
    store(args.model_name,
          args.run_uid,
          args.framework,
          args.framework_version,
          args.runtime_version,
          args.output_model_uid_path)
