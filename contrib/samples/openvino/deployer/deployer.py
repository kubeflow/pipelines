import kfp.dsl as dsl


@dsl.pipeline(
    name='OVMS deployment pipeline',
    description='Deploy OpenVINO Model Server instance in Kubernetes'
)
def openvino_predict(
        model_export_path = dsl.PipelineParam(name='model-export-path', value='gs://intelai_public_models/resnet_50_i8'),
        server_name = dsl.PipelineParam(name='server-name', value='resnet'),
        log_level = dsl.PipelineParam(name='log-level', value='DEBUG'),
        batch_size = dsl.PipelineParam(name='batch-size', value='auto'),
        model_version_policy = dsl.PipelineParam(name='model-version-policy', value='{"latest": { "num_versions":2 }}'),
        replicas = dsl.PipelineParam(name='replicas', value=1),
        images_list = dsl.PipelineParam(name='evaluation-images-list', value='https://raw.githubusercontent.com/IntelAI/OpenVINO-model-server/master/example_client/input_images.txt'),
        image_path_prefix = dsl.PipelineParam(name='image-path-prefix', value='https://github.com/IntelAI/OpenVINO-model-server/raw/master/example_client/'),
        model_input_name = dsl.PipelineParam(name='model-input-name', value='data'),
        model_output_name = dsl.PipelineParam(name='model-output-name', value='prob'),
        model_input_size = dsl.PipelineParam(name='model-input-size', value=224)):


    """A one-step pipeline."""
    deploy = dsl.ContainerOp(
        name='Deploy OpenVINO Model Server',
        image='gcr.io/constant-cubist-173123/inference_server/ml_deployer:13',
        command=['./deploy.sh'],
        arguments=[
            '--model-export-path', model_export_path,
            '--server-name', server_name,
            '--log-level', log_level,
            '--batch-size', batch_size,
            '--model-version-policy', model_version_policy,
            '--replicas', replicas],
        file_outputs={'output':'/tmp/server_endpoint'})

    dsl.ContainerOp(
        name='Evaluate OpenVINO Model Server',
        image='gcr.io/constant-cubist-173123/inference_server/ml_deployer:13',
        command=['./evaluate.py'],
        arguments=[
            '--images_list', images_list,
            '--grpc_endpoint', deploy.output,
            '--input_name', model_input_name,
            '--output_name', model_output_name,
            '--size', model_input_size,
            '--image_path_prefix', image_path_prefix],
        file_outputs={})

if __name__ == '__main__':
    import kfp.compiler as compiler
    compiler.Compiler().compile(openvino_predict, __file__ + '.tar.gz')