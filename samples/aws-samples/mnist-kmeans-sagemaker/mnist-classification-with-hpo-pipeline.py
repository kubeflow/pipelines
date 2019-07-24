#!/usr/bin/env python3

import kfp
from kfp import components
from kfp import dsl
from kfp.aws import use_aws_secret

sagemaker_hpo_op = components.load_component_from_file('../../../components/aws/sagemaker/hyperparameter_tuning/component.yaml')
sagemaker_train_op = components.load_component_from_file('../../../components/aws/sagemaker/train/component.yaml')
sagemaker_model_op = components.load_component_from_file('../../../components/aws/sagemaker/model/component.yaml')
sagemaker_deploy_op = components.load_component_from_file('../../../components/aws/sagemaker/deploy/component.yaml')
sagemaker_batch_transform_op = components.load_component_from_file('../../../components/aws/sagemaker/batch_transform/component.yaml')

@dsl.pipeline(
    name='MNIST Classification with HPO pipeline',
    description='MNIST Classification using KMEANS in SageMaker'
)
def mnist_classification(region='us-west-2',
    image='',
    algorithm_name='K-Means',
    training_input_mode='File',
    metric_definitions='{}',
    strategy='Bayesian',
    metric_name='test:msd',
    metric_type='Minimize',
    early_stopping_type='Off',
    static_parameters='{"k": "10", "feature_dim": "784"}',
    integer_parameters='[{"Name": "mini_batch_size", "MinValue": "450", "MaxValue": "550"}, {"Name": "extra_center_factor", "MinValue": "10", "MaxValue": "20"}]',
    continuous_parameters='[]',
    categorical_parameters='[{"Name": "init_method", "Values": ["random", "kmeans++"]}]',
    hpo_channels='[{"ChannelName": "train", "DataSource": {"S3DataSource": {"S3Uri": "s3://carowang-kfp-mnist/mnist_kmeans_example/data", "S3DataType": "S3Prefix", "S3DataDistributionType": "FullyReplicated"}}, "ContentType": "", "CompressionType": "None", "RecordWrapperType": "None", "InputMode": "File"}, {"ChannelName": "test", "DataSource": {"S3DataSource": {"S3Uri": "s3://carowang-kfp-mnist/mnist_kmeans_example/data", "S3DataType": "S3Prefix", "S3DataDistributionType": "FullyReplicated"}}, "ContentType": "", "CompressionType": "None", "RecordWrapperType": "None", "InputMode": "File"}]',
    output_location='s3://carowang-kfp-mnist/mnist_kmeans_example/output',
    output_encryption_key='',
    instance_type='ml.p2.16xlarge',
    instance_count='1',
    volume_size='50',
    max_num_jobs='1',
    max_parallel_jobs='1',
    max_run_time='3600',
    vpc_security_group_ids='',
    vpc_subnets='',
    network_isolation='True',
    traffic_encryption='False',
    warm_start_type='',
    parent_hpo_jobs='',
    hpo_tags='{}',
    training_channels='[{"ChannelName": "train", "DataSource": {"S3DataSource": {"S3Uri": "s3://carowang-kfp-mnist/mnist_kmeans_example/data", "S3DataType": "S3Prefix", "S3DataDistributionType": "FullyReplicated"}}, "ContentType": "", "CompressionType": "None", "RecordWrapperType": "None", "InputMode": "File"}]',
    batch_transform_instance_type='ml.m4.xlarge',
    batch_transform_input='s3://carowang-kfp-mnist/mnist_kmeans_example/input',
    batch_transform_data_type='S3Prefix',
    batch_transform_content_type='text/csv',
    batch_transform_compression_type='None',
    batch_transform_ouput='s3://carowang-kfp-mnist/mnist_kmeans_example/output',
    max_concurrent='4',
    max_payload='6',
    batch_strategy='MultiRecord',
    batch_transform_split_type='Line',
    role_arn='arn:aws:iam::841569659894:role/service-role/AmazonSageMaker-ExecutionRole-20190502T144803'
    ):

    hpo = sagemaker_hpo_op(
        region=region,
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
        channels=hpo_channels,
        output_location=output_location,
        output_encryption_key=output_encryption_key,
        instance_type=instance_type,
        instance_count=instance_count,
        volume_size=volume_size,
        max_num_jobs=max_num_jobs,
        max_parallel_jobs=max_parallel_jobs,
        max_run_time=max_run_time,
        vpc_security_group_ids=vpc_security_group_ids,
        vpc_subnets=vpc_subnets,
        network_isolation=network_isolation,
        traffic_encryption=traffic_encryption,
        warm_start_type=warm_start_type,
        parent_hpo_jobs=parent_hpo_jobs,
        tags=hpo_tags,
        role=role_arn,
    ).apply(use_aws_secret('aws-secret', 'AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'))

    training = sagemaker_train_op(
        region=region,
        image=image,
        training_input_mode=training_input_mode,
        hyperparameters=hpo.outputs['best_hyperparameters'],
        channels=training_channels,
        instance_type=instance_type,
        instance_count=instance_count,
        volume_size=volume_size,
        resource_encryption_key=resource_encryption_key,
        max_run_time=max_run_time,
        model_artifact_path=output_location,
        role=role_arn,
    ).apply(use_aws_secret('aws-secret', 'AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'))

    create_model = sagemaker_model_op(
        region=region,
        image=image,
        model_artifact_url=training.outputs['model_artifact_url'],
        model_name=training.outputs['job_name'],
        role=role_arn
    ).apply(use_aws_secret('aws-secret', 'AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'))

    prediction = sagemaker_deploy_op(
        region=region,
        model_name=create_model.output
    ).apply(use_aws_secret('aws-secret', 'AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'))

    batch_transform = sagemaker_batch_transform_op(
        region=region,
        model_name=create_model.output,
        max_concurrent=max_concurrent,
        max_payload=max_payload,
        batch_strategy=batch_strategy,
        input_location=batch_transform_input,
        data_type=batch_transform_data_type,
        content_type=batch_transform_content_type,
        split_type=batch_transform_split_type,
        compression_type=batch_transform_compression_type,
        output_location=batch_transform_ouput,
        instance_type=batch_transform_instance_type,
        instance_count=instance_count
    ).apply(use_aws_secret('aws-secret', 'AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'))

if __name__ == '__main__':
    kfp.compiler.Compiler().compile(mnist_classification, __file__ + '.zip')
