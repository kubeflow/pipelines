# Copyright 2019 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import json
import argparse
import os

from kubernetes import client

from kfserving import KFServingClient
from kfserving import constants
from kfserving import V1alpha1ModelSpec
from kfserving import V1alpha1TensorflowSpec
from kfserving import V1alpha1PyTorchSpec
from kfserving import V1alpha1SKLearnSpec
from kfserving import V1alpha1XGBoostSpec
from kfserving import V1alpha1TensorRTSpec
from kfserving import V1alpha1CustomSpec
from kfserving import V1alpha1KFServiceSpec
from kfserving import V1alpha1KFService


def ModelSpec(framework, model_uri):
    if framework == 'tensorflow':
        return V1alpha1ModelSpec(tensorflow=V1alpha1TensorflowSpec(model_uri=model_uri))
    elif framework == 'pytorch':
        return V1alpha1ModelSpec(pytorch=V1alpha1PyTorchSpec(model_uri=model_uri))
    elif framework == 'sklearn':
        return V1alpha1ModelSpec(sklearn=V1alpha1SKLearnSpec(model_uri=model_uri))
    elif framework == 'xgboost':
        return V1alpha1ModelSpec(xgboost=V1alpha1XGBoostSpec(model_uri=model_uri))
    elif framework == 'tensorrt':
        return V1alpha1ModelSpec(tensorrt=V1alpha1TensorRTSpec(model_uri=model_uri))
    elif framework == 'custom':
        # TODO: implement custom container spec with more args.
        # return V1alpha1ModelSpec(custom=V1alpha1CustomSpec(container=containerSpec))
        raise("Custom framework is not yet implemented")
    else:
        raise("Error: No matching framework: " + framework)

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--action', type=str, help='Action to execute on KFServing', default='create')
    parser.add_argument('--model-name', type=str, help='Name to give to the deployed model', default="")
    parser.add_argument('--default-model-uri', type=str, help='Path of the S3, GCS or PVC directory containing default model.')
    parser.add_argument('--canary-model-uri', type=str, help='Optional path of the S3, GCS or PVC directory containing canary model.', default="")
    parser.add_argument('--canary-model-traffic', type=str, help='Optional Traffic to be sent to the default model', default='0')
    parser.add_argument('--namespace', type=str, help='Kubernetes namespace where the KFServing service is deployed.', default='kubeflow')
    parser.add_argument('--framework', type=str, help='Model Serving Framework', default='tensorflow')
    parser.add_argument('--output_path', type=str, help='Path to store URI output')
    args = parser.parse_args()

    action = args.action.lower()
    model_name = args.model_name
    default_model_uri = args.default_model_uri
    canary_model_uri = args.canary_model_uri
    canary_model_traffic = args.canary_model_traffic
    namespace = args.namespace
    framework = args.framework.lower()
    output_path = args.output_path

    default_model_spec = ModelSpec(framework, default_model_uri)
    metadata = client.V1ObjectMeta(name=model_name, namespace=namespace)

    # Create Canary deployment if canary model uri is provided.
    if canary_model_uri:
        canary_model_spec = ModelSpec(framework, canary_model_uri)
        kfsvc = V1alpha1KFService(api_version=constants.KFSERVING_GROUP + '/' + constants.KFSERVING_VERSION,
                                  kind=constants.KFSERVING_KIND,
                                  metadata=metadata,
                                  spec=V1alpha1KFServiceSpec(default=default_model_spec,
                                                             canary=canary_model_spec,
                                                             canary_traffic_percent=int(canary_model_traffic)))
    else:
        kfsvc = V1alpha1KFService(api_version=constants.KFSERVING_GROUP + '/' + constants.KFSERVING_VERSION,
                                  kind=constants.KFSERVING_KIND,
                                  metadata=metadata,
                                  spec=V1alpha1KFServiceSpec(default=default_model_spec))
    KFServing = KFServingClient()
  
    if action == 'create':
        KFServing.create(kfsvc)
    elif action  == 'update':
        KFServing.patch(model_name, kfsvc)
    elif action  == 'delete':
        KFServing.delete(model_name, namespace=namespace)
    else:
        raise("Error: No matching action: " + action)

    model_status = KFServing.get(model_name, namespace=namespace)
    print(model_status)

    if not os.path.exists(os.path.dirname(output_path)):
        os.makedirs(os.path.dirname(output_path))
    with open(output_path, "w") as report:
        report.write(json.dumps(model_status))
