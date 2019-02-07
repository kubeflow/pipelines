The `Watson Train and Serve` sample pipeline runs training, storing and deploying a Tensorflow model with MNIST handwriting recognition using [IBM Watson Studio](https://www.ibm.com/cloud/watson-studio) and [IBM Watson Machine Learning](https://www.ibm.com/cloud/machine-learning) service.

# Requirements

This sample requires the user to have provisioned a machine learning service on Watson, a S3 cloud storage set up and the service credentials configured in the [`creds.ini` file](https://github.ibm.com/AIOpsPipeline/kfp-samples/blob/master/wml-containers/creds.ini)

However, if users want to use their own machine learning services and S3 cloud storage, following are the required steps.

* IBM Watson Machine Learning service instance

To create a machine learning service, go to [IBM Cloud](https://console.bluemix.net), login with IBM account id first. From the `Catalog` page, click on `AI` tab on the left side to go to this [page](https://console.bluemix.net/catalog/?category=ai). Then click on the [`Machine Learning`](https://console.bluemix.net/catalog/services/machine-learning) link and follow the instructions to create the service.

Once the service is created, from service's `Dashboard`, follow the instruction to generate `service credentials`. Refer to IBM Cloud [documents](https://console.bluemix.net/docs/) for help if needed. Collect the `url`, `username`, `password` and `instance_id` info from the service credentials as these will be required to access the service.

If the user already has a IBM Watson Studio service with IBM Cloud, go to that service and from there follow the instruction to create a `Project` to hook up the machine learning service created above. Watson Studio allows the user to see all the models and deployments trained and deployed through Watson Machine Learning serice.

* An S3 cloud storage

Watson Machine Learning service loads datasets from S3 cloud storage and stores models and other artifacts to S3 cloud storage. Users can use any S3 cloud storage they already preserve. Users can also create a S3 cloud storage with `IBM Cloud Object Storage` service by following this [link](https://console.bluemix.net/catalog/services/cloud-object-storage).

Collect the `endpoint`, `access_key_id` and `secret_access_key` fields from the service credentials for the S3 cloud storage. Create the service credentials first if not existed.

Create two buckets, one for storing the train datasets and one for storing the model outputs.

* Set up access credentials

This pipeline sample reads the credentials from a file hosted in a github repo. Refer to [`creds.ini`]() file and input user's specific credentials. Then upload the file to a github repo the user has access.

To access the credentials file, the user should provide a github access token and the link to the raw content of the file. Modify the `GITHUB_TOKEN` and `CONFIG_FILE_URL` variables in the [`wml-pipeline.py`]() file with the user's access token and link.

# The datasets

This pipeline sample uses the [MNIST](http://yann.lecun.com/exdb/mnist) datasets.

# Compling the pipeline template

Follow the guide to [building a pipeline](https://www.kubeflow.org/docs/pipelines/build-pipeline/) to install the Kubeflow Pipelines SDK, then run the following command to compile the sample [`wml-pipeline.py`](https://github.ibm.com/AIOpsPipeline/kfp-samples/blob/master/wml-containers/wml-pipeline.py) file into a workflow specification.

dsl-compile --py watson_train_serve_pipeline.py --output wml-pipeline.tar.gz
```

Then, submit `wml-pipeline.tar.gz` to the kubeflow pipeline UI. From there you can create different experiments and runs using the Watson pipeline definition.