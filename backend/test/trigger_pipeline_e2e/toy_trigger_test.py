#!/usr/bin/env python3
"""Minimal E2E for dsl.trigger_pipeline against a local KFP cluster.

Prereqs:
  - KFP API at http://localhost:8888 (kubectl port-forward svc/ml-pipeline 8888:8888)
  - Local SDK + pipeline-spec with TriggerPipelineSpec
  - Cluster apiserver + launcher images that implement TriggerPipeline
"""

import time

import kfp

if not hasattr(kfp, "__version__"):
    kfp.__version__ = "2.15.2"

from kfp import compiler, dsl
from kfp.client import Client

HOST = "http://localhost:8888"


@dsl.component(base_image="python:3.11-slim")
def say_hello(name: str) -> str:
    message = f"hello {name}"
    print(message)
    return message


def main():
    client = Client(host=HOST)
    ts = int(time.time())
    child_name = f"trigger-e2e-child-{ts}"
    parent_name = f"trigger-e2e-parent-{ts}"

    @dsl.pipeline(name=child_name)
    def child_pipeline(name: str = "world") -> str:
        return say_hello(name=name).output

    @dsl.pipeline(name=parent_name)
    def parent_pipeline(name: str = "trigger") -> None:
        dsl.trigger_pipeline(
            pipeline_name=child_name,
            arguments={"name": name},
            wait_for_completion=True,
            poke_interval_seconds=5,
        )

    child_path = "/tmp/trigger_e2e_child.yaml"
    parent_path = "/tmp/trigger_e2e_parent.yaml"
    compiler.Compiler().compile(child_pipeline, child_path)
    compiler.Compiler().compile(parent_pipeline, parent_path)
    parent_yaml = open(parent_path).read()
    if "triggerPipeline" not in parent_yaml:
        raise SystemExit("parent IR missing triggerPipeline")
    print(f"compiled parent IR bytes={len(parent_yaml)}")

    print("Uploading child pipeline...")
    child = client.upload_pipeline(
        pipeline_package_path=child_path, pipeline_name=child_name
    )
    print(f"  child pipeline_id={child.pipeline_id}")

    print("Uploading parent pipeline...")
    parent = client.upload_pipeline(
        pipeline_package_path=parent_path, pipeline_name=parent_name
    )
    print(f"  parent pipeline_id={parent.pipeline_id}")

    print("Creating parent run...")
    run = client.create_run_from_pipeline_package(
        pipeline_file=parent_path,
        arguments={"name": "kind"},
        run_name=f"trigger-e2e-{ts}",
        enable_caching=False,
    )
    run_id = run.run_id
    print(f"  parent run_id={run_id}")

    print("Waiting for parent run...")
    state = None
    for i in range(90):
        detail = client.get_run(run_id)
        state = getattr(detail, "state", None) or getattr(
            getattr(detail, "run", None), "state", None
        )
        print(f"  [{i}] state={state}")
        if state in ("SUCCEEDED", "FAILED", "ERROR", "CANCELED"):
            break
        time.sleep(5)
    else:
        raise SystemExit("timeout waiting for parent run")

    print(f"Final parent state: {state}")
    runs = client.list_runs(page_size=10, sort_by="created_at desc")
    print("Recent runs:")
    for item in getattr(runs, "runs", None) or []:
        print(
            f"  {item.run_id[:8]} display={item.display_name} state={item.state}"
        )

    if state != "SUCCEEDED":
        raise SystemExit(1)
    print("E2E PASSED")


if __name__ == "__main__":
    main()
