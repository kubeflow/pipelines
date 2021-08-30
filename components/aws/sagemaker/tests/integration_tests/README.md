## Requirements
1. [Docker](https://www.docker.com/)
1. [IAM Role](https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-roles.html) with a SageMakerFullAccess, RoboMakerFullAccess and AmazonS3FullAccess
1. IAM User credentials with SageMakerFullAccess, RoboMakerFullAccess, AWSCloudFormationFullAccess, IAMFullAccess, AmazonEC2FullAccess, AmazonS3FullAccess permissions
2. The SageMaker WorkTeam and GroundTruth Component tests expect that at least one private workteam already exists in the region where you are running these tests. 


## Creating S3 buckets with datasets

1. In the following Python script, change the bucket name and run the [`s3_sample_data_creator.py`](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/mnist-kmeans-sagemaker#the-sample-dataset) to create an S3 bucket with the sample mnist dataset in the region where you want to run the tests.
2. To prepare the dataset for the SageMaker GroundTruth Component test, follow the steps in the [GroundTruth Sample README](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/ground_truth_pipeline_demo#prep-the-dataset-label-categories-and-ui-template).
3. To prepare the processing script for the SageMaker Processing Component tests, upload the `scripts/kmeans_preprocessing.py` script to your bucket. This can be done by replacing `<my-bucket>` with your bucket name and running `aws s3 cp scripts/kmeans_preprocessing.py s3://<my-bucket>/mnist_kmeans_example/processing_code/kmeans_preprocessing.py`
4. Prepare RoboMaker Simulation App sources and Robot App sources and place them in the data bucket under the `/robomaker` key. The easiest way to create the files you need is to copy them from the public buckets that are used to store the [RoboMaker Hello World](https://console.aws.amazon.com/robomaker/home?region=us-east-1#sampleSimulationJobs) demos:
    ```bash
   aws s3 cp s3://aws-robomaker-samples-us-east-1-1fd12c306611/hello-world/melodic/gazebo9/1.4.0.62/1.2.0/simulation_ws.tar .
   aws s3 cp ./simulation_ws.tar s3://<your_bucket_name>/robomaker/simulation_ws.tar
   aws s3 cp s3://aws-robomaker-samples-us-east-1-1fd12c306611/hello-world/melodic/gazebo9/1.4.0.62/1.2.0/robot_ws.tar .
   aws s3 cp ./robot_ws.tar s3://<your_bucket_name>/robomaker/robot_ws.tar
   ```
    The files in the `/robomaker` directory on S3 should follow this pattern:
    ```
    /robomaker/simulation_ws.tar
    /robomaker/robot_ws.tar
    ```
5. Prepare RLEstimator sources and place them in the data bucket under the `/rlestimator` key. The easiest way to create the files you need is to follow the notebooks outlined in the [RLEstimator Samples README](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/rlestimator_pipeline/README.md).
    The files in the `/rlestimator` directory on S3 should follow this pattern:
    ```
    /rlestimator/sourcedir.tar.gz
    ```

## Step to run integration tests
1. Copy the `.env.example` file to `.env` and in the following steps modify the fields of this new file:
    1. Configure the AWS credentials fields with those of your IAM User.
    1. Update the `SAGEMAKER_EXECUTION_ROLE_ARN` with that of your role created earlier.
    1. Update the `S3_DATA_BUCKET` parameter with the name of the bucket created earlier.
    1. (Optional) If you have already created an EKS cluster for testing, replace the `EKS_EXISTING_CLUSTER` field with it's name.
1. Build the image by doing the following:
    1. Navigate to the root of this github directory.
    1. Run `docker build . -f components/aws/sagemaker/tests/integration_tests/Dockerfile -t amazon/integration_test`
1. Run the image, injecting your environment variable files and mounting the repo files into the container:
    1. Run `docker run -v <path_to_this_repo_on_your_machine>:/pipelines --env-file components/aws/sagemaker/tests/integration_tests/.env amazon/integration_test`