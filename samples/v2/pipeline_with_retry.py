import os

from kfp import dsl

from kfp import compiler

_KFP_PACKAGE_PATH = os.getenv('KFP_PACKAGE_PATH')

@dsl.component(kfp_package_path=_KFP_PACKAGE_PATH)
def verify_retries(retries: str) -> bool:
    if retries != '2':
        raise Exception('Number of retries has not reached two yet.')
    return True

@dsl.component(kfp_package_path=_KFP_PACKAGE_PATH)
def print_op(text: str) -> str:
    print(text)
    return text


@dsl.pipeline()
def retry_pipeline():
    task1 = verify_retries(retries="{{retries}}").set_retry(num_retries=2)
    task2 = print_op(text='test').set_retry(num_retries=2, backoff_duration="1 second", backoff_factor=1, backoff_max_duration="1 second")
    task3 = print_op(text='test').set_retry(num_retries=0)

if __name__ == '__main__':
    compiler.Compiler.compile(
        pipeline_func=retry_pipeline,
        package_path='pipeline_with_retry.yaml')

