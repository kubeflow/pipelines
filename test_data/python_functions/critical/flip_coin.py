# Copyright 2022 The Kubeflow Authors
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


@dsl.component()
def random_num(low: int, high: int) -> int:
    """Generate a random number between low and high."""
    import random  # noqa: PLC0415

    result = random.randint(low, high)
    print(result)
    return result


@dsl.component()
def flip_coin() -> str:
    """Flip a coin and output heads or tails randomly."""
    import random  # noqa: PLC0415

    result = "heads" if random.randint(0, 1) == 0 else "tails"
    print(result)
    return result


@dsl.component()
def print_msg(msg: str):
    """Print a message."""
    print(msg)


@dsl.pipeline(
    name="conditional-execution-pipeline",
    description="Shows how to use dsl.If().",
)
def flipcoin_pipeline():
    flip = flip_coin().set_caching_options(False)
    with dsl.If(flip.output == "heads"):
        random_num_head = random_num(low=0, high=9).set_caching_options(False)
        with dsl.If(random_num_head.output > 5):
            print_msg(msg="heads and %s > 5!" % random_num_head.output).set_caching_options(False)
        with dsl.If(random_num_head.output <= 5):
            print_msg(msg="heads and %s <= 5!" % random_num_head.output).set_caching_options(False)

    with dsl.If(flip.output == "tails"):
        random_num_tail = random_num(low=10, high=19).set_caching_options(False)
        with dsl.If(random_num_tail.output > 15):
            print_msg(msg="tails and %s > 15!" % random_num_tail.output).set_caching_options(False)
        with dsl.If(random_num_tail.output <= 15):
            print_msg(msg="tails and %s <= 15!" % random_num_tail.output).set_caching_options(False)


if __name__ == "__main__":
    compiler.Compiler().compile(flipcoin_pipeline, package_path=__file__.replace(".py", ".yaml"))
