#!/usr/bin/env python3

import kfp
from kfp import components
from kfp import dsl
from kfp.aws import use_aws_secret

sagemaker_train_op = components.load_component_from_file('../../../components/aws/sagemaker/train/component.yaml')
sagemaker_model_op = components.load_component_from_file('../../../components/aws/sagemaker/model/component.yaml')
sagemaker_deploy_op = components.load_component_from_file('../../../components/aws/sagemaker/deploy/component.yaml')
sagemaker_batch_transform_op = components.load_component_from_file('../../../components/aws/sagemaker/batch_transform/component.yaml')

@dsl.pipeline(
    name='MNIST Classification pipeline',
    description='MNIST Classification using KMEANS in SageMaker'
)
def mnist_classification(region='us-west-2',
    image='174872318107.dkr.ecr.us-west-2.amazonaws.com/kmeans:1',
    dataset_path='s3://kubeflow-pipeline-data/mnist_kmeans_example/data',
    instance_type='ml.c4.8xlarge',
    instance_count='2',
    volume_size='50',
    model_output_path='s3://kubeflow-pipeline-data/mnist_kmeans_example/model',
    batch_transform_input='s3://kubeflow-pipeline-data/mnist_kmeans_example/input',
    batch_transform_ouput='s3://kubeflow-pipeline-data/mnist_kmeans_example/output',
    role_arn=''
    ):

    training = sagemaker_train_op(
        region=region,
        image=image,
        instance_type=instance_type,
        instance_count=instance_count,
        volume_size=volume_size,
        dataset_path=dataset_path,
        model_artifact_path=model_output_path,
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
        input_location=batch_transform_input,
        output_location=batch_transform_ouput
    ).apply(use_aws_secret('aws-secret', 'AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'))

if __name__ == '__main__':
    kfp.compiler.Compiler().compile(mnist_classification, __file__ + '.zip')