from kfp import dsl
from typing import Dict, Any


@dsl.component(
    base_image='gcr.io/ml-pipeline/google-cloud-pipeline-components:2.3.1')
def vertex_ai_request(
    project: str,
    location: str,
    resource: str,
    method: str,
    query_parameters: dict = {},
    path_parameters: dict = {},
    api_version: str = 'v1',
):
    import requests
    import os

    def get_access_token():
        url = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token"
        headers = {"Metadata-Flavor": "Google"}
        response = requests.get(url, headers=headers)
        response.raise_for_status()
        return response.json()["access_token"]

    ENDPOINT_BASE = f'https://{location}-aiplatform.googleapis.com/{api_version}/projects/{project}/locations/{location}'

    method_map = {
        'datasets': {
            'methods': {
                'get': ('GET', 'datasets/{name}'),
                'list': ('GET', 'datasets'),
                # 'create': ('POST',),
                # 'delete': ('DELETE',),
                # 'export': ('POST',),
                # 'import': ('POST'),
                # 'patch': ('PATCH',),
                # 'searchDataItems': ('GET',),
            }
        }
    }
    print(
        "Hitting endpoint: ",
        os.path.join(
            ENDPOINT_BASE, method_map[resource]['methods'][method][1].format(
                project=project, **path_parameters)))
    headers = {
        "Authorization": f"Bearer {get_access_token()}",
        "Content-Type": "application/json"
    }
    response = requests.request(
        method=method_map[resource]['methods'][method][0],
        url=os.path.join(
            ENDPOINT_BASE, method_map[resource]['methods'][method][1].format(
                project=project, **path_parameters)),
        headers=headers,
        params=query_parameters,
    )
    response.raise_for_status()

    print("Got response:")
    print(response.json())
    print(
        "Here is where we would persist the artifact output based on the method used and the response payload."
    )


VertexAIRequest = vertex_ai_request


@dsl.pipeline
def demo_pipeline():
    op1 = VertexAIRequest(
        project='cjmccarthy-kfp',
        location='us-central1',
        resource='datasets',
        method='get',
        path_parameters={'name': '8622262432879869952'})


if __name__ == '__main__':
    import datetime
    import warnings
    import webbrowser

    from google.cloud import aiplatform

    from kfp import compiler

    warnings.filterwarnings('ignore')
    ir_file = __file__.replace('.py', '.yaml')
    compiler.Compiler().compile(
        pipeline_func=demo_pipeline, package_path=ir_file)
    pipeline_name = __file__.split('/')[-1].replace('_', '-').replace('.py', '')
    display_name = datetime.datetime.now().strftime('%m-%d-%Y-%H-%M-%S')
    job_id = f'{pipeline_name}-{display_name}'
    aiplatform.PipelineJob(
        template_path=ir_file,
        pipeline_root='gs://cjmccarthy-kfp-default-bucket',
        display_name=pipeline_name,
        job_id=job_id).submit()
    url = f'https://console.cloud.google.com/vertex-ai/locations/us-central1/pipelines/runs/{pipeline_name}-{display_name}?project=271009669852'
    webbrowser.open_new_tab(url)