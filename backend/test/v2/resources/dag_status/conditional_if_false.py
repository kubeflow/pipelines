import kfp
from kfp import dsl


@dsl.component()
def check_condition() -> bool:
    """Component that returns False to skip the If branch."""
    print("Checking condition: always returns False")
    return False


@dsl.component()
def execute_if_task(message: str) -> str:
    """Component that should NOT execute when If condition is False."""
    print(f"If branch executed: {message}")
    return f"If result: {message}"


@dsl.pipeline(name="conditional-if-false", description="Simple If condition that is False to test DAG status updates")
def conditional_if_false_pipeline():
    """
    Simple conditional pipeline with If statement that evaluates to False.
    
    """
    condition_task = check_condition().set_caching_options(enable_caching=False)
    
    # If condition is False, this task should NOT execute
    with dsl.If(condition_task.output == True):
        if_task = execute_if_task(message="this should not execute").set_caching_options(enable_caching=False)


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        conditional_if_false_pipeline, 
        "conditional_if_false.yaml"
    ) 