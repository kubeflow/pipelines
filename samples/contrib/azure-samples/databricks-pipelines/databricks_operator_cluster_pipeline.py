"""Create a cluster in Databricks. Then submit a one-time Run to that cluster."""
import kfp.dsl as dsl
import kfp.compiler as compiler

def create_cluster(cluster_name):
    return dsl.ResourceOp(
        name="createcluster",
        k8s_resource={
            "apiVersion": "databricks.microsoft.com/v1alpha1",
            "kind": "Dcluster",
            "metadata": {
                "name":cluster_name,
            },
            "spec":{
                "cluster_name": cluster_name,
                "spark_version": "5.3.x-scala2.11",
                "node_type_id": "Standard_D3_v2",
                "spark_conf": {
                    "spark.speculation": "true"
                },
                "num_workers": 2
            }
        },
        action="create",
        success_condition="status.cluster_info.state in (RUNNING, TERMINATED, UNKNOWN)",
        attribute_outputs={
            "name": "{.metadata.name}",
            "cluster_id": "{.status.cluster_info.cluster_id}",
            "cluster_name": "{.status.cluster_info.cluster_name}",
            "state": "{.status.cluster_info.state}"
        }
    )

def submit_run(run_name, cluster_id, parameter):
    return dsl.ResourceOp(
        name="submitrun",
        k8s_resource={
            "apiVersion": "databricks.microsoft.com/v1alpha1",
            "kind": "Run",
            "metadata": {
                "name": run_name,
            },
            "spec":{
                "run_name": run_name,
                "existing_cluster_id": cluster_id,
                "libraries": [{"jar": "dbfs:/my-jar.jar"}],
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
        action="delete",
    )

def delete_cluster(cluster_name):
    return dsl.ResourceOp(
        name="deletecluster",
        k8s_resource={
            "apiVersion": "databricks.microsoft.com/v1alpha1",
            "kind": "Dcluster",
            "metadata": {
                "name": cluster_name
            }
        },
        action="delete"
    )

@dsl.pipeline(
    name="DatabricksCluster",
    description="A toy pipeline that computes an approximation to pi with Azure Databricks."
)
def calc_pipeline(cluster_name="test-cluster", run_name="test-run", parameter="10"):
    create_cluster_task = create_cluster(cluster_name)
    submit_run_task = submit_run(run_name, create_cluster_task.outputs["cluster_id"], parameter)
    delete_run_task = delete_run(run_name)
    delete_run_task.after(submit_run_task)
    delete_cluster_task = delete_cluster(cluster_name)
    delete_cluster_task.after(delete_run_task)

if __name__ == "__main__":
    compiler.Compiler().compile(calc_pipeline, __file__ + ".tar.gz")
