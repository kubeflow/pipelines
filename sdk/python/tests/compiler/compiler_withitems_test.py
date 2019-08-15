from typing import Iterable, Text

from kfp.compiler import compiler
import kfp.dsl as dsl

# reproducible UUIDs: https://stackoverflow.com/a/56757552/9357327
import uuid
import random
# -------------------------------------------
# Remove this block to generate different
# UUIDs everytime you run this code.
# This block should be right below the uuid
# import.
rd = random.Random()
rd.seed(0)
uuid.uuid4 = lambda: uuid.UUID(int=rd.getrandbits(128))
# -------------------------------------------


if __name__ == '__main__':
    @dsl.pipeline(name='my-pipeline', description='A pipeline with multiple pipeline params.')
    def pipeline(my_pipe_param=10):
        loop_args = [{'a': 1, 'b': 2}, {'a': 10, 'b': 20}]
        with dsl.ParallelFor(loop_args) as item:
            op1 = dsl.ContainerOp(
                name="my-in-coop1",
                image="library/bash:4.4.23",
                command=["sh", "-c"],
                arguments=["echo op1 %s %s" % (item.a, my_pipe_param)],
            )

            op2 = dsl.ContainerOp(
                name="my-in-coop2",
                image="library/bash:4.4.23",
                command=["sh", "-c"],
                arguments=["echo op2 %s" % item.b],
            )

        op_out = dsl.ContainerOp(
            name="my-out-cop",
            image="library/bash:4.4.23",
            command=["sh", "-c"],
            arguments=["echo %s" % my_pipe_param],
        )

    yaml_text = compiler.Compiler().compile(pipeline, None)
    print(yaml_text)

    import kfp
    import time
    client = kfp.Client(host='127.0.0.1:8080/pipeline')
    print(client.list_experiments())

    pkg_path = '/tmp/witest_pkg.tar.gz'
    compiler.Compiler().compile(pipeline, package_path=pkg_path)
    exp = client.create_experiment('withitems_exp')
    client.run_pipeline(
        experiment_id=exp.id,
        job_name=f'withitems_job_{time.time()}',
        pipeline_package_path=pkg_path,
        params={'my-pipe-param': 11},
    )
