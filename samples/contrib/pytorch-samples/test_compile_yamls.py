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

import unittest
import json
from kfp import components
from kfp.components import load_component_from_file, load_component_from_url
from kfp import dsl
from kfp import compiler

bert_components = {
    "component_bert_prep":
        components.
        load_component_from_file("bert/yaml/pre_process/component.yaml"),
    "component_bert_train":
        components.load_component_from_file("bert/yaml/train/component.yaml"),
}
cifar_components = {
    "component_cifar10_prep":
        components.
        load_component_from_file("./cifar10/yaml/pre_process/component.yaml"),
    "component_cifar10_train":
        components.
        load_component_from_file("./cifar10/yaml/train/component.yaml"),
}
component_tb = load_component_from_file("common/tensorboard/component.yaml")
component_deploy = load_component_from_file("common/deploy/component.yaml")
prepare_tensorboard_op = load_component_from_file(
    "./common/tensorboard/component.yaml"
)
component_minio = components.load_component_from_file(
    "./common/minio/component.yaml"
)
pred_op = load_component_from_file("./common/prediction/component.yaml")


class ComponentCompileTest(unittest.TestCase):

    def setUp(self):
        super(ComponentCompileTest, self).setUp()
        self.INPUT_REQUEST = "https://kubeflow-dataset.s3.us-east-2.amazonaws.com/" \
                             "cifar10_input/input.json"
        self.DEPLOY_NAME_BERT = "bertserve"
        self.NAMESPACE = "kubeflow-user-example-com"
        self.EXPERIMENT = "Default"
        self.MINIO_ENDPOINT = "http://minio-service.kubeflow:9000"
        self.LOG_BUCKET = "mlpipeline"
        self.TENSORBOARD_IMAGE = "public.ecr.aws/y1x1p2u5/tboard:latest"
        self.DEPLOY_NAME_CIFAR = "torchserve"
        self.MODEL_NAME = "cifar10"
        self.ISVC_NAME = (
            self.DEPLOY_NAME_CIFAR + "." + self.NAMESPACE + "." + "example.com"
        )
        self.COOKIE = ""
        self.INGRESS_GATEWAY = (
            "http://istio-ingressgateway.istio-system.svc.cluster.local"
        )

    def test_cifar10_compile(self):

        @dsl.pipeline(
            name="Training Cifar10 pipeline",
            description="Cifar 10 dataset pipeline",
        )
        def pytorch_cifar10(
            minio_endpoint=self.MINIO_ENDPOINT,
            log_bucket=self.LOG_BUCKET,
            log_dir=f"tensorboard/logs/{dsl.RUN_ID_PLACEHOLDER}",
            mar_path=f"mar/{dsl.RUN_ID_PLACEHOLDER}/model-store",
            config_prop_path=f"mar/{dsl.RUN_ID_PLACEHOLDER}/config",
            model_uri=f"s3://mlpipeline/mar/{dsl.RUN_ID_PLACEHOLDER}",
            tf_image=self.TENSORBOARD_IMAGE,
            deploy=self.DEPLOY_NAME_CIFAR,
            isvc_name=self.ISVC_NAME,
            model=self.MODEL_NAME,
            namespace=self.NAMESPACE,
            confusion_matrix_log_dir=f"confusion_matrix"
            f"/{dsl.RUN_ID_PLACEHOLDER}/",
            checkpoint_dir=f"checkpoint_dir/cifar10",
            input_req=self.INPUT_REQUEST,
            cookie=self.COOKIE,
            ingress_gateway=self.INGRESS_GATEWAY,
        ):
            pod_template_spec = json.dumps({
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
            })

            prepare_tb_task = prepare_tensorboard_op(
                log_dir_uri=f"s3://{log_bucket}/{log_dir}",
                image=tf_image,
                pod_template_spec=pod_template_spec,
            ).set_display_name("Visualization")
            component_cifar10_prep = cifar_components["component_cifar10_prep"]
            prep_task = (
                component_cifar10_prep().after(prepare_tb_task).
                set_display_name("Preprocess & Transform")
            )
            component_cifar10_train = cifar_components["component_cifar10_train"
                                                      ]
            train_task = (
                component_cifar10_train(
                    input_data=prep_task.outputs["output_data"],
                    profiler="pytorch",
                    confusion_matrix_url=
                    f"minio://{log_bucket}/{confusion_matrix_log_dir}",
                    # For GPU set gpu count and accelerator type
                    gpus=0,
                    accelerator="None",
                ).after(prep_task).set_display_name("Training")
            )

            minio_tb_upload = (
                component_minio(
                    bucket_name="mlpipeline",
                    folder_name=log_dir,
                    input_path=train_task.outputs["tensorboard_root"],
                    filename="",
                ).after(train_task
                       ).set_display_name("Tensorboard Events Pusher")
            )

            minio_checkpoint_dir_upload = (
                component_minio(
                    bucket_name="mlpipeline",
                    folder_name=checkpoint_dir,
                    input_path=train_task.outputs["checkpoint_dir"],
                    filename="",
                ).after(train_task).set_display_name("checkpoint_dir Pusher")
            )

            minio_mar_upload = (
                component_minio(
                    bucket_name="mlpipeline",
                    folder_name=mar_path,
                    input_path=train_task.outputs["checkpoint_dir"],
                    filename="cifar10_test.mar",
                ).after(train_task).set_display_name("Mar Pusher")
            )

            minio_config_upload = (
                component_minio(
                    bucket_name="mlpipeline",
                    folder_name=config_prop_path,
                    input_path=train_task.outputs["checkpoint_dir"],
                    filename="config.properties",
                ).after(train_task).set_display_name("Conifg Pusher")
            )

            model_uri = str(model_uri)
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
                component_deploy(
                    action="apply", inferenceservice_yaml=isvc_yaml
                ).after(minio_mar_upload).set_display_name("Deployer")
            )
            pred_task = (
                pred_op(
                    host_name=self.ISVC_NAME,
                    input_request=self.INPUT_REQUEST,
                    cookie=self.COOKIE,
                    url=self.INGRESS_GATEWAY,
                    model=self.MODEL_NAME,
                    inference_type="predict",
                ).after(deploy_task).set_display_name("Prediction")
            )
            explain_task = (
                pred_op(
                    host_name=self.ISVC_NAME,
                    input_request=self.INPUT_REQUEST,
                    cookie=self.COOKIE,
                    url=self.INGRESS_GATEWAY,
                    model=self.MODEL_NAME,
                    inference_type="explain",
                ).after(pred_task).set_display_name("Explanation")
            )

        compiler.Compiler().compile(
            pytorch_cifar10, "pytorch.tar.gz", type_check=True
        )

    def test_bert_compile(self):

        @dsl.pipeline(
            name="Training pipeline", description="Sample training job test"
        )
        def pytorch_bert(
            minio_endpoint=self.MINIO_ENDPOINT,
            log_bucket=self.LOG_BUCKET,
            log_dir=f"tensorboard/logs/{dsl.RUN_ID_PLACEHOLDER}",
            mar_path=f"mar/{dsl.RUN_ID_PLACEHOLDER}/model-store",
            config_prop_path=f"mar/{dsl.RUN_ID_PLACEHOLDER}/config",
            model_uri=f"s3://mlpipeline/mar/{dsl.RUN_ID_PLACEHOLDER}",
            tf_image=self.TENSORBOARD_IMAGE,
            deploy=self.DEPLOY_NAME_BERT,
            namespace=self.NAMESPACE,
            confusion_matrix_log_dir=f"confusion_matrix/{dsl.RUN_ID_PLACEHOLDER}/",
            num_samples=1000,
        ):
            prepare_tb_task = component_tb(
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
                                    "value": "minio",
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
            component_bert_prep = bert_components["component_bert_prep"]
            prep_task = (
                component_bert_prep().after(prepare_tb_task).
                set_display_name("Preprocess & Transform")
            )
            component_bert_train = bert_components["component_bert_train"]
            train_task = (
                component_bert_train(
                    input_data=prep_task.outputs["output_data"],
                    profiler="pytorch",
                    confusion_matrix_url=
                    f"minio://{log_bucket}/{confusion_matrix_log_dir}",
                    num_samples=num_samples,
                    # For GPU set gpu count and accelerator type
                    gpus=0,
                    accelerator="None",
                ).after(prep_task).set_display_name("Training")
            )

            minio_tb_upload = (
                component_minio(
                    bucket_name="mlpipeline",
                    folder_name=log_dir,
                    input_path=train_task.outputs["tensorboard_root"],
                    filename="",
                ).after(train_task
                       ).set_display_name("Tensorboard Events Pusher")
            )
            minio_mar_upload = (
                component_minio(
                    bucket_name="mlpipeline",
                    folder_name=mar_path,
                    input_path=train_task.outputs["checkpoint_dir"],
                    filename="bert_test.mar",
                ).after(train_task).set_display_name("Mar Pusher")
            )
            minio_config_upload = (
                component_minio(
                    bucket_name="mlpipeline",
                    folder_name=config_prop_path,
                    input_path=train_task.outputs["checkpoint_dir"],
                    filename="config.properties",
                ).after(train_task).set_display_name("Conifg Pusher")
            )

            model_uri = str(model_uri)
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
                component_deploy(
                    action="apply", inferenceservice_yaml=isvc_yaml
                ).after(minio_mar_upload).set_display_name("Deployer")
            )

        compiler.Compiler().compile(
            pytorch_bert, "pytorch.tar.gz", type_check=True
        )
