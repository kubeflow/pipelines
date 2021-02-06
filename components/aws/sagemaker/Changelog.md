# Change log for AWS SageMaker Components

The version of the AWS SageMaker Components is determined by the docker image tag used in YAML spec   
Repository:  https://hub.docker.com/repository/docker/amazon/aws-sagemaker-kfp-components

---------------------------------------------
**Change log for version 1.1.0**
- Add SageMaker RLEstimator component
- Add RoboMaker create/delete simulation application, cerate simulation job components

> Pull requests : [#4813](https://github.com/kubeflow/pipelines/pull/4813/)

**Change log for version 1.0.0**
- First release to guarantee backward compatibility within major version
- Internally refactored components

> Pull requests : [#4336](https://github.com/kubeflow/pipelines/pull/4336/)

**Change log for version 0.9.0**
- Add functionality to Update SageMaker Endpoint for Deploy component

> Pull requests : [#4424](https://github.com/kubeflow/pipelines/pull/4424/)

**Change log for version 0.8.0**
- Add functionality to configure SageMaker Debugger for Training component

> Pull requests : [#4283](https://github.com/kubeflow/pipelines/pull/4283/)


**Change log for version 0.7.0**
- Add functionality to assume role when sending SageMaker requests

>  Pull requests : [#4212](https://github.com/kubeflow/pipelines/pull/4212)


**Change log for version 0.6.0**
- Add functionality to stop SageMaker jobs on run termination

>  Pull requests : [#4167](https://github.com/kubeflow/pipelines/pull/4167)


**Change log for version 0.5.3**
- Add static error string in case of error fetching logs

>  Pull requests : [#4056](https://github.com/kubeflow/pipelines/pull/4056)


**Change log for version 0.5.2**
- Modified outputs to use newer `outputPath` syntax

> Pull requests : [#4119](https://github.com/kubeflow/pipelines/pull/4119)


**Change log for version 0.5.1**
- Update region support for GroundTruth component
- Make `label_category_config` an optional parameter in Ground Truth component

> Pull requests : [#3932](https://github.com/kubeflow/pipelines/pull/3932)


**Change log for version 0.5.0**
- Print SageMaker logs in KFP UI for Train, Transform and Process component

> Pull requests : [#3954](https://github.com/kubeflow/pipelines/pull/3954)


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


**Change log for version 0.2.0 (Apr 14, 2020)**
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


