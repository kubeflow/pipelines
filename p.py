from kfp import dsl


@dsl.component
def identity(string: str) -> str:
    return string


@dsl.pipeline
def my_pipeline(string: str = 'string'):
    op1 = identity(string=string)
    op2 = identity(string=op1.output)


if __name__ == '__main__':
    import datetime
    import warnings
    import webbrowser

    from google.cloud import aiplatform

    from kfp import compiler

    warnings.filterwarnings('ignore')
    ir_file = __file__.replace('.py', '.yaml')
    compiler.Compiler().compile(pipeline_func=my_pipeline,
                                package_path=ir_file)
