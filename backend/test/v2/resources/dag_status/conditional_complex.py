import kfp
from kfp import dsl


@dsl.component()
def get_value(input_value: int) -> int:
    """Component that returns the input value to test different conditions."""
    print(f"Received input value: {input_value}")
    return input_value


@dsl.component()
def execute_if_task(message: str) -> str:
    """Component that executes when If condition is True (value == 1)."""
    print(f"If branch executed: {message}")
    return f"If result: {message}"


@dsl.component()
def execute_elif_task(message: str) -> str:
    """Component that executes when Elif condition is True (value == 2)."""
    print(f"Elif branch executed: {message}")
    return f"Elif result: {message}"


@dsl.component()
def execute_else_task(message: str) -> str:
    """Component that executes when all conditions are False (value != 1,2)."""
    print(f"Else branch executed: {message}")
    return f"Else result: {message}"


@dsl.pipeline(name="conditional-complex", description="Complex If/Elif/Else condition to test DAG status updates")
def conditional_complex_pipeline(test_value: int = 2):
    """
    Complex conditional pipeline with If/Elif/Else statements.
    
    This tests the issue where total_dag_tasks counts ALL branches (If + Elif + Else)
    instead of just the executed branch.
    
    Expected execution path:
    - test_value=1 → If branch
    - test_value=2 → Elif branch  
    - test_value=other → Else branch
    """
    # Get a value to test conditions against
    value_task = get_value(input_value=test_value).set_caching_options(enable_caching=False)
    
    # Multiple conditional branches - only ONE should execute
    with dsl.If(value_task.output == 1):
        if_task = execute_if_task(message="value was 1").set_caching_options(enable_caching=False)
    
    with dsl.Elif(value_task.output == 2):
        elif_task = execute_elif_task(message="value was 2").set_caching_options(enable_caching=False)
    
    with dsl.Else():
        else_task = execute_else_task(message="value was something else").set_caching_options(enable_caching=False)


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        conditional_complex_pipeline, 
        "conditional_complex.yaml"
    ) 