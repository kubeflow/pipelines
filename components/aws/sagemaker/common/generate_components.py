#!/usr/bin/env python3
"""A command line tool for generating component specification files."""
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import argparse

from common.component_compiler import SageMakerComponentCompiler
import common.sagemaker_component as component_module


COMPONENT_DIRECTORIES = [
    "batch_transform",
    "create_simulation_app",
    "delete_simulation_app",
    "deploy",
    "ground_truth",
    "hyperparameter_tuning",
    "model",
    "process",
    "rlestimator",
    "simulation_job",
    "simulation_job_batch",
    "train",
    "workteam",
]


def parse_arguments():
    """Parse command line arguments."""

    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--tag", type=str, required=True, help="The component container tag."
    )
    parser.add_argument(
        "--image",
        type=str,
        required=False,
        default="amazon/aws-sagemaker-kfp-components",
        help="The component container image.",
    )
    parser.add_argument(
        "--check",
        type=bool,
        required=False,
        default=False,
        help="Dry-run to compare against the existing files.",
    )

    args = parser.parse_args()
    return args


class ComponentCollectorContext:
    """Context for collecting components registered using their decorators."""

    def __enter__(self):
        component_specs = []

        def add_component(func):
            component_specs.append(func)
            return func

        # Backup previous handler
        self.old_handler = component_module._component_decorator_handler
        component_module._component_decorator_handler = add_component
        return component_specs

    def __exit__(self, *args):
        component_module._component_decorator_handler = self.old_handler


def compile_spec_file(component_file, spec_dir, args):
    """Attempts to compile a component specification file into a YAML spec.

    Writes a `component.yaml` file into a file one directory above where the
    specification file exists. For example if the spec is in `/my/spec/src`,
    it will create a file `/my/spec/component.yaml`.

    Args:
        component_file: A path to a component definition.
        spec_dir: The path containing the specification.
        args: Optional arguments as defined by the command line.
        check: Dry-run and check that the files match the expected output.
    """
    output_path = Path(spec_dir.parent, "component.yaml")
    relative_path = component_file.relative_to(root)
    # Remove extension
    relative_module = os.path.splitext(str(relative_path))[0]

    with ComponentCollectorContext() as component_metadatas:
        # Import the file using the path relative to the root
        __import__(relative_module.replace("/", "."))

    if len(component_metadatas) != 1:
        raise ValueError(
            f"Expected exactly 1 ComponentMetadata in {component_file}, found {len(component_metadatas)}"
        )

    if args.check:
        return SageMakerComponentCompiler.check(
            component_metadatas[0],
            str(relative_path),
            str(output_path.resolve()),
            component_image_tag=args.tag,
            component_image_uri=args.image,
        )

    SageMakerComponentCompiler.compile(
        component_metadatas[0],
        str(relative_path),
        str(output_path.resolve()),
        component_image_tag=args.tag,
        component_image_uri=args.image,
    )


if __name__ == "__main__":
    import os
    from pathlib import Path

    args = parse_arguments()

    cwd = Path(os.path.join(os.getcwd(), os.path.dirname(__file__)))
    root = cwd.parent

    for component in COMPONENT_DIRECTORIES:
        component_dir = Path(root, component)
        component_src_dir = Path(component_dir, "src")
        components = sorted(component_src_dir.glob("*_component.py"))

        if len(components) < 1:
            raise ValueError(f"Unable to find _component.py file for {component}")
        elif len(components) > 1:
            raise ValueError(f"Found multiple _component.py files for {component}")

        result = compile_spec_file(components[0], component_src_dir, args)

        if args.check and result:
            print(result)
            raise ValueError(
                f'Difference found between to the existing spec for the "{component}" component'
            )
