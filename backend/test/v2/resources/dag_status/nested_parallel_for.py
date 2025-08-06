import kfp
from kfp import dsl

@dsl.component()
def parent_setup() -> str:
    """Setup task in parent context."""
    print("Setting up parent pipeline for nested ParallelFor")
    return "parent_ready_for_parallel"

@dsl.component()
def parallel_worker(item: str, context: str) -> str:
    """Worker component for parallel execution."""
    print(f"Processing {item} in {context} context")
    return f"Processed {item}"

@dsl.component()
def nested_aggregator(context: str) -> str:
    """Aggregates results from nested parallel execution."""
    print(f"Aggregating results in {context} context")
    return f"Aggregated results for {context}"

@dsl.component()
def parent_finalize(setup_result: str, nested_result: str) -> str:
    """Final task in parent context."""
    print(f"Finalizing: {setup_result} + {nested_result}")
    return "nested_parallel_complete"

@dsl.pipeline(name="nested-parallel-for", description="Nested pipeline with ParallelFor to test hierarchical DAG status updates")
def nested_parallel_for_pipeline():
    """
    Pipeline with nested ParallelFor execution.
    
    This tests how DAG status updates work when ParallelFor loops
    are nested within conditional blocks or component groups.
    
    Structure:
    - Parent setup
    - Nested context containing:
      - ParallelFor loop (outer)
        - ParallelFor loop (inner) 
    - Parent finalize
    """
    # Parent context setup
    setup_task = parent_setup().set_caching_options(enable_caching=False)
    
    # Nested execution context
    with dsl.If(setup_task.output == "parent_ready_for_parallel"):
        # Outer ParallelFor loop
        with dsl.ParallelFor(items=['batch1', 'batch2', 'batch3']) as outer_item:
            # Inner ParallelFor loop within each outer iteration
            with dsl.ParallelFor(items=['task-a', 'task-b']) as inner_item:
                worker_task = parallel_worker(
                    item=inner_item, 
                    context=outer_item
                ).set_caching_options(enable_caching=False)
            
            # Aggregate results for this batch
            batch_aggregator = nested_aggregator(
                context=outer_item
            ).set_caching_options(enable_caching=False)
        
        # Final aggregation of all nested results
        final_aggregator = nested_aggregator(
            context="all_batches"
        ).set_caching_options(enable_caching=False)
        
        # Parent context finalization
        finalize_task = parent_finalize(
            setup_result=setup_task.output,
            nested_result=final_aggregator.output
        ).set_caching_options(enable_caching=False)

if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        nested_parallel_for_pipeline, 
        "nested_parallel_for.yaml"
    ) 