from kfp import dsl
from kfp import kubernetes


@dsl.component
def comp():
    pass


@dsl.pipeline
def my_pipeline():
    task = comp()
    kubernetes.use_secret_as_volume(
        task, secret_name='my-secret', mount_path='/mnt/my_vol')


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(my_pipeline, __file__.replace('.py', '.yaml'))
