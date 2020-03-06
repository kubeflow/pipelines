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
    #wml_url = "https://us-south.ml.cloud.ibm.com"
    #wml_apikey = "v09FbvTcyr2xKhlWrNfCQTz-o8zMYle2BILoTu-RMD0q"
    #wml_instance_id = "8fbafb51-7048-4439-b71b-dea35f6a15c2"

    wml_data_source_type = getSecret("/app/secrets/wml_data_source_type")

    #wml_data_source_type = "s3"
    
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

    #cos = Minio("s3.us-south.cloud-object-storage.appdomain.cloud",
    #           access_key = "4312cfd3cc7746c8a8afc7c4ad63597c",
    #           secret_key = "d9558e49dacaf8f7c7739391c7f6a8a750fe3f30f8b04f40",
    #           secure = True)
    cos.fget_object(cos_input_bucket, wml_train_code, model_code)
    #cos.fget_object("kevin-cloud-object-storage-trainingoutput-standard-s7y", wml_train_code, model_code)

    # set up the WML client
    wml_credentials = {
                       "url": wml_url,
                       "instance_id": wml_instance_id,
                       "apikey": wml_apikey
                      }
    client = WatsonMachineLearningAPIClient ( wml_credentials )
    # define the model
    #print("NAME-wml_run_definition: ", wml_run_definition)
    #print("VERSION-wml_framework_version: ", wml_framework_version)
    #print("FILEPATH-wml_train_code: ", wml_train_code)
    #print("PLATFORM-name-wml_framework_name: ", wml_framework_name)
    #print("PLATFORM-name-version-wml_framework_version: ", wml_framework_version)

    #data = [wml_framework_version]
    #jsArraydata = JSON.parse(data)

    lib_meta = {
        client.runtimes.LibraryMetaNames.NAME: wml_run_definition,
        client.runtimes.LibraryMetaNames.VERSION: wml_framework_version,
        client.runtimes.LibraryMetaNames.FILEPATH: model_code,
        client.runtimes.LibraryMetaNames.PLATFORM: {"name": wml_framework_name, "versions": [wml_framework_version]}
    }
    #client.runtimes.delete_library("a2383b9d-ccd5-434e-960a-7fbbfa3ca698")
    #client.runtimes.delete_library("7d256a33-93e6-468b-b3a2-8bca863b6353")
    #client.runtimes.delete_library("7d3219e3-b0c5-4fcf-b122-77bdf7eafd20")
    #client.runtimes.delete_library("ddfeeabe-1965-47e3-a6ac-419c1a4cd9e9")
    #client.runtimes.delete_library("9e34c77d-2540-4243-954b-80476c1e432c")
    #client.runtimes.delete_library("2c134df7-1aec-4f53-8ef9-084ca986ba30")
    #client.runtimes.delete_library("89e1f8bf-ab76-4398-bf95-6f64fbab7c3e")
    #client.runtimes.delete_library("7eb074a1-b211-466b-ba9a-e57c45d522d9")
    #client.runtimes.delete_library("788b542d-edf8-4589-8e9c-f898ebc8dfb6")
    #client.runtimes.delete_library("916594db-4e7e-47e8-9fb0-e2745600061d")


    #custom_library_details = client.runtimes.store_library({
    #    client.runtimes.LibraryMetaNames.NAME: wml_run_definition,
    #    client.runtimes.LibraryMetaNames.VERSION: wml_framework_version,
    #    client.runtimes.LibraryMetaNames.FILEPATH: wml_train_code,
    #    client.runtimes.LibraryMetaNames.PLATFORM: {"name": wml_framework_name, "versions": wml_framework_version}
    #})

    custom_library_details = client.runtimes.store_library(lib_meta)
    custom_library_uid = client.runtimes.get_library_uid(custom_library_details)
    #print("custom_library_uid", custom_library_uid)

    #print("get_library_details: ", client.runtimes.get_library_details(custom_library_uid))

    #print("list_libraries: ",client.runtimes.list_libraries())

    #definition_details = client.repository.store_definition( model_code, meta_props=lib_meta )
    #definition_uid     = client.repository.get_definition_uid( definition_details )


    # metadata = {
    #    client.repository.ModelMetaNames.NAME                                       : wml_run_definition,
        # client.repository.DefinitionMetaNames.AUTHOR_NAME       : wml_author_name,
    #    client.repository.ModelMetaNames.AUTHOR_NAME: wml_author_name,
        # client.repository.ModelMetaNames.FRAMEWORK_LIBRARIES: [{'name': wml_framework_name, 'version': wml_framework_version}],
    #    client.repository.ModelMetaNames.FRAMEWORK_NAME: wml_framework_name,
    #    client.repository.ModelMetaNames.FRAMEWORK_VERSION: wml_framework_version,
        # client.repository.DefinitionMetaNames.FRAMEWORK_VERSION : wml_framework_version,
    #    client.repository.PipelineMetaNames.RUNTIMES                                : [{'name': wml_runtime_name, 'version': wml_runtime_version}],
        # client.repository.DefinitionMetaNames.RUNTIME_VERSION   : wml_runtime_version,
    #    client.repository.PipelineMetaNames.COMMAND                                 : wml_execution_command
    #}

    #definition_details = client.repository.store_definition( model_code, meta_props=metadata )
    #definition_uid     = client.repository.get_definition_uid( definition_details )
    # print( "definition_uid: ", definition_uid )

    # create a pipeline with the model definitions included
    doc = {
    "doc_type": "pipeline",
    "version": "2.0",
    "primary_pipeline": "dlaas_only",
    "pipelines": [
      {
        "id": "dlaas_only",
        "runtime_ref": "hybrid",
        "nodes": [
          {
            "id": "training",
            "type": "model_node",
            "op": "dl_train",
            "runtime_ref": "DL",
            "inputs": [
            ],
            "outputs": [],
            "parameters": {
              "name": "tf-mnist",
              "description": "Simple MNIST model implemented in Tensorflow for DL",
              "command": "python3 convolutional_network.py --trainImagesFile ${DATA_DIR}/train-images-idx3-ubyte.gz --trainLabelsFile ${DATA_DIR}/train-labels-idx1-ubyte.gz --testImagesFile ${DATA_DIR}/t10k-images-idx3-ubyte.gz  --testLabelsFile ${DATA_DIR}/t10k-labels-idx1-ubyte.gz --learningRate 0.001 --trainingIters 400000",
              "training_lib_href": "/v4/libraries/"+custom_library_uid,
              "compute": {
                "name": "k80",
                "nodes": 1
                }
             }
           }
         ]
      }
    ],
    "schemas": [
      {
        "id": "schema1",
        "fields": [
          {
            "name": "text",
            "type": "string"
          }
        ]
      }
    ],
    "runtimes": [
      {
        "id": "DL",
        "name": "tensorflow",
        "version": "1.14-py3.6"
      }
     ]
    }

    metadata = {
            client.repository.PipelineMetaNames.NAME: wml_run_name,
            client.repository.PipelineMetaNames.DOCUMENT: doc


            }


    pipeline_id = client.pipelines.get_uid(client.repository.store_pipeline( meta_props=metadata))
        
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
    
    client.training.ConfigurationMetaNames.TRAINING_DATA_REFERENCES:
        [{
            "name": "training_input_data",
            "type": wml_data_source_type,
            "connection": {
                "endpoint_url": cos_endpoint,
                "access_key_id": cos_access_key,
                "secret_access_key": cos_secret_key
            },
          "location": {
              "bucket": cos_input_bucket
          },
          "schema": {
              "id": "id123_schema",
              "fields": [
                  {
                      "name": "text",
                      "type": "string"
                  }
              ]
          }
        }],
    client.training.ConfigurationMetaNames.PIPELINE_UID: pipeline_id 
    }

    training_id = client.training.get_uid(client.training.run(meta_props=metadata))
    
    #print("training_id", client.training.get_details(training_id))

    #print("get status", client.training.get_status(training_id))

    # for v4
    run_details = client.training.get_details(training_id)
    run_uid     = training_id

    with open("/tmp/run_uid", "w") as f:
        f.write(run_uid)
    f.close()

    # print logs
    client.training.monitor_logs(run_uid)
    client.training.monitor_metrics(run_uid)

    # checking the result
    status = client.training.get_status( run_uid )
    print("status: ", status)
    while status['state'] != 'completed':
        time.sleep(20)
        status = client.training.get_status( run_uid )
        print("status: ", status)
    print(status)

    # Get training details
    training_details = client.training.get_details(run_uid)
    print("training_details", training_details)
 
    
    #json_data = json.loads(training_details)

    #pprint.pprint(json_data)

    with open("/tmp/training_uid", "w") as f:
        #training_uid = training_details['entity']['training_results_reference']['location']['model_location']
        training_uid = training_uid = training_details['entity']['results_reference']['location']['training']
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
