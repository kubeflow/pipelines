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

def store(wml_model_name, run_uid):
    from watson_machine_learning_client import WatsonMachineLearningAPIClient

    # retrieve credentials
    with open("/app/secrets/wml_url", 'r') as f:
        wml_url = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/wml_username", 'r') as f:
        wml_username = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/wml_password", 'r') as f:
        wml_password = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/wml_instance_id", 'r') as f:
        wml_instance_id = f.readline().strip('\'')
    f.close()
    
    # set up the WML client
    wml_credentials = {
                       "url": wml_url,
                       "username": wml_username,
                       "password": wml_password,
                       "instance_id": wml_instance_id
                      }
    client = WatsonMachineLearningAPIClient( wml_credentials )
        
    # store the model
    stored_model_name    = wml_model_name
    stored_model_details = client.repository.store_model( run_uid, stored_model_name )
    model_uid            = client.repository.get_model_uid( stored_model_details )
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
    args = parser.parse_args()
    store(args.model_name, args.run_uid)
