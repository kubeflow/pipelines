# Jenkins CI Sample

## Overview
This sample use Jenkins to implement continuous integration for a simple pipeline printing "hello world" to console. 
This sample use curl to interact with kubeflow pipeline. An Alternative to use sdk to can be found in the mnist sample.

## Usage
To use this sample, you need to:
* Deploy kubeflow pipeline on GCP.
* Expose ml-pipeline in your workloads after deploying kubeflow pipeline.
* Create your gs bucket, and set it public
* Replace the constants in jenkinsfile to your own configuration.
* Deploy Jenkins on your machine or cloud
* Set up a Jenkins pipeline with the jenkins file in the folder.
* Connect Jenkins to your github repo
