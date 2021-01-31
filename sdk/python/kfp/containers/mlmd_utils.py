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
import ml_metadata as mlmd
from ml_metadata.proto import metadata_store_pb2
from ml_metadata.proto import metadata_store_service_pb2
import random
import time
from typing import Dict, List, Optional

# Number of times to retry initialization of connection.
_MAX_INIT_RETRY = 10

# Context type, the following three types of contexts are supported:
#  - pipeline level context is shared within one pipeline, across multiple
#    pipeline runs.
#  - pipeline run level context is shared within one pipeline run, across
#    all component executions in that pipeline run.
_CONTEXT_TYPE_PIPELINE = 'pipeline'
_CONTEXT_TYPE_PIPELINE_RUN = 'run'
# Keys of execution type properties.
_EXECUTION_TYPE_KEY_COMPONENT_ID = 'component_id'
# Keys for artifact properties.
_ARTIFACT_TYPE_KEY_STATE = 'state'


class ArtifactState(object):
  """Enumeration of possible Artifact states."""
  # Indicates that there is a pending execution producing the artifact.
  PENDING = 'pending'
  # Indicates that the artifact ready to be consumed.
  PUBLISHED = 'published'
  # Indicates that the no data at the artifact uri, though the artifact is not
  # marked as deleted.
  MISSING = 'missing'
  # Indicates that the artifact should be garbage collected.
  MARKED_FOR_DELETION = 'MARKED_FOR_DELETION'
  # Indicates that the artifact has been garbage collected.
  DELETED = 'deleted'



def _get_artifact_state(
    artifact: metadata_store_pb2.Artifact) -> Optional[str]:
  """Gets artifact state string if available."""
  if _ARTIFACT_TYPE_KEY_STATE in artifact.properties:
    return artifact.properties[_ARTIFACT_TYPE_KEY_STATE].string_value
  elif _ARTIFACT_TYPE_KEY_STATE in artifact.custom_properties:
    return artifact.custom_properties[_ARTIFACT_TYPE_KEY_STATE].string_value
  else:
    return None


class Metadata(object):
  """Helper class to handle interaction with metadata store."""
  def __init__(
      self,
      connection_config: metadata_store_pb2.MetadataStoreClientConfig):
    for _ in range(_MAX_INIT_RETRY):
      try:
        self._store = mlmd.MetadataStore(connection_config)
      except RuntimeError as err:
        time.sleep(random.random())
        continue
      else:
        return

    raise RuntimeError('Failed to establish MLMD connection with error.')

  @property
  def store(self) -> mlmd.MetadataStore:
    """Returns underlying MetadataStore."""
    if self._store is None:
      raise RuntimeError('Metadata object is not instantiated.')
    return self._store

  def get_pipeline_context(
      self, pipeline_name: str
  ) -> Optional[metadata_store_pb2.Context]:
    """Gets the context for the pipeline run.
    Args:
      pipeline_name: pipeline name for the current pipeline run.
    Returns:
      a matched context or None
    """
    return self.store.get_context_by_type_and_name(
        _CONTEXT_TYPE_PIPELINE, pipeline_name)

  def get_run_context(self, run_name: str):
    """Gets the context for the pipeline run.
    Args:
      run_name: run name for the current pipeline run.
    Returns:
      a matched context or None
    """
    return self.store.get_context_by_type_and_name(
        _CONTEXT_TYPE_PIPELINE_RUN, run_name)

  def get_qualified_artifacts(
      self,
      run_name: str,
      producer_component_id: Optional[str] = None,
      output_key: Optional[str] = None,
  ) -> List[metadata_store_service_pb2.ArtifactAndType]:
    """Gets qualified artifacts that have the right producer info.
    Args:
      run_name: pipeline context constraints to filter artifacts
      producer_component_id: producer constraint to filter artifacts
      output_key: output key constraint to filter artifacts
    Returns:
      A list of Artifact, containing qualified artifacts.
    """

    def _match_producer_component_id(
        execution: metadata_store_pb2.Execution,
        component_id: str) -> bool:
      if component_id:
        return execution.properties[
                 _EXECUTION_TYPE_KEY_COMPONENT_ID].string_value == component_id
      else:
        return True

    def _match_output_key(event: metadata_store_pb2.Event, key: str) -> bool:
      if key:
        assert len(event.path.steps) == 2, 'Event must have two path steps.'
        return (event.type == metadata_store_pb2.Event.OUTPUT and
                event.path.steps[0].key == key)
      else:
        return event.type == metadata_store_pb2.Event.OUTPUT

    # TODO(numerology): Consider validating artifact type before query.
    # Gets the executions that are associated with all contexts.
    assert run_name, 'Must specify the name of pipeline text.'
    executions_dict = {}
    run_context = self.get_run_context(run_name=run_name)
    executions = self.store.get_executions_by_context(run_context.id)
    executions_dict.update(dict((e.id, e) for e in executions))
    executions_within_context = executions_dict.values()

    # Filters the executions to match producer component id.
    qualified_producer_executions = [
        e.id
        for e in executions_within_context
        if _match_producer_component_id(e, producer_component_id)
    ]
    # Gets the output events that have the matched output key.
    qualified_output_events = [
        ev for ev in self.store.get_events_by_execution_ids(
            qualified_producer_executions) if _match_output_key(ev, output_key)
    ]

    # Gets the candidate artifacts from output events.
    candidate_artifacts = self.store.get_artifacts_by_id(
        list(set(ev.artifact_id for ev in qualified_output_events)))
    # Filters the artifacts that have the right artifact type and state.
    qualified_artifacts = [
        a for a in candidate_artifacts if
        (_get_artifact_state(a) == ArtifactState.PUBLISHED)]

    return qualified_artifacts



