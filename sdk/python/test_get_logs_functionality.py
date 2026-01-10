#!/usr/bin/env python3
"""
Functional test for get_logs() method.
Requires a running Kubeflow Pipelines instance.
"""

from kfp import dsl, compiler, Client
import time


# 1. Define a simple pipeline
@dsl.component
def hello_world(message: str) -> str:
    print(f"Hello: {message}")
    return message


@dsl.component
def goodbye_world(message: str) -> str:
    print(f"Goodbye: {message}")
    return message


@dsl.pipeline(name='test-logs-pipeline')
def test_pipeline(text: str = "World"):
    task1 = hello_world(message=text)
    task2 = goodbye_world(message=task1.output)


# 2. Compile pipeline
compiler.Compiler().compile(test_pipeline, 'test_pipeline.yaml')

# 3. Create client and run pipeline
try:
    client = Client(host='http://localhost:8080')

    print("Creating pipeline run...")
    run = client.create_run_from_pipeline_func(
        test_pipeline, arguments={'text': 'Testing get_logs functionality'})

    print(f"Run created: {run.run_id}")
    print("Waiting for pipeline to complete...")

    # Wait for completion
    run.wait_for_run_completion(timeout=300)

    print(f"Pipeline status: {run.state}")

    # TEST 1: Get logs from all components
    print("\n" + "=" * 60)
    print("TEST 1: Get logs from all components")
    print("=" * 60)
    try:
        all_logs = client.get_logs(run_id=run.run_id)
        print(f"Found {len(all_logs)} components:")
        for component_name, logs in all_logs.items():
            print(f"\nComponent: {component_name}")
            print(f"Logs (first 200 chars): {logs[:200]}")
            assert logs, f"No logs found for {component_name}"
        print("\nTEST 1: PASSED")
    except Exception as e:
        print(f"TEST 1: FAILED - {e}")

    # TEST 2: Get logs from specific component
    print("\n" + "=" * 60)
    print("TEST 2: Get logs from specific component")
    print("=" * 60)
    try:
        component_list = list(all_logs.keys())
        if component_list:
            component_name = component_list[0]
            logs = client.get_logs(
                run_id=run.run_id, component_name=component_name)
            print(f"Component: {component_name}")
            print(f"Logs (first 200 chars): {logs[:200]}")
            assert logs, f"No logs found for {component_name}"
            print("TEST 2: PASSED")
        else:
            print("TEST 2: SKIPPED (no components found)")
    except Exception as e:
        print(f"TEST 2: FAILED - {e}")

    # TEST 3: Error handling - non-existent run
    print("\n" + "=" * 60)
    print("TEST 3: Error handling - non-existent run")
    print("=" * 60)
    try:
        client.get_logs(run_id='nonexistent-run-xyz')
        print("TEST 3: FAILED (should have raised ValueError)")
    except ValueError as e:
        if "not found" in str(e):
            print(f"Correctly raised ValueError: {e}")
            print("TEST 3: PASSED")
        else:
            print(f"TEST 3: FAILED (wrong error message): {e}")
    except Exception as e:
        print(f"TEST 3: FAILED (unexpected error): {e}")

    # TEST 4: Error handling - non-existent component
    print("\n" + "=" * 60)
    print("TEST 4: Error handling - non-existent component")
    print("=" * 60)
    try:
        client.get_logs(
            run_id=run.run_id, component_name='nonexistent-component')
        print("TEST 4: FAILED (should have raised ValueError)")
    except ValueError as e:
        if "not found" in str(e):
            print(f"Correctly raised ValueError: {e}")
            print("TEST 4: PASSED")
        else:
            print(f"TEST 4: FAILED (wrong error message): {e}")
    except Exception as e:
        print(f"TEST 4: FAILED (unexpected error): {e}")

    print("\n" + "=" * 60)
    print("ALL TESTS COMPLETED!")
    print("=" * 60)

except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
