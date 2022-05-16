from kfp import dsl


@dsl.component(
    base_image='python:3.7-alpine',
    packages_to_install=['numpy'],
    install_kfp_package=True,
    kfp_package_path='https://pypi.org/simple/kfp',
)
def my_component(string: str, number: int) -> str:
    string *= number
    return string


@dsl.component(
    base_image='python:3.7-alpine',
    packages_to_install=['numpy'],
    install_kfp_package=True,
    kfp_package_path='https://pypi.org/simple/kfp',
)
def another(string: str, number: int) -> str:
    string *= number
    return string


@dsl.pipeline(
    name='pipeline',
    pipeline_root='gs://my-bucket',
)
def my_pipeline(string: str, number: int):
    op1 = my_component(string=string, number=number)
    op1.set_env_variable('MY_ENV_VAR', 'my-value')
    op1.set_cpu_limit('2')
    op1.set_memory_limit('16Gi')
    op1.set_display_name('my-display-name')
    op1.add_node_selector_constraint('TPU_V3')
    op1.set_gpu_limit('3')
    op1.set_caching_options(True)

    another(string=op1.output, number=number)

    op2 = my_component(string=op1.output, number=number)


if __name__ == '__main__':
    import datetime
    import warnings
    import webbrowser

    from google.cloud import aiplatform
    from kfp import compiler

    warnings.filterwarnings('ignore')
    ir_file = __file__.replace('.py', '.yaml')
    compiler.Compiler().compile(pipeline_func=my_pipeline, package_path=ir_file)
    # pipeline_name = __file__.split('/')[-1].replace('_', '-').replace('.py', '')
    # display_name = datetime.datetime.now().strftime('%m-%d-%Y-%H-%M-%S')
    # job_id = f'{pipeline_name}-{display_name}'
    # aiplatform.PipelineJob(
    #     template_path=ir_file,
    #     pipeline_root='gs://cjmccarthy-kfp-default-bucket',
    #     display_name=pipeline_name,
    #     job_id=job_id).submit()
    # url = f'https://console.cloud.google.com/vertex-ai/locations/us-central1/pipelines/runs/{pipeline_name}-{display_name}?project=271009669852'
    # webbrowser.open_new_tab(url)
