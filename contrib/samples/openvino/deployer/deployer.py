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
        replicas = dsl.PipelineParam(name='replicas', value=1)):

    """A one-step pipeline."""
    dsl.ContainerOp(
        name='openvino-model-server-deployer',
        image='gcr.io/constant-cubist-173123/inference_server/ml_deployer:9',
        command=['./deploy.sh'],
        arguments=[
            '--model-export-path', model_export_path,
            '--server-name', server_name,
            '--log-level', log_level,
            '--batch-size', batch_size,
            '--model-version-policy', model_version_policy,
            '--replicas', replicas],
        file_outputs={'output':'/tmp/server_endpoint'})

if __name__ == '__main__':
    import kfp.compiler as compiler
    compiler.Compiler().compile(openvino_predict, __file__ + '.tar.gz')