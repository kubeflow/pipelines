from kfp import dsl, compiler
from kfp.dsl import PipelineConfig

config = PipelineConfig()
config.semaphore_key = 'semaphore'

@dsl.component
def comp():
    pass

@dsl.pipeline(pipeline_config=config)
def pipeline_with_semaphore():
    task = comp()


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_semaphore,
        package_path=__file__.replace('.py', '.yaml'))