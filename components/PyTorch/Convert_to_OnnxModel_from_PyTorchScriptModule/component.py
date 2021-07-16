from kfp.components import create_component_from_func, InputPath, OutputPath


def convert_to_onnx_from_pytorch_script_module(
    model_path: InputPath('PyTorchScriptModule'),
    converted_model_path: OutputPath('OnnxModel'),
    list_of_input_shapes: list,
):
    '''Creates fully-connected network in PyTorch ScriptModule format'''
    import torch
    model = torch.jit.load(model_path)
    example_inputs = [
        torch.ones(*input_shape)
        for input_shape in list_of_input_shapes
    ]
    example_outputs = model.forward(*example_inputs)
    torch.onnx.export(
        model=model,
        args=example_inputs,
        f=converted_model_path,
        verbose=True,
        training=torch.onnx.TrainingMode.EVAL,
        example_outputs=example_outputs,
    )

    
if __name__ == '__main__':
    convert_to_onnx_from_pytorch_script_module_op = create_component_from_func(
        convert_to_onnx_from_pytorch_script_module,
        output_component_file='component.yaml',
        base_image='pytorch/pytorch:1.7.1-cuda11.0-cudnn8-runtime',
        packages_to_install=[],
        annotations={
            "author": "Alexey Volkov <alexey.volkov@ark-kun.com>",
            "canonical_location": "https://raw.githubusercontent.com/Ark-kun/pipeline_components/master/components/PyTorch/Convert_to_OnnxModel_from_PyTorchScriptModule/component.yaml",
        },
    )