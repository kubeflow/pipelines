from kfp import dsl
from google_cloud_pipeline_components.v1.dataflow import DataflowPythonJobOp
from google_cloud_pipeline_components.v1.wait_gcp_resources import WaitGcpResourcesOp


@dsl.pipeline
def my_pipeline():
    """Sample pipeline for dataflow python job."""

    dataflow_python_op = DataflowPythonJobOp(
        project='186556260430',
        location='us-central1',
        requirements_file_path='gs://ml-pipeline-playground/samples/dataflow/wc/requirements.txt',
        python_module_path='gs://cjmccarthy-managed-pipelines-test/foo.py',
        temp_location='gs://cjmccarthy-managed-pipelines-test/dest',
    )
    WaitGcpResourcesOp(gcp_resources=dataflow_python_op.outputs['gcp_resources']
                      ).ignore_upstream_failure()


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
        project='186556260430',
        location='us-central1',
        template_path=ir_file,
        pipeline_root='gs://cjmccarthy-managed-pipelines-test',
        display_name=pipeline_name,
        job_id=job_id).submit()
    url = f'https://console.cloud.google.com/vertex-ai/locations/us-central1/pipelines/runs/{pipeline_name}-{display_name}?project=186556260430'
    webbrowser.open_new_tab(url)