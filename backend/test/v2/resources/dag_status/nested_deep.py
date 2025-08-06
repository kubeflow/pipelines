import kfp
from kfp import dsl

"""
DEEP NESTED PIPELINE - 5-Level Hierarchy Test
==============================================

This pipeline creates the most complex nested structure possible to test 
DAG status updates across deep hierarchical contexts with mixed constructs.

PIPELINE HIERARCHY STRUCTURE:

Level 1: ROOT PIPELINE ──────────────────────────────────────┐
│                                                            │
├─ [setup] ──────────────────────────────────────────────┐   │
│                                                        │   │
├─ Level 2: CONDITIONAL CONTEXT                          │   │
│  │                                                     │   │
│  ├─ [controller] ──────────────────────────────────┐   │   │
│  │                                                 │   │   │
│  ├─ IF(level2_ready):                              │   │   │
│  │  │                                              │   │   │
│  │  ├─ Level 3: BATCH PARALLEL FOR                 │   │   │
│  │  │  │                                           │   │   │
│  │  │  ├─ FOR batch_a:                             │   │   │
│  │  │  │  │                                        │   │   │
│  │  │  │  ├─ Level 4: TASK PARALLEL FOR            │   │   │
│  │  │  │  │  │                                     │   │   │
│  │  │  │  │  ├─ FOR task_1:                        │   │   │
│  │  │  │  │  │  ├─ [worker(batch_a, task_1)]       │   │   │
│  │  │  │  │  │  ├─ Level 5: [get_condition()]      │   │   │
│  │  │  │  │  │  └─ Level 5: [processor_A]          │   │   │
│  │  │  │  │  │                                     │   │   │
│  │  │  │  │  ├─ FOR task_2:                        │   │   │
│  │  │  │  │  │  ├─ [worker(batch_a, task_2)]       │   │   │
│  │  │  │  │  │  ├─ Level 5: [get_condition()]      │   │   │
│  │  │  │  │  │  └─ Level 5: [processor_A]          │   │   │
│  │  │  │  │  │                                     │   │   │
│  │  │  │  │  └─ FOR task_3:                        │   │   │
│  │  │  │  │     ├─ [worker(batch_a, task_3)]       │   │   │
│  │  │  │  │     ├─ Level 5: [get_condition()]      │   │   │
│  │  │  │  │     └─ Level 5: [processor_A]          │   │   │
│  │  │  │  │                                        │   │   │
│  │  │  │  └─ [aggregator(batch_a)]                 │   │   │
│  │  │  │                                           │   │   │
│  │  │  └─ FOR batch_b:                             │   │   │
│  │  │     │                                        │   │   │
│  │  │     ├─ Level 4: TASK PARALLEL FOR            │   │   │
│  │  │     │  │                                     │   │   │
│  │  │     │  ├─ FOR task_1:                        │   │   │
│  │  │     │  │  ├─ [worker(batch_b, task_1)]       │   │   │
│  │  │     │  │  ├─ Level 5: [get_condition()]      │   │   │
│  │  │     │  │  └─ Level 5: [processor_A]          │   │   │
│  │  │     │  │                                     │   │   │
│  │  │     │  ├─ FOR task_2:                        │   │   │
│  │  │     │  │  ├─ [worker(batch_b, task_2)]       │   │   │
│  │  │     │  │  ├─ Level 5: [get_condition()]      │   │   │
│  │  │     │  │  └─ Level 5: [processor_A]          │   │   │
│  │  │     │  │                                     │   │   │
│  │  │     │  └─ FOR task_3:                        │   │   │
│  │  │     │     ├─ [worker(batch_b, task_3)]       │   │   │
│  │  │     │     ├─ Level 5: [get_condition()]      │   │   │
│  │  │     │     └─ Level 5: [processor_A]          │   │   │
│  │  │     │                                        │   │   │
│  │  │     └─ [aggregator(batch_b)]                 │   │   │
│  │  │                                              │   │   │
│  │  └─ [level2_finalizer] ─────────────────────────┘   │   │
│  │                                                     │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                            │ 
└─ [level1_finalizer] ───────────────────────────────────────┘

EXECUTION MATH:
- Level 1: 2 tasks (setup + finalizer)
- Level 2: 2 tasks (controller + finalizer)
- Level 3: 2 tasks (aggregator × 2 batches)
- Level 4: 6 tasks (worker × 2 batches × 3 tasks)
- Level 5: 12 tasks (condition + processor × 6 workers)
Total Expected: 24 tasks

BUG: total_dag_tasks = 0 (hierarchy traversal completely broken)

TASK COUNT CALCULATION:
- Level 1: 2 tasks (setup + finalizer)
- Level 2: 2 tasks (controller + finalizer)  
- Level 3: 2 tasks (aggregator × 2 batches)
- Level 4: 6 tasks (worker × 2 batches × 3 tasks)
- Level 5: 6 tasks (condition check × 6 workers)
- Level 5: 6 tasks (processor × 6 workers, only branch A executes)
────────────────────────────────────────────────────────
EXPECTED total_dag_tasks = 24 tasks

BUG SYMPTOM:
- Actual total_dag_tasks = 0 (hierarchy traversal fails)
- DAG state stuck in RUNNING (can't track completion)
"""

# =============================================================================
# COMPONENT DEFINITIONS - Building blocks for each hierarchy level
# =============================================================================

@dsl.component()
def level1_setup() -> str:
    """
    ROOT LEVEL: Initialize the entire pipeline hierarchy.
    
    This represents the entry point for the most complex nested structure.
    Sets up the foundation for 4 additional levels of nesting below.
    """
    print("LEVEL 1: Setting up root pipeline context")
    return "level1_ready"

@dsl.component()
def level2_controller(input_from_level1: str) -> str:
    """
    CONTROLLER LEVEL: Orchestrates nested batch processing.
    
    Takes input from root level and decides whether to proceed with
    the complex nested batch and task processing in levels 3-5.
    """
    print(f"LEVEL 2: Controller received '{input_from_level1}' - initiating nested processing")
    return "level2_ready"

@dsl.component()
def level3_worker(batch: str, task: str) -> str:
    """
    WORKER LEVEL: Individual task execution within batch context.
    
    Each worker processes one task within one batch. With 2 batches × 3 tasks,
    this creates 6 parallel worker instances, each feeding into level 5 conditionals.
    """
    print(f"LEVEL 4: Worker executing batch='{batch}', task='{task}'")
    return f"level3_result_{batch}_{task}"

@dsl.component()
def level4_processor(worker_result: str, condition_value: int) -> str:
    """
    PROCESSOR LEVEL: Conditional processing of worker results.
    
    Applies different processing logic based on condition value.
    Each of the 6 workers feeds into this, creating 6 processor instances
    (all using branch A since condition always == 1).
    """
    branch = "A" if condition_value == 1 else "B"
    print(f"LEVEL 5: Processor {branch} handling '{worker_result}' (condition={condition_value})")
    return f"level4_processed_{worker_result}_branch_{branch}"

@dsl.component()
def get_deep_condition() -> int:
    """
    CONDITION PROVIDER: Returns condition for deep nested branching.
    
    Always returns 1, ensuring all 6 workers take the same conditional path.
    This creates predictable behavior for testing DAG status calculation.
    """
    print("LEVEL 5: Deep condition check (always returns 1)")
    return 1

@dsl.component()
def level3_aggregator(level: str) -> str:
    """
    BATCH AGGREGATOR: Collects results from all tasks within a batch.
    
    Each batch (batch_a, batch_b) gets its own aggregator instance,
    creating 2 aggregator tasks that summarize the work done in levels 4-5.
    """
    print(f"LEVEL 3: Aggregating results for batch '{level}'")
    return f"level3_aggregated_{level}"

@dsl.component()
def level2_finalizer(controller_result: str, aggregated_result: str) -> str:
    """
    CONTROLLER FINALIZER: Completes nested batch processing context.
    
    Runs after all batch processing (levels 3-5) completes.
    Represents the exit point from the nested conditional context.
    """
    print(f"LEVEL 2: Finalizing controller - {controller_result} + {aggregated_result}")
    return "level2_finalized"

@dsl.component()
def level1_finalizer(setup_result: str, level2_result: str) -> str:
    """
    ROOT FINALIZER: Completes the entire pipeline hierarchy.
    
    This is the final task that should execute only after all 23 other
    tasks across all 5 levels have completed successfully.
    """
    print(f"LEVEL 1: Root finalizer - {setup_result} + {level2_result}")
    return "deep_nesting_complete"

# =============================================================================
# PIPELINE DEFINITION - 5-Level Deep Nested Structure
# =============================================================================

@dsl.pipeline(
    name="nested-deep", 
    description="Deep nested pipeline testing 5-level hierarchical DAG status updates with mixed ParallelFor and conditional constructs"
)
def nested_deep_pipeline():
    """
    DEEP NESTED PIPELINE - Maximum Complexity Test Case
    
    Creates a 5-level deep hierarchy combining:
    - Sequential dependencies (Level 1 → Level 2)
    - Conditional contexts (IF statements)  
    - Parallel batch processing (ParallelFor batches)
    - Parallel task processing (ParallelFor tasks within batches)
    - Deep conditional branching (IF/ELSE within each task)
    
    This structure creates exactly 24 tasks across 5 nested levels,
    representing the most complex scenario for total_dag_tasks calculation.
    
    EXECUTION FLOW:
    1. Level 1 setup (1 task)
    2. Level 2 controller decides to proceed (1 task)
    3. Enter conditional context: IF(level2_ready)
       4. Level 3: FOR each batch in [batch_a, batch_b] (2 iterations)
          5. Level 4: FOR each task in [task_1, task_2, task_3] (3×2=6 iterations)
             6. Worker processes batch+task combination (6 tasks)
             7. Level 5: Get condition value (6 tasks)
             8. Level 5: IF(condition==1) → Process A (6 tasks, B never executes)
          9. Level 3: Aggregate batch results (2 tasks)
       10. Level 2: Finalize nested processing (1 task)
    11. Level 1: Final completion (1 task)
    
    Expected total_dag_tasks: 24
    Actual total_dag_tasks (BUG): 0
    """
    
    # ─────────────────────────────────────────────────────────────────────────
    # LEVEL 1: ROOT PIPELINE CONTEXT
    # ─────────────────────────────────────────────────────────────────────────
    print("Starting Level 1: Root pipeline initialization")
    level1_task = level1_setup().set_caching_options(enable_caching=False)
    
    # ─────────────────────────────────────────────────────────────────────────
    # LEVEL 2: CONTROLLER CONTEXT  
    # ─────────────────────────────────────────────────────────────────────────
    print("Starting Level 2: Controller orchestration")
    level2_task = level2_controller(input_from_level1=level1_task.output).set_caching_options(enable_caching=False)
    
    # ═════════════════════════════════════════════════════════════════════════
    # BEGIN DEEP NESTING: Conditional entry into 3-level hierarchy
    # ═════════════════════════════════════════════════════════════════════════
    with dsl.If(level2_task.output == "level2_ready"):
        print("Entering deep nested context (Levels 3-5)")
        
        # ─────────────────────────────────────────────────────────────────────
        # LEVEL 3: BATCH PARALLEL PROCESSING
        # Creates 2 parallel branches, one for each batch
        # ─────────────────────────────────────────────────────────────────────
        with dsl.ParallelFor(items=['batch_a', 'batch_b']) as batch:
            print(f"Level 3: Processing batch {batch}")
            
            # ─────────────────────────────────────────────────────────────────
            # LEVEL 4: TASK PARALLEL PROCESSING  
            # Creates 3 parallel workers per batch = 6 total workers
            # ─────────────────────────────────────────────────────────────────
            with dsl.ParallelFor(items=['task_1', 'task_2', 'task_3']) as task:
                print(f"Level 4: Processing {batch}/{task}")
                
                # Individual worker for this batch+task combination
                worker_result = level3_worker(batch=batch, task=task).set_caching_options(enable_caching=False)
                
                # ─────────────────────────────────────────────────────────────
                # LEVEL 5: DEEP CONDITIONAL PROCESSING
                # Each worker gets conditional processing based on dynamic condition
                # ─────────────────────────────────────────────────────────────
                print(f"Level 5: Conditional processing for {batch}/{task}")
                condition_task = get_deep_condition().set_caching_options(enable_caching=False)
                
                # Conditional branch A: Complex processing (condition == 1)
                with dsl.If(condition_task.output == 1):
                    processor_a = level4_processor(
                        worker_result=worker_result.output, 
                        condition_value=condition_task.output
                    ).set_caching_options(enable_caching=False)
                    
                # Conditional branch B: Alternative processing (condition != 1)  
                # NOTE: This branch never executes since condition always == 1
                with dsl.Else():
                    processor_b = level4_processor(
                        worker_result=worker_result.output,
                        condition_value=0
                    ).set_caching_options(enable_caching=False)
            
            # ─────────────────────────────────────────────────────────────────
            # LEVEL 3 COMPLETION: Aggregate results for this batch
            # Runs after all 3 tasks (and their L5 conditionals) complete
            # ─────────────────────────────────────────────────────────────────
            batch_aggregator = level3_aggregator(level=batch).set_caching_options(enable_caching=False)
        
        # ─────────────────────────────────────────────────────────────────────
        # LEVEL 2 COMPLETION: Finalize after all batch processing
        # Runs after both batches (and all their nested tasks) complete
        # ─────────────────────────────────────────────────────────────────────
        level2_finalizer_task = level2_finalizer(
            controller_result=level2_task.output,
            aggregated_result="all_batches_complete"  # Placeholder for aggregated results
        ).set_caching_options(enable_caching=False)
    
    # ─────────────────────────────────────────────────────────────────────────
    # LEVEL 1 COMPLETION: Root pipeline finalization
    # Should only execute after ALL 23 tasks in the nested hierarchy complete
    # ─────────────────────────────────────────────────────────────────────────
    level1_finalizer_task = level1_finalizer(
        setup_result=level1_task.output,
        level2_result="level2_context_complete"  # Placeholder for level 2 results
    ).set_caching_options(enable_caching=False)

if __name__ == "__main__":
    # Compile the deep nested pipeline for DAG status testing
    print("Compiling deep nested pipeline...")
    print("Expected task count: 24 across 5 hierarchy levels")
    print("Bug symptom: total_dag_tasks=0, DAG stuck in RUNNING state")
    
    kfp.compiler.Compiler().compile(
        nested_deep_pipeline, 
        "nested_deep.yaml"
    ) 