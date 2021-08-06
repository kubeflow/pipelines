# Sample installation

1. Create a kubernetes cluster

Create an AWS EKS cluster:
```
eksctl create cluster \
--name AWS-KFP \
--version 1.17 \
--region us-west-2 \
--nodegroup-name linux-nodes \
--node-type m5.xlarge \
--nodes 2 \
--nodes-min 1 \
--nodes-max 4 \
--managed
```

2. Create a storage bucket

To create an S3 bucket, pick a name and region and replace `<YOUR_S3_BUCKET_NAME>`:
```
export S3_BUCKET=<YOUR_S3_BUCKET_NAME>
export AWS_REGION=us-west-2
aws s3 mb s3://$S3_BUCKET --region $AWS_REGION
```

3. Prepare a database

Follow this [doc](https://www.kubeflow.org/docs/aws/rds/#deploy-amazon-rds-mysql-in-your-environment)
to set up AWS RDS instance. The RDS instance is created as part of an AWS CloudFormation Stack.

To get the database instance identifier, replace `STACK` with the name of the stack you just created:
```
$ aws cloudformation describe-stack-resources --stack-name STACK --query "StackResources[?LogicalResourceId=='MyDB'].PhysicalResourceId" --output text
abcdefghijklmno
```

To get the database host address, specify the DBInstanceIdentifier you just found:
```
$ aws rds describe-db-instances --query "DBInstances[?DBInstanceIdentifier=='abcdefghijklmno'].Endpoint.Address" --output text
abcdefghijklmno.pqrstuvwxyzz.us-west-2.rds.amazonaws.com
```

4. Customize your values

Edit the following files:
  - [params.env](params.env)
  - [secret.env](secret.env)
  - [minio-artifact-secret-patch.env](minio-artifact-secret-patch.env)

5. Deploy

```
kubectl apply -k ../../cluster-scoped-resources
kubectl wait crd/applications.app.k8s.io --for condition=established --timeout=60s

kubectl apply -k ./
kubectl wait applications/pipeline -n kubeflow --for condition=Ready --timeout=1800s
```

Replace `apply` with `delete` in the commands above to delete a
partial failed deployment so you can re-apply.

Forward a local port so you can access the web UI via `localhost:8080`:
```
kubectl port-forward -n kubeflow svc/ml-pipeline-ui 8080:80
```
