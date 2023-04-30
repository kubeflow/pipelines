# Description: This script deploys a SageMaker endpoint with data capture enabled.
# Example: python set_up_endpoint.py --s3_data_bucket <bucket_name> --sagemaker_role_arn <role_arn> --region <region>

from datetime import datetime
import os
import boto3

from sagemaker import image_uris


def deploy_endpoint(sagemaker_client, s3_data_bucket, sagemaker_role_arn, region):
    model_name = datetime.now().strftime("%Y-%m-%d-%H-%M-%S") + "-model"
    endpoint_name = datetime.now().strftime("%Y-%m-%d-%H-%M-%S") + "-endpoint-v2"
    endpoint_config_name = (
        datetime.now().strftime("%Y-%m-%d-%H-%M-%S") + "-endpoint-config"
    )

    # create sagemaker model
    image_uri = image_uris.retrieve("xgboost", region, "0.90-1")
    print("Creating model with image uri: ", image_uri)
    create_model_api_response = sagemaker_client.create_model(
        ModelName=model_name,
        PrimaryContainer={
            "Image": image_uri,
            "ModelDataUrl": f"s3://{s3_data_bucket}/model-monitor/xgb-churn-prediction-model.tar.gz",
            "Environment": {},
        },
        ExecutionRoleArn=sagemaker_role_arn,
    )

    # create sagemaker endpoint config
    print("Creating endpoint config with name: ", endpoint_config_name)
    create_endpoint_config_api_response = sagemaker_client.create_endpoint_config(
        EndpointConfigName=endpoint_config_name,
        ProductionVariants=[
            {
                "VariantName": "variant-1",
                "ModelName": model_name,
                "InitialInstanceCount": 1,
                "InstanceType": "ml.m5.large",
            },
        ],
        DataCaptureConfig={
            "EnableCapture": True,
            "CaptureOptions": [{"CaptureMode": "Input"}, {"CaptureMode": "Output"}],
            "InitialSamplingPercentage": 100,
            "DestinationS3Uri": f"s3://{s3_data_bucket}/model-monitor/datacapture",
        },
    )

    # create sagemaker endpoint
    create_endpoint_api_response = sagemaker_client.create_endpoint(
        EndpointName=endpoint_name,
        EndpointConfigName=endpoint_config_name,
    )
    print("Creating endpoint with name: ", endpoint_name, "Please wait for 3 mins...")

    try:
        sagemaker_client.get_waiter("endpoint_in_service").wait(
            EndpointName=endpoint_name
        )
    finally:
        resp = sagemaker_client.describe_endpoint(EndpointName=endpoint_name)
        endpoint_status = resp["EndpointStatus"]
        endpoint_arn = resp["EndpointArn"]
        print(f"Deployed endpoint {endpoint_arn}, ended with status {endpoint_status}")

        if endpoint_status != "InService":
            message = sagemaker_client.describe_endpoint(EndpointName=endpoint_name)[
                "FailureReason"
            ]
            print(
                "Endpoint deployment failed with the following error: {}".format(
                    message
                )
            )
            raise Exception("Endpoint deployment failed")

    return endpoint_name


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--s3_data_bucket",
        type=str,
        help="S3 bucket name",
        required=True,
    )
    parser.add_argument(
        "--sagemaker_role_arn",
        type=str,
        help="SageMaker role ARN",
        required=True,
    )
    parser.add_argument(
        "--region",
        type=str,
        help="AWS region",
        required=True,
    )
    args = parser.parse_args()

    sagemaker_client = boto3.client("sagemaker", region_name=args.region)

    endpoint_name = deploy_endpoint(
        sagemaker_client, args.s3_data_bucket, args.sagemaker_role_arn, args.region
    )
    os.environ.setdefault("ENDPOINT_NAME", endpoint_name)
    print("Endpoint deployed: ", endpoint_name)
