#!/usr/bin/env/python3
#
# Copyright (c) Facebook, Inc. and its affiliates.
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
"""Pipeline for bert classification example."""

import json
from kfp.onprem import use_k8s_secret
from kfp import components
from kfp.components import load_component_from_file
from kfp import dsl
from kfp import compiler

INGRESS_GATEWAY = "http://istio-ingressgateway.istio-system.svc.cluster.local"
AUTH = ""
NAMESPACE = "kubeflow-user-example-com"
COOKIE = "authservice_session=" + AUTH

MINIO_ENDPOINT = "http://minio-service.kubeflow:9000"
LOG_BUCKET = "mlpipeline"
TENSORBOARD_IMAGE = "public.ecr.aws/pytorch-samples/tboard:latest"

DEPLOY_NAME = "bertserve"
MODEL_NAME = "bert"

ISVC_NAME = DEPLOY_NAME + "." + NAMESPACE + "." + "example.com"
INPUT_REQUEST = (
    "https://kubeflow-dataset.s3.us-east-2.amazonaws.com"
    "/cifar10/input.json"
)

YAML_FOLDER_PATH = "bert/yaml"
YAML_COMMON_FOLDER = "common"

prepare_tensorboard_op = load_component_from_file(
    "yaml/tensorboard_component.yaml"
)  # pylint: disable=not-callable
prep_op = components.load_component_from_file("yaml/preprocess_component.yaml")  # pylint: disable=not-callable
train_op = components.load_component_from_file("yaml/train_component.yaml")  # pylint: disable=not-callable
deploy_op = load_component_from_file("yaml/deploy_component.yaml")  # pylint: disable=not-callable
pred_op = components.load_component_from_file("yaml/prediction_component.yaml")  # pylint: disable=not-callable

minio_op = components.load_component_from_file("yaml/minio_component.yaml")  # pylint: disable=not-callable


@dsl.pipeline(name="Training pipeline", description="Sample training job test")
def pytorch_bert( # pylint: disable=too-many-arguments
    minio_endpoint=MINIO_ENDPOINT,
    log_bucket=LOG_BUCKET,
    log_dir=f"tensorboard/logs/{dsl.RUN_ID_PLACEHOLDER}",
    mar_path=f"mar/{dsl.RUN_ID_PLACEHOLDER}/model-store",
    config_prop_path=f"mar/{dsl.RUN_ID_PLACEHOLDER}/config",
    model_uri=f"s3://mlpipeline/mar/{dsl.RUN_ID_PLACEHOLDER}",
    tf_image=TENSORBOARD_IMAGE,
    deploy=DEPLOY_NAME,
    namespace=NAMESPACE,
    confusion_matrix_log_dir=f"confusion_matrix/{dsl.RUN_ID_PLACEHOLDER}/",
    num_samples=1000,
    max_epochs=1
):
    """Thid method defines the pipeline tasks and operations"""
    prepare_tb_task = prepare_tensorboard_op(
        log_dir_uri=f"s3://{log_bucket}/{log_dir}",
        image=tf_image,
        pod_template_spec=json.dumps({
            "spec": {
                "containers": [{
                    "env": [
                        {
                            "name": "AWS_ACCESS_KEY_ID",
                            "valueFrom": {
                                "secretKeyRef": {
                                    "name": "mlpipeline-minio-artifact",
                                    "key": "accesskey",
                                }
                            },
                        },
                        {
                            "name": "AWS_SECRET_ACCESS_KEY",
                            "valueFrom": {
                                "secretKeyRef": {
                                    "name": "mlpipeline-minio-artifact",
                                    "key": "secretkey",
                                }
                            },
                        },
                        {
                            "name": "AWS_REGION",
                            "value": "minio"
                        },
                        {
                            "name": "S3_ENDPOINT",
                            "value": f"{minio_endpoint}",
                        },
                        {
                            "name": "S3_USE_HTTPS",
                            "value": "0"
                        },
                        {
                            "name": "S3_VERIFY_SSL",
                            "value": "0"
                        },
                    ]
                }]
            }
        }),
    ).set_display_name("Visualization")

    prep_task = (
        prep_op().after(prepare_tb_task
                       ).set_display_name("Preprocess & Transform")
    )
    confusion_matrix_url = f"minio://{log_bucket}/{confusion_matrix_log_dir}"
    script_args = f"model_name=bert.pth," \
                  f"num_samples={num_samples}," \
                  f"confusion_matrix_url={confusion_matrix_url}"
    # For gpus, set number of gpus and accelerator type
    ptl_args = f"max_epochs={max_epochs}," \
               "profiler=pytorch," \
               "gpus=0," \
               "accelerator=None"
    train_task = (
        train_op(
            input_data=prep_task.outputs["output_data"],
            script_args=script_args,
            ptl_arguments=ptl_args
        ).after(prep_task).set_display_name("Training")
    )
    # For GPU uncomment below line and set GPU limit and node selector
    # ).set_gpu_limit(1).add_node_selector_constraint
    # ('cloud.google.com/gke-accelerator','nvidia-tesla-p4')

    (
        minio_op(
            bucket_name="mlpipeline",
            folder_name=log_dir,
            input_path=train_task.outputs["tensorboard_root"],
            filename="",
        ).after(train_task).set_display_name("Tensorboard Events Pusher")
    )
    minio_mar_upload = (
        minio_op(
            bucket_name="mlpipeline",
            folder_name=mar_path,
            input_path=train_task.outputs["checkpoint_dir"],
            filename="bert_test.mar",
        ).after(train_task).set_display_name("Mar Pusher")
    )
    (
        minio_op(
            bucket_name="mlpipeline",
            folder_name=config_prop_path,
            input_path=train_task.outputs["checkpoint_dir"],
            filename="config.properties",
        ).after(train_task).set_display_name("Conifg Pusher")
    )

    model_uri = str(model_uri)
    # pylint: disable=unused-variable
    isvc_yaml = """
    apiVersion: "serving.kubeflow.org/v1beta1"
    kind: "InferenceService"
    metadata:
      name: {}
      namespace: {}
    spec:
      predictor:
        serviceAccountName: sa
        pytorch:
          storageUri: {}
          resources:
            limits:
              memory: 4Gi   
    """.format(deploy, namespace, model_uri)

    # For GPU inference use below yaml with gpu count and accelerator
    gpu_count = "1"
    accelerator = "nvidia-tesla-p4"
    isvc_gpu_yaml = """
    apiVersion: "serving.kubeflow.org/v1beta1"
    kind: "InferenceService"
    metadata:
      name: {}
      namespace: {}
    spec:
      predictor:
        serviceAccountName: sa
        pytorch:
          storageUri: {}
          resources:
            limits:
              memory: 4Gi   
              nvidia.com/gpu: {}
          nodeSelector:
            cloud.google.com/gke-accelerator: {}
""".format(deploy, namespace, model_uri, gpu_count, accelerator)
    # Update inferenceservice_yaml for GPU inference
    deploy_task = (
        deploy_op(action="apply", inferenceservice_yaml=isvc_yaml
                 ).after(minio_mar_upload).set_display_name("Deployer")
    )

    dsl.get_pipeline_conf().add_op_transformer(
        use_k8s_secret(
            secret_name="mlpipeline-minio-artifact",
            k8s_secret_key_to_env={
                "secretkey": "MINIO_SECRET_KEY",
                "accesskey": "MINIO_ACCESS_KEY",
            },
        )
    )


if __name__ == "__main__":
    compiler.compiler.Compiler().compile(
        pytorch_bert, package_path="pytorch_bert.yaml"
    )
