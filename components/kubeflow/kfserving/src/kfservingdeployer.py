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
import requests
import re

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
    else:
        raise("Error: No matching framework: " + framework)


def customModelSpec(custom_model_spec):
    env = [client.V1EnvVar(name=i['name'], value=i['value']) for i in custom_model_spec['env']] if custom_model_spec.get('env', '') else None
    ports = [client.V1ContainerPort(container_port=int(custom_model_spec.get('port', '')))] if custom_model_spec.get('port', '') else None
    containerSpec = client.V1Container(
        name=custom_model_spec.get('name', 'custom-container'),
        image=custom_model_spec['image'],
        env=env,
        ports=ports,
        command=custom_model_spec.get('command', None),
        args=custom_model_spec.get('args', None),
        image_pull_policy=custom_model_spec.get('image_pull_policy', None),
        working_dir=custom_model_spec.get('working_dir', None)
    )
    return V1alpha1ModelSpec(custom=V1alpha1CustomSpec(container=containerSpec))


def kfserving_deployment(metadata, default_model_spec, canary_model_spec=None, canary_model_traffic=None):
    return V1alpha1KFService(api_version=constants.KFSERVING_GROUP + '/' + constants.KFSERVING_VERSION,
                             kind=constants.KFSERVING_KIND,
                             metadata=metadata,
                             spec=V1alpha1KFServiceSpec(default=default_model_spec,
                                                        canary=canary_model_spec,
                                                        canary_traffic_percent=canary_model_traffic))


def deploy_model(action, model_name, default_model_uri, canary_model_uri, canary_model_traffic, namespace, framework, default_custom_model_spec, canary_custom_model_spec, autoscaling_target=0):
    if int(autoscaling_target) != 0:
        annotations = {"autoscaling.knative.dev/target": str(autoscaling_target)}
    else:
        annotations = None
    metadata = client.V1ObjectMeta(name=model_name, namespace=namespace, annotations=annotations)
    if framework != 'custom':
        default_model_spec = ModelSpec(framework, default_model_uri)
    else:
        default_model_spec = customModelSpec(default_custom_model_spec)
    # Create Canary deployment if canary model uri is provided.
    if framework != 'custom' and canary_model_uri:
        canary_model_spec = ModelSpec(framework, canary_model_uri)
        kfsvc = kfserving_deployment(metadata, default_model_spec, canary_model_spec, canary_model_traffic)
    elif framework == 'custom' and canary_custom_model_spec:
        canary_model_spec = customModelSpec(canary_custom_model_spec)
        kfsvc = kfserving_deployment(metadata, default_model_spec, canary_model_spec, canary_model_traffic)
    else:
        kfsvc = kfserving_deployment(metadata, default_model_spec)

    KFServing = KFServingClient()

    if action == 'create':
        KFServing.create(kfsvc)
    elif action == 'update':
        KFServing.patch(model_name, kfsvc)
    elif action == 'delete':
        KFServing.delete(model_name, namespace=namespace)
    else:
        raise("Error: No matching action: " + action)

    model_status = KFServing.get(model_name, namespace=namespace)
    return model_status

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--action', type=str, help='Action to execute on KFServing', default='create')
    parser.add_argument('--model-name', type=str, help='Name to give to the deployed model', default="")
    parser.add_argument('--default-model-uri', type=str, help='Path of the S3, GCS or PVC directory containing default model.')
    parser.add_argument('--canary-model-uri', type=str, help='Optional path of the S3, GCS or PVC directory containing canary model.', default="")
    parser.add_argument('--canary-model-traffic', type=str, help='Optional Traffic to be sent to the default model', default='0')
    parser.add_argument('--namespace', type=str, help='Kubernetes namespace where the KFServing service is deployed.', default='kubeflow')
    parser.add_argument('--framework', type=str, help='Model Serving Framework', default='tensorflow')
    parser.add_argument('--default-custom-model-spec', type=json.loads, help='Custom runtime default custom model container spec', default={})
    parser.add_argument('--canary-custom-model-spec', type=json.loads, help='Custom runtime canary custom model container spec', default={})
    parser.add_argument('--kfserving-deployer-api', type=str, help='kfserving remote deployer api endpoint', default='')
    parser.add_argument('--autoscaling-target', type=str, help='Autoscaling target number', default='0')
    parser.add_argument('--output_path', type=str, help='Path to store URI output')
    args = parser.parse_args()

    url = re.compile(r"https?://")

    action = args.action.lower()
    model_name = args.model_name
    default_model_uri = args.default_model_uri
    canary_model_uri = args.canary_model_uri
    canary_model_traffic = int(args.canary_model_traffic)
    namespace = args.namespace
    framework = args.framework.lower()
    output_path = args.output_path
    default_custom_model_spec = args.default_custom_model_spec
    canary_custom_model_spec = args.canary_custom_model_spec
    kfserving_deployer_api = url.sub('', args.kfserving_deployer_api)
    autoscaling_target = int(args.autoscaling_target)

    if kfserving_deployer_api:
        formData = {
            "action": action,
            "model_name": model_name,
            "default_model_uri": default_model_uri,
            "canary_model_uri": canary_model_uri,
            "canary_model_traffic": canary_model_traffic,
            "namespace": namespace,
            "framework": framework,
            "default_custom_model_spec": default_custom_model_spec,
            "canary_custom_model_spec": canary_custom_model_spec,
            "autoscaling_target": autoscaling_target
            }
        response = requests.post("http://" + kfserving_deployer_api + "/deploy-model", json=formData)
        model_status = response.json()
    else:
        model_status = deploy_model(
            action=action,
            model_name=model_name,
            default_model_uri=default_model_uri,
            canary_model_uri=canary_model_uri,
            canary_model_traffic=canary_model_traffic,
            namespace=namespace,
            framework=framework,
            default_custom_model_spec=default_custom_model_spec,
            canary_custom_model_spec=canary_custom_model_spec,
            autoscaling_target=autoscaling_target
        )
    print(model_status)
    try:
        print(model_status['status']['url'] + ' is the knative domain header. $ISTIO_INGRESS_ENDPOINT are defined in the below commands')
        print('Sample test commands: ')
        print('# Note: If Istio Ingress gateway is not served with LoadBalancer, use $CLUSTER_NODE_IP:31380 as the ISTIO_INGRESS_ENDPOINT')
        print('ISTIO_INGRESS_ENDPOINT=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath=\'{.status.loadBalancer.ingress[0].ip}\')')
        print('curl -X GET -H "Host: ' + url.sub('', model_status['status']['url']) + '" $ISTIO_INGRESS_ENDPOINT')
    except:
        print('Model is not ready, check the logs for the Knative URL status.')
    if not os.path.exists(os.path.dirname(output_path)):
        os.makedirs(os.path.dirname(output_path))
    with open(output_path, "w") as report:
        report.write(json.dumps(model_status))
