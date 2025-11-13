import kfp.dsl as dsl
from kfp import compiler

@dsl.component(base_image='registry.access.redhat.com/ubi9/python-311:latest', packages_to_install=["requests"])
def submit_request(url: str):
    import requests
    import os

    # If running pipeline in multi-user mode, skip submitting request to external address.
    multi_user = os.getenv('MULTI_USER', 'false')
    if multi_user.lower() != 'true':
        return

    scheme = os.getenv('SCHEME', 'http')
    if scheme not in ['http', 'https']:
        raise Exception(f"Invalid scheme: {scheme}")

    url = f'{scheme}://{url}'
    # Submit HTTP or HTTPS request to in-cluster endpoint and external address, depending on the scheme specified by env var.
    try:
        response = requests.get(url, verify=True, timeout=5)
        print(response.status_code)
    except Exception as e:
        raise Exception(f"Failed to request {url}: {e}")
    if response.status_code != 200:
        raise Exception(f"Failed to request {url}: {response.status_code}")


@dsl.pipeline(name='pipeline-with-external-request', description='A pipeline that requests both in-cluster & external URLs.')
def pipeline_with_submit_request():
    # When TLS is enabled, verifies both the system CA bundle and the cert-manager CA (Mounted by the compiler when apiserver is TLS-enabled)
    # are trusted by the Python executor. If non TLS-enabled, executes HTTP requests.

    submit_request(url='httpbin.org/get').set_caching_options(False)
    submit_request(url='ml-pipeline:8888/apis/v2beta1/healthz').set_caching_options(False)


if __name__ == '__main__':
    # execute only if run as a script
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_submit_request,
        package_path='../../test_data/pipeline_files/valid/critical/pipeline_with_submit_request.yaml'
    )
