import kfp.dsl as dsl


@dsl.pipeline(
  name='Prediction pipeline',
  description='Generate slim models and optimize them with OpenVINO'
)
def tf_slim_optimize(
        model_name: dsl.PipelineParam,
        num_classes: dsl.PipelineParam,
        checkpoint_url: dsl.PipelineParam,
        batch_size: dsl.PipelineParam,
        export_dir: dsl.PipelineParam,
        saved_model_dir: dsl.PipelineParam,
        mo_options: dsl.PipelineParam):

    slim = dsl.ContainerOp(
     name='tf-slim',
     image='<tf-slim component image>',
     command=['python', 'slim_model.py'],
     arguments=[
         '--model_name', model_name,
         '--batch_size', batch_size,
         '--checkpoint_url', checkpoint_url,
         '--num_classes', num_classes,
         '--saved_model_dir', saved_model_dir,
         '--export_dir', export_dir],
     file_outputs={'saved-model-dir': '/tmp/saved_model_dir.txt'})

    dsl.ContainerOp(
     name='tf-slim',
     image='<model optimizer component image>',
     command=['convert_model.py'],
     arguments=[
        '--input_path', '%s/saved_model.pb' % slim.output,
        '--mo_options', mo_options,
        '--output_path', slim.output],
     file_outputs={})


if __name__ == '__main__':
    import kfp.compiler as compiler
    compiler.Compiler().compile(tf_slim_optimize, __file__ + '.tar.gz')
