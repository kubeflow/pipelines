# kfp-functional-test

## Updating python dependencies

[pip-tools](https://github.com/jazzband/pip-tools) is used to manage python
dependencies. To update dependencies:
1. edit [requirements.in](requirements.in)
1. run
    ```
    ../../backend/update_requirements.sh <requirements.in >requirements.txt
    ```
    to update and pin the transitive dependencies.

## Run kfp-functional-test in local

### Via python

1. run
   ```
    gcloud auth application-default login
   ```
    acquire new user credentials to use for Application Default Credentials.

1. go to the root directory of kubeflow pipelines project, run
   ```
   cd {YOUR_ROOT_DIRECTORY_OF_KUBEFLOW_PIPELINES}
   python3 ./test/kfp-functional-test/run_kfp_functional_test.py  --host "https://$(curl https://raw.githubusercontent.com/kubeflow/testing/master/test-infra/kfp/endpoint)"
   ```

### Via docker
1. run
    ```
    gcloud auth application-default login
    ```
    acquire new user credentials to use for Application Default Credentials. 
    Credentials saved to file with {CREDENTIALS_PATH} similar to: [$HOME/.config/gcloud/application_default_credentials.json]

1. copy the Credentials to the temp folder

   ````
    cp {CREDENTIALS_PATH} /tmp/keys/{FILENAME}.json
   ```
1. Provide authentication credentials by setting the environment variable GOOGLE_APPLICATION_CREDENTIALS. 
   Replace [PATH] with the file path of the JSON file that contains your credentials.
   run 

   ```
   export GOOGLE_APPLICATION_CREDENTIALS="/tmp/keys/{FILENAME}.json"
   ```

1. go to the root directory of kubeflow pipelines project and run
   ```
   cd {YOUR_ROOT_DIRECTORY_OF_KUBEFLOW_PIPELINES}
   docker run -it -v $(pwd):/tmp/src -w /tmp/src -e GOOGLE_APPLICATION_CREDENTIALS=/tmp/keys/{FILENAME}.json \
   -v $GOOGLE_APPLICATION_CREDENTIALS:/tmp/keys/{FILENAME}.json:ro \
   python:3.7-slim /tmp/src/test/kfp-functional-test/kfp-functional-test.sh
   ```