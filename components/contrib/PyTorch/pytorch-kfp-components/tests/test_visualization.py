#!/usr/bin/env/python3
#
# Copyright (c) Facebook, Inc. and its affiliates.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Unit tests for visualization component."""
import os
import json
import tempfile
from unittest.mock import patch
import mock
from pytorch_kfp_components.components.visualization.component import Visualization
from pytorch_kfp_components.components.visualization.executor import Executor
import pytest

metdata_dir = tempfile.mkdtemp()


@pytest.fixture(scope="class")
def viz_params():
    """Setting visualization parameters.

    Returns:
        viz_param : dict of visualization parameters.
    """
    markdown_params = {
        "storage": "dummy-storage",
        "source": {
            "dummy_key": "dummy_value"
        },
    }

    viz_param = {
        "mlpipeline_ui_metadata":
        os.path.join(metdata_dir, "mlpipeline_ui_metadata.json"),
        "mlpipeline_metrics":
        os.path.join(metdata_dir, "mlpipeline_metrics"),
        "confusion_matrix_dict": {},
        "test_accuracy":
        99.05,
        "markdown":
        markdown_params,
    }
    return viz_param


@pytest.fixture(scope="class")
def confusion_matrix_params():
    """Setting the confusion matrix parameters.

    Returns:
        confusion_matrix_params : Dict of confusion matrix parmas
    """
    confusion_matrix_param = {
        "actuals": ["1", "2", "3", "4"],
        "preds": ["2", "3", "4", "0"],
        "classes": ["dummy", "dummy"],
        "url": "minio://dummy_bucket/folder_name",
    }
    return confusion_matrix_param


def generate_visualization(viz_params: dict):  #pylint: disable=redefined-outer-name
    """Generates the visualization object.

    Returns:
        output_dict : output dict of vizualization obj.
    """
    viz_obj = Visualization(
        mlpipeline_ui_metadata=viz_params["mlpipeline_ui_metadata"],
        mlpipeline_metrics=viz_params["mlpipeline_metrics"],
        confusion_matrix_dict=viz_params["confusion_matrix_dict"],
        test_accuracy=viz_params["test_accuracy"],
        markdown=viz_params["markdown"],
    )

    return viz_obj.output_dict


@pytest.mark.parametrize(
    "viz_key",
    [
        "confusion_matrix_dict",
        "test_accuracy",
        "markdown",
    ],
)
def test_invalid_type_viz_params(viz_params, viz_key):  #pylint: disable=redefined-outer-name
    """Test visualization for invalid parameter type."""
    viz_params[viz_key] = "dummy"
    if viz_key == "test_accuracy":
        expected_type = "<class 'float'>"
    else:
        expected_type = "<class 'dict'>"
    expected_exception_msg = f"{viz_key} must be of type {expected_type} but" \
                             f" received as {type(viz_params[viz_key])}"
    with pytest.raises(TypeError, match=expected_exception_msg):
        generate_visualization(viz_params)


@pytest.mark.parametrize(
    "viz_key",
    [
        "mlpipeline_ui_metadata",
        "mlpipeline_metrics",
    ],
)
def test_invalid_type_metadata_path(viz_params, viz_key):  #pylint: disable=redefined-outer-name
    """Test visualization with invalid metadata path."""

    viz_params[viz_key] = ["dummy"]
    expected_exception_msg = f"{viz_key} must be of type <class 'str'> " \
                             f"but received as {type(viz_params[viz_key])}"
    with pytest.raises(TypeError, match=expected_exception_msg):
        generate_visualization(viz_params)


@pytest.mark.parametrize(
    "viz_key",
    [
        "mlpipeline_ui_metadata",
        "mlpipeline_metrics",
    ],
)
def test_default_metadata_path(viz_params, viz_key):  #pylint: disable=redefined-outer-name
    """Test visualization with default metadata path."""
    viz_params[viz_key] = None
    expected_output = {
        "mlpipeline_ui_metadata": "/mlpipeline-ui-metadata.json",
        "mlpipeline_metrics": "/mlpipeline-metrics.json",
    }
    with patch(
            "test_visualization.generate_visualization",
            return_value=expected_output,
    ):
        output_dict = generate_visualization(viz_params)
    assert output_dict == expected_output


def test_custom_metadata_path(viz_params, tmpdir):  #pylint: disable=redefined-outer-name
    """Test visualization with custom metadata path."""
    metadata_ui_path = os.path.join(str(tmpdir), "mlpipeline_ui_metadata.json")
    metadata_metrics_path = os.path.join(str(tmpdir),
                                         "mlpipeline_metrics.json")
    viz_params["mlpipeline_ui_metadata"] = metadata_ui_path
    viz_params["mlpipeline_metrics"] = metadata_metrics_path
    output_dict = generate_visualization(viz_params)
    assert output_dict is not None
    assert output_dict["mlpipeline_ui_metadata"] == metadata_ui_path
    assert output_dict["mlpipeline_metrics"] == metadata_metrics_path
    assert os.path.exists(metadata_ui_path)
    assert os.path.exists(metadata_metrics_path)


def test_setting_all_keys_to_none(viz_params):  #pylint: disable=redefined-outer-name
    """Test visialization with all parameters set to None tyoe."""
    for key in viz_params.keys():
        viz_params[key] = None

    expected_exception_msg = r"Any one of these keys should be set -" \
                             r" confusion_matrix_dict, test_accuracy, markdown"
    with pytest.raises(ValueError, match=expected_exception_msg):
        generate_visualization(viz_params)


def test_accuracy_metric(viz_params):  #pylint: disable=redefined-outer-name
    """Test for getting proper accuracy metric."""
    output_dict = generate_visualization(viz_params)
    assert output_dict is not None
    metadata_metric_file = viz_params["mlpipeline_metrics"]
    assert os.path.exists(metadata_metric_file)
    with open(metadata_metric_file) as file:
        data = json.load(file)
    assert data["metrics"][0]["numberValue"] == viz_params["test_accuracy"]


def test_markdown_storage_invalid_datatype(viz_params):  #pylint: disable=redefined-outer-name
    """Test for passing invalid markdown storage datatype."""
    viz_params["markdown"]["storage"] = ["test"]
    expected_exception_msg = (
        r"storage must be of type <class 'str'> but received as {}".format(
            type(viz_params["markdown"]["storage"])))
    with pytest.raises(TypeError, match=expected_exception_msg):
        generate_visualization(viz_params)


def test_markdown_source_invalid_datatype(viz_params):  #pylint: disable=redefined-outer-name
    """Test for passing invalid markdown source datatype."""
    viz_params["markdown"]["source"] = "test"
    expected_exception_msg = (
        r"source must be of type <class 'dict'> but received as {}".format(
            type(viz_params["markdown"]["source"])))
    with pytest.raises(TypeError, match=expected_exception_msg):
        generate_visualization(viz_params)


@pytest.mark.parametrize(
    "markdown_key",
    [
        "source",
        "storage",
    ],
)
def test_markdown_source_missing_key(viz_params, markdown_key):  #pylint: disable=redefined-outer-name
    """Test with markdown source missing keys."""
    del viz_params["markdown"][markdown_key]
    expected_exception_msg = r"Missing mandatory key - {}".format(markdown_key)
    with pytest.raises(ValueError, match=expected_exception_msg):
        generate_visualization(viz_params)


def test_markdown_success(viz_params):  #pylint: disable=redefined-outer-name
    """Test for successful markdown generation."""
    output_dict = generate_visualization(viz_params)
    assert output_dict is not None
    assert "mlpipeline_ui_metadata" in output_dict
    assert os.path.exists(output_dict["mlpipeline_ui_metadata"])
    with open(output_dict["mlpipeline_ui_metadata"]) as file:
        data = file.read()
    assert "dummy_key" in data
    assert "dummy_value" in data


def test_different_storage_value(viz_params):  #pylint: disable=redefined-outer-name
    """Test for different storgae values for markdown."""
    viz_params["markdown"]["storage"] = "inline"
    output_dict = generate_visualization(viz_params)
    assert output_dict is not None
    assert "mlpipeline_ui_metadata" in output_dict
    assert os.path.exists(output_dict["mlpipeline_ui_metadata"])
    with open(output_dict["mlpipeline_ui_metadata"]) as file:
        data = file.read()
    assert "inline" in data


def test_multiple_metadata_appends(viz_params):  #pylint: disable=redefined-outer-name
    """Test for multiple metadata append."""
    if os.path.exists(viz_params["mlpipeline_ui_metadata"]):
        os.remove(viz_params["mlpipeline_ui_metadata"])

    if os.path.exists(viz_params["mlpipeline_metrics"]):
        os.remove(viz_params["mlpipeline_metrics"])
    generate_visualization(viz_params)
    generate_visualization(viz_params)
    output_dict = generate_visualization(viz_params)
    assert output_dict is not None
    assert "mlpipeline_ui_metadata" in output_dict
    assert os.path.exists(output_dict["mlpipeline_ui_metadata"])
    with open(output_dict["mlpipeline_ui_metadata"]) as file:
        data = json.load(file)
    assert len(data["outputs"]) == 3


@pytest.mark.parametrize(
    "cm_key",
    ["actuals", "preds", "classes", "url"],
)
def test_confusion_matrix_invalid_types(
        viz_params,  #pylint: disable=redefined-outer-name
        confusion_matrix_params,  #pylint: disable=redefined-outer-name
        cm_key):
    """Test for invalid type keys for confusion matrix."""
    confusion_matrix_params[cm_key] = {"test": "dummy"}
    viz_params["confusion_matrix_dict"] = confusion_matrix_params
    with pytest.raises(TypeError):
        generate_visualization(viz_params)


@pytest.mark.parametrize(
    "cm_key",
    ["actuals", "preds", "classes", "url"],
)
def test_confusion_matrix_optional_check(
        viz_params,  #pylint: disable=redefined-outer-name
        confusion_matrix_params,  #pylint: disable=redefined-outer-name
        cm_key):
    """Tests for passing confusion matrix keys as optional."""
    confusion_matrix_params[cm_key] = {}
    viz_params["confusion_matrix_dict"] = confusion_matrix_params
    expected_error_msg = f"{cm_key} is not optional. " \
                         f"Received value: {confusion_matrix_params[cm_key]}"
    with pytest.raises(ValueError, match=expected_error_msg):
        generate_visualization(viz_params)


@pytest.mark.parametrize(
    "cm_key",
    ["actuals", "preds", "classes", "url"],
)
def test_confusion_matrix_missing_check(
        viz_params,  #pylint: disable=redefined-outer-name
        confusion_matrix_params,  #pylint: disable=redefined-outer-name
        cm_key):
    """Tests for missing confusion matrix keys."""
    del confusion_matrix_params[cm_key]
    viz_params["confusion_matrix_dict"] = confusion_matrix_params
    expected_error_msg = f"Missing mandatory key - {cm_key}"
    with pytest.raises(ValueError, match=expected_error_msg):
        generate_visualization(viz_params)


def test_confusion_matrix_success(viz_params, confusion_matrix_params):  #pylint: disable=redefined-outer-name
    """Test for successful confusion matrix generation."""
    if os.path.exists(viz_params["mlpipeline_ui_metadata"]):
        os.remove(viz_params["mlpipeline_ui_metadata"])
    viz_params["confusion_matrix_dict"] = confusion_matrix_params
    with mock.patch.object(Executor, "_upload_confusion_matrix_to_minio"):
        output_dict = generate_visualization(viz_params)

    assert output_dict is not None
    assert "mlpipeline_ui_metadata" in output_dict
    assert os.path.exists(output_dict["mlpipeline_ui_metadata"])
    with open(output_dict["mlpipeline_ui_metadata"]) as file:
        data = file.read()
    assert "confusion_matrix" in data
