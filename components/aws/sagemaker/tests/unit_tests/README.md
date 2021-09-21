# Unit tests for AWS SageMaker KFP Components  

## How to run these tests 

### Method 1 : Run these tests using docker 

1. Clone the git repo 
    ```
    git clone https://github.com/kubeflow/pipelines.git
    ```
2. Build the dockerfile  
    ```
    cd pipelines
    docker build . -f ./components/aws/sagemaker/tests/unit_tests/Dockerfile -t amazon/unit-test-aws-sagemaker-kfp-components
    ```
3. Run all unit tests
   ```
   docker run -it -v <path_to_this_repo_on_your_machine>:/app/ amazon/unit-test-aws-sagemaker-kfp-components:latest
   ```
   This runs the tests against a mounted volume from your host machine. This means you can edit the files and rerun the tests immediately without having to rebuild the docker container.
   
--------------

### Method 2 : Run these tests locally

1. Clone the git repo 
    ```
    git clone https://github.com/kubeflow/pipelines.git
    ```
2. Install the pip packages required for testing 
    ```
    cd pipelines/components/aws/sagemaker/
   
    pip install -r requirements.txt 
    ```
3. Run all unit tests 
    ```
    cd tests/unit_tests/
   
    ./run_unit_tests.sh
    ```