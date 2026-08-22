#!/usr/bin/env python3
"""Upload stable manual-trigger-child/parent for UI smoke checks."""

import time
from typing import NamedTuple

import kfp

if not hasattr(kfp, "__version__"):
    kfp.__version__ = "2.15.2"

from kfp import compiler, dsl
from kfp.client import Client

HOST = "http://localhost:8888"
CHILD = "manual-trigger-child"
PARENT = "manual-trigger-parent"

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


def main() -> None:
    version = f"manual-{int(time.time())}"

    @dsl.pipeline(name=CHILD)
    def child_pipeline(name: str = "world") -> ChildOutputs:
        result = greet_and_count(name=name)
        squared = square_char_count(char_count=result.outputs["char_count"])
        return ChildOutputs(
            message=result.outputs["message"],
            char_count=result.outputs["char_count"],
            char_count_squared=squared.output,
        )

    @dsl.pipeline(name=PARENT)
    def parent_pipeline(name: str = "manual") -> None:
        trigger = dsl.trigger_pipeline(
            pipeline_name=CHILD,
            arguments={"name": name},
            wait_for_completion=True,
            poke_interval_seconds=5,
        )
        summarize_trigger(
            run_id=trigger.outputs["run_id"],
            state=trigger.outputs["state"],
            pipeline_version_id=trigger.outputs["pipeline_version_id"],
        )

    child_path = "/tmp/manual_trigger_child.yaml"
    parent_path = "/tmp/manual_trigger_parent.yaml"
    compiler.Compiler().compile(child_pipeline, child_path)
    compiler.Compiler().compile(parent_pipeline, parent_path)

    client = Client(host=HOST)

    def upsert(path: str, name: str, version_name: str):
        pipelines = client.list_pipelines(page_size=100).pipelines or []
        match = next((p for p in pipelines if p.display_name == name), None)
        if match is None:
            p = client.upload_pipeline(pipeline_package_path=path, pipeline_name=name)
            print(f"uploaded pipeline {name} id={p.pipeline_id}")
            v = client.upload_pipeline_version(
                pipeline_package_path=path,
                pipeline_id=p.pipeline_id,
                pipeline_version_name=version_name,
            )
            print(f"  version={v.pipeline_version_id} name={version_name}")
            return p.pipeline_id, v.pipeline_version_id
        v = client.upload_pipeline_version(
            pipeline_package_path=path,
            pipeline_id=match.pipeline_id,
            pipeline_version_name=version_name,
        )
        print(
            f"version on {name} id={match.pipeline_id} "
            f"version={v.pipeline_version_id} name={version_name}"
        )
        return match.pipeline_id, v.pipeline_version_id

    cid, cvid = upsert(child_path, CHILD, version)
    pid, pvid = upsert(parent_path, PARENT, version)
    print(f"VERSION_LABEL {version}")
    print(f"CHILD_ID {cid}")
    print(f"CHILD_VERSION_ID {cvid}")
    print(f"PARENT_ID {pid}")
    print(f"PARENT_VERSION_ID {pvid}")
    print("UI http://localhost:8081")
    print(f"UI_PARENT http://localhost:8081/#/pipelines/details/{pid}")
    print(f"UI_CHILD http://localhost:8081/#/pipelines/details/{cid}")
    print(
        "CREATE_RUN "
        f"http://localhost:8081/#/runs/new?pipelineId={pid}&pipelineVersionId={pvid}"
    )
    print(
        "Child graph: greet_and_count -> square_char_count(char_count); "
        "outputs message, char_count, char_count_squared."
    )
    print(
        "Parent graph: trigger_pipeline -> summarize_trigger(run_id, state, "
        "pipeline_version_id)."
    )


if __name__ == "__main__":
    main()
