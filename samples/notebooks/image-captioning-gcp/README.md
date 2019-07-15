# Image Captioning TF 2.0

## About
This notebook is an example of how to convert an existing Tensorflow notebook into a Kubeflow pipeline using jupyter notebook.  Specifically, this notebook takes an example tensorflow notebook, [image captioning with attention](https://colab.sandbox.google.com/github/tensorflow/docs/blob/master/site/en/r2/tutorials/text/image_captioning.ipynb), and creates a kubeflow pipeline.  

## Setup

### Setup notebook server
This pipeline requires you to [setup a notebook server](https://www.kubeflow.org/docs/notebooks/setup/) in the Kubeflow UI.  After you are setup, upload this notebook and then run it in the notebook server.

### Create a GCS bucket
This pipeline requires a GCS bucket.  If you haven't already, [create a GCS bucket](https://cloud.google.com/storage/docs/creating-buckets) to run the notebook.  Make sure to create the storage bucket in the same project that you are running Kubeflow on to have the proper permissions by default.  You can also create a GCS bucket by running `gsutil mb -p <project_name> gs://<bucket_name>`.

### Upload the notebook in the Kubeflow UI
In order to run this pipeline, make sure to upload the notebook to your notebook server in the Kubeflow UI.  You can clone this repo in the Jupyter notebook server by connecting to the notebook server and then selecting New > Terminal.  In the terminal type `git clone https://github.com/kubeflow/pipelines.git`.

## Outputs
Below are some screenshots of the final pipeline and the model outputs.

![pipeline-screenshot](https://user-images.githubusercontent.com/17008638/61160416-41694f80-a4b4-11e9-9317-5a92f625c173.png)

![attention-screenshot](https://user-images.githubusercontent.com/17008638/61160441-59d96a00-a4b4-11e9-809b-f3df7cbe0dae.PNG)