from kfp import compiler
from kfp import dsl


@dsl.component
def component_a():
    print('Component A')

@dsl.component
def component_b():
    print ('Component B')

@dsl.pipeline(name='nested-pipeline')
def nested_pipeline():
    component_a()
    component_b().set_retry(num_retries=2)

@dsl.pipeline(name='hello-world')
def pipeline():
    nested_pipeline().set_retry(num_retries=1, backoff_max_duration='1800s', backoff_factor=1)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline,
        package_path=__file__.replace('.py', '.json'))

