"""This module is useful for generating yaml files for the withParams tests and for running unformal
compiler tests during development."""
import time

from kfp.compiler import compiler
from kfp import dsl
from kfp.dsl import _for_loop


class Coder:
    def __init__(self, ):
        self._code_id = 0

    def get_code(self, ):
        self._code_id += 1
        return '{code:0{num_chars:}d}'.format(code=self._code_id, num_chars=_for_loop.LoopArguments.NUM_CODE_CHARS)


dsl.ParallelFor._get_unique_id_code = Coder().get_code


if __name__ == '__main__':
    do_output = True

    params = {}
    if do_output:
        @dsl.pipeline(name='my-pipeline')
        def pipeline():
            op0 = dsl.ContainerOp(
                name="my-out-cop0",
                image='python:alpine3.6',
                command=["sh", "-c"],
                    arguments=['python -c "import json; import sys; json.dump([{\'a\': 1, \'b\': 2}, {\'a\': 10, \'b\': 20}], open(\'/tmp/out.json\', \'w\'))"'],
                file_outputs={'out': '/tmp/out.json'},
            )

            with dsl.ParallelFor(op0.output) as item:
                op1 = dsl.ContainerOp(
                    name="my-in-cop1",
                    image="library/bash:4.4.23",
                    command=["sh", "-c"],
                    arguments=["echo do output op1 item.a: %s" % item.a],
                )

            op_out = dsl.ContainerOp(
                name="my-out-cop2",
                image="library/bash:4.4.23",
                command=["sh", "-c"],
                arguments=["echo do output op2, outp: %s" % op0.output],
            )

        job_name = f'do-output=TRUE-passed-{time.time()}'
    else:
        @dsl.pipeline(name='my-pipeline')
        def pipeline(loopidy_doop=[{'a': 1, 'b': 2}, {'a': 10, 'b': 20}]):
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
                    arguments=["echo no output global op1, item: %s" % item.a],
                ).after(op0)

            op_out = dsl.ContainerOp(
                name="my-out-cop2",
                image="library/bash:4.4.23",
                command=["sh", "-c"],
                arguments=["echo no output global op2, outp: %s" % op0.output],
            )

        job_name = f'do-output=FALSE-global-{time.time()}'

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
