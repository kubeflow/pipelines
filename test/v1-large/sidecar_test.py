import kfp
import kfp.dsl as dsl

from conftest import runner


@dsl.pipeline(
    name="pipeline-with-sidecar",
    description=
    "A pipeline that demonstrates how to add a sidecar to an operation."
)
def pipeline_with_sidecar():
    # sidecar with sevice that reply "hello world" to any GET request
    echo = dsl.Sidecar(
        name="echo",
        image="nginx:1.13",
        command=["nginx", "-g", "daemon off;"],
    )

    # container op with sidecar
    op1 = dsl.ContainerOp(
        name="download",
        image="busybox:latest",
        command=["sh", "-c"],
        arguments=[
            "until wget http://localhost:80 -O /tmp/results.txt; do sleep 5; done && cat /tmp/results.txt"
        ],
        sidecars=[echo],
        file_outputs={"downloaded": "/tmp/results.txt"},
    )

def test_pipeline_with_sidecar():
    runner(pipeline_with_sidecar)


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(pipeline_with_sidecar, __file__ + '.yaml')
