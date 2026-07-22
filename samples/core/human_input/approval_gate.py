# Copyright 2024 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Sample: static approval gate with YES/NO human input.

Demonstrates a pipeline that pauses at a human-input checkpoint and waits for
a user to choose YES or NO before continuing.  This is sample 1 of 2 for the
human-input (intermediate parameters) feature.

Usage::

    python approval_gate.py        # compile only
    python approval_gate.py --run  # compile + submit via KFP client

Acceptance criteria verified by this sample:
  1. Pipeline compiles successfully (contains a suspend template).
  2. On submission the run enters Suspended state.
  3. Via the KFP UI or API the user selects YES/NO and the run resumes.
  4. The downstream task receives the chosen value.
"""
from kfp import compiler
from kfp import dsl
from kfp.dsl import human_input, Output, Artifact


@dsl.component
def notify_start() -> str:
    """Emit a message before the approval step."""
    msg = "Pipeline started – waiting for approval."
    print(msg)
    return msg


@dsl.component
def process_decision(decision: str, start_message: str) -> str:
    """Process the human-supplied approval decision.

    Args:
        decision: YES or NO, supplied interactively through the KFP UI.
        start_message: message emitted before the approval step.

    Returns:
        A summary string describing what happened.
    """
    if decision.upper() == "YES":
        result = f"Approved! ({start_message})"
    else:
        result = f"Rejected. ({start_message})"
    print(result)
    return result


@dsl.pipeline(name="approval-gate-pipeline")
def approval_gate_pipeline():
    """Pipeline with a static YES/NO approval gate.

    The pipeline pauses at the *approval-gate* task.  A human must visit the
    KFP UI, choose a value, and click *Submit & Resume* before the downstream
    *process-decision* task runs.
    """
    start_task = notify_start()

    # Human-input checkpoint: the workflow suspends here until a human supplies
    # a value for the 'decision' parameter.
    gate = human_input(
        name="approval-gate",
        parameters={
            "decision": dsl.HumanInputParameterSpec(
                description="Approval decision",
                default="NO",
                enum=["YES", "NO"],
            ),
        },
    )

    # The downstream task consumes the decision directly.
    process_decision(
        decision=gate.outputs["decision"],
        start_message=start_task.output,
    )


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser()
    parser.add_argument("--run", action="store_true", help="Submit the pipeline to a KFP server")
    parser.add_argument("--host", default="http://localhost:3000", help="KFP API server host")
    args = parser.parse_args()

    output_file = "approval_gate.yaml"
    compiler.Compiler().compile(approval_gate_pipeline, output_file)
    print(f"Compiled pipeline written to {output_file}")

    if args.run:
        import kfp

        client = kfp.Client(host=args.host)
        run = client.create_run_from_pipeline_func(approval_gate_pipeline)
        print(f"Submitted run: {run.run_id}")
        print(
            f"Visit the KFP UI, navigate to the run, click the 'approval-gate' "
            f"node, choose YES or NO, and click 'Submit & Resume'."
        )
