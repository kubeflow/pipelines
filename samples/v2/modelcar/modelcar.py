#!/usr/bin/env python3
# Copyright 2025 The Kubeflow Authors
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
"""Pipeline that imports a model in the Modelcar format from an OCI registry."""

import os

from kfp import compiler
from kfp import dsl
from kfp.dsl import component

@dsl.component()
def build_model_car(model: dsl.Output[dsl.Model]):
    # Simulate pushing the Modelcar to an OCI registry
    model.uri = "oci://registry.domain.local/org/repo:v1.0"

@dsl.component()
def get_model_files_list(input_model: dsl.Input[dsl.Model]) -> str:
    import os
    import os.path

    if not os.path.exists(input_model.path):
        raise RuntimeError(f"The model does not exist at: {input_model.path}")

    expected_files = {
        "added_tokens.json",
        "config.json",
        "generation_config.json",
        "merges.txt",
        "model.safetensors",
        "normalizer.json",
        "preprocessor_config.json",
        "special_tokens_map.json",
        "tokenizer.json",
        "tokenizer_config.json",
        "vocab.json",
    }

    filesInPath = set(os.listdir(input_model.path))

    if not filesInPath.issuperset(expected_files):
        raise RuntimeError(
            "The model does not have expected files: "
            + ", ".join(sorted(expected_files.difference(filesInPath)))
        )

    return ", ".join(sorted(filesInPath))


@dsl.pipeline(name="pipeline-with-modelcar-model")
def pipeline_modelcar_test(
    model_uri: str = "oci://registry.domain.local/modelcar:test",
):
    model_source_oci_task = dsl.importer(
        artifact_uri=model_uri, artifact_class=dsl.Model
    )

    get_model_files_list(input_model=model_source_oci_task.output).set_caching_options(False)

    build_model_car().set_caching_options(False)


if __name__ == "__main__":
    compiler.Compiler().compile(
        pipeline_func=pipeline_modelcar_test, package_path=__file__ + ".yaml"
    )
