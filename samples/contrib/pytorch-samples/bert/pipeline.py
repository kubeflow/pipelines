#!/usr/bin/env python
# coding: utf-8

import kfp
import json
from kfp.onprem import use_k8s_secret
from kfp import components
from kfp.components import load_component_from_file
from kfp import dsl
from kfp import compiler


yaml_folder_path = "examples/bert/yaml"
yaml_common_folder = "examples/common"

prepare_tensorboard_op = load_component_from_file(f"{yaml_common_folder}/tensorboard/component.yaml")
prep_op = components.load_component_from_file(f"{yaml_folder_path}/pre_process/component.yaml")
train_op = components.load_component_from_file(f"{yaml_folder_path}/train/component.yaml")
deploy_op = load_component_from_file(f"{yaml_common_folder}/deploy/component.yaml")


minio_op = components.load_component_from_file(f"{yaml_common_folder}/minio/component.yaml")


@dsl.pipeline(name="Training pipeline", description="Sample training job test")
def pytorch_bert(
    minio_endpoint="http://minio-service.kubeflow:9000",
    log_bucket="mlpipeline",
    log_dir=f"tensorboard/logs/{dsl.RUN_ID_PLACEHOLDER}",
    mar_path=f"mar/{dsl.RUN_ID_PLACEHOLDER}/model-store",
    config_prop_path=f"mar/{dsl.RUN_ID_PLACEHOLDER}/config",
    model_uri=f"s3://mlpipeline/mar/{dsl.RUN_ID_PLACEHOLDER}",
    tf_image="jagadeeshj/tb_plugin:v1.8",
    deploy="bertserve",
    namespace="kubeflow-user-example-com",
    confusion_matrix_log_dir=f"confusion_matrix/{dsl.RUN_ID_PLACEHOLDER}/",
    num_samples=1000
):

    prepare_tb_task = prepare_tensorboard_op(
        log_dir_uri=f"s3://{log_bucket}/{log_dir}",
        image=tf_image,
        pod_template_spec=json.dumps(
            {
                "spec": {
                    "containers": [
                        {
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
                                {"name": "AWS_REGION", "value": "minio"},
                                {"name": "S3_ENDPOINT", "value": f"{minio_endpoint}"},
                                {"name": "S3_USE_HTTPS", "value": "0"},
                                {"name": "S3_VERIFY_SSL", "value": "0"},
                            ]
                        }
                    ]
                }
            }
        ),
    ).set_display_name("Visualization")

    prep_task = prep_op().after(prepare_tb_task).set_display_name("Preprocess & Transform")
    train_task = (
        train_op(
            input_data=prep_task.outputs["output_data"],
            profiler="pytorch",
            confusion_matrix_url=f"minio://{log_bucket}/{confusion_matrix_log_dir}",
            num_samples=num_samples
        )
        .apply(
            use_k8s_secret(
                secret_name="mlpipeline-minio-artifact",
                k8s_secret_key_to_env={
                    "secretkey": "MINIO_SECRET_KEY",
                    "accesskey": "MINIO_ACCESS_KEY",
                },
            )
        )
        .after(prep_task)
        .set_display_name("Training")
    )

    minio_tb_upload = (
        minio_op(
            bucket_name="mlpipeline",
            folder_name=log_dir,
            input_path=train_task.outputs["tensorboard_root"],
            filename="",
        )
        .apply(
            use_k8s_secret(
                secret_name="mlpipeline-minio-artifact",
                k8s_secret_key_to_env={
                    "secretkey": "MINIO_SECRET_KEY",
                    "accesskey": "MINIO_ACCESS_KEY",
                },
            )
        )
        .after(train_task)
        .set_display_name("Tensorboard Events Pusher")
    )
    minio_mar_upload = (
        minio_op(
            bucket_name="mlpipeline",
            folder_name=mar_path,
            input_path=train_task.outputs["checkpoint_dir"],
            filename="bert_test.mar",
        )
        .apply(
            use_k8s_secret(
                secret_name="mlpipeline-minio-artifact",
                k8s_secret_key_to_env={
                    "secretkey": "MINIO_SECRET_KEY",
                    "accesskey": "MINIO_ACCESS_KEY",
                },
            )
        )
        .after(train_task)
        .set_display_name("Mar Pusher")
    )
    minio_config_upload = (
        minio_op(
            bucket_name="mlpipeline",
            folder_name=config_prop_path,
            input_path=train_task.outputs["checkpoint_dir"],
            filename="config.properties",
        )
        .apply(
            use_k8s_secret(
                secret_name="mlpipeline-minio-artifact",
                k8s_secret_key_to_env={
                    "secretkey": "MINIO_SECRET_KEY",
                    "accesskey": "MINIO_ACCESS_KEY",
                },
            )
        )
        .after(train_task)
        .set_display_name("Conifg Pusher")
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
    """.format(
        deploy, namespace, model_uri
    )
    deploy_task = (
        deploy_op(action="apply", inferenceservice_yaml=isvc_yaml)
        .after(minio_mar_upload)
        .set_display_name("Deployer")
    )


if __name__ == "__main__":
    kfp.compiler.compiler.Compiler().compile(pytorch_bert, package_path="pytorch_bert.yaml")
