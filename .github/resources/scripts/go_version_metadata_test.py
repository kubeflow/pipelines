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
"""Unit tests for structure-aware Go metadata discovery."""

import unittest

from go_version_metadata import (docker_go_runtime_sources,
                                 yaml_mapping_values)


class GoVersionMetadataTest(unittest.TestCase):

    def test_malformed_flow_collection_terminates(self):
        self.assertEqual(
            yaml_mapping_values('steps: [}\n', ('uses',)),
            {'uses': []},
        )

    def test_discovers_every_golang_stage(self):
        stages, arguments = docker_go_runtime_sources(
            'FROM golang:1.27.0 AS builder\n'
            'FROM golang:1.26.0 AS stale\n')

        self.assertEqual(stages,
                         ['golang:1.27.0', 'golang:1.26.0'])
        self.assertEqual(arguments, [])

    def test_resolves_global_arg_defaults_used_by_from(self):
        stages, arguments = docker_go_runtime_sources(
            'ARG GO_REPOSITORY=docker.io/library/golang\n'
            'FROM ${GO_REPOSITORY}:1.26.0 AS stale\n')

        self.assertEqual(stages, ['${GO_REPOSITORY}:1.26.0'])
        self.assertEqual(arguments, ['GO_REPOSITORY'])


if __name__ == '__main__':
    unittest.main()
