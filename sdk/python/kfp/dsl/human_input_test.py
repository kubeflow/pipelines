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
"""Unit tests for kfp.dsl.human_input."""
import json
import unittest

from kfp import dsl
from kfp.dsl.human_input import HUMAN_INPUT_SENTINEL_IMAGE, _HumanInputComponent
from kfp.dsl.structures import HumanInputParameterSpec, HumanInputSpec


class TestHumanInputParameterSpec(unittest.TestCase):
    """Tests for HumanInputParameterSpec dataclass."""

    def test_defaults(self):
        spec = HumanInputParameterSpec()
        self.assertIsNone(spec.description)
        self.assertIsNone(spec.default)
        self.assertIsNone(spec.enum)

    def test_all_fields(self):
        spec = HumanInputParameterSpec(
            description="Choose a side",
            default="YES",
            enum=["YES", "NO"],
        )
        self.assertEqual(spec.description, "Choose a side")
        self.assertEqual(spec.default, "YES")
        self.assertEqual(spec.enum, ["YES", "NO"])

    def test_enum_can_be_set_without_default(self):
        spec = HumanInputParameterSpec(enum=["a", "b"])
        self.assertIsNone(spec.default)
        self.assertEqual(spec.enum, ["a", "b"])


class TestHumanInputSpec(unittest.TestCase):
    """Tests for HumanInputSpec dataclass."""

    def test_parameters_field(self):
        spec = HumanInputSpec(
            parameters={"decision": HumanInputParameterSpec(enum=["YES", "NO"])},
        )
        self.assertIn("decision", spec.parameters)

    def test_empty_parameters(self):
        spec = HumanInputSpec(parameters={})
        self.assertEqual(spec.parameters, {})


class TestHumanInputFunction(unittest.TestCase):
    """Tests for the dsl.human_input() factory function."""

    def _compile(self, pipeline_func):
        from kfp import compiler
        import tempfile, os, yaml

        with tempfile.TemporaryDirectory() as tmpdir:
            path = os.path.join(tmpdir, "pipeline.yaml")
            compiler.Compiler().compile(pipeline_func, path)
            with open(path) as f:
                return yaml.safe_load(f)

    def test_returns_pipeline_task(self):
        """human_input() inside a pipeline compiles without error."""

        @dsl.pipeline
        def my_pipeline():
            dsl.human_input(
                name="approval-gate",
                parameters={"decision": HumanInputParameterSpec(enum=["YES", "NO"])},
            )

        compiled = self._compile(my_pipeline)
        self.assertIn("deploymentSpec", compiled)

    def test_output_is_accessible(self):
        """The task's outputs dict must expose the declared parameters."""

        @dsl.component
        def consume(value: str) -> str:
            return value

        @dsl.pipeline
        def my_pipeline():
            gate = dsl.human_input(
                name="gate",
                parameters={"status": HumanInputParameterSpec(default="pending")},
            )
            consume(value=gate.outputs["status"])

        compiled = self._compile(my_pipeline)
        task_names = list(compiled.get("root", {}).get("dag", {}).get("tasks", {}).keys())
        self.assertIn("gate", task_names)
        self.assertIn("consume", task_names)


class TestHumanInputCompilation(unittest.TestCase):
    """Integration-level tests: compile a pipeline and validate the output."""

    def _compile(self, pipeline_func):
        from kfp import compiler
        import tempfile, os, yaml

        with tempfile.TemporaryDirectory() as tmpdir:
            path = os.path.join(tmpdir, "pipeline.yaml")
            compiler.Compiler().compile(pipeline_func, path)
            with open(path) as f:
                return yaml.safe_load(f)

    def test_sentinel_image_in_deployment_spec(self):
        """The compiled YAML must contain the sentinel image in the deployment spec."""

        @dsl.pipeline
        def my_pipeline():
            dsl.human_input(
                name="approval-gate",
                parameters={"decision": HumanInputParameterSpec(enum=["YES", "NO"])},
            )

        compiled = self._compile(my_pipeline)

        # Find the executor for the human-input component.
        deployment_spec = compiled.get("deploymentSpec", {})
        executors = deployment_spec.get("executors", {})
        images = [
            e.get("container", {}).get("image", "")
            for e in executors.values()
        ]
        self.assertIn(HUMAN_INPUT_SENTINEL_IMAGE, images,
                      f"Expected sentinel image {HUMAN_INPUT_SENTINEL_IMAGE!r} in executors: {images}")

    def test_parameter_metadata_in_args(self):
        """args[0] must be a JSON dict mapping parameter names to their metadata."""

        @dsl.pipeline
        def my_pipeline():
            dsl.human_input(
                name="approval-gate",
                parameters={
                    "decision": HumanInputParameterSpec(
                        description="Approve?",
                        default="NO",
                        enum=["YES", "NO"],
                    ),
                },
            )

        compiled = self._compile(my_pipeline)

        deployment_spec = compiled.get("deploymentSpec", {})
        executors = deployment_spec.get("executors", {})

        found_args = None
        for executor in executors.values():
            container = executor.get("container", {})
            if container.get("image") == HUMAN_INPUT_SENTINEL_IMAGE:
                found_args = container.get("args", [])
                break

        self.assertIsNotNone(found_args, "Sentinel executor not found")
        self.assertGreater(len(found_args), 0, "args must be non-empty")

        meta = json.loads(found_args[0])
        self.assertIn("decision", meta)
        decision_meta = meta["decision"]
        self.assertEqual(decision_meta.get("description"), "Approve?")
        self.assertEqual(decision_meta.get("default"), "NO")
        self.assertEqual(decision_meta.get("enum"), ["YES", "NO"])

    def test_output_parameters_declared(self):
        """The component must declare output parameters matching the human-input keys."""

        @dsl.pipeline
        def my_pipeline():
            dsl.human_input(
                name="gate",
                parameters={
                    "choice": HumanInputParameterSpec(),
                    "reason": HumanInputParameterSpec(description="Reason for choice"),
                },
            )

        compiled = self._compile(my_pipeline)

        # Look at the component spec for the gate task.
        components = compiled.get("components", {})
        # The component name follows the naming convention comp-<task-name>.
        gate_spec = components.get("comp-gate", {})
        output_defs = gate_spec.get("outputDefinitions", {})
        params = output_defs.get("parameters", {})
        self.assertIn("choice", params, f"'choice' not in output params: {list(params)}")
        self.assertIn("reason", params, f"'reason' not in output params: {list(params)}")

    def test_downstream_task_receives_output(self):
        """A downstream component can consume the human-input task output."""

        @dsl.component
        def consume(value: str) -> str:
            return f"Got: {value}"

        @dsl.pipeline
        def my_pipeline():
            gate = dsl.human_input(
                name="approval-gate",
                parameters={"decision": HumanInputParameterSpec(enum=["YES", "NO"])},
            )
            consume(value=gate.outputs["decision"])

        # Should compile without error.
        compiled = self._compile(my_pipeline)
        # The consume component must appear in the compiled YAML.
        task_names = list(compiled.get("root", {})
                          .get("dag", {})
                          .get("tasks", {})
                          .keys())
        self.assertIn("approval-gate", task_names)
        self.assertIn("consume", task_names)

    def test_approval_gate_sample_compiles(self):
        """The approval-gate sample compiles without errors."""
        from samples.core.human_input.approval_gate import approval_gate_pipeline
        compiled = self._compile(approval_gate_pipeline)
        self.assertIn("deploymentSpec", compiled)


if __name__ == "__main__":
    unittest.main()
