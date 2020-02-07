# Mnist CI Pipeline

## What you can learn in this sample
* CI process of a simple but general ML pipeline.
* Launch a tensorboard as one pipeline step
* Data passing between steps


## What needs to be done before run
* Create a secret following the troubleshooting parts in [https://github.com/kubeflow/pipelines/tree/master/manifests/kustomize]()
* Set up a trigger in cloud build, and link it to your github repo
* Enable "Kubernetes Engine Developer" in cloud build setting
* Replace the CLOUDSDK_COMPUTE_ZONE, CLOUDSDK_CONTAINER_CLUSTER in cloudbuild.yaml with your own zone and cluster
* Substitute the 'substitution' field in cloudbuild.yaml:

`_GCR_PATH`: '[YOUR CLOUD REGISTRY], for example: gcr.io/my-project' \
`_GS_BUCKET`: '[YOUR GS BUCKET TO STORE PIPELINE AND LAUNCH TENSORBOARD], for example: gs://my-bucket'\
`_PIPELINE_ID`: '[PIPELINE ID TO CREATE A VERSION ON], get it on kfp UI' \

* Set your container registy public or grant cloud registry access to cloud build and kubeflow pipeline
* Set your gs bucket public or grant cloud storage access to cloud build and kubeflow pipeline
* Try a commit to your repo, then you can observe the build process triggered automatically 



