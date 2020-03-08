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

def store(wml_model_name, run_uid, training_uid):
    from watson_machine_learning_client import WatsonMachineLearningAPIClient

    # retrieve credentials
    wml_url = getSecret("/app/secrets/wml_url")
    wml_instance_id = getSecret("/app/secrets/wml_instance_id")
    wml_apikey = getSecret("/app/secrets/wml_apikey")
    wml_data_source_type = getSecret("/app/secrets/wml_data_source_type")

    # set up the WML client
    wml_credentials = {
                       "url": wml_url,
                       "instance_id": wml_instance_id,
                       "apikey": wml_apikey
                      }
    client = WatsonMachineLearningAPIClient(wml_credentials)

    # store the model
    meta_props_tf={
     client.repository.ModelMetaNames.NAME: wml_model_name,
     client.repository.ModelMetaNames.RUNTIME_UID : "tensorflow_1.14-py3.6",
     client.repository.ModelMetaNames.TYPE: "tensorflow_1.14"
    }
    # stored_model_name    = wml_model_name
    # stored_model_details = client.repository.store_model( run_uid, stored_model_name)
    model_details = client.repository.store_model(run_uid, meta_props=meta_props_tf)

    #model_uid = client.repository.get_model_uid( stored_model_details )
    model_uid = client.repository.get_model_uid(model_details)
    print( "model_uid: ", model_uid )

    with open("/tmp/model_uid", "w") as f:
        f.write(model_uid)
    f.close()

    import time
    time.sleep(120)

if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument('--model-name', type=str, required=True)
    parser.add_argument('--run-uid', type=str, required=True)
    parser.add_argument('--training-uid', type=str, required=True)
    args = parser.parse_args()
    store(args.model_name, args.run_uid, args.training_uid)
