# Mnist CI Pipeline

## What you can learn in this sample
* CI process of a simple but general ML pipeline.
* Launch a tensorboard as one pipeline step
* Data passing between steps


## What needs to be done before run
* Create a secret following the troubleshooting parts in [https://github.com/kubeflow/pipelines/tree/master/manifests/kustomize]()
* Set up a trigger in cloud build, and link it to your github repo
* Substitute the 'substitution' field in cloudbuild.yaml:

`_GCR_PATH`: '[YOUR CLOUD REGISTRY], for example: gcr.io/my-project' \
`_GS_BUCKET`: '[YOUR GS BUCKET TO STORE PIPELINE AND LAUNCH TENSORBOARD], for example: gs://my-bucket'\
`_PIPELINE_ID`: '[PIPELINE ID TO CREATE A VERSION ON], get it on kfp UI' \
`_HOST_NAME`: '[EXTERNAL HOST NAME OF ml-pipeline service], read README.md for more info.'

For the constant **_HOST_NAME**, you need to expose the 'ml-pipeline' service to external network so that the cloud build can create a http request to the kfp server to create a pipeline or its new version. You can achieve this by:
- Enter **pantheon -> kubernetes engine -> workloads**
- Find 'ml-pipeline' deployment
- Click into the deployment. Click into the pod. Expose the pod and choose 'Load balancer' in the field 'service type'.
- Wait for a seconds. Get the endpoints in 'Exposing services'. This will be the _HOST_NAME to put in the cloudbuild.yaml file.

* Set your container registy public
* Set the gs bucket public to view, so that cloud build can access it
* Try a commit to your repo, then you can observe the build process triggered automatically 



