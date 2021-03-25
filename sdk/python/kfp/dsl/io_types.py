# Copyright 2021 Google LLC
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
"""Classes for input/output types in KFP SDK.

These are only compatible with v2 Pipelines.
"""

import os
from typing import Dict, Optional, Type, TypeVar


class Artifact(object):
  """Generic Artifact class.

  This class is meant to represent the metadata around an input or output
  machine-learning Artifact. Artifacts have URIs, which can either be a location
  on disk (or Cloud storage) or some other resource identifier such as
  an API resource name.

  Artifacts carry a `metadata` field, which is a dictionary for storing
  metadata related to this artifact.
  """
  TYPE_NAME = "system.Artifact"

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    """Initializes the Artifact with the given name, URI and metadata."""
    self.uri = uri or ''
    self.name = name or ''
    self.metadata = metadata or {}


class Model(Artifact):
  """An artifact representing an ML Model."""
  TYPE_NAME = "kfp.Model"

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    super().__init__(uri=uri, name=name, metadata=metadata)

  @property
  def framework(self) -> str:
    return self._get_framework()

  def _get_framework(self) -> str:
    return self.metadata.get('framework', '')

  @framework.setter
  def framework(self, framework: str):
    self._set_framework(framework)

  def _set_framework(self, framework: str):
    self.metadata['framework'] = framework


class Dataset(Artifact):
  """An artifact representing an ML Dataset."""
  TYPE_NAME = "kfp.Dataset"

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    super().__init__(uri=uri, name=name, metadata=metadata)


class Metrics(Artifact):
  """Represent a simple base Artifact type to store key-value scalar metrics.
  """
  TYPE_NAME = "kfp.Metrics"

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    super().__init__(uri=uri, name=name, metadata=metadata)

  def log_metric(self, metric: str, value: float):
    """Sets a custom scalar metric.

    Args:
      metric: Metric key
      value: Value of the metric.
    """
    self.metadata[metric] = value


T = TypeVar('T', bound=Artifact)

_GCS_LOCAL_MOUNT_PREFIX = '/gcs/'


class _IOArtifact():
  """Internal wrapper class for representing Input/Output Artifacts."""

  def __init__(self, artifact_type: Type[T], artifact: Optional[T] = None):
    self.type = artifact_type
    self._artifact = artifact

  def get(self) -> T:
    return self._artifact

  @property
  def uri(self):
    return self._artifact.uri

  @uri.setter
  def uri(self, uri):
    self._artifact.uri = uri

  @property
  def path(self):
    return self._get_path()

  @path.setter
  def path(self, path):
    self._set_path(path)

  def _get_path(self) -> str:
    if self._artifact.uri.startswith('gs://'):
      return _GCS_LOCAL_MOUNT_PREFIX + self._artifact.uri[len('gs://'):]

  def _set_path(self, path):
    if path.startswith(_GCS_LOCAL_MOUNT_PREFIX):
      path = 'gs://' + path[len(_GCS_LOCAL_MOUNT_PREFIX):]
    self._artifact.uri = path


class InputArtifact(_IOArtifact):

  def __init__(self, artifact_type: Type[T], artifact: Optional[T] = None):
    super().__init__(artifact_type=artifact_type, artifact=artifact)


class OutputArtifact(_IOArtifact):

  def __init__(self, artifact_type: Type[T], artifact: Optional[T] = None):
    super().__init__(artifact_type=artifact_type, artifact=artifact)
    if artifact is not None:
      os.makedirs(self.path, exist_ok=True)
      self.path = os.path.join(self.path, 'data')


_SCHEMA_TITLE_TO_TYPE: Dict[str, Artifact] = {
    x.TYPE_NAME: x for x in [Artifact, Model, Dataset, Metrics]
}


def create_runtime_artifact(runtime_artifact: Dict) -> Artifact:
  """Creates an Artifact instance from the specified RuntimeArtifact.

  Args:
    runtime_artifact: Dictionary representing JSON-encoded RuntimeArtifact.
  """
  schema_title = runtime_artifact.get('type', {}).get('schemaTitle', '')

  artifact_type = _SCHEMA_TITLE_TO_TYPE.get(schema_title)
  if not artifact_type:
    artifact_type = Artifact
  return artifact_type(
      uri=runtime_artifact.get('uri', ''),
      name=runtime_artifact.get('name', ''),
      metadata=runtime_artifact.get('metadata', {}),
  )
