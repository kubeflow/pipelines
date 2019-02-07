# place storing the configuration secrets
GITHUB_TOKEN = '9db6d59731b5d32f794df2adeb9784bb402019b3'
CONFIG_FILE_URL = 'https://raw.githubusercontent.com/adrian555/kfp-secrets/master/creds.ini'

# generate default secret name
import os
secret_name = 'ai-pipeline-' + os.path.splitext(os.path.basename(CONFIG_FILE_URL))[0]

# create pipelines
import kfp.dsl as dsl
import ai_pipeline_params as params

@dsl.pipeline(
    name='KFP on WML training',
    description='Kubeflow pipelines leveraging WML to perform tensorflow image recognition.'
)
def kfp_wml_pipeline(
):
    # op1 - this operation will create the credentials as secrets to be used by other operations
    config_op = dsl.ContainerOp(
        name="config",
        image="aipipeline/wml-config",
        command=['python3'],
        arguments=['/app/config.py', 
                   '--token', GITHUB_TOKEN, 
                   '--url', CONFIG_FILE_URL],
        file_outputs={'secret-name' : '/tmp/'+secret_name}
    )
    
    # op2 - this operation trains the model with the model codes and data saved in the S3 buckets
    train_op = dsl.ContainerOp(
        name="train",
        image="aipipeline/wml-train",
        command=['python3'],
        arguments=['/app/wml-train.py',
                   '--config', config_op.output,
                   '--train-code', 'tf-model.zip',
                   '--execution-command', '\'python3 convolutional_network.py --trainImagesFile ${DATA_DIR}/train-images-idx3-ubyte.gz --trainLabelsFile ${DATA_DIR}/train-labels-idx1-ubyte.gz --testImagesFile ${DATA_DIR}/t10k-images-idx3-ubyte.gz --testLabelsFile ${DATA_DIR}/t10k-labels-idx1-ubyte.gz --learningRate 0.001 --trainingIters 20000\''],
        file_outputs={'run-uid' : '/tmp/run_uid'}).apply(params.use_ai_pipeline_params(secret_name))
    
    # op3 - this operation stores the model trained above
    store_op = dsl.ContainerOp(
        name="store",
        image="aipipeline/wml-store",
        command=['python3'],
        arguments=['/app/wml-store.py', 
                   '--run-uid', train_op.output,
                   '--model-name', 'python-tensorflow-mnist'],
        file_outputs={'model-uid' : '/tmp/model_uid'}).apply(params.use_ai_pipeline_params(secret_name))
    
    # op4 - this operation deploys the model to a web service and run scoring with the payload in S3
    deploy_op = dsl.ContainerOp(
        name="deploy",
        image="aipipeline/wml-deploy",
        command=['python3'],
        arguments=['/app/wml-deploy.py',
                   '--model-uid', store_op.output,
                   '--model-name', 'python-tensorflow-mnist',
                   '--scoring-payload', 'tf-mnist-test-payload.json'],
        file_outputs={'output' : '/tmp/output'}).apply(params.use_ai_pipeline_params(secret_name))

if __name__ == '__main__':
    # compile the pipeline
    import kfp.compiler as compiler
    pipeline_filename = kfp_wml_pipeline.__name__ + '.tar.gz'
    compiler.Compiler().compile(kfp_wml_pipeline, pipeline_filename)
