from typing import List

import kfp
from kfp import dsl


@dsl.component()
def generate_items(count: int) -> List[str]:
    """Generate a list of items based on count."""
    items = [f"item-{i}" for i in range(count)]
    print(f"Generated {len(items)} items: {items}")
    return items


@dsl.component()
def process_item(item: str) -> str:
    """Process a single item."""
    print(f"Processing {item}")
    return f"Processed: {item}"


@dsl.pipeline(name="parallel-for-dynamic", description="Dynamic ParallelFor loop with runtime-determined iterations")
def parallel_for_dynamic_pipeline(iteration_count: int = 3):
    """
    Dynamic ParallelFor pipeline with runtime-determined iteration count.
    """
    # First generate the list of items dynamically
    items_task = generate_items(count=iteration_count)

    # Then process each item in parallel
    with dsl.ParallelFor(items=items_task.output) as item:
        process_task = process_item(item=item)
