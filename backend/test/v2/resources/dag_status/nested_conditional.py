import kfp
from kfp import dsl

@dsl.component()
def parent_setup(mode: str) -> str:
    """Setup task that determines execution mode."""
    print(f"Setting up parent pipeline in {mode} mode")
    return mode

@dsl.component()
def get_condition_value(mode: str) -> int:
    """Returns a value based on mode for conditional testing."""
    if mode == "development":
        value = 1
    elif mode == "staging":
        value = 2
    else:  # production
        value = 3
    print(f"Condition value for {mode}: {value}")
    return value

@dsl.component()
def development_task() -> str:
    """Task executed in development mode."""
    print("Executing development-specific task")
    return "dev_task_complete"

@dsl.component()
def staging_task() -> str:
    """Task executed in staging mode."""
    print("Executing staging-specific task")
    return "staging_task_complete"

@dsl.component()
def production_task() -> str:
    """Task executed in production mode."""
    print("Executing production-specific task")
    return "prod_task_complete"

@dsl.component()
def nested_conditional_task(branch_result: str) -> str:
    """Task that runs within nested conditional context."""
    print(f"Running nested task with: {branch_result}")
    return f"nested_processed_{branch_result}"

@dsl.component()
def parent_finalize(setup_result: str, nested_result: str) -> str:
    """Final task in parent context."""
    print(f"Finalizing: {setup_result} + {nested_result}")
    return "nested_conditional_complete"

@dsl.pipeline(name="nested-conditional", description="Nested pipeline with complex conditionals to test hierarchical DAG status updates")
def nested_conditional_pipeline(execution_mode: str = "development"):
    """
    Pipeline with nested conditional execution.
    
    This tests how DAG status updates work when conditional logic
    is nested within other conditional blocks or component groups.
    
    Structure:
    - Parent setup (determines mode)
    - Outer conditional based on setup result
      - Inner conditionals (If/Elif/Else) based on mode value
        - Nested tasks within each branch
    - Parent finalize
    """
    # Parent context setup
    setup_task = parent_setup(mode=execution_mode).set_caching_options(enable_caching=False)
    
    # Outer conditional context
    with dsl.If(setup_task.output != ""):
        # Get value for nested conditionals
        condition_value = get_condition_value(mode=setup_task.output).set_caching_options(enable_caching=False)
        
        # Nested conditional structure (If/Elif/Else)
        with dsl.If(condition_value.output == 1):
            dev_task = development_task().set_caching_options(enable_caching=False)
            # Nested task within development branch
            nested_dev = nested_conditional_task(branch_result=dev_task.output).set_caching_options(enable_caching=False)
            branch_result = nested_dev.output
            
        with dsl.Elif(condition_value.output == 2):
            staging_task_instance = staging_task().set_caching_options(enable_caching=False)
            # Nested task within staging branch
            nested_staging = nested_conditional_task(branch_result=staging_task_instance.output).set_caching_options(enable_caching=False)
            branch_result = nested_staging.output
            
        with dsl.Else():
            prod_task = production_task().set_caching_options(enable_caching=False)
            # Nested task within production branch
            nested_prod = nested_conditional_task(branch_result=prod_task.output).set_caching_options(enable_caching=False)
            branch_result = nested_prod.output
        
        # Parent context finalization
        finalize_task = parent_finalize(
            setup_result=setup_task.output,
            nested_result="nested_branch_complete"  # Placeholder since branch_result scope is limited
        ).set_caching_options(enable_caching=False)

if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        nested_conditional_pipeline, 
        "nested_conditional.yaml"
    ) 