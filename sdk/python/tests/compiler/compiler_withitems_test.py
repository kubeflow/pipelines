from kfp.compiler import compiler
import kfp.dsl as dsl
from kfp.dsl import _for_loop


class Coder:
    def __init__(self, ):
        self._code_id = 0

    def get_code(self, ):
        self._code_id += 1
        return '{code:0{num_chars:}d}'.format(code=self._code_id, num_chars=_for_loop.LoopArguments.NUM_CODE_CHARS)

dsl.ParallelFor._get_unique_id_code = Coder().get_code


@dsl.pipeline(name='my-pipeline')
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


# @dsl.pipeline(name='my-pipeline')
# def pipeline(my_pipe_param=10):
#     loop_args = [{'a': 1, 'b': 2}, {'a': 10, 'b': 20}]
#     with dsl.ParallelFor(loop_args) as item:
#         op1 = dsl.ContainerOp(
#             name="my-in-coop1",
#             image="library/bash:4.4.23",
#             command=["sh", "-c"],
#             arguments=["echo op1 %s %s" % (item.a, my_pipe_param)],
#         )
#
#         with dsl.ParallelFor([100, 200, 300]) as inner_item:
#             op11 = dsl.ContainerOp(
#                 name="my-inner-inner-coop",
#                 image="library/bash:4.4.23",
#                 command=["sh", "-c"],
#                 arguments=["echo op1 %s %s %s" % (item.a, inner_item, my_pipe_param)],
#             )
#
#         op2 = dsl.ContainerOp(
#             name="my-in-coop2",
#             image="library/bash:4.4.23",
#             command=["sh", "-c"],
#             arguments=["echo op2 %s" % item.b],
#         )
#
#     op_out = dsl.ContainerOp(
#         name="my-out-cop",
#         image="library/bash:4.4.23",
#         command=["sh", "-c"],
#         arguments=["echo %s" % my_pipe_param],
#     )


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
