# Copyright 2019 kubeflow.org.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
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
import time
import yaml
from distutils.util import strtobool

from kubernetes import client

from kfserving import KFServingClient
from kfserving import constants
from kfserving import V1alpha2EndpointSpec
from kfserving import V1alpha2PredictorSpec
from kfserving import V1alpha2TensorflowSpec
from kfserving import V1alpha2PyTorchSpec
from kfserving import V1alpha2SKLearnSpec
from kfserving import V1alpha2XGBoostSpec
from kfserving.models.v1alpha2_onnx_spec import V1alpha2ONNXSpec
from kfserving import V1alpha2TensorRTSpec
from kfserving import V1alpha2CustomSpec
from kfserving import V1alpha2InferenceServiceSpec
from kfserving import V1alpha2InferenceService


def yamlStr(str):
    if str == "" or str == None:
        return None
    else:
        return yaml.safe_load(str)


def EndpointSpec(framework, storage_uri, service_account):
    if framework == "tensorflow":
        return V1alpha2EndpointSpec(
            predictor=V1alpha2PredictorSpec(
                tensorflow=V1alpha2TensorflowSpec(storage_uri=storage_uri),
                service_account_name=service_account,
            )
        )
    elif framework == "pytorch":
        return V1alpha2EndpointSpec(
            predictor=V1alpha2PredictorSpec(
                pytorch=V1alpha2PyTorchSpec(storage_uri=storage_uri),
                service_account_name=service_account,
            )
        )
    elif framework == "sklearn":
        return V1alpha2EndpointSpec(
            predictor=V1alpha2PredictorSpec(
                sklearn=V1alpha2SKLearnSpec(storage_uri=storage_uri),
                service_account_name=service_account,
            )
        )
    elif framework == "xgboost":
        return V1alpha2EndpointSpec(
            predictor=V1alpha2PredictorSpec(
                xgboost=V1alpha2XGBoostSpec(storage_uri=storage_uri),
                service_account_name=service_account,
            )
        )
    elif framework == "onnx":
        return V1alpha2EndpointSpec(
            predictor=V1alpha2PredictorSpec(
                onnx=V1alpha2ONNXSpec(storage_uri=storage_uri),
                service_account_name=service_account,
            )
        )
    elif framework == "tensorrt":
        return V1alpha2EndpointSpec(
            predictor=V1alpha2PredictorSpec(
                tensorrt=V1alpha2TensorRTSpec(storage_uri=storage_uri),
                service_account_name=service_account,
            )
        )
    else:
        raise ("Error: No matching framework: " + framework)


def customEndpointSpec(custom_model_spec, service_account):
    env = (
        [
            client.V1EnvVar(name=i["name"], value=i["value"])
            for i in custom_model_spec["env"]
        ]
        if custom_model_spec.get("env", "")
        else None
    )
    ports = (
        [client.V1ContainerPort(container_port=int(custom_model_spec.get("port", "")), protocol="TCP")]
        if custom_model_spec.get("port", "")
        else None
    )
    resources = (
        client.V1ResourceRequirements(
            requests=(custom_model_spec["resources"]["requests"]
                      if custom_model_spec.get('resources', {}).get('requests')
                      else None
                      ),
            limits=(custom_model_spec["resources"]["limits"]
                    if custom_model_spec.get('resources', {}).get('limits')
                    else None
                    ),
        )
        if custom_model_spec.get("resources", {})
        else None
    )
    containerSpec = client.V1Container(
        name=custom_model_spec.get("name", "custom-container"),
        image=custom_model_spec["image"],
        env=env,
        ports=ports,
        command=custom_model_spec.get("command", None),
        args=custom_model_spec.get("args", None),
        image_pull_policy=custom_model_spec.get("image_pull_policy", None),
        working_dir=custom_model_spec.get("working_dir", None),
        resources=resources
    )
    return V1alpha2EndpointSpec(
        predictor=V1alpha2PredictorSpec(
            custom=V1alpha2CustomSpec(container=containerSpec),
            service_account_name=service_account,
        )
    )


def InferenceService(
    metadata, default_model_spec, canary_model_spec=None, canary_model_traffic=None
):
    return V1alpha2InferenceService(
        api_version=constants.KFSERVING_GROUP + "/" + constants.KFSERVING_VERSION,
        kind=constants.KFSERVING_KIND,
        metadata=metadata,
        spec=V1alpha2InferenceServiceSpec(
            default=default_model_spec,
            canary=canary_model_spec,
            canary_traffic_percent=canary_model_traffic,
        ),
    )


def deploy_model(
    action,
    model_name,
    default_model_uri,
    canary_model_uri,
    canary_model_traffic,
    namespace,
    framework,
    default_custom_model_spec,
    canary_custom_model_spec,
    service_account,
    autoscaling_target=0,
    enable_istio_sidecar=True,
    inferenceservice_yaml={},
    watch_timeout=120,
):
    KFServing = KFServingClient()

    if inferenceservice_yaml:
        # Overwrite name and namespace if exist
        if namespace:
            inferenceservice_yaml['metadata']['namespace'] = namespace
        if model_name:
            inferenceservice_yaml['metadata']['name'] = model_name
        kfsvc = inferenceservice_yaml
    else:
        # Create annotation
        annotations = {}
        if int(autoscaling_target) != 0:
            annotations["autoscaling.knative.dev/target"] = str(autoscaling_target)
        if not enable_istio_sidecar:
            annotations["sidecar.istio.io/inject"] = 'false'
        if not annotations:
            annotations = None
        metadata = client.V1ObjectMeta(
            name=model_name, namespace=namespace, annotations=annotations
        )

        # Create Default deployment if default model uri is provided.
        if framework != "custom" and default_model_uri:
            default_model_spec = EndpointSpec(framework, default_model_uri, service_account)
        elif framework == "custom" and default_custom_model_spec:
            default_model_spec = customEndpointSpec(
                default_custom_model_spec, service_account
            )

        # Create Canary deployment if canary model uri is provided.
        if framework != "custom" and canary_model_uri:
            canary_model_spec = EndpointSpec(framework, canary_model_uri, service_account)
            kfsvc = InferenceService(
                metadata, default_model_spec, canary_model_spec, canary_model_traffic
            )
        elif framework == "custom" and canary_custom_model_spec:
            canary_model_spec = customEndpointSpec(
                canary_custom_model_spec, service_account
            )
            kfsvc = InferenceService(
                metadata, default_model_spec, canary_model_spec, canary_model_traffic
            )
        else:
            kfsvc = InferenceService(metadata, default_model_spec)

    def create(kfsvc, model_name, namespace):
        KFServing.create(kfsvc, namespace=namespace)
        time.sleep(1)
        KFServing.get(model_name, namespace=namespace, watch=True, timeout_seconds=watch_timeout)

    def update(kfsvc, model_name, namespace):
        KFServing.patch(model_name, kfsvc, namespace=namespace)
        time.sleep(1)
        KFServing.get(model_name, namespace=namespace, watch=True, timeout_seconds=watch_timeout)

    if action == "create":
        create(kfsvc, model_name, namespace)
    elif action == "update":
        update(kfsvc, model_name, namespace)
    elif action == "apply":
        try:
            create(kfsvc, model_name, namespace)
        except:
            update(kfsvc, model_name, namespace)
    elif action == "rollout":
        if inferenceservice_yaml:
            raise("Rollout is not supported for inferenceservice yaml")
        KFServing.rollout_canary(
            model_name,
            canary=canary_model_spec,
            percent=canary_model_traffic,
            namespace=namespace,
            watch=True,
            timeout_seconds=watch_timeout,
        )
    elif action == "promote":
        KFServing.promote(
            model_name, namespace=namespace, watch=True, timeout_seconds=watch_timeout
        )
    elif action == "delete":
        KFServing.delete(model_name, namespace=namespace)
    else:
        raise ("Error: No matching action: " + action)

    model_status = KFServing.get(model_name, namespace=namespace)
    return model_status


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--action", type=str, help="Action to execute on KFServing", default="create"
    )
    parser.add_argument(
        "--model-name", type=str, help="Name to give to the deployed model", default=""
    )
    parser.add_argument(
        "--default-model-uri",
        type=str,
        help="Path of the S3, GCS or PVC directory containing default model.",
    )
    parser.add_argument(
        "--canary-model-uri",
        type=str,
        help="Optional path of the S3, GCS or PVC directory containing canary model.",
        default="",
    )
    parser.add_argument(
        "--canary-model-traffic",
        type=str,
        help="Optional Traffic to be sent to the default model",
        default="0",
    )
    parser.add_argument(
        "--namespace",
        type=str,
        help="Kubernetes namespace where the KFServing service is deployed.",
        default="anonymous",
    )
    parser.add_argument(
        "--framework", type=str, help="Model Serving Framework", default="tensorflow"
    )
    parser.add_argument(
        "--default-custom-model-spec",
        type=json.loads,
        help="Custom runtime default custom model container spec",
        default={},
    )
    parser.add_argument(
        "--canary-custom-model-spec",
        type=json.loads,
        help="Custom runtime canary custom model container spec",
        default={},
    )
    parser.add_argument(
        "--kfserving-endpoint",
        type=str,
        help="kfserving remote deployer api endpoint",
        default="",
    )
    parser.add_argument(
        "--autoscaling-target", type=str, help="Autoscaling target number", default="0"
    )
    parser.add_argument(
        "--service-account",
        type=str,
        help="Service account containing s3 credentials",
        default="",
    )
    parser.add_argument(
        "--enable-istio-sidecar",
        type=strtobool,
        help="Whether to inject istio sidecar",
        default="True"
    )
    parser.add_argument(
        "--inferenceservice_yaml",
        type=yamlStr,
        help="Raw InferenceService serialized YAML for deployment",
        default={}
    )
    parser.add_argument("--output-path", type=str, help="Path to store URI output")
    parser.add_argument("--watch-timeout",
                        type=str,
                        help="Timeout seconds for watching until InferenceService becomes ready.",
                        default="120")
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
    kfserving_endpoint = url.sub("", args.kfserving_endpoint)
    autoscaling_target = int(args.autoscaling_target)
    service_account = args.service_account
    enable_istio_sidecar = args.enable_istio_sidecar
    inferenceservice_yaml = args.inferenceservice_yaml
    watch_timeout = int(args.watch_timeout)

    if kfserving_endpoint:
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
            "autoscaling_target": autoscaling_target,
            "service_account": service_account,
            "enable_istio_sidecar": enable_istio_sidecar,
            "inferenceservice_yaml": inferenceservice_yaml
        }
        response = requests.post(
            "http://" + kfserving_endpoint + "/deploy-model", json=formData
        )
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
            autoscaling_target=autoscaling_target,
            service_account=service_account,
            enable_istio_sidecar=enable_istio_sidecar,
            inferenceservice_yaml=inferenceservice_yaml,
            watch_timeout=watch_timeout
        )
    print(model_status)
    # Check whether the model is ready
    for condition in model_status["status"]["conditions"]:
        if condition['type'] == 'Ready':
            if condition['status'] == 'True':
                print('Model is ready\n')
                break
            else:
                print('Model is timed out, please check the inferenceservice events for more details.')
                exit(1)
    try:
        print(
            model_status["status"]["url"] + " is the knative domain."
        )
        print("Sample test commands: \n")
        # model_status['status']['url'] is like http://flowers-sample.kubeflow.example.com/v1/models/flowers-sample
        print(
            "curl -v -X GET %s" % model_status["status"]["url"]
        )

        print("\nIf the above URL is not accessible, it's recommended to setup Knative with a configured DNS.\n"\
              "https://knative.dev/docs/install/installing-istio/#configuring-dns")
    except:
        print("Model is not ready, check the logs for the Knative URL status.")
        exit(1)
    if not os.path.exists(os.path.dirname(output_path)):
        os.makedirs(os.path.dirname(output_path))
    with open(output_path, "w") as report:
        report.write(json.dumps(model_status))
