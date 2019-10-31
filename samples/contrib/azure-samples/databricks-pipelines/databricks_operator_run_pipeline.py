"""Submit a one-time Run with implicit cluster creation to Databricks."""
import kfp.dsl as dsl
import kfp.compiler as compiler

def submit_run(run_name, parameter):
    return dsl.ResourceOp(
        name="submitrun",
        k8s_resource={
            "apiVersion": "databricks.microsoft.com/v1alpha1",
            "kind": "Run",
            "metadata": {
                "name":run_name,
            },
            "spec":{
                "run_name": run_name,
                "new_cluster": {
                    "spark_version": "5.3.x-scala2.11",
                    "node_type_id": "Standard_D3_v2",
                    "num_workers": 2
                },
                "libraries": [{"jar": "dbfs:/docs/sparkpi.jar"}],
                "spark_jar_task": {
                    "main_class_name": "com.databricks.ComputeModels",
                    "parameters": [parameter]
                }
            },
        },
        action="create",
        success_condition="status.metadata.state.life_cycle_state in (TERMINATED, SKIPPED, INTERNAL_ERROR)",
        attribute_outputs={
            "name": "{.metadata.name}",
            "job_id": "{.status.metadata.job_id}",
            "number_in_job": "{.status.metadata.number_in_job}",
            "run_id": "{.status.metadata.run_id}",
            "run_name": "{.status.metadata.run_name}",
            "life_cycle_state": "{.status.metadata.state.life_cycle_state}",
            "result_state": "{.status.metadata.state.result_state}",
            "notebook_output_result": "{.status.notebook_output.result}",
            "notebook_output_truncated": "{.status.notebook_output.truncated}",
            "error": "{.status.error}"
        }
    )

def delete_run(run_name):
    return dsl.ResourceOp(
        name="deleterun",
        k8s_resource={
            "apiVersion": "databricks.microsoft.com/v1alpha1",
            "kind": "Run",
            "metadata": {
                "name": run_name
            }
        },
        action="delete"
    )

@dsl.pipeline(
    name="DatabricksRun",
    description="A toy pipeline that computes an approximation to pi with Azure Databricks."
)
def calc_pipeline(run_name="test-run", parameter="10"):
    submit_run_task = submit_run(run_name, parameter)
    delete_run_task = delete_run(run_name)
    delete_run_task.after(submit_run_task)

if __name__ == "__main__":
    compiler.Compiler().compile(calc_pipeline, __file__ + ".tar.gz")
