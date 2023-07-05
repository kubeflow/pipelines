import kfp.dsl as dsl


@dsl.pipeline(
  name='Prediction pipeline',
  description='Execute prediction operation for the dataset from numpy file and test accuracy and latency'
)
def openvino_predict(
        model_bin='gs://intelai_public_models/resnet_50_i8/1/resnet_50_i8.bin',
        model_xml='gs://intelai_public_models/resnet_50_i8/1/resnet_50_i8.xml',
        generated_model_dir='gs://your-bucket/folder',
        input_numpy_file='gs://intelai_public_models/images/imgs.npy',
        label_numpy_file='gs://intelai_public_models/images/lbs.npy',
        batch_size=1,
        scale_div=1,
        scale_sub=0
    ):

    """A one-step pipeline."""
    dsl.ContainerOp(
     name='openvino-predict',
     image='gcr.io/constant-cubist-173123/inference_server/ml_predict:5',
     command=['python3', 'predict.py'],
     arguments=[
         '--model_bin', model_bin,
         '--model_xml', model_xml,
         '--input_numpy_file', input_numpy_file,
         '--label_numpy_file', label_numpy_file,
         '--batch_size', batch_size,
         '--scale_div', scale_div,
         '--scale_sub', scale_sub,
         '--output_folder', generated_model_dir],
     file_outputs={})

if __name__ == '__main__':
    import kfp.compiler as compiler
    compiler.Compiler().compile(openvino_predict, __file__ + '.tar.gz')
