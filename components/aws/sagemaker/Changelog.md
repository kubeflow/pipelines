# Change log for AWS SageMaker Components

The version of the AWS SageMaker Components is determined by the docker image tag used in YAML spec   
Repository:  https://hub.docker.com/repository/docker/amazon/aws-sagemaker-kfp-components

---------------------------------------------
**Change log for version 0.4.1**
- Fix breaking bug in HPO component

> Pull requests : [#4010](https://github.com/kubeflow/pipelines/pull/4010)


**Change log for version 0.4.0**
- Add new component for SageMaker Processing Jobs

> Pull requests : [#3944](https://github.com/kubeflow/pipelines/pull/3944)


**Change log for version 0.3.1**
- Explicitly specify component field types

> Pull requests : [#3683](https://github.com/kubeflow/pipelines/pull/3683)


**Change log for version 0.3.0**
- Remove data_location parameters from all components
	  (Use "channels" parameter instead)

> Pull requests : [#3518](https://github.com/kubeflow/pipelines/pull/3518)


**Change log for version 2.0 (Apr 14, 2020)**
- Fix bug in Ground Truth component
- Add user agent header to boto3 client
  
> Pull requests: [#3474](https://github.com/kubeflow/pipelines/pull/3474), [#3487](https://github.com/kubeflow/pipelines/pull/3487)


---------------------------------------------

## Old

These are the old images which were in https://hub.docker.com/r/redbackthomson/aws-kubeflow-sagemaker/tags

**Change log 20200402**
- Fix for vpc issue 
- Add license files 
- Use AmazonLinux instead of Ubuntu 
- Pin the pip packages 

	
> Pull requests: [#3374](https://github.com/kubeflow/pipelines/pull/3374), [#3397](https://github.com/kubeflow/pipelines/pull/3397)

No change log available for older images 
Please check git log 


