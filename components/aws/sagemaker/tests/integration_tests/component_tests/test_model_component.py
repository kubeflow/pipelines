# Copyright 2020 Amazon.com, Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You
# may not use this file except in compliance with the License. A copy of
# the License is located at
#
# http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is
# distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF
# ANY KIND, either express or implied. See the License for the specific
# language governing permissions and limitations under the License.

import pytest
import os
import utils

from utils import kfp_client_utils
from utils import minio_utils
from utils import sagemaker_utils


@pytest.mark.parametrize(
    "test_file_dir",
    [
        pytest.param(
            "resources/config/kmeans-mnist-model", marks=pytest.mark.canary_test
        )
    ],
)
def test_createmodel(kfp_client, experiment_id, sagemaker_client, test_file_dir):

    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )

    # Generate random prefix for model name to avoid errors if model with same name exists
    test_params["Arguments"]["model_name"] = input_model_name = (
        utils.generate_random_string(5) + "-" + test_params["Arguments"]["model_name"]
    )
    print(f"running test with model_name: {input_model_name}")

    _, _, workflow_json = kfp_client_utils.compile_run_monitor_pipeline(
        kfp_client,
        experiment_id,
        test_params["PipelineDefinition"],
        test_params["Arguments"],
        download_dir,
        test_params["TestName"],
        test_params["Timeout"],
    )

    outputs = {"sagemaker-create-model": ["model_name"]}

    output_files = minio_utils.artifact_download_iterator(
        workflow_json, outputs, download_dir
    )

    output_model_name = utils.read_from_file_in_tar(
        output_files["sagemaker-create-model"]["model_name"], "model_name.txt"
    )
    print(f"model_name: {output_model_name}")
    assert output_model_name == input_model_name
    assert (
        sagemaker_utils.describe_model(sagemaker_client, input_model_name) is not None
    )

    utils.remove_dir(download_dir)
