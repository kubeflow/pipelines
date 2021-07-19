kfp_endpoint = None

import kfp
from kfp import components

download_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/240543e483076ae718f82c6f280441daa2f041fd/components/web/Download/component.yaml')
run_notebook_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/4ebce5f643b6af5639053ea7eaed52b02bf7e928/components/notebooks/Run_notebook_using_papermill/component.yaml')

def notebook_pipeline():
    notebook = download_op('https://raw.githubusercontent.com/kubeflow/pipelines/93fc34474bf989998cf19445149aca2847eee763/components/notebooks/samples/test_notebook.ipynb').output    
    
    run_notebook_op(
        notebook=notebook,
        parameters={'param1': 'value 1'},
        input_data="Optional. Pass output of any component here. Can be a directory.",
        packages_to_install=["matplotlib"],
    )

if __name__ == '__main__':
    pipelin_run = kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(notebook_pipeline, arguments={})
