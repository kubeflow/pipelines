from typing import List

from kfp import dsl


@dsl.component
def get_list() -> List[str]:
    return ['a', 'b', 'c']


@dsl.component
def my_component(string: str) -> str:
    print(string)
    return string


@dsl.component
def aggregator(strings: List[str]) -> str:
    return strings[0]


@dsl.pipeline(name='pipeline')
def my_pipeline():
    l = get_list()
    with dsl.ParallelFor(['a', 'b', 'c']) as s:
        task = my_component(string=s)
    # res = [my_component(string=string) for string in dsl.ParallelFor(l.output)]
    aggregator(strings=l.output)


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
