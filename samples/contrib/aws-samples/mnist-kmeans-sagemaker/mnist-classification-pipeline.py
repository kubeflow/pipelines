#!/usr/bin/env python3

import kfp
from kfp import components
from kfp import dsl
from kfp.aws import use_aws_secret

sagemaker_hpo_op = components.load_component_from_file('../../../../components/aws/sagemaker/hyperparameter_tuning/component.yaml')
sagemaker_train_op = components.load_component_from_file('../../../../components/aws/sagemaker/train/component.yaml')
sagemaker_model_op = components.load_component_from_file('../../../../components/aws/sagemaker/model/component.yaml')
sagemaker_deploy_op = components.load_component_from_file('../../../../components/aws/sagemaker/deploy/component.yaml')
sagemaker_batch_transform_op = components.load_component_from_file('../../../../components/aws/sagemaker/batch_transform/component.yaml')

@dsl.pipeline(
    name='MNIST Classification pipeline',
    description='MNIST Classification using KMEANS in SageMaker'
)
def mnist_classification(region='us-west-2',
    image='174872318107.dkr.ecr.us-west-2.amazonaws.com/kmeans:1',
    training_input_mode='File',
    hpo_strategy='Bayesian',
    hpo_metric_name='test:msd',
    hpo_metric_type='Minimize',
    hpo_early_stopping_type='Off',
    hpo_static_parameters='{"k": "10", "feature_dim": "784"}',
    hpo_integer_parameters='[{"Name": "mini_batch_size", "MinValue": "500", "MaxValue": "600"}, {"Name": "extra_center_factor", "MinValue": "10", "MaxValue": "20"}]',
    hpo_continuous_parameters='[]',
    hpo_categorical_parameters='[{"Name": "init_method", "Values": ["random", "kmeans++"]}]',
    hpo_channels='[{"ChannelName": "train", \
                "DataSource": { \
                    "S3DataSource": { \
                        "S3Uri": "s3://kubeflow-pipeline-data/mnist_kmeans_example/train_data",  \
                        "S3DataType": "S3Prefix", \
                        "S3DataDistributionType": "FullyReplicated" \
                        } \
                    }, \
                "ContentType": "", \
                "CompressionType": "None", \
                "RecordWrapperType": "None", \
                "InputMode": "File"}, \
               {"ChannelName": "test", \
                "DataSource": { \
                    "S3DataSource": { \
                        "S3Uri": "s3://kubeflow-pipeline-data/mnist_kmeans_example/test_data", \
                        "S3DataType": "S3Prefix", \
                        "S3DataDistributionType": "FullyReplicated" \
                        } \
                    }, \
                "ContentType": "", \
                "CompressionType": "None", \
                "RecordWrapperType": "None", \
                "InputMode": "File"}]',
    hpo_spot_instance='False',
    hpo_max_wait_time='3600',
    hpo_checkpoint_config='{}',
    output_location='s3://kubeflow-pipeline-data/mnist_kmeans_example/output',
    output_encryption_key='',
    instance_type='ml.p2.16xlarge',
    instance_count='1',
    volume_size='50',
    hpo_max_num_jobs='9',
    hpo_max_parallel_jobs='3',
    max_run_time='3600',
    endpoint_url='',
    network_isolation='True',
    traffic_encryption='False',
    train_channels='[{"ChannelName": "train", \
                "DataSource": { \
                    "S3DataSource": { \
                        "S3Uri": "s3://kubeflow-pipeline-data/mnist_kmeans_example/train_data",  \
                        "S3DataType": "S3Prefix", \
                        "S3DataDistributionType": "FullyReplicated" \
                        } \
                    }, \
                "ContentType": "", \
                "CompressionType": "None", \
                "RecordWrapperType": "None", \
                "InputMode": "File"}]',
    train_spot_instance='False',
    train_max_wait_time='3600',
    train_checkpoint_config='{}',
    batch_transform_instance_type='ml.m4.xlarge',
    batch_transform_input='s3://kubeflow-pipeline-data/mnist_kmeans_example/input',
    batch_transform_data_type='S3Prefix',
    batch_transform_content_type='text/csv',
    batch_transform_compression_type='None',
    batch_transform_ouput='s3://kubeflow-pipeline-data/mnist_kmeans_example/output',
    batch_transform_max_concurrent='4',
    batch_transform_max_payload='6',
    batch_strategy='MultiRecord',
    batch_transform_split_type='Line',
    role_arn=''
    ):

    hpo = sagemaker_hpo_op(
        region=region,
        endpoint_url=endpoint_url,
        image=image,
        training_input_mode=training_input_mode,
        strategy=hpo_strategy,
        metric_name=hpo_metric_name,
        metric_type=hpo_metric_type,
        early_stopping_type=hpo_early_stopping_type,
        static_parameters=hpo_static_parameters,
        integer_parameters=hpo_integer_parameters,
        continuous_parameters=hpo_continuous_parameters,
        categorical_parameters=hpo_categorical_parameters,
        channels=hpo_channels,
        output_location=output_location,
        output_encryption_key=output_encryption_key,
        instance_type=instance_type,
        instance_count=instance_count,
        volume_size=volume_size,
        max_num_jobs=hpo_max_num_jobs,
        max_parallel_jobs=hpo_max_parallel_jobs,
        max_run_time=max_run_time,
        network_isolation=network_isolation,
        traffic_encryption=traffic_encryption,
        spot_instance=hpo_spot_instance,
        max_wait_time=hpo_max_wait_time,
        checkpoint_config=hpo_checkpoint_config,
        role=role_arn,
    ).apply(use_aws_secret('aws-secret', 'AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'))

    training = sagemaker_train_op(
        region=region,
        endpoint_url=endpoint_url,
        image=image,
        training_input_mode=training_input_mode,
        hyperparameters=hpo.outputs['best_hyperparameters'],
        channels=train_channels,
        instance_type=instance_type,
        instance_count=instance_count,
        volume_size=volume_size,
        max_run_time=max_run_time,
        model_artifact_path=output_location,
        output_encryption_key=output_encryption_key,
        network_isolation=network_isolation,
        traffic_encryption=traffic_encryption,
        spot_instance=train_spot_instance,
        max_wait_time=train_max_wait_time,
        checkpoint_config=train_checkpoint_config,
        role=role_arn,
    ).apply(use_aws_secret('aws-secret', 'AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'))

    create_model = sagemaker_model_op(
        region=region,
        endpoint_url=endpoint_url,
        model_name=training.outputs['job_name'],
        image=training.outputs['training_image'],
        model_artifact_url=training.outputs['model_artifact_url'],
        network_isolation=network_isolation,
        role=role_arn
    ).apply(use_aws_secret('aws-secret', 'AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'))

    prediction = sagemaker_deploy_op(
        region=region,
        endpoint_url=endpoint_url,
        model_name_1=create_model.output,
    ).apply(use_aws_secret('aws-secret', 'AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'))

    batch_transform = sagemaker_batch_transform_op(
        region=region,
        endpoint_url=endpoint_url,
        model_name=create_model.output,
        instance_type=batch_transform_instance_type,
        instance_count=instance_count,
        max_concurrent=batch_transform_max_concurrent,
        max_payload=batch_transform_max_payload,
        batch_strategy=batch_strategy,
        input_location=batch_transform_input,
        data_type=batch_transform_data_type,
        content_type=batch_transform_content_type,
        split_type=batch_transform_split_type,
        compression_type=batch_transform_compression_type,
        output_location=batch_transform_ouput
    ).apply(use_aws_secret('aws-secret', 'AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'))

if __name__ == '__main__':
    kfp.compiler.Compiler().compile(mnist_classification, __file__ + '.zip')
