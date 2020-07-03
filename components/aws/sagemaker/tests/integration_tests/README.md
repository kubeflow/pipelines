## Requirements
1. [Docker](https://www.docker.com/)
1. [IAM Role](https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-roles.html) with a SageMakerFullAccess and AmazonS3FullAccess
1. IAM User credentials with SageMakerFullAccess, AWSCloudFormationFullAccess, IAMFullAccess, AmazonEC2FullAccess, AmazonS3FullAccess permissions
2. The SageMaker WorkTeam and GroundTruth Component tests expect that at least one private workteam already exists in the region where you are running these tests. 


## Creating S3 buckets with datasets

1. In the following Python script, change the bucket name and run the [`s3_sample_data_creator.py`](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/mnist-kmeans-sagemaker#the-sample-dataset) to create an S3 bucket with the sample mnist dataset in the region where you want to run the tests.
2. To prepare the dataset for the SageMaker GroundTruth Component test, follow the steps in the `[GroundTruth Sample README](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/ground_truth_pipeline_demo#prep-the-dataset-label-categories-and-ui-template)`.
3. To prepare the processing script for the SageMaker Processing Component tests, upload the `scripts/kmeans_preprocessing.py` script to your bucket. This can be done by replacing `<my-bucket> with your bucket name and running `aws s3 cp scripts/kmeans_preprocessing.py s3://<my-bucket>/mnist_kmeans_example/processing_code/kmeans_preprocessing.py`


## Step to run integration tests
1. Copy the `.env.example` file to `.env` and in the following steps modify the fields of this new file:
    1. Configure the AWS credentials fields with those of your IAM User.
    1. Update the `SAGEMAKER_EXECUTION_ROLE_ARN` with that of your role created earlier.
    1. Update the `S3_DATA_BUCKET` parameter with the name of the bucket created earlier.
    1. (Optional) If you have already created an EKS cluster for testing, replace the `EKS_EXISTING_CLUSTER` field with it's name.
1. Build the image by doing the following:
    1. Navigate to the root of this github directory.
    1. Run `docker build . -f components/aws/sagemaker/tests/integration_tests/Dockerfile -t amazon/integration_test`
1. Run the image, injecting your environment variable files:
    1. Navigate to the `components/aws` directory.
    1. Run `docker run --env-file components/aws/sagemaker/tests/integration_tests/.env amazon/integration_test`