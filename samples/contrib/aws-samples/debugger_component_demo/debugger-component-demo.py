#!/usr/bin/env python3

# Uncomment the apply(use_aws_secret()) below if you are not using OIDC
# more info : https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/README.md

import kfp
import json
import os
import copy
from kfp import components
from kfp import dsl
from kfp.aws import use_aws_secret


cur_file_dir = os.path.dirname(__file__)
components_dir = os.path.join(cur_file_dir, '../../../../components/aws/sagemaker/')

sagemaker_train_op = components.load_component_from_file(components_dir + '/train/component.yaml')

debug_hook = {
    'S3OutputPath':'s3://kubeflow-pipeline-data/mnist_kmeans_example/hook_config'
}

debug_hook['CollectionConfigurations'] = []

collection_list = {
    'feature_importance' : {
        'save_interval': '5'
    }, 
    'losses' : {
        'save_interval': '10'
    },
    'average_shap': {
        'save_interval': '5'
    },
    'metrics': {
        'save_interval': '5'
    }
}

for key, val in collection_list.items():
    debug_hook['CollectionConfigurations'].append({'CollectionName': key, 'CollectionParameters': val})

loss_rule = {
    'RuleConfigurationName': 'LossNotDecreasing',
    'RuleEvaluatorImage': '503895931360.dkr.ecr.us-east-1.amazonaws.com/sagemaker-debugger-rules:latest',
    'RuleParameters': {
        'rule_to_invoke': 'LossNotDecreasing',
        'tensor_regex': '.*'
    }
}

overtraining_rule = {
    'RuleConfigurationName': 'Overtraining',
    'RuleEvaluatorImage': '503895931360.dkr.ecr.us-east-1.amazonaws.com/sagemaker-debugger-rules:latest',
    'RuleParameters': {
        'rule_to_invoke': 'Overtraining',
        'patience_train': '10',
        'patience_validation': '20'
    }
}

debug_rule_configurations=[loss_rule, overtraining_rule]

bad_hyperparameters = {
    'objective': 'reg:squarederror',
    'max_depth': '5', 
    'eta': '0', 
    'gamma': '4', 
    'min_child_weight': '6', 
    'silent': '0', 
    'subsample': '0.7', 
    'num_round': '50'
}

channelObjList = []

channelObj = {
    'ChannelName': '',
    'DataSource': {
        'S3DataSource': {
            'S3Uri': '',
            'S3DataType': 'S3Prefix',
            'S3DataDistributionType': 'FullyReplicated'
        }
    },
    'ContentType': 'text/csv',
    'CompressionType': 'None',
    'RecordWrapperType': 'None',
    'InputMode': 'File'
}

channelObj['ChannelName'] = 'train'
channelObj['DataSource']['S3DataSource']['S3Uri'] = 's3://kubeflow-pipeline-data/mnist_kmeans_example/input/valid_data.csv'
channelObjList.append(copy.deepcopy(channelObj))


@dsl.pipeline(
    name='xgboost-mnist-debugger',
    description='SageMaker training job test with debugger'
)
def training(
        region='us-east-1',
        endpoint_url='',
        image='683313688378.dkr.ecr.us-east-1.amazonaws.com/sagemaker-xgboost:0.90-2-cpu-py3',
        training_input_mode='File',
        hyperparameters=bad_hyperparameters,
        channels=channelObjList,
        instance_type='ml.m5.2xlarge',
        instance_count=1,
        volume_size=50,
        max_run_time=3600,
        model_artifact_path='s3://kubeflow-pipeline-data/mnist_kmeans_example/output/model',
        output_encryption_key='',
        network_isolation=True,
        traffic_encryption=False,
        spot_instance=False,
        max_wait_time=3600,
        checkpoint_config={},
        debug_hook_config=debug_hook,
        debug_rule_config=debug_rule_configurations,
        tensorboard_output_config={},
        role=''
        ):
    training = sagemaker_train_op(
        region=region,
        endpoint_url=endpoint_url,
        image=image,
        training_input_mode=training_input_mode,
        hyperparameters=hyperparameters,
        channels=channels,
        instance_type=instance_type,
        instance_count=instance_count,
        volume_size=volume_size,
        max_run_time=max_run_time,
        model_artifact_path=model_artifact_path,
        output_encryption_key=output_encryption_key,
        network_isolation=network_isolation,
        traffic_encryption=traffic_encryption,
        spot_instance=spot_instance,
        max_wait_time=max_wait_time,
        checkpoint_config=checkpoint_config,
        debug_hook_config=debug_hook_config,
        debug_rule_config=debug_rule_config,
        tensorboard_output_config=tensorboard_output_config,
        role=role,
    )#.apply(use_aws_secret('aws-secret', 'AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'))


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(training, __file__ + '.zip')
