# Copyright 2026 The Kubeflow Authors
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

import os
import unittest
from unittest import mock

import generate_proto


class GenerateProtoTest(unittest.TestCase):

    def test_output_path_in_pkg_dir(self):
        source = os.path.join(generate_proto.PROTO_DIR, "pipeline_spec.proto")
        expected_output = os.path.join(generate_proto.PKG_DIR, "pipeline_spec_pb2.py")

        with mock.patch("os.path.exists", return_value=True), \
             mock.patch("os.path.getmtime", side_effect=lambda p: 100 if p == source else 200), \
             mock.patch("subprocess.call") as mock_subprocess:
            # File exists and is up to date -> should NOT call protoc
            generate_proto.generate_proto(source)
            mock_subprocess.assert_not_called()

    def test_regenerates_when_output_missing(self):
        source = os.path.join(generate_proto.PROTO_DIR, "pipeline_spec.proto")
        expected_output = os.path.join(generate_proto.PKG_DIR, "pipeline_spec_pb2.py")

        def mock_exists(path):
            if path == source:
                return True
            if path == expected_output:
                return False
            return False

        with mock.patch("os.path.exists", side_effect=mock_exists), \
             mock.patch.object(generate_proto, "PROTOC", "/usr/bin/protoc"), \
             mock.patch("subprocess.call", return_value=0) as mock_subprocess:
            generate_proto.generate_proto(source)
            mock_subprocess.assert_called_once_with([
                "/usr/bin/protoc",
                f"-I{generate_proto.PROTO_DIR}",
                f"--python_out={generate_proto.PKG_DIR}",
                source,
            ])

    def test_regenerates_when_source_newer(self):
        source = os.path.join(generate_proto.PROTO_DIR, "pipeline_spec.proto")
        expected_output = os.path.join(generate_proto.PKG_DIR, "pipeline_spec_pb2.py")

        def mock_mtime(path):
            if path == source:
                return 2000  # source is newer
            return 1000  # output is older

        with mock.patch("os.path.exists", return_value=True), \
             mock.patch("os.path.getmtime", side_effect=mock_mtime), \
             mock.patch.object(generate_proto, "PROTOC", "/usr/bin/protoc"), \
             mock.patch("subprocess.call", return_value=0) as mock_subprocess:
            generate_proto.generate_proto(source)
            mock_subprocess.assert_called_once_with([
                "/usr/bin/protoc",
                f"-I{generate_proto.PROTO_DIR}",
                f"--python_out={generate_proto.PKG_DIR}",
                source,
            ])


if __name__ == "__main__":
    unittest.main()
