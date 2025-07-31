import kfp
from kfp import dsl

@dsl.component()
def parent_setup() -> str:
    """Setup task in parent context."""
    print("Setting up parent pipeline")
    return "parent_setup_complete"

@dsl.component()
def child_setup() -> str:
    """Setup task in child pipeline."""
    print("Child pipeline setup")
    return "child_setup_complete"

@dsl.component()
def child_worker(input_data: str) -> str:
    """Worker task in child pipeline."""
    print(f"Child worker processing: {input_data}")
    return f"child_processed_{input_data}"

@dsl.component()
def child_finalizer(setup_result: str, worker_result: str) -> str:
    """Finalizer task in child pipeline."""
    print(f"Child finalizer: {setup_result} + {worker_result}")
    return "child_pipeline_complete"

@dsl.pipeline()
def child_pipeline(input_value: str = "default_input") -> str:
    """
    Child pipeline that will be converted to a component.
    
    This creates an actual nested DAG execution.
    """
    # Child pipeline execution flow
    setup_task = child_setup().set_caching_options(enable_caching=False)
    
    worker_task = child_worker(input_data=input_value).set_caching_options(enable_caching=False)
    worker_task.after(setup_task)
    
    finalizer_task = child_finalizer(
        setup_result=setup_task.output,
        worker_result=worker_task.output
    ).set_caching_options(enable_caching=False)
    
    return finalizer_task.output

@dsl.component()
def parent_finalize(parent_input: str, child_input: str) -> str:
    """Finalization task in parent context."""
    print(f"Finalizing parent with inputs: {parent_input}, {child_input}")
    return "parent_finalize_complete"

@dsl.pipeline(name="nested-simple", description="Real nested pipeline: parent calls child pipeline to test hierarchical DAG status updates")
def nested_simple_pipeline():
    """
    Parent pipeline that calls a real child pipeline.
    
    This creates true nested DAG execution where:
    - Parent DAG manages the overall flow
    - Child DAG handles sub-workflow execution
    
    This tests the issue where DAG status updates don't properly
    traverse the parent → child DAG hierarchy.
    """
    # Parent context setup
    setup_task = parent_setup().set_caching_options(enable_caching=False)
    
    # Call child pipeline as a component - this creates REAL nesting!
    # In KFP v2, you can directly call a pipeline as a component
    child_pipeline_task = child_pipeline(
        input_value="data_from_parent"
    ).set_caching_options(enable_caching=False)
    child_pipeline_task.after(setup_task)
    
    # Parent context finalization using child results
    finalize_task = parent_finalize(
        parent_input=setup_task.output,
        child_input=child_pipeline_task.output
    ).set_caching_options(enable_caching=False)

if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        nested_simple_pipeline, 
        "nested_simple.yaml"
    ) 