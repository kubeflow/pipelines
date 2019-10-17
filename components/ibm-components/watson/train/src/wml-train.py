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

# define the function to train a model on wml

def getSecret(secret):
    with open(secret, 'r') as f:
        res = f.readline().strip('\'')
    f.close()
    return res

def train(args):
    from watson_machine_learning_client import WatsonMachineLearningAPIClient
    from minio import Minio
    from urllib.parse import urlsplit
    import os,time

    wml_train_code = args.train_code
    wml_execution_command = args.execution_command.strip('\'')
    wml_framework_name = args.framework if args.framework else 'tensorflow'
    wml_framework_version = args.framework_version if args.framework_version else '1.13'
    wml_runtime_name = args.runtime if args.runtime else 'python'
    wml_runtime_version = args.runtime_version if args.runtime_version else '3.6'
    wml_run_definition = args.run_definition if args.run_definition else 'python-tensorflow-definition'
    wml_run_name = args.run_name if args.run_name else 'python-tensorflow-run'
    wml_author_name = args.author_name if args.author_name else 'default-author'

    # retrieve credentials
    wml_url = getSecret("/app/secrets/wml_url")
    wml_apikey = getSecret("/app/secrets/wml_apikey")
    wml_instance_id = getSecret("/app/secrets/wml_instance_id")

    wml_data_source_type = getSecret("/app/secrets/wml_data_source_type")

    cos_endpoint = getSecret("/app/secrets/cos_endpoint")
    cos_endpoint_parts = urlsplit(cos_endpoint)
    if bool(cos_endpoint_parts.scheme):
        cos_endpoint_hostname = cos_endpoint_parts.hostname
    else:
        cos_endpoint_hostname = cos_endpoint
        cos_endpoint = 'https://' + cos_endpoint
    cos_access_key = getSecret("/app/secrets/cos_access_key")
    cos_secret_key = getSecret("/app/secrets/cos_secret_key")
    cos_input_bucket = getSecret("/app/secrets/cos_input_bucket")
    cos_output_bucket = getSecret("/app/secrets/cos_output_bucket")

    # download model code
    model_code = os.path.join('/app', wml_train_code)

    cos = Minio(cos_endpoint_hostname,
               access_key = cos_access_key,
               secret_key = cos_secret_key,
               secure = True)
    cos.fget_object(cos_input_bucket, wml_train_code, model_code)

    # set up the WML client
    wml_credentials = {
                       "url": wml_url,
                       "instance_id": wml_instance_id,
                       "apikey": wml_apikey
                      }
    client = WatsonMachineLearningAPIClient( wml_credentials )

    # define the model
    metadata = {
        client.repository.DefinitionMetaNames.NAME              : wml_run_definition,
        client.repository.DefinitionMetaNames.AUTHOR_NAME       : wml_author_name,
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
        client.training.ConfigurationMetaNames.NAME        : wml_run_name,
        client.training.ConfigurationMetaNames.AUTHOR_NAME : wml_author_name,
        client.training.ConfigurationMetaNames.TRAINING_DATA_REFERENCE : {
            "connection" : {
                "endpoint_url"      : cos_endpoint,
                "access_key_id"     : cos_access_key,
                "secret_access_key" : cos_secret_key
            },
            "source" : {
                "bucket" : cos_input_bucket,
            },
            "type" : wml_data_source_type
        },
        client.training.ConfigurationMetaNames.TRAINING_RESULTS_REFERENCE: {
            "connection" : {
                "endpoint_url"      : cos_endpoint,
                "access_key_id"     : cos_access_key,
                "secret_access_key" : cos_secret_key
            },
            "target" : {
                "bucket" : cos_output_bucket,
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

    # Get training details
    training_details = client.training.get_details(run_uid)
    with open("/tmp/training_uid", "w") as f:
        training_uid = training_details['entity']['training_results_reference']['location']['model_location']
        f.write(training_uid)
    f.close()

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
    parser.add_argument('--author-name', type=str)
    parser.add_argument('--config', type=str, default="secret_name")
    args = parser.parse_args()
    # Check secret name is not empty
    if (not args.config):
        print("Secret for this pipeline is not properly created, exiting with status 1...")
        exit(1)
    train(args)
