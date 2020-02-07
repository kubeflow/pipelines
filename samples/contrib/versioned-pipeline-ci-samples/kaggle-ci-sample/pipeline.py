import kfp.dsl as dsl
import kfp.components as components
from kfp.gcp import use_gcp_secret

@dsl.pipeline(
    name = "kaggle pipeline",
    description = "kaggle pipeline that go from download data, train model to display result"
)
def kaggle_houseprice(
    bucket_name: str,
    commit_sha: str
):
    import os

    downloadDataOp = components.load_component_from_file('./download_dataset/component.yaml')
    downloadDataStep = downloadDataOp(bucket_name=bucket_name).apply(use_gcp_secret('user-gcp-sa'))

    stepVisualizeTable = dsl.ContainerOp(
        name = 'visualize dataset in table',
        image = os.path.join(args.gcr_address, 'kaggle_visualize_table:latest'),
        command = ['python', 'visualize.py'],
        arguments = ['--train_file_path', '%s' % downloadDataStep.outputs['train_dataset']],
        output_artifact_paths={'mlpipeline-ui-metadata': '/mlpipeline-ui-metadata.json'}
    ).apply(use_gcp_secret('user-gcp-sa'))

    stepVisualizeHTML = dsl.ContainerOp(
        name = 'visualize dataset in html',
        image = os.path.join(args.gcr_address, 'kaggle_visualize_html:latest'),
        command = ['python', 'visualize.py'],
        arguments = ['--train_file_path', '%s' % downloadDataStep.outputs['train_dataset'],
                     '--commit_sha', commit_sha,
                     '--bucket_name', bucket_name],
        output_artifact_paths={'mlpipeline-ui-metadata': '/mlpipeline-ui-metadata.json'}
    ).apply(use_gcp_secret('user-gcp-sa'))

    stepTrainModel = dsl.ContainerOp(
        name = 'train model',
        image = os.path.join(args.gcr_address, 'kaggle_train:latest'),
        command = ['python', 'train.py'],
        arguments = ['--train_file',  '%s' % downloadDataStep.outputs['train_dataset'], 
                     '--test_file', '%s' % downloadDataStep.outputs['test_dataset'],
                     '--output_bucket', bucket_name
                     ],
        file_outputs = {'result': '/result_path.txt'}
    ).apply(use_gcp_secret('user-gcp-sa'))

    stepSubmitResult = dsl.ContainerOp(
        name = 'submit result to kaggle competition',
        image = os.path.join(args.gcr_address, 'kaggle_submit:latest'),
        command = ['python', 'submit_result.py'],
        arguments = ['--result_file', '%s' % stepTrainModel.outputs['result'],
                     '--submit_message', 'submit']
    ).apply(use_gcp_secret('user-gcp-sa'))

if __name__ == '__main__':
    import kfp.compiler as compiler
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument('--gcr_address', type = str)
    args = parser.parse_args()
    compiler.Compiler().compile(kaggle_houseprice, __file__ + '.zip')