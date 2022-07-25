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
#pylint: disable=not-callable,unused-variable
"""Test for component compilation."""
import os
import unittest
import json
import pytest
from kfp import components
from kfp.components import load_component_from_file
from kfp import dsl
from kfp import compiler

tests_dir, _ = os.path.split(os.path.abspath(__file__))

templates_dir = os.path.join(os.path.dirname(tests_dir), "templates")

components_dir = tests_dir.split("PyTorch")[0].rstrip("/")

BERT_COMPONENTS = {
    "component_bert_prep":
        components.
        load_component_from_file(f"{templates_dir}/preprocess_component.yaml"),
    "component_bert_train":
        components.
        load_component_from_file(f"{templates_dir}/train_component.yaml"),
}
CIFAR_COMPONENTS = {
    "component_cifar10_prep":
        components.
        load_component_from_file(f"{templates_dir}/preprocess_component.yaml"),
    "component_cifar10_train":
        components.
        load_component_from_file(f"{templates_dir}/train_component.yaml"),
}
COMPONENT_TB = load_component_from_file(
    f"{templates_dir}/tensorboard_component.yaml"
)
COMPONENT_DEPLOY = load_component_from_file(
    f"{components_dir}/kserve/component.yaml"
)
COMPONENT_MNIO = components.load_component_from_file(
    f"{templates_dir}/minio_component.yaml"
)
PRED_OP = load_component_from_file(f"{templates_dir}/prediction_component.yaml")


class ComponentCompileTest(unittest.TestCase):  #pylint: disable=too-many-instance-attributes
    """Test cases for compilation of yamls."""

    def setUp(self):
        """Set up."""
        super().setUp()
        self.input_request = "./compile_test.json"
        self.deploy_name_bert = "bertserve"
        self.namespace = "kubeflow-user-example-com"
        self.experiment = "Default"
        self.minio_endpoint = "http://minio-service.kubeflow:9000"
        self.log_bucket = "mlpipeline"
        self.tensorboard_image = "public.ecr.aws/pytorch-samples/tboard:latest"
        self.deploy_name_cifar = "torchserve"
        self.model_name = "cifar10"
        self.isvc_name = (
            self.deploy_name_cifar + "." + self.namespace + "." + "example.com"
        )
        self.cookie = ""
        self.ingress_gateway = (
            "http://istio-ingressgateway.istio-system.svc.cluster.local"
        )

    def test_cifar10_compile(self):
        """Test Cifar10 yamls compile."""

        @dsl.pipeline(
            name="Training Cifar10 pipeline",
            description="Cifar 10 dataset pipeline",
        )  #pylint: disable=too-many-arguments,too-many-locals
        def pytorch_cifar10(
            minio_endpoint=self.minio_endpoint,
            log_bucket=self.log_bucket,
            log_dir=f"tensorboard/logs/{dsl.RUN_ID_PLACEHOLDER}",
            mar_path=f"mar/{dsl.RUN_ID_PLACEHOLDER}/model-store",
            config_prop_path=f"mar/{dsl.RUN_ID_PLACEHOLDER}/config",
            model_uri=f"s3://mlpipeline/mar/{dsl.RUN_ID_PLACEHOLDER}",
            tf_image=self.tensorboard_image,
            deploy=self.deploy_name_cifar,
            namespace=self.namespace,
            confusion_matrix_log_dir=f"confusion_matrix"
            f"/{dsl.RUN_ID_PLACEHOLDER}/",
            checkpoint_dir="checkpoint_dir/cifar10",
        ):
            """Cifar10 pipelines."""
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

            prepare_tb_task = COMPONENT_TB(
                log_dir_uri=f"s3://{log_bucket}/{log_dir}",
                image=tf_image,
                pod_template_spec=pod_template_spec,
            ).set_display_name("Visualization")
            component_cifar10_prep = CIFAR_COMPONENTS["component_cifar10_prep"]
            prep_task = (
                component_cifar10_prep().after(prepare_tb_task).
                set_display_name("Preprocess & Transform")
            )
            component_cifar10_train = CIFAR_COMPONENTS["component_cifar10_train"
                                                      ]
            confusion_matrix_url = \
                f"minio://{log_bucket}/{confusion_matrix_log_dir}"
            script_args = f"model_name=resnet.pth," \
                          f"confusion_matrix_url={confusion_matrix_url}"
            # For gpus, set number of gpus and accelerator type
            ptl_args = "max_epochs=1, " \
                       "gpus=0, " \
                       "accelerator=None, " \
                       "profiler=pytorch"
            train_task = (
                component_cifar10_train(
                    input_data=prep_task.outputs["output_data"],
                    script_args=script_args,
                    ptl_arguments=ptl_args
                ).after(prep_task).set_display_name("Training")
            )

            minio_tb_upload = (
                COMPONENT_MNIO(
                    bucket_name="mlpipeline",
                    folder_name=log_dir,
                    input_path=train_task.outputs["tensorboard_root"],
                    filename="",
                ).after(train_task
                       ).set_display_name("Tensorboard Events Pusher")
            )

            minio_checkpoint_dir_upload = (
                COMPONENT_MNIO(
                    bucket_name="mlpipeline",
                    folder_name=checkpoint_dir,
                    input_path=train_task.outputs["checkpoint_dir"],
                    filename="",
                ).after(train_task).set_display_name("checkpoint_dir Pusher")
            )

            minio_mar_upload = (
                COMPONENT_MNIO(
                    bucket_name="mlpipeline",
                    folder_name=mar_path,
                    input_path=train_task.outputs["checkpoint_dir"],
                    filename="cifar10_test.mar",
                ).after(train_task).set_display_name("Mar Pusher")
            )

            minio_config_upload = (
                COMPONENT_MNIO(
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
                COMPONENT_DEPLOY(
                    action="apply", inferenceservice_yaml=isvc_yaml
                ).after(minio_mar_upload).set_display_name("Deployer")
            )
            pred_task = (
                PRED_OP(
                    host_name=self.isvc_name,
                    input_request=self.input_request,
                    cookie=self.cookie,
                    url=self.ingress_gateway,
                    model=self.model_name,
                    inference_type="predict",
                ).after(deploy_task).set_display_name("Prediction")
            )
            explain_task = (
                PRED_OP(
                    host_name=self.isvc_name,
                    input_request=self.input_request,
                    cookie=self.cookie,
                    url=self.ingress_gateway,
                    model=self.model_name,
                    inference_type="explain",
                ).after(pred_task).set_display_name("Explanation")
            )

        compiler.Compiler().compile(
            pytorch_cifar10, "pytorch.tar.gz", type_check=True
        )

    def test_bert_compile(self):
        """Test bert yamls compilation."""

        @dsl.pipeline(
            name="Training pipeline", description="Sample training job test"
        )  #pylint: disable=too-many-arguments,too-many-locals
        def pytorch_bert(
            minio_endpoint=self.minio_endpoint,
            log_bucket=self.log_bucket,
            log_dir=f"tensorboard/logs/{dsl.RUN_ID_PLACEHOLDER}",
            mar_path=f"mar/{dsl.RUN_ID_PLACEHOLDER}/model-store",
            config_prop_path=f"mar/{dsl.RUN_ID_PLACEHOLDER}/config",
            model_uri=f"s3://mlpipeline/mar/{dsl.RUN_ID_PLACEHOLDER}",
            tf_image=self.tensorboard_image,
            deploy=self.deploy_name_bert,
            namespace=self.namespace,
            confusion_matrix_log_dir=
            f"confusion_matrix/{dsl.RUN_ID_PLACEHOLDER}/",
            num_samples=1000,
            max_epochs=1
        ):
            """Bert Pipeline."""
            prepare_tb_task = COMPONENT_TB(
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
            component_bert_prep = BERT_COMPONENTS["component_bert_prep"]
            prep_task = (
                component_bert_prep().after(prepare_tb_task).
                set_display_name("Preprocess & Transform")
            )
            component_bert_train = BERT_COMPONENTS["component_bert_train"]
            confusion_matrix_url = \
                f"minio://{log_bucket}/{confusion_matrix_log_dir}"
            script_args = f"model_name=bert.pth," \
                          f"num_samples={num_samples}," \
                          f"confusion_matrix_url={confusion_matrix_url}"
            # For gpus, set number of gpus and accelerator type
            ptl_args = f"max_epochs={max_epochs}," \
                       "profiler=pytorch," \
                       "gpus=0," \
                       "accelerator=None"
            train_task = (
                component_bert_train(
                    input_data=prep_task.outputs["output_data"],
                    script_args=script_args,
                    ptl_arguments=ptl_args
                ).after(prep_task).set_display_name("Training")
            )

            minio_tb_upload = (
                COMPONENT_MNIO(
                    bucket_name="mlpipeline",
                    folder_name=log_dir,
                    input_path=train_task.outputs["tensorboard_root"],
                    filename="",
                ).after(train_task
                       ).set_display_name("Tensorboard Events Pusher")
            )
            minio_mar_upload = (
                COMPONENT_MNIO(
                    bucket_name="mlpipeline",
                    folder_name=mar_path,
                    input_path=train_task.outputs["checkpoint_dir"],
                    filename="bert_test.mar",
                ).after(train_task).set_display_name("Mar Pusher")
            )
            minio_config_upload = (
                COMPONENT_MNIO(
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
                COMPONENT_DEPLOY(
                    action="apply", inferenceservice_yaml=isvc_yaml
                ).after(minio_mar_upload).set_display_name("Deployer")
            )

        compiler.Compiler().compile(
            pytorch_bert, "pytorch.tar.gz", type_check=True
        )

    def test_cifar10_compile_fail(self):
        """Test Cifar10 yamls compile."""

        @dsl.pipeline(
            name="Training Cifar10 pipeline",
            description="Cifar 10 dataset pipeline",
        )  #pylint: disable=too-many-arguments,too-many-locals
        def pytorch_cifar10(
            minio_endpoint=self.minio_endpoint,
            log_bucket=self.log_bucket,
            log_dir=f"tensorboard/logs/{dsl.RUN_ID_PLACEHOLDER}",
            mar_path=f"mar/{dsl.RUN_ID_PLACEHOLDER}/model-store",
            config_prop_path=f"mar/{dsl.RUN_ID_PLACEHOLDER}/config",
            model_uri=f"s3://mlpipeline/mar/{dsl.RUN_ID_PLACEHOLDER}",
            tf_image=self.tensorboard_image,
            deploy=self.deploy_name_cifar,
            namespace=self.namespace,
            confusion_matrix_log_dir=f"confusion_matrix"
            f"/{dsl.RUN_ID_PLACEHOLDER}/",  #pylint: disable=f-string-without-interpolation
            checkpoint_dir=f"checkpoint_dir/cifar10",
        ):
            """Cifar10 pipelines."""
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

            prepare_tb_task = COMPONENT_TB(
                log_dir_uri=f"s3://{log_bucket}/{log_dir}",
                image=tf_image,
                pod_template_spec=pod_template_spec,
            ).set_display_name("Visualization")
            component_cifar10_prep = ""
            prep_task = (
                component_cifar10_prep().after(prepare_tb_task).
                set_display_name("Preprocess & Transform")
            )
            confusion_matrix_url = \
                f"minio://{log_bucket}/{confusion_matrix_log_dir}"
            script_args = f"model_name=resnet.pth," \
                          f"confusion_matrix_url={confusion_matrix_url}"
            # For gpus, set number of gpus and accelerator type
            ptl_args = "max_epochs=1, " \
                       "gpus=0, " \
                       "accelerator=None, " \
                       "profiler=pytorch"
            component_cifar10_train = ""
            train_task = (
                component_cifar10_train(
                    input_data=prep_task.outputs["output_data"],
                    script_args=script_args,
                    ptl_arguments=ptl_args
                ).after(prep_task).set_display_name("Training")
            )

            minio_tb_upload = (
                COMPONENT_MNIO(
                    bucket_name="mlpipeline",
                    folder_name=log_dir,
                    input_path=train_task.outputs["tensorboard_root"],
                    filename="",
                ).after(train_task
                       ).set_display_name("Tensorboard Events Pusher")
            )

            minio_checkpoint_dir_upload = (
                COMPONENT_MNIO(
                    bucket_name="mlpipeline",
                    folder_name=checkpoint_dir,
                    input_path=train_task.outputs["checkpoint_dir"],
                    filename="",
                ).after(train_task).set_display_name("checkpoint_dir Pusher")
            )

            minio_mar_upload = (
                COMPONENT_MNIO(
                    bucket_name="mlpipeline",
                    folder_name=mar_path,
                    input_path=train_task.outputs["checkpoint_dir"],
                    filename="cifar10_test.mar",
                ).after(train_task).set_display_name("Mar Pusher")
            )

            minio_config_upload = (
                COMPONENT_MNIO(
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
                COMPONENT_DEPLOY(
                    action="apply", inferenceservice_yaml=isvc_yaml
                ).after(minio_mar_upload).set_display_name("Deployer")
            )
            pred_task = (
                PRED_OP(
                    host_name=self.isvc_name,
                    input_request=self.input_request,
                    cookie=self.cookie,
                    url=self.ingress_gateway,
                    model=self.model_name,
                    inference_type="predict",
                ).after(deploy_task).set_display_name("Prediction")
            )
            explain_task = (
                PRED_OP(
                    host_name=self.isvc_name,
                    input_request=self.input_request,
                    cookie=self.cookie,
                    url=self.ingress_gateway,
                    model=self.model_name,
                    inference_type="explain",
                ).after(pred_task).set_display_name("Explanation")
            )

        with pytest.raises(TypeError):
            compiler.Compiler().compile(
                pytorch_cifar10, "pytorch.tar.gz", type_check=True
            )

    def test_bert_compile_fail(self):
        """Test bert yamls compilation."""

        @dsl.pipeline(
            name="Training pipeline", description="Sample training job test"
        )  #pylint: disable=too-many-arguments,too-many-locals
        def pytorch_bert(
            minio_endpoint=self.minio_endpoint,
            log_bucket=self.log_bucket,
            log_dir=f"tensorboard/logs/{dsl.RUN_ID_PLACEHOLDER}",
            mar_path=f"mar/{dsl.RUN_ID_PLACEHOLDER}/model-store",
            config_prop_path=f"mar/{dsl.RUN_ID_PLACEHOLDER}/config",
            model_uri=f"s3://mlpipeline/mar/{dsl.RUN_ID_PLACEHOLDER}",
            tf_image=self.tensorboard_image,
            deploy=self.deploy_name_bert,
            namespace=self.namespace,
            confusion_matrix_log_dir=
            f"confusion_matrix/{dsl.RUN_ID_PLACEHOLDER}/",
            num_samples=1000,
            max_epochs=1
        ):
            """Bert Pipeline."""
            prepare_tb_task = COMPONENT_TB(
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
            component_bert_prep = ""
            prep_task = (
                component_bert_prep().after(prepare_tb_task).
                set_display_name("Preprocess & Transform")
            )
            confusion_matrix_url = \
                f"minio://{log_bucket}/{confusion_matrix_log_dir}"
            script_args = f"model_name=bert.pth," \
                          f"num_samples={num_samples}," \
                          f"confusion_matrix_url={confusion_matrix_url}"
            # For gpus, set number of gpus and accelerator type
            ptl_args = f"max_epochs={max_epochs}," \
                       "profiler=pytorch," \
                       "gpus=0," \
                       "accelerator=None"
            component_bert_train = ""
            train_task = (
                component_bert_train(
                    input_data=prep_task.outputs["output_data"],
                    script_args=script_args,
                    ptl_arguments=ptl_args
                ).after(prep_task).set_display_name("Training")
            )

            minio_tb_upload = (
                COMPONENT_MNIO(
                    bucket_name="mlpipeline",
                    folder_name=log_dir,
                    input_path=train_task.outputs["tensorboard_root"],
                    filename="",
                ).after(train_task
                       ).set_display_name("Tensorboard Events Pusher")
            )
            minio_mar_upload = (
                COMPONENT_MNIO(
                    bucket_name="mlpipeline",
                    folder_name=mar_path,
                    input_path=train_task.outputs["checkpoint_dir"],
                    filename="bert_test.mar",
                ).after(train_task).set_display_name("Mar Pusher")
            )
            minio_config_upload = (
                COMPONENT_MNIO(
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
                COMPONENT_DEPLOY(
                    action="apply", inferenceservice_yaml=isvc_yaml
                ).after(minio_mar_upload).set_display_name("Deployer")
            )

        with pytest.raises(TypeError):
            compiler.Compiler().compile(
                pytorch_bert, "pytorch.tar.gz", type_check=True
            )
