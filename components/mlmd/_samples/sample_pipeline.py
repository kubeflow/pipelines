kfp_endpoint = None

import kfp
from kfp import components


get_context_types_op   = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/8a7b5a7940690f431c87f9e506e32db1b62c5632/components/mlmd/Get_context_types/component.yaml')
get_execution_types_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/8a7b5a7940690f431c87f9e506e32db1b62c5632/components/mlmd/Get_execution_types/component.yaml')
get_artifact_types_op  = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/8a7b5a7940690f431c87f9e506e32db1b62c5632/components/mlmd/Get_artifact_types/component.yaml')

get_contexts_op        = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/8a7b5a7940690f431c87f9e506e32db1b62c5632/components/mlmd/Get_contexts/component.yaml')
get_executions_op      = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/8a7b5a7940690f431c87f9e506e32db1b62c5632/components/mlmd/Get_executions/component.yaml')
get_artifacts_op       = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/8a7b5a7940690f431c87f9e506e32db1b62c5632/components/mlmd/Get_artifacts/component.yaml')

get_context_by_type_and_name_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/8a7b5a7940690f431c87f9e506e32db1b62c5632/components/mlmd/Get_context_by_type_and_name/component.yaml')


def mlmd_pipeline():
    context_types = get_context_types_op().output
    execution_types = get_execution_types_op().output
    artifact_types = get_artifact_types_op().output

    contexts = get_contexts_op(
        # type_name="KfpRun",  # Optional
    ).output
    executions = get_executions_op(
        # type_name="components.Chicago Taxi Trips dataset@sha256=95f8395dcd533d3856860ef592024daebcd236dee784d2e70bdd470ee36ed3c8",  # Optional
        # context_id=1234,  # Optional
        # context_name="continuous-training-pipeline-252hz",  # Optional
    ).output
    artifacts = get_artifacts_op(
        # type_name="String",  # Optional
        # context_id=1234,  # Optional
        # context_name="continuous-training-pipeline-252hz",  # Optional
    ).output


if __name__ == '__main__':
    kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(mlmd_pipeline, arguments={})
