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

import argparse
import ast
from distutils.util import strtobool
import json
import os
import sys
import time
import yaml

from kubernetes import client
from kubernetes.client import V1Container
from kubernetes.client import V1EnvVar
from kubernetes.client.models import V1ResourceRequirements

from kserve import constants
from kserve import KServeClient
from kserve import V1beta1InferenceService
from kserve import V1beta1InferenceServiceSpec
from kserve import V1beta1LightGBMSpec
from kserve import V1beta1ONNXRuntimeSpec
from kserve import V1beta1PMMLSpec
from kserve import V1beta1PredictorSpec
from kserve import V1beta1SKLearnSpec
from kserve import V1beta1TFServingSpec
from kserve import V1beta1TorchServeSpec
from kserve import V1beta1TritonSpec
from kserve import V1beta1XGBoostSpec
from kserve import V1beta1TransformerSpec
from kserve.api.watch import isvc_watch


AVAILABLE_FRAMEWORKS = {
    'tensorflow': V1beta1TFServingSpec,
    'pytorch': V1beta1TorchServeSpec,
    'sklearn': V1beta1SKLearnSpec,
    'xgboost': V1beta1XGBoostSpec,
    'onnx': V1beta1ONNXRuntimeSpec,
    'triton': V1beta1TritonSpec,
    'pmml': V1beta1PMMLSpec,
    'lightgbm': V1beta1LightGBMSpec
}


def create_predictor_spec(framework, runtime_version, resource_requests, resource_limits, 
                          storage_uri, canary_traffic_percent, service_account, min_replicas, 
                          max_replicas, containers, request_timeout):
    """
    Create and return V1beta1PredictorSpec to be used in a V1beta1InferenceServiceSpec
    object.
    """

    predictor_spec = V1beta1PredictorSpec(
        service_account_name=service_account,
        min_replicas=(min_replicas
                      if min_replicas >= 0
                      else None
                     ),
        max_replicas=(max_replicas
                      if max_replicas > 0 and max_replicas >= min_replicas
                      else None
                     ),
        containers=(containers or None),
        canary_traffic_percent=canary_traffic_percent,
        timeout=request_timeout
    )
    # If the containers field was set, then this is custom model serving.
    if containers:
        return predictor_spec

    if framework not in AVAILABLE_FRAMEWORKS:
        raise ValueError("Error: No matching framework: " + framework)

    setattr(
        predictor_spec,
        framework,
        AVAILABLE_FRAMEWORKS[framework](
            storage_uri=storage_uri, 
            resources=V1ResourceRequirements(
                requests=resource_requests,
                limits=resource_limits
            ),
            runtime_version=runtime_version
        )
    )
    return predictor_spec


def create_transformer_spec(resource_requests, resource_limits, docker_image, 
                            image_args, storage_uri, service_account, min_replicas, 
                            max_replicas, request_timeout):
    """
    Create and return V1beta1TransformerSpec to be used in a V1beta1InferenceServiceSpec
    object.
    """
    if docker_image:
        return V1beta1TransformerSpec(
            min_replicas=(min_replicas if min_replicas >= 0 else None),
            max_replicas=(max_replicas if max_replicas > 0 and max_replicas >= min_replicas else None),
            service_account_name=service_account,
            timeout=request_timeout,
            containers=[V1Container(
                name="kserve-transformer",
                image=docker_image,
                args=(image_args if image_args else None),
                env=[(V1EnvVar(name="STORAGE_URI", value=storage_uri) if storage_uri else None)],
                resources=V1ResourceRequirements(
                    requests=resource_requests, 
                    limits=resource_limits
                )
            )]
        )
    else:
        return None


def create_custom_container_spec(custom_model_spec):
    """
    Given a JSON container spec, return a V1Container object
    representing the container. This is used for passing in
    custom server images. The expected format for the input is:

    { "image": "test/containerimage",
      "port":5000,
      "name": "custom-container" }
    """

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
    return client.V1Container(
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


def create_inference_service(metadata, transformer_spec, predictor_spec):
    """
    Build and return V1beta1InferenceService object.
    """
    return V1beta1InferenceService(
        api_version=constants.KSERVE_V1BETA1,
        kind=constants.KSERVE_KIND,
        metadata=metadata,
        spec=V1beta1InferenceServiceSpec(
            transformer=transformer_spec,
            predictor=predictor_spec
        ),
    )


def submit_api_request(kserve_client, action, name, isvc, namespace=None,
                       watch=False, timeout_seconds=300):
    """
    Creates or updates a Kubernetes custom object. This code is borrowed from the
    KServeClient.create/patch methods as using those directly doesn't allow for
    sending in dicts as the InferenceService object which is needed for supporting passing
    in raw InferenceService serialized YAML.
    """
    custom_obj_api = kserve_client.api_instance
    args = [constants.KSERVE_GROUP, constants.KSERVE_V1BETA1_VERSION,
            namespace, constants.KSERVE_PLURAL]
    if action == 'update':
        outputs = custom_obj_api.patch_namespaced_custom_object(*args, name, isvc)
    else:
        outputs = custom_obj_api.create_namespaced_custom_object(*args, isvc)

    if watch:
        # Sleep 3 to avoid status still be True within a very short time.
        time.sleep(3)
        isvc_watch(
            name=outputs['metadata']['name'],
            namespace=namespace,
            timeout_seconds=timeout_seconds)
    else:
        return outputs


def perform_action(action, model_name, namespace, pred_model_uri, pred_canary_traffic_percent, 
                   pred_framework, pred_runtime_version, pred_resource_requests, 
                   pred_resource_limits, pred_custom_model_spec, service_account, 
                   inferenceservice_yaml, pred_request_timeout, transf_resource_requests, 
                   transf_uri, transf_request_timeout, transf_resource_limits, transf_image, 
                   transf_args, autoscaling_target=0, enable_istio_sidecar=True, 
                   watch_timeout=300, pred_min_replicas=0, pred_max_replicas=0, 
                   transf_min_replicas=0, transf_max_replicas=0):
    """
    Perform the specified action. If the action is not 'delete' and `inferenceService_yaml`
    was provided, the dict representation of the YAML will be sent directly to the
    Kubernetes API. Otherwise, a V1beta1InferenceService object will be built using the
    provided input and then sent for creation/update.
    :return InferenceService JSON output
    """
    kserve_client = KServeClient()

    if inferenceservice_yaml:
        # Overwrite name and namespace if exists
        if namespace:
            inferenceservice_yaml['metadata']['namespace'] = namespace

        if model_name:
            inferenceservice_yaml['metadata']['name'] = model_name
        else:
            model_name = inferenceservice_yaml['metadata']['name']

        isvc = inferenceservice_yaml

    elif action != 'delete':
        # Create annotations
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

        # If a custom model container spec was provided, build the V1Container
        # object using it.
        pred_containers = []
        if pred_custom_model_spec:
            pred_containers = [create_custom_container_spec(pred_custom_model_spec)]

        # Build the V1beta1PredictorSpec and V1beta1TransformerSpec
        predictor_spec = create_predictor_spec(
            pred_framework, pred_runtime_version, pred_resource_requests, pred_resource_limits, 
            pred_model_uri, pred_canary_traffic_percent, service_account, pred_min_replicas, 
            pred_max_replicas, pred_containers, pred_request_timeout
        )

        transformer_spec = create_transformer_spec(
            transf_resource_requests, transf_resource_limits, transf_image, transf_args, 
            transf_uri, service_account, transf_min_replicas, transf_max_replicas,
            transf_request_timeout
        )
        
        isvc = create_inference_service(metadata, transformer_spec, predictor_spec)

    if action == "create":
        submit_api_request(kserve_client, 'create', model_name, isvc, namespace,
                           watch=True, timeout_seconds=watch_timeout)
    elif action == "update":
        submit_api_request(kserve_client, 'update', model_name, isvc, namespace,
                           watch=True, timeout_seconds=watch_timeout)
    elif action == "apply":
        try:
            submit_api_request(kserve_client, 'create', model_name, isvc, namespace,
                               watch=True, timeout_seconds=watch_timeout)
        except Exception:
            submit_api_request(kserve_client, 'update', model_name, isvc, namespace,
                               watch=True, timeout_seconds=watch_timeout)
    elif action == "delete":
        kserve_client.delete(model_name, namespace=namespace)
    else:
        raise ("Error: No matching action: " + action)

    model_status = kserve_client.get(model_name, namespace=namespace)
    return model_status


def main():
    """
    This parses arguments passed in from the CLI and performs the corresponding action.
    """
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--action", type=str, help="Action to execute on KServe", default="create"
    )
    parser.add_argument(
        "--model-name", type=str, help="Name to give to the deployed InferenceService"
    )
    parser.add_argument(
        "--namespace",
        type=str,
        help="Kubernetes namespace where the InferenceService is deployed",
        default="",
    )
    parser.add_argument(
        "--autoscaling-target", type=str, help="Autoscaling target number", default="0"
    )
    parser.add_argument(
        "--enable-istio-sidecar",
        type=strtobool,
        help="Whether to inject istio sidecar",
        default="True"
    )
    parser.add_argument(
        "--enable-isvc-status",
        type=strtobool,
        help="Specifies whether to store the inference service status as the output parameter",
        default="True"
    )
    parser.add_argument(
        "--inferenceservice-yaml",
        type=yaml.safe_load,
        help="Raw InferenceService serialized YAML for deployment",
        default="{}"
    )
    parser.add_argument(
        "--watch-timeout",
        type=str,
        help="Timeout seconds for watching until InferenceService becomes ready",
        default="300"
    )
    parser.add_argument(
        "--service-account",
        type=str,
        help="Service account containing AWS S3, GCP or ABS credentials",
        default="",
    )

    parser.add_argument(
        "--pred-min-replicas", 
        type=str, 
        help="Minimum number of Predictor replicas", 
        default="-1"
    )
    parser.add_argument(
        "--pred-max-replicas", 
        type=str, 
        help="Maximum number of Predictor replicas", 
        default="-1"
    )
    parser.add_argument(
        "--pred-model-uri",
        type=str,
        help="Path of the S3, GCS or ABS compatible directory containing the Predictor model",
    )
    parser.add_argument(
        "--pred-canary-traffic-percent",
        type=str,
        help="The traffic split percentage between the candidate model and the last ready model",
        default="100",
    )
    parser.add_argument(
        "--pred-framework",
        type=str,
        help="Model serving framework to use for the Predictor. Available frameworks: " +
             str(list(AVAILABLE_FRAMEWORKS.keys())),
        default=""
    )
    parser.add_argument(
        "--pred-runtime-version",
        type=str,
        help="Runtime Version of Machine Learning Framework",
        default="latest"
    )
    parser.add_argument(
        "--pred-resource-requests",
        type=json.loads,
        help="CPU and Memory requests for the Predictor",
        default='{"cpu": "0.5", "memory": "512Mi"}',
    )
    parser.add_argument(
        "--pred-resource-limits",
        type=json.loads,
        help="CPU and Memory limits for the Predictor",
        default='{"cpu": "1", "memory": "1Gi"}',
    )
    parser.add_argument(
        "--pred-request-timeout",
        type=str,
        help="Specifies the number of seconds to wait before timing out a request to the Predictor",
        default="60"
    )
    parser.add_argument(
        "--pred-custom-model-spec",
        type=json.loads,
        help="The container spec for a custom Predictor runtime",
        default="{}",
    )

    parser.add_argument(
        "--transf-min-replicas", 
        type=str, 
        help="Minimum number of Transformer replicas", 
        default="-1"
    )
    parser.add_argument(
        "--transf-max-replicas", 
        type=str, 
        help="Maximum number of Transformer replicas", 
        default="-1"
    )
    parser.add_argument(
        "--transf-image",
        type=str,
        help="Docker image used for the Transformer pod container",
    )
    parser.add_argument(
        "--transf-args",
        type=str,
        help="Arguments to the entrypoint of the Transformer pod container, overwrites CMD",
    )
    parser.add_argument(
        "--transf-uri",
        type=str,
        help="Path of the S3, GCS or ABS compatible directory containing the Transformer. Not necessary if the whole pre-/postprocessing logic is in the docker image",
    )
    parser.add_argument(
        "--transf-resource-requests",
        type=json.loads,
        help="CPU and Memory requests for the Transformer",
        default='{"cpu": "0.5", "memory": "512Mi"}',
    )
    parser.add_argument(
        "--transf-resource-limits",
        type=json.loads,
        help="CPU and Memory limits for the Transformer",
        default='{"cpu": "1", "memory": "1Gi"}',
    )
    parser.add_argument(
        "--transf-request-timeout",
        type=str,
        help="Specifies the number of seconds to wait before timing out a request to the Transformer",
        default="60"
    )

    parser.add_argument("--output-path", type=str, help="Path to store URI output")

    args = parser.parse_args()

    action = args.action.lower()
    model_name = args.model_name
    namespace = args.namespace
    autoscaling_target = int(args.autoscaling_target)
    enable_istio_sidecar = args.enable_istio_sidecar
    enable_isvc_status = args.enable_isvc_status
    inferenceservice_yaml = args.inferenceservice_yaml
    watch_timeout = int(args.watch_timeout)
    service_account = args.service_account
    pred_min_replicas = int(args.pred_min_replicas)
    pred_max_replicas = int(args.pred_max_replicas)
    pred_model_uri = args.pred_model_uri
    pred_canary_traffic_percent = int(args.pred_canary_traffic_percent)
    pred_framework = args.pred_framework.lower()
    pred_runtime_version = args.pred_runtime_version.lower()
    pred_resource_requests = args.pred_resource_requests
    pred_resource_limits = args.pred_resource_limits
    pred_request_timeout = int(args.pred_request_timeout)
    pred_custom_model_spec = args.pred_custom_model_spec
    transf_min_replicas = int(args.transf_min_replicas)
    transf_max_replicas = int(args.transf_max_replicas)
    transf_image = args.transf_image
    transf_args = ast.literal_eval(args.transf_args) 
    transf_uri = args.transf_uri
    transf_resource_requests = args.transf_resource_requests
    transf_resource_limits = args.transf_resource_limits
    transf_request_timeout = int(args.transf_request_timeout)
    output_path = args.output_path

    # Default the namespace.
    if not namespace:
        namespace = 'anonymous'
        # If no namespace was provided, but one is listed in the YAML, use that.
        if inferenceservice_yaml and inferenceservice_yaml.get('metadata', {}).get('namespace'):
            namespace = inferenceservice_yaml['metadata']['namespace']

    # Only require model name when an Isvc YAML was not provided.
    if not inferenceservice_yaml and not model_name:
        parser.error('{} argument is required when performing "{}" action'.format(
            'model_name', action
    ))
    # If the action isn't a delete, require 'model-uri' and 'framework' only if an Isvc YAML
    # or custom model container spec are not provided.
    if action != 'delete':
        if not inferenceservice_yaml and not pred_custom_model_spec and not (pred_model_uri and pred_framework):
            parser.error('Arguments for {} and {} are required when performing "{}" action'.format(
                'pred_model_uri', 'pred_framework', action
        ))

    model_status = perform_action(
        action=action,
        model_name=model_name,
        namespace=namespace,
        autoscaling_target=autoscaling_target,
        enable_istio_sidecar=enable_istio_sidecar,
        inferenceservice_yaml=inferenceservice_yaml,
        watch_timeout=watch_timeout,
        service_account=service_account,
        pred_min_replicas=pred_min_replicas,
        pred_max_replicas=pred_max_replicas,
        pred_model_uri=pred_model_uri,
        pred_canary_traffic_percent=pred_canary_traffic_percent,
        pred_framework=pred_framework,
        pred_runtime_version=pred_runtime_version,
        pred_resource_requests=pred_resource_requests,
        pred_resource_limits=pred_resource_limits,
        pred_custom_model_spec=pred_custom_model_spec,
        pred_request_timeout=pred_request_timeout,
        transf_min_replicas=transf_min_replicas,
        transf_max_replicas=transf_max_replicas,
        transf_image=transf_image,
        transf_args=transf_args,
        transf_uri=transf_uri,
        transf_resource_requests=transf_resource_requests,
        transf_resource_limits=transf_resource_limits,
        transf_request_timeout=transf_request_timeout
    )

    print(model_status)

    if action != 'delete':
        # Check whether the model is ready
        for condition in model_status["status"]["conditions"]:
            if condition['type'] == 'Ready':
                if condition['status'] == 'True':
                    print('Model is ready\n')
                    break
                print('Model is timed out, please check the InferenceService events for more details.')
                sys.exit(1)
        try:
            print(model_status["status"]["url"] + " is the Knative domain.")
            print("Sample test commands: \n")
            # model_status['status']['url'] is like http://flowers-sample.kubeflow.example.com/v1/models/flowers-sample
            print("curl -v -X GET %s" % model_status["status"]["url"])
            print("\nIf the above URL is not accessible, it's recommended to setup Knative with a configured DNS.\n"
                  "https://knative.dev/docs/install/installing-istio/#configuring-dns")
        except Exception:
            print("Model is not ready, check the logs for the Knative URL status.")
            sys.exit(1)

    if output_path:
        if not enable_isvc_status:
            model_status = {}
        else:
            try:
                # Remove some less needed fields to reduce output size.
                del model_status['metadata']['managedFields']
                del model_status['status']['conditions']
                if sys.getsizeof(model_status) > 3000:
                    del model_status['components']['predictor']['address']['url']
                    del model_status['components']['predictor']['latestCreatedRevision']
                    del model_status['components']['predictor']['latestReadyRevision']
                    del model_status['components']['predictor']['latestRolledoutRevision']
                    del model_status['components']['predictor']['url']
                    del model_status['spec']
            except KeyError:
                pass

        if not os.path.exists(os.path.dirname(output_path)):
            os.makedirs(os.path.dirname(output_path))
        with open(output_path, "w") as report:
            report.write(json.dumps(model_status, indent=4))


if __name__ == "__main__":
    main()
