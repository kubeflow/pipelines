import kfp.dsl as dsl
import ai_pipeline_params as params

# generate default secret name
secret_name = 'kfp-creds'


# create pipeline
@dsl.pipeline(
  name='FfDL pipeline',
  description='A pipeline for machine learning workflow using Fabric for Deep Learning and Seldon.'
)
def ffdlPipeline(
    GITHUB_TOKEN=dsl.PipelineParam(name='github-token',
                                   value=''),
    CONFIG_FILE_URL=dsl.PipelineParam(name='config-file-url',
                                      value='https://raw.githubusercontent.com/user/repository/branch/creds.ini'),
    model_def_file_path=dsl.PipelineParam(name='model-def-file-path',
                                          value='gender-classification.zip'),
    manifest_file_path=dsl.PipelineParam(name='manifest-file-path',
                                         value='manifest.yml'),
    model_deployment_name=dsl.PipelineParam(name='model-deployment-name',
                                            value='gender-classifier'),
    model_class_name=dsl.PipelineParam(name='model-class-name',
                                       value='ThreeLayerCNN'),
    model_class_file=dsl.PipelineParam(name='model-class-file',
                                       value='gender_classification.py')
):
    """A pipeline for end to end machine learning workflow."""
    config_op = dsl.ContainerOp(
        name="config",
        image="aipipeline/wml-config",
        command=['python3'],
        arguments=['/app/config.py',
                   '--token', GITHUB_TOKEN,
                   '--url', CONFIG_FILE_URL,
                   '--name', secret_name],
        file_outputs={'secret-name': '/tmp/' + secret_name}
    )

    train = dsl.ContainerOp(
     name='train',
     image='aipipeline/ffdl-train:0.6',
     command=['sh', '-c'],
     arguments=['echo %s > /tmp/logs.txt; python -u train.py --model_def_file_path %s --manifest_file_path %s;'
                % (config_op.output, model_def_file_path, manifest_file_path)],
     file_outputs={'output': '/tmp/training_id.txt'}).apply(params.use_ai_pipeline_params(secret_name))

    serve = dsl.ContainerOp(
     name='serve',
     image='aipipeline/ffdl-serve:0.11',
     command=['sh', '-c'],
     arguments=['python -u serve.py --model_id %s --deployment_name %s --model_class_name %s --model_class_file %s;'
                % (train.output, model_deployment_name, model_class_name, model_class_file)],
     file_outputs={'output': '/tmp/deployment_result.txt'}).apply(params.use_ai_pipeline_params(secret_name))


if __name__ == '__main__':
    import kfp.compiler as compiler
    compiler.Compiler().compile(ffdlPipeline, __file__ + '.zip')
