import kfp
kfp.__version__ = "2.14.3"
kfp.TYPE_CHECK = True

from kfp import compiler, dsl

@dsl.component
def echo(item: str) -> str:
    return item

@dsl.component
def collect(items: list) -> str:
    return str(items)


@dsl.pipeline(name="repro-parallelfor-named-fixed")
def pipeline_named():
    with dsl.ParallelFor(items=["1", "2", "3"], name="My Fixed Custom Loop") as item:
        work = echo(item=item)
    collect(items=dsl.Collected(work.output))

if __name__ == "__main__":
    compiler.Compiler().compile(pipeline_named, "pipeline_named.yaml")
    print("Successfully compiled both test configurations!")
