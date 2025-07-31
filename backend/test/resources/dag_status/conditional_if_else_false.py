import kfp
from kfp import dsl


@dsl.component()
def check_condition() -> bool:
    """Component that returns False to trigger the Else branch."""
    print("Checking condition: always returns False")
    return False


@dsl.component()
def execute_if_task(message: str) -> str:
    """Component that should NOT execute when If condition is False."""
    print(f"If branch executed: {message}")
    return f"If result: {message}"


@dsl.component()
def execute_else_task(message: str) -> str:
    """Component that executes when If condition is False."""
    print(f"Else branch executed: {message}")
    return f"Else result: {message}"


@dsl.pipeline(name="conditional-if-else-false", description="If/Else condition where If is False to test DAG status updates")
def conditional_if_else_false_pipeline():
    """
    If/Else conditional pipeline where If condition evaluates to False.
    
    This tests the issue where total_dag_tasks counts both If AND Else branches
    instead of just the executed Else branch.
    """
    # Check condition (always False)
    condition_task = check_condition().set_caching_options(enable_caching=False)
    
    # If condition is False, execute else_task (if_task should NOT execute)
    with dsl.If(condition_task.output == True):
        if_task = execute_if_task(message="if should not execute").set_caching_options(enable_caching=False)
    with dsl.Else():
        else_task = execute_else_task(message="else branch executed").set_caching_options(enable_caching=False)


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        conditional_if_else_false_pipeline, 
        "conditional_if_else_false.yaml"
    ) 