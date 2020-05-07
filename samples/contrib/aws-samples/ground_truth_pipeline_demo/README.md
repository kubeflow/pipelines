The `mini-image-classification-pipeline.py` sample runs a pipeline to demonstrate usage for the create workteam, Ground Truth, and train components.

This sample is based on [this example](https://github.com/awslabs/amazon-sagemaker-examples/blob/master/ground_truth_labeling_jobs/from_unlabeled_data_to_deployed_machine_learning_model_ground_truth_demo_image_classification/from_unlabeled_data_to_deployed_machine_learning_model_ground_truth_demo_image_classification.ipynb).

The sample goes through the workflow of creating a private workteam, creating data labeling jobs for that team, and running a training job using the new labeled data.


## Prep the dataset, label categories, and UI template

For this demo, you will be using a very small subset of the [Google Open Images dataset](https://storage.googleapis.com/openimages/web/index.html).

Run the following to download `openimgs-annotations.csv`:
```bash
wget --no-verbose https://storage.googleapis.com/openimages/2018_04/test/test-annotations-human-imagelabels-boxable.csv -O openimgs-annotations.csv
```
Create a s3 bucket and run [this python script](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/ground_truth_pipeline_demo/prep_inputs.py) to get the images and generate `train.manifest`, `validation.manifest`, `class_labels.json`, and `instuctions.template`.


## Amazon Cognito user groups

From Cognito note down Pool ID, User Group Name and client ID  
You need this information to fill arguments user_pool, user_groups and client_ID

[Official doc for Amazon Cognito](https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-getting-started.html)

For this demo you can create a new user pool (if you don't have one already).   

AWS console -> Amazon SageMaker -> Ground Truth, Labeling workforces -> Private -> Create Private Team -> Give it "KFP-ground-truth-demo-pool" name and use your email address -> Create Private team -> Click on the radio button and from summary note down the "Amazon Cognito user pool", "App client" and "Labeling portal sign-in URL" -> click on the team name that you created and note down "Amazon Cognito user group"

Use the info that you noted down to fill arguments for the pipeline  
user_pool = Amazon Cognito user pool  
user_groups = Amazon Cognito user group  
client_ID = App client  

> Note : Once you start a run on the pipeline you will receive the ground_truth labeling jobs at "Labeling portal sign-in URL" link 

## IAM Roles

We need two IAM roles to run AWS KFP components. You only have to do this once. (Re-use the Role ARNs if you have done this before)

**Role 1]** For KFP pods to access AWS Sagemaker. Here are the steps to create it.
1. Enable OIDC support on the EKS cluster
   ```
   eksctl utils associate-iam-oidc-provider --cluster <cluster_name> \
    --region <cluster_region> --approve
   ```
2. Take note of the [OIDC](https://openid.net/connect/) issuer URL. This URL is in the form `oidc.eks.<region>.amazonaws.com/id/<OIDC_ID>` . Note down the URL.
   ```
   aws eks describe-cluster --name <cluster_name> --query "cluster.identity.oidc.issuer" --output text
   ```
3. Create a file named trust.json with the following content.   
   Replace `<OIDC_URL>` with your OIDC issuer URL **(Don’t include https://)** and `<AWS_account_number>` with your AWS account number. 
   ```
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Principal": {
           "Federated": "arn:aws:iam::<AWS_account_number>:oidc-provider/<OIDC_URL>"
         },
         "Action": "sts:AssumeRoleWithWebIdentity",
         "Condition": {
           "StringEquals": {
             "<OIDC_URL>:aud": "sts.amazonaws.com",
             "<OIDC_URL>:sub": "system:serviceaccount:kubeflow:pipeline-runner"
           }
         }
       }
     ]
   }
   ```
4. Create an IAM role using trust.json. Make a note of the ARN returned in the output.
   ```
   aws iam create-role --role-name kfp-example-pod-role --assume-role-policy-document file://trust.json
   aws iam attach-role-policy --role-name kfp-example-pod-role --policy-arn arn:aws:iam::aws:policy/AmazonSageMakerFullAccess
   aws iam get-role --role-name kfp-example-pod-role --output text --query 'Role.Arn'
   ```
5. Edit your pipeline-runner service account.
   ```
   kubectl edit -n kubeflow serviceaccount pipeline-runner
   ```
   Add `eks.amazonaws.com/role-arn: <role_arn>` to annotations, then save the file. Example:   
   ```
   apiVersion: v1
   kind: ServiceAccount
   metadata:
     annotations:
       eks.amazonaws.com/role-arn: <role_arn>
       kubectl.kubernetes.io/last-applied-configuration: |
         {"apiVersion":"v1","kind":"ServiceAccount","metadata":{"annotations":{},"labels":{"app":"pipeline-runner","app.kubernetes.io/component":"pipelines-runner","app.kubernetes.io/instance":"pipelines-runner-0.2.0","app.kubernetes.io/managed-by":"kfctl","app.kubernetes.io/name":"pipelines-runner","app.kubernetes.io/part-of":"kubeflow","app.kubernetes.io/version":"0.2.0"},"name":"pipeline-runner","namespace":"kubeflow"}}
     creationTimestamp: "2020-04-16T05:48:06Z"
     labels:
       app: pipeline-runner
       app.kubernetes.io/component: pipelines-runner
       app.kubernetes.io/instance: pipelines-runner-0.2.0
       app.kubernetes.io/managed-by: kfctl
       app.kubernetes.io/name: pipelines-runner
       app.kubernetes.io/part-of: kubeflow
       app.kubernetes.io/version: 0.2.0
     name: pipeline-runner
     namespace: kubeflow
     resourceVersion: "11787"
     selfLink: /api/v1/namespaces/kubeflow/serviceaccounts/pipeline-runner
     uid: d86234bd-7fa5-11ea-a8f2-02934be6dc88
   secrets:
   - name: pipeline-runner-token-dkjrk
   ```
**Role 2]** For sagemaker job to access S3 buckets and other Sagemaker services. This Role ARN is given as an input to the components.
   ```
   SAGEMAKER_EXECUTION_ROLE_NAME=kfp-example-sagemaker-execution-role

   TRUST="{ \"Version\": \"2012-10-17\", \"Statement\": [ { \"Effect\": \"Allow\", \"Principal\": { \"Service\": \"sagemaker.amazonaws.com\" }, \"Action\": \"sts:AssumeRole\" } ] }"
   aws iam create-role --role-name ${SAGEMAKER_EXECUTION_ROLE_NAME}--assume-role-policy-document "$TRUST"
   aws iam attach-role-policy --role-name ${SAGEMAKER_EXECUTION_ROLE_NAME} --policy-arn arn:aws:iam::aws:policy/AmazonSageMakerFullAccess
   aws iam attach-role-policy --role-name ${SAGEMAKER_EXECUTION_ROLE_NAME} --policy-arn arn:aws:iam::aws:policy/AmazonS3FullAccess

   aws iam get-role --role-name ${SAGEMAKER_EXECUTION_ROLE_NAME} --output text --query 'Role.Arn'

   # Note down the role arn which is of the form 
   arn:aws:iam::<AWS_acc_num>:role/<role-name>
   ```


## Compiling the pipeline template

Follow the guide to [building a pipeline](https://www.kubeflow.org/docs/guides/pipelines/build-pipeline/) to install the Kubeflow Pipelines SDK, then run the following command to compile the sample Python into a workflow specification. The specification takes the form of a YAML file compressed into a `.tar.gz` file.


```bash
dsl-compile --py mini-image-classification-pipeline.py --output mini-image-classification-pipeline.tar.gz
```

## Deploying the pipeline

Open the Kubeflow pipelines UI. Create a new pipeline, and then upload the compiled specification (`.tar.gz` file) as a new pipeline template.

The pipeline requires several arguments - replace `role_arn`, Amazon Cognito information, and the S3 input paths with your settings, and run the pipeline.

> Note : team_name, ground_truth_train_job_name and ground_truth_validation_job_name need to be unique or else pipeline will error out if the names already exist

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
