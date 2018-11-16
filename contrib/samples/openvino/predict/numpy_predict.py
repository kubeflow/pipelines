import kfp.dsl as dsl


@dsl.pipeline(
  name='Prediction pipeline',
  description='Execute prediction operation for the dataset from numpy file'
)
def openvino_predict(
        model_xml: dsl.PipelineParam,
        model_bin: dsl.PipelineParam,
        input_numpy_file: dsl.PipelineParam,
        output_folder: dsl.PipelineParam):
    """A one-step pipeline."""
    dsl.ContainerOp(
     name='openvino-predict',
     image='<image name>',
     command=['python3', 'predict.py'],
     arguments=[
         '--model_bin', model_bin,
         '--model_xml', model_xml,
         '--input_numpy_file', input_numpy_file,
         '--output_folder', output_folder],
     file_outputs={})


if __name__ == '__main__':
    import kfp.compiler as compiler
    compiler.Compiler().compile(openvino_predict, __file__ + '.tar.gz')
