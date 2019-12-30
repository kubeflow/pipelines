# Test SDK Client

## Overview

This folder includes a test on sdk client creating pipelines and pipeline versions in batches.
## Usage

To test the sdk client on creating pipeline and its versions, you need to first:

* Make sure your command line is connected to a cluster with kubeflow pipeline already deployed.
* Put a pipeline.zip file in your cloud storage bucke·t. Expose the bucket public. If you store your `pipeline.zip` in `gs：//my-project/pipeline.zip`, your url for the pipeline.zip package should be： `https://storage.googleapis.com/my-project/pipeline.zip`
* Use the command line. For example, if you would like to create 100 pipelines, and 100 versions for each pipeline, use this command below:
```bash
python test_create_pipeline_and_version.py
--pipeline_url [YOUR URL FOR YOUR PIPELINE PACKAGE]
--create_pipeline_num 100
--create_pipeline_version_num 100
```