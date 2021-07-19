# Copyright 2021 The Kubeflow Authors
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
"""Sample pipeline for legacy data passing."""

import kfp
from kfp.components import create_component_from_func, InputPath, OutputPath
from kfp.v2 import compiler


# Components
# Produce
@create_component_from_func
def produce_anything(data_path: OutputPath()):
  with open(data_path, "w") as f:
    f.write("produce_anything")


@create_component_from_func
def produce_something(data_path: OutputPath("Something")):
  with open(data_path, "w") as f:
    f.write("produce_something")


@create_component_from_func
def produce_string() -> str:
  return "produce_string"


# Consume as value
@create_component_from_func
def consume_anything_as_value(data):
  print("consume_anything_as_value: " + data)


@create_component_from_func
def consume_something_as_value(data: "Something"):
  print("consume_something_as_value: " + data)


@create_component_from_func
def consume_string_as_value(data: str):
  print("consume_string_as_value: " + data)


# Consume as file
@create_component_from_func
def consume_anything_as_file(data_path: InputPath()):
  with open(data_path) as f:
    print("consume_anything_as_file: " + f.read())


@create_component_from_func
def consume_something_as_file(data_path: InputPath("Something")):
  with open(data_path) as f:
    print("consume_something_as_file: " + f.read())


@create_component_from_func
def consume_string_as_file(data_path: InputPath(str)):
  with open(data_path) as f:
    print("consume_string_as_file: " + f.read())


# Pipeline
@kfp.dsl.pipeline(name="legacy-data-passing-pipeline")
def data_passing_pipeline(
    anything_param="anything_param",
    something_param: "Something" = "something_param",
    string_param: str = "string_param",
):
  produced_anything = produce_anything().output
  produced_something = produce_something().output
  produced_string = produce_string().output

  # Pass constant value; consume as value
  consume_anything_as_value("constant")
  consume_something_as_value("constant")
  consume_string_as_value("constant")

  # Pass constant value; consume as file
  consume_anything_as_file("constant")
  consume_something_as_file("constant")
  consume_string_as_file("constant")

  # Pass pipeline parameter; consume as value
  consume_anything_as_value(anything_param)
  consume_anything_as_value(something_param)
  consume_anything_as_value(string_param)
  consume_something_as_value(anything_param)
  consume_something_as_value(something_param)
  consume_string_as_value(anything_param)
  consume_string_as_value(string_param)

  # Pass pipeline parameter; consume as file
  consume_anything_as_file(anything_param)
  consume_anything_as_file(something_param)
  consume_anything_as_file(string_param)
  consume_something_as_file(anything_param)
  consume_something_as_file(something_param)
  consume_string_as_file(anything_param)
  consume_string_as_file(string_param)

  # Pass task output; consume as value
  consume_anything_as_value(produced_anything)
  consume_anything_as_value(produced_something)
  consume_anything_as_value(produced_string)
  consume_something_as_value(produced_anything)
  consume_something_as_value(produced_something)
  consume_string_as_value(produced_anything)
  consume_string_as_value(produced_string)

  # Pass task output; consume as file
  consume_anything_as_file(produced_anything)
  consume_anything_as_file(produced_something)
  consume_anything_as_file(produced_string)
  consume_something_as_file(produced_anything)
  consume_something_as_file(produced_something)
  consume_string_as_file(produced_anything)
  consume_string_as_file(produced_string)


if __name__ == "__main__":
  compiler.Compiler().compile(
      pipeline_func=data_passing_pipeline,
      package_path=__file__.replace(".py", ".json"))
