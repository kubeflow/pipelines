"""Import an item into a Databricks Workspace."""
import kfp.dsl as dsl
import kfp.compiler as compiler
import databricks

def import_workspace_item(item_name):
    return databricks.ImportWorkspaceItemOp(
        name="importworkspaceitem",
        item_name=item_name,
        content="cHJpbnQoImhlbGxvLCB3b3JsZCIpCgoK",
        path="/Users/alejacma@microsoft.com/ScalaExampleNotebook",
        language="SCALA",
        file_format="SOURCE"
    )

def delete_workspace_item(item_name):
    return databricks.DeleteWorkspaceItemOp(
        name="deleteworkspaceitem",
        item_name=item_name
    )

@dsl.pipeline(
    name="DatabricksWorkspaceItem",
    description="A toy pipeline that imports some source code into a Databricks Workspace."
)
def calc_pipeline(item_name="test-item"):
    import_workspace_item_task = import_workspace_item(item_name)
    delete_workspace_item_task = delete_workspace_item(item_name)
    delete_workspace_item_task.after(import_workspace_item_task)

if __name__ == "__main__":
    compiler.Compiler().compile(calc_pipeline, __file__ + ".tar.gz")
