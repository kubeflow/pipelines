#!/usr/bin/env python3


import kfp
import json
import os
import copy
from kfp import components
from kfp import dsl
from kfp.aws import use_aws_secret


components_dir = os.path.join(os.path.dirname(__file__), '../../../../components/aws/sagemaker/')
sagemaker_hpo_op = components.load_component_from_file(os.path.join(components_dir, 'hyperparameter_tuning/component.yaml'))


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
    'CompressionType': 'None',
    'RecordWrapperType': 'None',
    'InputMode': 'File'
}

channelObj['ChannelName'] = 'train'
channelObj['DataSource']['S3DataSource']['S3Uri'] = 's3://kubeflow-pipeline-data/mnist_kmeans_example/train_data'
channelObjList.append(copy.deepcopy(channelObj))
channelObj['ChannelName'] = 'test'
channelObj['DataSource']['S3DataSource']['S3Uri'] = 's3://kubeflow-pipeline-data/mnist_kmeans_example/test_data'
channelObjList.append(copy.deepcopy(channelObj))


@dsl.pipeline(
    name='MNIST HPO test pipeline',
    description='SageMaker hyperparameter tuning job test'
)
def hpo_test(region='us-east-1',
    job_name='HPO-kmeans-sample',
    image='',
    algorithm_name='K-Means',
    training_input_mode='File',
    metric_definitions={},
    strategy='Bayesian',
    metric_name='test:msd',
    metric_type='Minimize',
    early_stopping_type='Off',
    static_parameters={"k": "10", "feature_dim": "784"},
    integer_parameters=[{"Name": "mini_batch_size", "MinValue": "450", "MaxValue": "550"}, \
                         {"Name": "extra_center_factor", "MinValue": "10", "MaxValue": "20"}],
    continuous_parameters=[],
    categorical_parameters=[{"Name": "init_method", "Values": ["random", "kmeans++"]}],
    channels=channelObjList,
    output_location='s3://kubeflow-pipeline-data/mnist_kmeans_example/output',
    output_encryption_key='',
    instance_type='ml.m5.2xlarge',
    instance_count=1,
    volume_size=50,
    max_num_jobs=1,
    max_parallel_jobs=1,
    resource_encryption_key='',
    max_run_time=3600,
    vpc_security_group_ids='',
    vpc_subnets='',
    endpoint_url='',
    network_isolation=True,
    traffic_encryption=False,
    warm_start_type='',
    parent_hpo_jobs='',
    spot_instance=False,
    max_wait_time=3600,
    checkpoint_config={},
    tags={},
    role_arn='',
    ):

    training = sagemaker_hpo_op(
        region=region,
        endpoint_url=endpoint_url,
        job_name=job_name,
        image=image,
        training_input_mode=training_input_mode,
        algorithm_name=algorithm_name,
        metric_definitions=metric_definitions,
        strategy=strategy,
        metric_name=metric_name,
        metric_type=metric_type,
        early_stopping_type=early_stopping_type,
        static_parameters=static_parameters,
        integer_parameters=integer_parameters,
        continuous_parameters=continuous_parameters,
        categorical_parameters=categorical_parameters,
        channels=channels,
        output_location=output_location,
        output_encryption_key=output_encryption_key,
        instance_type=instance_type,
        instance_count=instance_count,
        volume_size=volume_size,
        max_num_jobs=max_num_jobs,
        max_parallel_jobs=max_parallel_jobs,
        resource_encryption_key=resource_encryption_key,
        max_run_time=max_run_time,
        vpc_security_group_ids=vpc_security_group_ids,
        vpc_subnets=vpc_subnets,
        network_isolation=network_isolation,
        traffic_encryption=traffic_encryption,
        warm_start_type=warm_start_type,
        parent_hpo_jobs=parent_hpo_jobs,
        spot_instance=spot_instance,
        max_wait_time=max_wait_time,
        checkpoint_config=checkpoint_config,
        tags=tags,
        role=role_arn,
    )


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(hpo_test, __file__ + '.zip')
