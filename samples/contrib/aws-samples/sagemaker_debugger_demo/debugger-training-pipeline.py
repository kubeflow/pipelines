#!/usr/bin/env python3

import kfp
import json
import os
import copy
from kfp import components
from kfp import dsl


cur_file_dir = os.path.dirname(__file__)
components_dir = os.path.join(cur_file_dir, "../../../../components/aws/sagemaker/")

sagemaker_train_op = components.load_component_from_file(
    components_dir + "/train/component.yaml"
)


def training_input(input_name, s3_uri, content_type):
    return {
        "ChannelName": input_name,
        "DataSource": {"S3DataSource": {"S3Uri": s3_uri, "S3DataType": "S3Prefix"}},
        "ContentType": content_type,
    }


def training_debug_hook(s3_uri, collection_dict):
    return {
        "S3OutputPath": s3_uri,
        "CollectionConfigurations": format_collection_config(collection_dict),
    }


def format_collection_config(collection_dict):
    output = []
    for key, val in collection_dict.items():
        output.append({"CollectionName": key, "CollectionParameters": val})
    return output


def training_debug_rules(rule_name, parameters, region):
    debugger_rule_accounts = {
        "ap-east-1": 199566480951,
        "ap-northeast-1": 430734990657,
        "ap-northeast-2": 578805364391,
        "ap-south-1": 904829902805,
        "ap-southeast-1": 972752614525,
        "ap-southeast-2": 184798709955,
        "ca-central-1": 519511493484,
        "cn-north-1": 618459771430,
        "cn-northwest-1": 658757709296,
        "eu-central-1": 482524230118,
        "eu-north-1": 314864569078,
        "eu-west-1": 929884845733,
        "eu-west-2": 250201462417,
        "eu-west-3": 447278800020,
        "me-south-1": 986000313247,
        "sa-east-1": 818342061345,
        "us-east-1": 503895931360,
        "us-east-2": 915447279597,
        "us-west-1": 685455198987,
        "us-west-2": 895741380848,
    }

    return {
        "RuleConfigurationName": rule_name,
        "RuleEvaluatorImage": f"{debugger_rule_accounts[region]}.dkr.ecr.{region}.amazonaws.com/sagemaker-debugger-rules:latest",
        "RuleParameters": parameters,
    }


collections = {
    "feature_importance": {"save_interval": "5"},
    "losses": {"save_interval": "10"},
    "average_shap": {"save_interval": "5"},
    "metrics": {"save_interval": "3"},
}


bad_hyperparameters = {
    "max_depth": "5",
    "eta": "0",
    "gamma": "4",
    "min_child_weight": "6",
    "silent": "0",
    "subsample": "0.7",
    "num_round": "50",
}


@dsl.pipeline(
    name="XGBoost Training Pipeline with bad hyperparameters",
    description="SageMaker training job test with debugger",
)
def training(
    role_arn="",
    bucket_name="my-bucket",
    region="us-east-1",
    image="683313688378.dkr.ecr.us-east-1.amazonaws.com/sagemaker-xgboost:0.90-2-cpu-py3",
):
    train_channels = [
        training_input(
            "train",
            f"s3://{bucket_name}/mnist_kmeans_example/input/valid_data.csv",
            "text/csv",
        )
    ]
    train_debug_rules = [
        training_debug_rules(
            "LossNotDecreasing",
            {"rule_to_invoke": "LossNotDecreasing", "tensor_regex": ".*"},
        ),
        training_debug_rules(
            "Overtraining",
            {
                "rule_to_invoke": "Overtraining",
                "patience_train": "10",
                "patience_validation": "20",
            },
        ),
    ]
    training = sagemaker_train_op(
        region=region,
        image=image,
        hyperparameters=bad_hyperparameters,
        channels=train_channels,
        instance_type="ml.m5.2xlarge",
        model_artifact_path=f"s3://{bucket_name}/mnist_kmeans_example/output/model",
        debug_hook_config=training_debug_hook(
            f"s3://{bucket_name}/mnist_kmeans_example/hook_config", collections
        ),
        debug_rule_config=train_debug_rules,
        role=role_arn,
    )


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(training, __file__ + ".zip")
