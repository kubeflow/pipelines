from kfp import dsl
from kfp.dsl import *
from typing import *

setup = []


@dsl.component
def genai_eval_component_prototype(string: str) -> str:
    import subprocess
    try:
        subprocess.check_call([
            "pip",
            "install",
            "--upgrade",
            "--force-reinstall",
            "/gcs/vertex_eval_sdk_private_releases/rapid_genai_evaluation/google_cloud_aiplatform-1.42.dev20240208+rapid.genai.evaluation-py2.py3-none-any.whl",
            "--no-warn-conflicts",
        ])
    except subprocess.CalledProcessError as e:
        print(f"Error: {e}")

    from google.cloud.aiplatform.private_preview.rapid_genai_evaluation.evaluate import evaluate
    import vertexai
    from google.cloud import aiplatform

    return string


if __name__ == '__main__':
    import datetime
    import warnings
    import webbrowser

    from google.cloud import aiplatform

    from kfp import compiler

    warnings.filterwarnings('ignore')
    ir_file = __file__.replace('.py', '.yaml')
    compiler.Compiler().compile(
        pipeline_func=genai_eval_component_prototype,
        package_path=ir_file,
    )
    pipeline_name = __file__.split('/')[-1].replace('_', '-').replace('.py', '')
    display_name = datetime.datetime.now().strftime('%m-%d-%Y-%H-%M-%S')
    job_id = f'{pipeline_name}-{display_name}'
    aiplatform.PipelineJob(
        project='managed-pipeline-test',
        location='us-central1',
        template_path=ir_file,
        pipeline_root='gs://cjmccarthy-managed-pipeline-test',
        display_name=pipeline_name,
        parameter_values={
            'string': 'bar'
        },
        job_id=job_id).submit()
    url = f'https://console.cloud.google.com/vertex-ai/locations/us-central1/pipelines/runs/{pipeline_name}-{display_name}?project=managed-pipeline-test'
    webbrowser.open_new_tab(url)
