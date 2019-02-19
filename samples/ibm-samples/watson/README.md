The `Watson Train and Serve` sample pipeline runs training, storing and deploying a Tensorflow model with MNIST handwriting recognition using [IBM Watson Studio](https://www.ibm.com/cloud/watson-studio) and [IBM Watson Machine Learning](https://www.ibm.com/cloud/machine-learning) service.

# Requirements

This sample requires the user to have provisioned a machine learning service on Watson, a S3 cloud storage set up and the service credentials configured in the creds.ini file.

To provision your own Watson Machine Learning services and S3 cloud storage, following are the required steps.

* IBM Watson Machine Learning service instance

To create a machine learning service, go to [IBM Cloud](https://console.bluemix.net), login with IBM account id first. From the `Catalog` page, click on `AI` tab on the left side to go to this [page](https://console.bluemix.net/catalog/?category=ai). Then click on the [`Machine Learning`](https://console.bluemix.net/catalog/services/machine-learning) link and follow the instructions to create the service.

Once the service is created, from service's `Dashboard`, follow the instruction to generate `service credentials`. Refer to IBM Cloud [documents](https://console.bluemix.net/docs/) for help if needed. Collect the `url`, `username`, `password` and `instance_id` info from the service credentials as these will be required to access the service.

* An S3 cloud storage

Watson Machine Learning service loads datasets from S3 cloud storage and stores model outputs and other artifacts to S3 cloud storage. Users can use any S3 cloud storage they already preserve. Users can also create a S3 cloud storage with `IBM Cloud Object Storage` service by following this [link](https://console.bluemix.net/catalog/services/cloud-object-storage).

Collect the `endpoint`, `access_key_id` and `secret_access_key` fields from the service credentials for the S3 cloud storage. Create the service credentials first if not existed. To ensure generating HMAC credentials, specify the following in the `Add Inline Configuration Parameters` field: `{"HMAC":true}`.

Create two buckets, one for storing the train datasets and model source codes, and one for storing the model outputs.

* Set up access credentials

This pipeline sample reads the credentials from a file hosted in a github repo. Refer to `creds.ini` file and input user's specific credentials. Then upload the file to a github repo the user has access.

Note: make sure the `s3_endpoint` value in the `creds.ini` file only contains valid endpoint without the `http://` or `https://` prefix.

To access the credentials file, the user should provide a github access token and the link to the raw content of the file. Modify the `GITHUB_TOKEN` and `CONFIG_FILE_URL` variables in the `watson_train_serve_pipeline.py` file with the user's access token and link.

# The datasets

This pipeline sample uses the [MNIST](http://yann.lecun.com/exdb/mnist) datasets, including [train-images-idx3-ubyte.gz](http://yann.lecun.com/exdb/mnist/train-images-idx3-ubyte.gz), [train-labels-idx1-ubyte.gz](http://yann.lecun.com/exdb/mnist/train-labels-idx1-ubyte.gz), [t10k-images-idx3-ubyte.gz](http://yann.lecun.com/exdb/mnist/t10k-images-idx3-ubyte.gz), and [t10k-labels-idx1-ubyte.gz](http://yann.lecun.com/exdb/mnist/t10k-labels-idx1-ubyte.gz).

If users are using their own S3 cloud storage instances, download these datasets and upload to the input bucket created above on the S3 cloud storage.

# Upload model code and datasets

Once the user has the model train source code ready, compress all files into one `zip` format file.

For example, run following command to compress the sample model train codes

```command line
pushd source/model-source-code
zip -j tf-model tf-model/convolutional_network.py tf-model/input_data.py
popd
```

This should create a `tf-model.zip` file.

Upload the model train code, together with the train datasets, to the input bucket created above in the S3 cloud storage.

# Upload model scoring payload

At the end of the deploy stage of this sample, `tf-mnist-test-payload.json` is used as the scoring payload to test the deployment. Upload this file to the input bucket in the S3 cloud storage.

# Compling the pipeline template

First, install the necessary Python package for setting up the access to the Watson Machine Learning service and S3 cloud storage, with following command

```command line
pip3 install ai_pipeline_params
```

Then follow the guide to [building a pipeline](https://www.kubeflow.org/docs/pipelines/build-pipeline/) to install the Kubeflow Pipelines SDK, and run the following command to compile the sample `watson_train_serve_pipeline.py` file into a workflow specification.

```
dsl-compile --py watson_train_serve_pipeline.py --output wml-pipeline.tar.gz
```

Then, submit `wml-pipeline.tar.gz` to the kubeflow pipeline UI. From there you can create different experiments and runs using the Watson pipeline definition.
