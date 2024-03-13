from kfp import dsl
from kfp import local
from kfp.dsl import Artifact
from kfp.dsl import Output

local.init(runner=local.SubprocessRunner())

project = '<your-project-here>'

source = """
def main():
    import sys
    import json
    import os

    executor_input = sys.argv[1]

    my_custom_uri = 'gs://{project}/foo/bar/blah.txt'
    local_path = my_custom_uri.replace('gs://', '/gcs/')

    # write artifact
    os.makedirs(os.path.dirname(local_path), exist_ok=True)
    with open(local_path, 'w+') as f:
        f.write('my custom artifact')


    # tell Pipelines backend where you wrote it
    executor_input_struct = json.loads(executor_input)
    artifact_to_override = executor_input_struct['outputs']['artifacts']['a']
    executor_output_path = executor_input_struct['outputs']['outputFile']
    artifact_to_override['artifacts'][0]['uri'] = my_custom_uri
    updated_executor_output = {'artifacts': {'a': artifact_to_override}}

    os.makedirs(os.path.dirname(executor_output_path), exist_ok=True)
    with open(executor_output_path, 'w+') as f:
        json.dump(updated_executor_output, f)

main()
"""


@dsl.container_component
def comp(
    a: Output[Artifact],
    executor_input: str = dsl.PIPELINE_TASK_EXECUTOR_INPUT_PLACEHOLDER,
):
    return dsl.ContainerSpec(
        image='python:3.8',
        command=['python', '-c'],
        args=[source, executor_input],
    )


@dsl.pipeline
def my_pipeline():
    comp()


# my_pipeline()
if __name__ == '__main__':
    import datetime
    import warnings
    import webbrowser

    from google.cloud import aiplatform
    from kfp import compiler

    warnings.filterwarnings('ignore')
    ir_file = __file__.replace('.py', '.yaml')
    compiler.Compiler().compile(pipeline_func=my_pipeline, package_path=ir_file)
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
