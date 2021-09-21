# This pipeline demonstrates and verifies all ways of data passing supported by KFP.

# %% [markdown]
# KFP has simple data passing model. There are three different kinds of arguments and tw different ways to consume an input argument. All combinations are supported.

# Any input can be consumed:
# * As value (`inputValue` placeholder)
# * As file (`inputPath` placeholder).

# Input argument can come from:
# * Constant value
# * Pipeline parameter
# * Upstream component output

# Combining these options there are 6 end-to-end data passing cases, each of which works regardless of type:

# 1. Pass constant value which gets consumed as value. (Common)
# 2. Pass constant value which gets consumed as file. (Often used in test pipelines)
# 3. Pass pipeline parameter that gets consumed as value. (Common)
# 4. Pass pipeline parameter that gets consumed as file. (Rare. Sometimes used with JSON parameters)
# 5. Pass task output that gets consumed as value. (Common)
# 6. Pass task output that gets consumed as file. (Common)

# The only restriction on types is that when both upstream output and downstream input have types, the types must match.

# %%
import kfp
from kfp.components import create_component_from_func, InputPath, OutputPath


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
def produce_something2() -> 'Something':
    return "produce_something2"


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
def consume_something_as_file(data_path: InputPath('Something')):
    with open(data_path) as f:
        print("consume_something_as_file: " + f.read())


@create_component_from_func
def consume_string_as_file(data_path: InputPath(str)):
    with open(data_path) as f:
        print("consume_string_as_file: " + f.read())


# Pipeline
@kfp.dsl.pipeline(name='data_passing_pipeline')
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
    kfp_endpoint = None
    kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(
        data_passing_pipeline, arguments={})
