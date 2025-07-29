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
    from pathlib import Path
    import os,time

    wml_train_code = args.train_code
    wml_execution_command = args.execution_command.strip('\'')
    wml_framework_name = args.framework if args.framework else 'tensorflow'
    wml_framework_version = args.framework_version if args.framework_version else '1.15'
    wml_runtime_name = args.runtime if args.runtime else 'python'
    wml_runtime_version = args.runtime_version if args.runtime_version else '3.6'
    wml_run_definition = args.run_definition if args.run_definition else 'python-tensorflow-definition'
    wml_run_name = args.run_name if args.run_name else 'python-tensorflow-run'
    wml_author_name = args.author_name if args.author_name else 'default-author'
    wml_compute_name = args.compute_name if args.compute_name else 'k80'
    wml_compute_nodes = args.compute_nodes if args.compute_nodes else '1'

    wml_runtime_version_v4 = wml_framework_version + '-py' + wml_runtime_version
    wml_compute_nodes_v4 = int(wml_compute_nodes)

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
               access_key=cos_access_key,
               secret_key=cos_secret_key,
               secure=True)

    cos.fget_object(cos_input_bucket, wml_train_code, model_code)

    # set up the WML client
    wml_credentials = {
                       "url": wml_url,
                       "instance_id": wml_instance_id,
                       "apikey": wml_apikey
                      }
    client = WatsonMachineLearningAPIClient(wml_credentials)
    # define the model
    lib_meta = {
        client.runtimes.LibraryMetaNames.NAME: wml_run_definition,
        client.runtimes.LibraryMetaNames.VERSION: wml_framework_version,
        client.runtimes.LibraryMetaNames.FILEPATH: model_code,
        client.runtimes.LibraryMetaNames.PLATFORM: {"name": wml_framework_name, "versions": [wml_framework_version]}
    }
    # check exisiting library
    library_details = client.runtimes.get_library_details()
    for library_detail in library_details['resources']:
        if library_detail['entity']['name'] == wml_run_definition:
            # Delete library if exist because we cannot update model_code
            uid = client.runtimes.get_library_uid(library_detail)
            client.repository.delete(uid)
            break
    custom_library_details = client.runtimes.store_library(lib_meta)
    custom_library_uid = client.runtimes.get_library_uid(custom_library_details)

    # create a pipeline with the model definitions included
    doc = {
        "doc_type": "pipeline",
        "version": "2.0",
        "primary_pipeline": wml_framework_name,
        "pipelines": [{
            "id": wml_framework_name,
            "runtime_ref": "hybrid",
            "nodes": [{
                "id": "training",
                "type": "model_node",
                "op": "dl_train",
                "runtime_ref": wml_run_name,
                "inputs": [],
                "outputs": [],
                "parameters": {
                    "name": "tf-mnist",
                    "description": wml_run_definition,
                    "command": wml_execution_command,
                    "training_lib_href": "/v4/libraries/"+custom_library_uid,
                    "compute": {
                        "name": wml_compute_name,
                        "nodes": wml_compute_nodes_v4
                    }
                }
            }]
        }],
        "runtimes": [{
            "id": wml_run_name,
            "name": wml_framework_name,
            "version": wml_runtime_version_v4
        }]
    }

    metadata = {
        client.repository.PipelineMetaNames.NAME: wml_run_name,
        client.repository.PipelineMetaNames.DOCUMENT: doc
    }
    pipeline_id = client.pipelines.get_uid(client.repository.store_pipeline(meta_props=metadata))
        
    client.pipelines.get_details(pipeline_id)

    # start the training run for v4
    metadata = {
        client.training.ConfigurationMetaNames.TRAINING_RESULTS_REFERENCE: {
            "name": "training-results-reference_name",
            "connection": {
                "endpoint_url": cos_endpoint,
                "access_key_id": cos_access_key,
                "secret_access_key": cos_secret_key
            },
            "location": {
                "bucket": cos_output_bucket
            },
            "type": wml_data_source_type
        },
        client.training.ConfigurationMetaNames.TRAINING_DATA_REFERENCES:[{
            "name": "training_input_data",
            "type": wml_data_source_type,
            "connection": {
                "endpoint_url": cos_endpoint,
                "access_key_id": cos_access_key,
                "secret_access_key": cos_secret_key
            },
            "location": {
                "bucket": cos_input_bucket
            }
        }],
        client.training.ConfigurationMetaNames.PIPELINE_UID: pipeline_id
    }

    training_id = client.training.get_uid(client.training.run(meta_props=metadata))
    print("training_id", client.training.get_details(training_id))
    print("get status", client.training.get_status(training_id))
    # for v4
    run_details = client.training.get_details(training_id)
    run_uid = training_id

    # print logs
    client.training.monitor_logs(run_uid)
    client.training.monitor_metrics(run_uid)

    # checking the result
    status = client.training.get_status(run_uid)
    print("status: ", status)
    while status['state'] != 'completed':
        time.sleep(20)
        status = client.training.get_status(run_uid)
    print(status)

    Path(args.output_run_uid_path).parent.mkdir(parents=True, exist_ok=True)
    Path(args.output_run_uid_path).write_text(run_uid)

    # Get training details
    training_details = client.training.get_details(run_uid)
    print("training_details", training_details)
 
    Path(args.output_training_uid_path).parent.mkdir(parents=True, exist_ok=True)
    Path(args.output_training_uid_path).write_text(run_uid)

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
    parser.add_argument('--compute-name', type=str)
    parser.add_argument('--compute-nodes', type=str)
    parser.add_argument('--output-run-uid-path', type=str, default="/tmp/run_uid")
    parser.add_argument('--output-training-uid-path', type=str, default="/tmp/training_uid")
    args = parser.parse_args()
    # Check secret name is not empty
    if (not args.config):
        print("Secret for this pipeline is not properly created, exiting with status 1...")
        exit(1)
    train(args)
