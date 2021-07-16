from kfp.components import create_component_from_func, InputPath, OutputPath


def convert_to_tensorflow_saved_model_from_onnx_model(
    model_path: InputPath('OnnxModel'),
    converted_model_path: OutputPath('TensorflowSavedModel'),
):
    import onnx
    import onnx_tf

    onnx_model = onnx.load(model_path)
    tf_rep = onnx_tf.backend.prepare(onnx_model)
    tf_rep.export_graph(converted_model_path)

    
if __name__ == '__main__':
    convert_to_tensorflow_saved_model_from_onnx_model_op = create_component_from_func(
        convert_to_tensorflow_saved_model_from_onnx_model,
        output_component_file='component.yaml',
        base_image='tensorflow/tensorflow:2.4.1',
        packages_to_install=['onnx-tf==1.7.0', 'onnx==1.8.0'],  # onnx-tf==1.7.0 is not compatible with onnx==1.8.1
        annotations={
            "author": "Alexey Volkov <alexey.volkov@ark-kun.com>",
            "canonical_location": "https://raw.githubusercontent.com/Ark-kun/pipeline_components/master/components/_converters/OnnxModel/to_TensorflowSavedModel/component.yaml",
        },
    )
