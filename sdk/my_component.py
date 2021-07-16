from kfp.components.structures import *

my_component_spec = ComponentSpec(
    inputs=[
        InputSpec(name="Input1", type="String", default="Hello"),
        InputSpec(name="Input2", type="CSV"),
    ],
    outputs=[
        OutputSpec(name="Output1", type="CSV"),
    ],
    implementation=ContainerImplementation(container=ContainerSpec(
        image="busybox",
        command=[
            "my_program.sh",
            "--input-1", InputValuePlaceholder(input_name="Input1"),
            "--input-2-path", InputPathPlaceholder(input_name="Input2"),
            "--output-1-path", OutputPathPlaceholder(output_name="Output1"),
        ],
    )),
)