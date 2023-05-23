
import kfp.dsl as dsl


@dsl.pipeline(
  name='Model-Optimization',
  description='Convert model using OpenVINO model optimizer'
)
def download_optimize_and_upload(
        input_path: dsl.PipelineParam,
        output_path: dsl.PipelineParam,
        mo_options: dsl.PipelineParam):
    """A one-step pipeline."""

    dsl.ContainerOp(
        name='mo',
        image='gcr.io/constant-cubist-173123/inference_server/ml_mo:12',
        command=['convert_model.py'],
        arguments=[
         '--input_path', input_path,
         '--output_path', output_path,
         '--mo_options', mo_options],
        file_outputs={'output': '/tmp/output_path.txt'})


if __name__ == '__main__':
    import kfp.compiler as compiler
    compiler.Compiler().compile(download_optimize_and_upload, __file__ + '.tar.gz')
