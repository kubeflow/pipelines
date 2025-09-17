# Copyright 2025 The Kubeflow Authors
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
from kfp import compiler, dsl


def util_func(msg: str) -> str:
    return f"{msg} from util_func"


def util_func2(msg: str) -> str:
    return f"{msg} from util_func2"


@dsl.component(
    additional_funcs=[util_func, util_func2])
def echo(msg: str):
    assert util_func(msg) == f"{msg} from util_func"
    assert util_func2(msg) == f"{msg} from util_func2"


@dsl.pipeline(
    name="pipeline-with-utils", description="A simple hello world pipeline")
def pipeline_with_utils(msg: str = "Hello, World!"):
    echo(msg=msg)


if __name__ == "__main__":
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_utils,
        package_path=__file__.replace(".py", ".yaml"),
    )
