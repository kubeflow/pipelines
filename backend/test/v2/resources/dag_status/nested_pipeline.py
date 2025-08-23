import kfp
import kfp.kubernetes
from kfp import dsl
from kfp.dsl import Artifact, Input, Output

@dsl.component()
def fail():
    import sys
    sys.exit(1)

@dsl.component()
def hello_world():
    print("hellow_world")

# Status for inner inner pipeline will be updated to fail
@dsl.pipeline(name="inner_inner_pipeline", description="")
def inner_inner_pipeline():
    fail()

# Status for inner pipeline stays RUNNING
@dsl.pipeline(name="inner__pipeline", description="")
def inner__pipeline():
    inner_inner_pipeline()

# Status for root stays RUNNING
@dsl.pipeline(name="outer_pipeline", description="")
def outer_pipeline():
    inner__pipeline()

if __name__ == "__main__":
    kfp.compiler.Compiler().compile(outer_pipeline, "nested_pipeline.yaml")
