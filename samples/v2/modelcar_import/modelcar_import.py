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

# In tests, we install a KFP package from the PR under test. Users should not
# normally need to specify `kfp_package_path` in their component definitions.
_KFP_PACKAGE_PATH = os.getenv("KFP_PACKAGE_PATH")


@dsl.component(kfp_package_path=_KFP_PACKAGE_PATH)
def get_model_files_list(model: dsl.Input[dsl.Model]) -> str:
    import os
    import os.path

    if not os.path.exists(model.path):
        raise RuntimeError(f"The model does not exist at: {model.path}")

    expected_files = {
        "added_tokens.json",
        "model.safetensors",
        "tokenizer.json",
        "config.json",
        "normalizer.json",
        "tokenizer_config.json",
        "generation_config.json",
        "preprocessor_config.json",
        "vocab.json",
        "merges.txt",
        "special_tokens_map.json",
    }

    filesInPath = set(os.listdir(model.path))

    if filesInPath != expected_files:
        raise RuntimeError(
            f"The model does not have expected files: {', '.join(sorted(expected_files.difference(filesInPath)))}"
        )

    return ", ".join(sorted(filesInPath))


@dsl.pipeline(name="pipeline-with-modelcar-model")
def pipeline_modelcar_import(
    model_uri: str = "oci://registry.domain.local/modelcar:latest",
):
    model_source_oci_task = dsl.importer(
        artifact_uri=model_uri, artifact_class=dsl.Model
    ).set_caching_options(False)

    get_model_files_list(model=model_source_oci_task.output).set_caching_options(False)


if __name__ == "__main__":
    compiler.Compiler().compile(
        pipeline_func=pipeline_modelcar_import, package_path=__file__ + ".yaml"
    )
