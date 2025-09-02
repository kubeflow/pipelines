import os

from kfp import dsl

from kfp import compiler

@dsl.component()
def verify_retries(retryCount: str, retries: str) -> bool:
    if retryCount != retries:
        raise Exception(f"Number of retries has not reached {retries} blank yet.")
    return True

@dsl.component()
def print_op(text: str) -> str:
    print(text)
    return text

@dsl.pipeline
def pipeline_single_component():
    task2 = verify_retries(retryCount="{{retries}}", retries='1')

@dsl.pipeline
def pipeline_single_component_with_retry():
    task2 = verify_retries(retryCount="{{retries}}", retries='1').set_retry(num_retries=1)

@dsl.pipeline
def pipeline_multi_component():
    task2 = verify_retries(retryCount="{{retries}}", retries='1')
    task2 = verify_retries(retryCount="{{retries}}", retries='2').set_retry(num_retries=2)

@dsl.pipeline
def retry_pipeline():
    task1 = verify_retries(retryCount="{{retries}}", retries='2').set_retry(num_retries=2)
    task2 = verify_retries(retryCount="{{retries}}", retries='2').set_retry(num_retries=2, backoff_duration="0s", backoff_factor=2, backoff_max_duration="3600s")
    task3 = print_op(text='test').set_retry(num_retries=0)

    #  retry with all args set at pipeline level only
    task4 = pipeline_single_component().set_retry(num_retries=1)
    # retry set at component level only
    task5 = pipeline_single_component_with_retry()
    # retry set at both component and pipeline level
    task6 = pipeline_multi_component().set_retry(num_retries=1)


if __name__ == '__main__':
    compiler.Compiler.compile(
        pipeline_func=retry_pipeline,
        package_path='pipeline_with_retry.yaml')

