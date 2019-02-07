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
# define the function to train a model on wml

def train(args):
    from watson_machine_learning_client import WatsonMachineLearningAPIClient
    import boto3
    import os,time

    wml_train_code = args.train_code
    wml_execution_command = args.execution_command.strip('\'')
    wml_framework_name = args.framework if args.framework else 'tensorflow'
    wml_framework_version = args.framework_version if args.framework_version else '1.5'
    wml_runtime_name = args.runtime if args.runtime else 'python'
    wml_runtime_version = args.runtime_version if args.runtime_version else '3.5'
    wml_run_definition = args.run_definition if args.run_definition else 'python-tensorflow-definition'
    wml_run_name = args.run_name if args.run_name else 'python-tensorflow-run'

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

    with open("/app/secrets/wml_data_source_type", 'r') as f:
        wml_data_source_type = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/s3_endpoint", 'r') as f:
        s3_endpoint = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/s3_access_key", 'r') as f:
        s3_access_key = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/s3_secret_key", 'r') as f:
        s3_secret_key = f.readline().strip('\'')
    f.close()
    
    with open("/app/secrets/s3_input_bucket", 'r') as f:
        s3_input_bucket = f.readline().strip('\'')
    f.close()

    with open("/app/secrets/s3_output_bucket", 'r') as f:
        s3_output_bucket = f.readline().strip('\'')
    f.close()

    # download model code
    model_code = os.path.join('/app', wml_train_code)
    
    s3 = boto3.resource('s3',
                        endpoint_url=s3_endpoint,
                        aws_access_key_id=s3_access_key,
                        aws_secret_access_key=s3_secret_key)
    s3.Bucket(s3_input_bucket).download_file(wml_train_code, model_code)

    # set up the WML client
    wml_credentials = {
                       "url": wml_url,
                       "username": wml_username,
                       "password": wml_password,
                       "instance_id": wml_instance_id
                      }
    client = WatsonMachineLearningAPIClient( wml_credentials )
        
    # define the model
    metadata = {
        client.repository.DefinitionMetaNames.NAME              : wml_run_definition,
        client.repository.DefinitionMetaNames.AUTHOR_EMAIL      : "wzhuang@us.ibm.com",
        client.repository.DefinitionMetaNames.FRAMEWORK_NAME    : wml_framework_name,
        client.repository.DefinitionMetaNames.FRAMEWORK_VERSION : wml_framework_version,
        client.repository.DefinitionMetaNames.RUNTIME_NAME      : wml_runtime_name,
        client.repository.DefinitionMetaNames.RUNTIME_VERSION   : wml_runtime_version,
        client.repository.DefinitionMetaNames.EXECUTION_COMMAND : wml_execution_command
    }

    definition_details = client.repository.store_definition( model_code, meta_props=metadata )
    definition_uid     = client.repository.get_definition_uid( definition_details )
    # print( "definition_uid: ", definition_uid )

    # define the run
    metadata = {
        client.training.ConfigurationMetaNames.NAME         : wml_run_name,
        client.training.ConfigurationMetaNames.AUTHOR_EMAIL : "wzhuang@us.ibm.com",
        client.training.ConfigurationMetaNames.TRAINING_DATA_REFERENCE : {
            "connection" : {
                "endpoint_url"      : s3_endpoint,
                "access_key_id"     : s3_access_key,
                "secret_access_key" : s3_secret_key
            },
            "source" : {
                "bucket" : s3_input_bucket,
            },
            "type" : wml_data_source_type
        },
        client.training.ConfigurationMetaNames.TRAINING_RESULTS_REFERENCE: {
            "connection" : {
                "endpoint_url"      : s3_endpoint,
                "access_key_id"     : s3_access_key,
                "secret_access_key" : s3_secret_key
            },
            "target" : {
                "bucket" : s3_output_bucket,
            },
            "type" : wml_data_source_type
        }
    }

    # start the training
    run_details = client.training.run( definition_uid, meta_props=metadata, asynchronous=True )
    run_uid     = client.training.get_run_uid( run_details )
    with open("/tmp/run_uid", "w") as f:
        f.write(run_uid)
    f.close()

    # print logs
    client.training.monitor_logs(run_uid)
    client.training.monitor_metrics(run_uid)

    # checking the result
    status = client.training.get_status( run_uid )
    while status['state'] != 'completed':
        time.sleep(20)
        status = client.training.get_status( run_uid )
    print(status)

if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument('--train-code', type=str, required=True)
    parser.add_argument('--execution-command', type=str, required=True)
    parser.add_argument('--framework', type=str)
    parser.add_argument('--framework-version', type=str)
    parser.add_argument('--runtime', type=str)
    parser.add_argument('--runtime-version', type=str)
    parser.add_argument('--run-definition', type=str)
    parser.add_argument('--run-name', type=str)
    parser.add_argument('--config', type=str)
    args = parser.parse_args()
    if (args.config != "created"):
        exit(1)
    train(args)
