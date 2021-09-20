#!/usr/bin/env/python3
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
"""Unit Tests for Minio Component."""
import tempfile
import os
import mock
import pytest
from pytorch_kfp_components.components.minio.component import MinIO
from pytorch_kfp_components.components.minio.executor import Executor

tmpdir = tempfile.mkdtemp()

with open(os.path.join(str(tmpdir), "dummy.txt"), "w") as fp:
    fp.write("dummy")

#pylint: disable=redefined-outer-name
#pylint: disable=invalid-name

@pytest.fixture(scope="class")
def minio_inputs():
    """Sets the minio inputs.

    Returns:
          minio_inputs : dict of inputs for minio uploads.
    """
    minio_input = {
        "bucket_name": "dummy",
        "source": f"{tmpdir}/dummy.txt",
        "destination": "dummy.txt",
        "endpoint": "localhost:9000",
    }
    return minio_input


def upload_to_minio(minio_inputs):
    """Uoloads the artifact to minio."""
    MinIO(
        source=minio_inputs["source"],
        bucket_name=minio_inputs["bucket_name"],
        destination=minio_inputs["destination"],
        endpoint=minio_inputs["endpoint"],
    )


@pytest.mark.parametrize(
    "key",
    ["source", "bucket_name", "destination", "endpoint"],
)
def test_minio_variables_invalid_type(minio_inputs, key):
    """Testing for invalid variable types."""
    minio_inputs[key] = ["test"]
    expected_exception_msg = f"{key} must be of type <class 'str'>" \
                             f" but received as {type(minio_inputs[key])}"
    with pytest.raises(TypeError, match=expected_exception_msg):
        upload_to_minio(minio_inputs)


@pytest.mark.parametrize(
    "key",
    ["source", "bucket_name", "destination", "endpoint"],
)
def test_minio_mandatory_param(minio_inputs, key):
    """Testing for invalid minio mandatory parameters."""
    minio_inputs[key] = None
    expected_exception_msg = (
        f"{key} is not optional. Received value: {minio_inputs[key]}")
    with pytest.raises(ValueError, match=expected_exception_msg):
        upload_to_minio(minio_inputs)


def test_missing_access_key(minio_inputs):
    """Test upload if minio access key is missing."""
    os.environ["MINIO_SECRET_KEY"] = "dummy"
    expected_exception_msg = "Environment variable MINIO_ACCESS_KEY not found"
    with pytest.raises(ValueError, match=expected_exception_msg):
        upload_to_minio(minio_inputs)

    os.environ.pop("MINIO_SECRET_KEY")


def test_missing_secret_key(minio_inputs):
    """Test upload if minio secret key is missing."""
    os.environ["MINIO_ACCESS_KEY"] = "dummy"
    expected_exception_msg = "Environment variable MINIO_SECRET_KEY not found"
    with pytest.raises(ValueError, match=expected_exception_msg):
        upload_to_minio(minio_inputs)
    os.environ.pop("MINIO_ACCESS_KEY")


def test_unreachable_endpoint(minio_inputs):
    """Testing unreachable minio endpoint with invalid minio creds."""
    os.environ["MINIO_ACCESS_KEY"] = "dummy"
    os.environ["MINIO_SECRET_KEY"] = "dummy"
    with pytest.raises(Exception, match="Max retries exceeded with url: "):
        upload_to_minio(minio_inputs)


def test_invalid_file_path(minio_inputs):
    """Test invalid source file path."""
    minio_inputs["source"] = "dummy"
    expected_exception_msg = (
        f"Input path - {minio_inputs['source']} does not exists")
    with pytest.raises(ValueError, match=expected_exception_msg):
        upload_to_minio(minio_inputs)


def test_minio_upload_file(minio_inputs):
    """Testing upload of files to minio."""
    with mock.patch.object(Executor, "upload_artifacts_to_minio") as client:
        client.return_value = []
        upload_to_minio(minio_inputs)
        client.asser_called_once()


def test_minio_upload_folder(minio_inputs):
    """Testing upload of folder to minio."""
    minio_inputs["source"] = tmpdir
    with mock.patch.object(Executor, "upload_artifacts_to_minio") as client:
        client.return_value = []
        upload_to_minio(minio_inputs)
        client.asser_called_once()
