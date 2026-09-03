#!/usr/bin/env python3
"""Minimal E2E for dsl.trigger_pipeline against a local KFP cluster.

Prereqs:
  - KFP API at http://localhost:8888 (kubectl port-forward svc/ml-pipeline 8888:8888)
  - Local SDK + pipeline-spec with TriggerPipelineSpec
  - Cluster apiserver + launcher images that implement TriggerPipeline
"""

import time
from typing import NamedTuple

import kfp

if not hasattr(kfp, "__version__"):
    kfp.__version__ = "2.15.2"

from kfp import compiler, dsl
from kfp.client import Client

HOST = "http://localhost:8888"

ChildOutputs = NamedTuple(
    "ChildOutputs",
    [
        ("message", str),
        ("char_count", int),
        ("char_count_squared", int),
    ],
)


@dsl.component(base_image="python:3.11-slim")
def greet_and_count(name: str) -> NamedTuple(
    "Outputs",
    [
        ("message", str),
        ("char_count", int),
    ],
):
    """Example child work: build a greeting and return a calculated length."""
    outputs = NamedTuple(
        "Outputs",
        [
            ("message", str),
            ("char_count", int),
        ],
    )
    message = f"hello {name}"
    char_count = len(name)
    print(f"{message} (char_count={char_count})")
    return outputs(message=message, char_count=char_count)


@dsl.component(base_image="python:3.11-slim")
def square_char_count(char_count: int) -> int:
    """Downstream child task: simple calculation on greet_and_count output."""
    squared = char_count * char_count
    print(f"char_count={char_count} -> squared={squared}")
    return squared


@dsl.component(base_image="python:3.11-slim")
def summarize_trigger(run_id: str, state: str, pipeline_version_id: str) -> str:
    """Parent task after trigger: uses trigger outputs for a simple summary."""
    summary = (
        f"triggered child run_id={run_id} state={state} "
        f"pipeline_version_id={pipeline_version_id}"
    )
    print(summary)
    return summary


def main():
    client = Client(host=HOST)
    ts = int(time.time())
    child_name = f"trigger-e2e-child-{ts}"
    parent_name = f"trigger-e2e-parent-{ts}"

    @dsl.pipeline(name=child_name)
    def child_pipeline(name: str = "world") -> ChildOutputs:
        result = greet_and_count(name=name)
        squared = square_char_count(char_count=result.outputs["char_count"])
        return ChildOutputs(
            message=result.outputs["message"],
            char_count=result.outputs["char_count"],
            char_count_squared=squared.output,
        )

    @dsl.pipeline(name=parent_name)
    def parent_pipeline(name: str = "trigger") -> None:
        trigger = dsl.trigger_pipeline(
            pipeline_name=child_name,
            arguments={"name": name},
            wait_for_completion=True,
            poke_interval_seconds=5,
        )
        summarize_trigger(
            run_id=trigger.outputs["run_id"],
            state=trigger.outputs["state"],
            pipeline_version_id=trigger.outputs["pipeline_version_id"],
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
    expected_child_prefix = f"{child_name}-from-trigger"
    runs = client.list_runs(page_size=20, sort_by="created_at desc")
    print("Recent runs:")
    child_version = None
    child_run_id = None
    for item in getattr(runs, "runs", None) or []:
        pvr = getattr(item, "pipeline_version_reference", None)
        version_id = getattr(pvr, "pipeline_version_id", None) if pvr else None
        print(
            f"  {item.run_id[:8]} display={item.display_name} "
            f"state={item.state} version={version_id}"
        )
        if (
            item.display_name
            and item.display_name.startswith(expected_child_prefix)
            and item.state == "SUCCEEDED"
            and child_version is None
        ):
            child_version = version_id
            child_run_id = item.run_id

    if state != "SUCCEEDED":
        raise SystemExit(1)
    if not child_run_id or not child_version:
        raise SystemExit(
            f"expected succeeded child run starting with {expected_child_prefix!r} "
            f"and a pipeline_version_id"
        )
    print(f"Child run_id={child_run_id}")
    print(f"Child pipeline_version_id={child_version}")
    print("E2E PASSED")


if __name__ == "__main__":
    main()
