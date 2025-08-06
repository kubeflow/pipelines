import kfp
from kfp import dsl


@dsl.component()
def check_condition() -> bool:
    """Component that returns True to trigger the If branch."""
    print("Checking condition: always returns True")
    return True


@dsl.component()
def execute_if_task(message: str) -> str:
    """Component that executes when If condition is True."""
    print(f"If branch executed: {message}")
    return f"If result: {message}"


@dsl.pipeline(name="conditional-if-true", description="Simple If condition that is True to test DAG status updates")
def conditional_if_true_pipeline():
    """
    Simple conditional pipeline with If statement that evaluates to True.
    
    This tests the issue where total_dag_tasks counts all branches
    instead of just the executed one.
    """
    # Check condition (always True)
    condition_task = check_condition().set_caching_options(enable_caching=False)
    
    # If condition is True, execute this task
    with dsl.If(condition_task.output == True):
        if_task = execute_if_task(message="condition was true").set_caching_options(enable_caching=False)


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        conditional_if_true_pipeline, 
        "conditional_if_true.yaml"
    ) 