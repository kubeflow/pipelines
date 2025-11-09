import functools

from kfp import dsl
from kfp.dsl import (
    Input,
    Output,
    Artifact,
    Dataset,
    component
)

base_image="quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0"
dsl.component = functools.partial(dsl.component, base_image=base_image)


@component
def create_dataset(
        condition_to_activate: str,
        output_dataset: Output[Dataset],
        condition_out: dsl.OutputPath(str)):
    with open(output_dataset.path, "w") as f:
        f.write(condition_to_activate)
    with open(condition_out, 'w') as f:
        f.write(condition_to_activate)

@component
def give_animal(want_animal: str, input_dataset: Input[Dataset], output_animal: Output[Artifact]):
    with open(input_dataset.path, "r") as f:
        data = f.read()
    assert data == "second"
    with open(output_animal.path, "w") as f:
        f.write(want_animal)

@component
def analyze_animal(animal_artifact: Input[Artifact], analysis_output: Output[Artifact]):
    with open(animal_artifact.path, "r") as f:
        data = f.read()
    assert data == "dog"
    with open(analysis_output.path, "w") as f:
        f.write(f'done_analyzing')

@component
def check_analysis(analysis_input: Input[Artifact]):
    with open(analysis_input.path, "r") as f:
        data = f.read()
    assert data == "done_analyzing"

@component
def check_animal(animal_input: Input[Artifact]):
    with open(animal_input.path, "r") as f:
        data = f.read()
    assert data == "dog"

@dsl.pipeline
def secondary_pipeline() -> Artifact:
    t0 = create_dataset(condition_to_activate="second")

    with dsl.If('first' == t0.outputs['condition_out']):
        give_cat = give_animal(want_animal="cat", input_dataset=t0.outputs['output_dataset'])
    with dsl.Elif('second' == t0.outputs['condition_out']):
        give_dog = give_animal(want_animal="dog", input_dataset=t0.outputs['output_dataset'])
        analyze_dog = analyze_animal(animal_artifact=give_dog.outputs["output_animal"])
    with dsl.Else():
        give_mouse = give_animal(want_animal="mouse", input_dataset=t0.outputs['output_dataset'])
        analyze_mouse = analyze_animal(animal_artifact=give_mouse.outputs["output_animal"])

    one_analysis = dsl.OneOf(
        analyze_dog.outputs['analysis_output'],
        analyze_mouse.outputs['analysis_output'],
    )

    return dsl.OneOf(
        give_cat.outputs['output_animal'],
        give_dog.outputs['output_animal'],
        give_mouse.outputs['output_animal'])


@dsl.pipeline
def primary_pipeline():
    secondary_pipeline_task = secondary_pipeline()
    # This actually fails today the rest pass
    check_animal(animal_input=secondary_pipeline_task.output)

if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=primary_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
