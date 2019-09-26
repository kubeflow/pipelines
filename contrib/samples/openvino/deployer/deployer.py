import kfp.dsl as dsl


@dsl.pipeline(
    name='OVMS deployment pipeline',
    description='Deploy OpenVINO Model Server instance in Kubernetes'
)
def openvino_predict(
        model_export_path='gs://intelai_public_models/resnet_50_i8',
        server_name='resnet',
        log_level='DEBUG',
        batch_size='auto',
        model_version_policy='{"latest": { "num_versions":2 }}',
        replicas=1,
        images_list='https://raw.githubusercontent.com/IntelAI/OpenVINO-model-server/master/example_client/input_images.txt',
        image_path_prefix='https://github.com/IntelAI/OpenVINO-model-server/raw/master/example_client/',
        model_input_name='data',
        model_output_name='prob',
        model_input_size=224
    ):


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