# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import kfp.dsl as dsl
from kfp.dsl import _for_loop


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


if __name__ == '__main__':
    from kfp import compiler
    import kfp
    import time
    client = kfp.Client(host='127.0.0.1:8080/pipeline')
    print(compiler.Compiler().compile(pipeline, package_path=None))

    pkg_path = '/tmp/witest_pkg.tar.gz'
    compiler.Compiler().compile(pipeline, package_path=pkg_path)
    exp = client.create_experiment('withparams_exp')
    client.run_pipeline(
        experiment_id=exp.id,
        job_name='withparam_output_dict_{}'.format(time.time()),
        pipeline_package_path=pkg_path,
        params={},
    )
