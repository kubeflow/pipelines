from kfp import dsl, compiler
@dsl.component()
def fail():
    import sys
    sys.exit(1)

@dsl.component()
def hello_world():
    print("hellow_world")

@dsl.component()
def post_msg():
    print(f"this is a message")

@dsl.component()
def output_msg() -> str:
    return "that"

@dsl.pipeline
def pipeline():
    output = output_msg().set_caching_options(enable_caching=False)
    # This will fail to report in the outer dag
    # Note that this dag will have multiple total_dag_tasks
    # But only one of them will be executed.
    with dsl.If('this' == output.output):
        hello_world().set_caching_options(enable_caching=False)
    with dsl.Else():
        fail().set_caching_options(enable_caching=False)

    # More nested dags
    with dsl.If('that' == output.output):
        with dsl.If('this' == output.output):
            hello_world().set_caching_options(enable_caching=False)
        with dsl.Elif('this2' == output.output):
            hello_world().set_caching_options(enable_caching=False)
        with dsl.Else():
            fail().set_caching_options(enable_caching=False)


compiler.Compiler().compile(
    pipeline_func=pipeline,
    package_path=__file__.replace('.py', '-v2.yaml'))
