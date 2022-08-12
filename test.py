from kfp.v2.dsl import component, Output, SlicedClassificationMetrics
from kfp.v2 import compiler
import kfp

# kfp_package_path=
# 'git+https://github.com/connor-mccarthy/pipelines.git@fix-sliced-classification-metrics#subdirectory=sdk/python',


@component()
def simple_metric_op(sliced_eval_metrics: Output[SlicedClassificationMetrics]):
    sliced_eval_metrics._sliced_metrics = {}
    sliced_eval_metrics.load_confusion_matrix('a slice',
                                              categories=['cat1', 'cat2'],
                                              matrix=[[1, 0], [2, 4]])


@kfp.dsl.pipeline(name="test-sliced-metric")
def metric_test_pipeline():
    m_op = simple_metric_op()


if __name__ == '__main__':
    import datetime
    import warnings
    import webbrowser

    from google.cloud import aiplatform

    from kfp import compiler

    warnings.filterwarnings('ignore')
    ir_file = __file__.replace('.py', '.json')
    # compiler.Compiler().compile(pipeline_func=my_pipeline,
    #                             package_path=ir_file)
    pipeline_name = __file__.split('/')[-1].replace('_',
                                                    '-').replace('.py', '')
    display_name = datetime.datetime.now().strftime('%m-%d-%Y-%H-%M-%S')
    job_id = f'{pipeline_name}-{display_name}'
    aiplatform.PipelineJob(template_path=ir_file,
                           pipeline_root='gs://cjmccarthy-kfp-default-bucket',
                           display_name=pipeline_name,
                           job_id=job_id).submit()
    url = f'https://console.cloud.google.com/vertex-ai/locations/us-central1/pipelines/runs/{pipeline_name}-{display_name}?project=271009669852'
    webbrowser.open_new_tab(url)

# if __name__ == '__main__':
#     import datetime
#     import warnings
#     import webbrowser

#     from kfp import Client

#     # warnings.filterwarnings('ignore')
#     # ir_file = __file__.replace('.py', '.yaml')
#     # compiler.Compiler().compile(pipeline_func=metric_test_pipeline,
#     #                             package_path=ir_file)
#     pipeline_name = __file__.split('/')[-1].replace('_',
#                                                     '-').replace('.py', '')
#     display_name = datetime.datetime.now().strftime('%m-%d-%Y-%H-%M-%S')
#     job_id = f'{pipeline_name}-{display_name}'
#     endpoint = 'https://3ac75517948b2661-dot-us-central1.pipelines.googleusercontent.com'
#     kfp_client = Client(host=endpoint)
#     run = kfp_client.create_run_from_pipeline_func(metric_test_pipeline,
#                                                    arguments={})
#     url = f'{endpoint}/#/runs/details/{run.run_id}'
#     print(url)
#     webbrowser.open_new_tab(url)
# compiler.Compiler().compile(
#     pipeline_func=metric_test_pipeline,
#     package_path="evaluation_pipeline.json",
# )