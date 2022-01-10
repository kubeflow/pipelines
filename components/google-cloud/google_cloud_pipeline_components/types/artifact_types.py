# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Classes for ML Metadata input/output Artifacts for tracking Google resources.
"""

from typing import Dict,  Optional
from kfp.v2 import dsl

class VertexModel(dsl.Artifact):
  """An artifact representing a Vertex Model."""
  TYPE_NAME = 'google.VertexModel'

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    super().__init__(uri=uri, name=name, metadata=metadata)

class VertexEndpoint(dsl.Artifact):
  """An artifact representing a Vertex Endpoint."""
  TYPE_NAME = 'google.VertexEndpoint'

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    super().__init__(uri=uri, name=name, metadata=metadata)

class VertexBatchPredictionJob(dsl.Artifact):
  """An artifact representing a Vertex BatchPredictionJob."""
  TYPE_NAME = 'google.VertexBatchPredictionJob'

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    super().__init__(uri=uri, name=name, metadata=metadata)

class VertexDataset(dsl.Artifact):
  """An artifact representing a Vertex Dataset."""
  TYPE_NAME = 'google.VertexDataset'

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    super().__init__(uri=uri, name=name, metadata=metadata)

class BQMLModel(dsl.Artifact):
  """An artifact representing a BQML Model."""
  TYPE_NAME = 'google.BQMLModel'

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    super().__init__(uri=uri, name=name, metadata=metadata)

class BQTable(dsl.Artifact):
  """An artifact representing a BQ Table."""
  TYPE_NAME = 'google.BQTable'

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    super().__init__(uri=uri, name=name, metadata=metadata)


class UnmanagedContainerModel(dsl.Artifact):
  """An artifact representing an unmanaged container model."""
  TYPE_NAME = 'google.UnmanagedContainerModel'

  def __init__(self, metadata: Optional[Dict] = None):
    super().__init__(metadata=metadata)
