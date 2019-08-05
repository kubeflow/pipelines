The `mini-image-classification-pipeline.py` sample runs a pipeline to demonstrate usage for the create workteam, Ground Truth, and train components.

This sample is based on [this example](https://github.com/awslabs/amazon-sagemaker-examples/blob/master/ground_truth_labeling_jobs/from_unlabeled_data_to_deployed_machine_learning_model_ground_truth_demo_image_classification/from_unlabeled_data_to_deployed_machine_learning_model_ground_truth_demo_image_classification.ipynb).

The sample goes through the workflow of creating a private workteam, creating data labeling jobs for that team, and running a training job using the new labeled data.


## Prep the dataset, label categories, and UI template

For this demo, you will be using a very small subset of the [Google Open Images dataset](https://storage.googleapis.com/openimages/web/index.html).

Run the following to download `openimgs-annotations.csv`:
```bash
wget https://storage.googleapis.com/openimages/2018_04/test/test-annotations-human-imagelabels-boxable.csv -O openimgs-annotations.csv
```
Create a s3 bucket and run [this python script](https://github.com/kubeflow/pipelines/tree/master/samples/aws-samples/ground_truth_pipeline_demo/prep_inputs.py) to get the images and generate `train.manifest`, `validation.manifest`, `class_labels.json`, and `instuctions.template`.


## Amazon Cognito user groups

To use the workteam component, you must have an Amazon Cognito user group(s) set up with the workers you want for the workteam.

For this demo, please create a user group with only yourself in it.

If this is your first time creating a private workteam in the region you are using, follow the steps below to set up a new user pool, user group, and app client ID on Amazon Cognito.
If you already have private workteams in the region you are using, skip step 1 and do steps 2 and 3 in the user pool that is already being used.

Steps:
1. Create a user pool in Amazon Cognito. Configure the user pool as needed, and make sure to create an app client. The Pool ID will be found under General settings.
2. After creating the user pool, go to the Users and Groups section and create a group. Create users for the team, and add those users to the group.
3. Under App integration > Domain name, create an Amazon Cognito domain for the user pool.


## SageMaker permission

In order to run this pipeline, we need to prepare an IAM Role to run Sagemaker jobs. You need this `role_arn` to run a pipeline. Check [here](https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-roles.html) for details.

This pipeline also use aws-secret to get access to Sagemaker services, please also make sure you have a `aws-secret` in the kubeflow namespace.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws-secret
  namespace: kubeflow
type: Opaque
data:
  AWS_ACCESS_KEY_ID: YOUR_BASE64_ACCESS_KEY
  AWS_SECRET_ACCESS_KEY: YOUR_BASE64_SECRET_ACCESS
```

> Note: To get base64 string, try `echo -n $AWS_ACCESS_KEY_ID | base64`


## Compiling the pipeline template

Follow the guide to [building a pipeline](https://www.kubeflow.org/docs/guides/pipelines/build-pipeline/) to install the Kubeflow Pipelines SDK, then run the following command to compile the sample Python into a workflow specification. The specification takes the form of a YAML file compressed into a `.tar.gz` file.

```bash
dsl-compile --py mini-image-classification-pipeline.py --output mini-image-classification-pipeline.tar.gz
```

## Deploying the pipeline

Open the Kubeflow pipelines UI. Create a new pipeline, and then upload the compiled specification (`.tar.gz` file) as a new pipeline template.

The pipeline requires several arguments - replace `role_arn`, Amazon Cognito information, and the S3 input paths with your settings, and run the pipeline.

If you are a new worker, you will receive an email with a link to the labeling portal and login information after the create workteam component completes.
During the execution of the two Ground Truth components (one for training data, one for validation data), the labeling jobs will appear in the portal and you will need to complete these jobs.  

After the pipeline finished, you may delete the user pool/ user group and the S3 bucket.


## Components source

Create Workteam:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/workteam/src)

Ground Truth Labeling:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/ground_truth/src)

Training:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/train/src)
