# Unit tests for AWS Sagemaker KFP Components 

These unit tests test the user input parsing part of the code.

## How to run these tests

1. Clone the git repo 
    ```
    # For now using my branch as an example 
    
    git clone https://github.com/kubeflow/pipelines.git
    ```
2. Install the pip pakages required for testing 
    ```
    cd pipelines/components/aws/sagemaker/unit_tests/
   
    pip install -r requirements.txt 
    ```
3. Run all unit tests 
    ```
    # while in the same directory pipelines/components/aws/sagemaker/unit_tests/
   
    ./run_all_tests.sh
    ```
   