import time
from typing import Iterable, Text

from kfp.compiler import compiler
from kfp import dsl
import kfp

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
    do_output = True

    if do_output:
        @dsl.pipeline(name='my-pipeline', description='A pipeline with multiple pipeline params.')
        def pipeline():
            op0 = dsl.ContainerOp(
                name="my-out-cop0",
                image='python:alpine3.6',
                command=["sh", "-c"],
                arguments=['python -c "import json; import sys; json.dump([i for i in range(20, 31)], open(\'/tmp/out.json\', \'w\'))"'],
                file_outputs={'out': '/tmp/out.json'},
            )

            # loop_args = [{'a': 1, 'b': 2}, {'a': 10, 'b': 20}]
            with dsl.ParallelFor(op0.output) as item:
                op1 = dsl.ContainerOp(
                    name="my-in-cop1",
                    image="library/bash:4.4.23",
                    command=["sh", "-c"],
                    arguments=["echo op1 %s" % item],
                )

            op_out = dsl.ContainerOp(
                name="my-out-cop2",
                image="library/bash:4.4.23",
                command=["sh", "-c"],
                arguments=["echo %s" % op0.output],
            )

        job_name = f'withparams_passed_param_{time.time()}'
        params = {}
    else:
        @dsl.pipeline(name='my-pipeline', description='A pipeline with multiple pipeline params.')
        def pipeline(loopidy_doop=[3, 5, 7, 9]):
            op0 = dsl.ContainerOp(
                name="my-out-cop0",
                image='python:alpine3.6',
                command=["sh", "-c"],
                arguments=['python -c "import json; import sys; json.dump([i for i in range(20, 31)], open(\'/tmp/out.json\', \'w\'))"'],
                file_outputs={'out': '/tmp/out.json'},
            )

            with dsl.ParallelFor(loopidy_doop) as item:
                op1 = dsl.ContainerOp(
                    name="my-in-cop1",
                    image="library/bash:4.4.23",
                    command=["sh", "-c"],
                    arguments=["echo op1 %s" % item],
                ).after(op0)

            op_out = dsl.ContainerOp(
                name="my-out-cop2",
                image="library/bash:4.4.23",
                command=["sh", "-c"],
                arguments=["echo %s" % op0.output],
            ).after(op1)

        job_name = f'withparams_global_param_{time.time()}'
        params = {}

    yaml_text = compiler.Compiler().compile(pipeline, None)
    print(yaml_text)

    import kfp
    import time
    client = kfp.Client(host='127.0.0.1:8080/pipeline')
    print(client.list_experiments())

    pkg_path = '/tmp/witest_pkg.tar.gz'
    compiler.Compiler().compile(pipeline, package_path=pkg_path)
    exp = client.create_experiment('withparams_exp')
    client.run_pipeline(
        experiment_id=exp.id,
        job_name=job_name,
        pipeline_package_path=pkg_path,
        params=params,
    )
