"""KFP DSL compiler with Vertex Specific Features.

This compiler is intended to compile PipelineSpec with Vertex Specifc features.

To ensure full compatibility with Vertex specifc functionalities,
Google first party pipelines should utilize this version of compiler.
"""

import os
from os import path
from typing import Any, Dict, Optional

from absl import logging
from google.protobuf import json_format
from google_cloud_pipeline_components.proto import template_metadata_pb2
from kfp import compiler as kfp_compiler
from kfp.dsl import base_component
from kfp.pipeline_spec import pipeline_spec_pb2
import yaml


class Compiler:
  """Compiles pipelines composed using the KFP SDK DSL to a YAML pipeline definition.

  The pipeline definition is `PipelineSpec IR
  <https://github.com/kubeflow/pipelines/blob/2060e38c5591806d657d85b53eed2eef2e5de2ae/api/v2alpha1/pipeline_spec.proto#L50>`_,
  the protobuf message that defines a pipeline.

  Example:
    ::

      @dsl.pipeline(
        name='name',
      )
      def my_pipeline(a: int, b: str = 'default value'):
          ...

      compiler.Compiler().compile(
          pipeline_func=my_pipeline,
          package_path='path/to/pipeline.yaml',
          pipeline_parameters={'a': 1},
      )
  """

  def merge_template_metadata_to_pipeline_spec_proto(
      self,
      template_metadata: Optional[template_metadata_pb2.TemplateMetadata],
      pipeline_spec_proto: pipeline_spec_pb2.PipelineSpec,
  ) -> pipeline_spec_pb2.PipelineSpec:
    """Merges TemplateMetadata into a PipelineSpec for execution on Google Cloud.

    This function prepares a PipelineSpec for execution on Google Cloud by
    incorporating TemplateMetadata into the platform-specific configuration. The
    metadata is converted to JSON and embedded within the 'google_cloud'
    platform
    configuration.

    Args:
        template_metadata: A TemplateMetadata object containing component
          metadata.
        pipeline_spec_proto: A PipelineSpec protobuf object representing the
          pipeline specification.

    Returns:
        A modified PipelineSpec protobuf object with the TemplateMetadata merged
        into the 'google_cloud' PlatformSpec configuration or the original
        PlatformSpec proto if the template_metadata is none.
    """
    if template_metadata is None:
      return pipeline_spec_proto
    template_metadata_json = json_format.MessageToJson(template_metadata)
    platform_spec_proto = pipeline_spec_pb2.PlatformSpec()
    platform_spec_proto.platform = "google_cloud"
    json_format.Parse(template_metadata_json, platform_spec_proto.config)
    pipeline_spec_proto.root.platform_specs.append(platform_spec_proto)
    return pipeline_spec_proto

  def parse_pipeline_spec_yaml(
      self,
      pipeline_spec_yaml_file: str,
  ) -> pipeline_spec_pb2.PipelineSpec:
    """Parse pipeline spec yaml parses to the proto.

    Args:
      pipeline_spec_yaml_file: Path to the pipeline spec yaml file.

    Returns:
      Proto parsed.

    Raises:
      ValueError: When the PipelineSpec is invalid.
    """
    with open(pipeline_spec_yaml_file, "r") as f:
      pipeline_spec_yaml = f.read()
    pipeline_spec_dict = yaml.safe_load(pipeline_spec_yaml)
    pipeline_spec_proto = pipeline_spec_pb2.PipelineSpec()
    try:
      json_format.ParseDict(pipeline_spec_dict, pipeline_spec_proto)
    except json_format.ParseError as e:
      raise ValueError(
          "Failed to parse %s . Please check if that is a valid YAML file"
          " parsing a pipelineSpec proto." % pipeline_spec_yaml_file
      ) from e
    if not pipeline_spec_proto.HasField("pipeline_info"):
      raise ValueError(
          "PipelineInfo field not found in the pipeline spec YAML file %s."
          % pipeline_spec_yaml_file
      )
    if not pipeline_spec_proto.pipeline_info.display_name:
      logging.warning(
          (
              "PipelineInfo.displayName field is empty in pipeline spec YAML"
              " file %s."
          ),
          pipeline_spec_yaml_file,
      )
    if not pipeline_spec_proto.pipeline_info.description:
      logging.warning(
          (
              "PipelineInfo.description field is empty in pipeline spec YAML"
              " file %s."
          ),
          pipeline_spec_yaml_file,
      )
    return pipeline_spec_proto

  def compile(
      self,
      pipeline_func: base_component.BaseComponent,
      package_path: str,
      pipeline_name: Optional[str] = None,
      pipeline_parameters: Optional[Dict[str, Any]] = None,
      type_check: bool = True,
      includ_vertex_specifc_features=True,
  ) -> None:
    """Compiles the pipeline or component function into IR YAML.

       By default, this compiler will compile any Vertex Specifc Features.

    Args:
        pipeline_func: Pipeline function constructed with the ``@dsl.pipeline``
          or component constructed with the ``@dsl.component`` decorator.
        package_path: Output YAML file path. For example,
          ``'~/my_pipeline.yaml'`` or ``'~/my_component.yaml'``.
        pipeline_name: Name of the pipeline.
        pipeline_parameters: Map of parameter names to argument values.
        type_check: Whether to enable type checking of component interfaces
          during compilation.
        includ_vertex_specifc_features: Whether to enable compiling Vertex
          Specific Features.
    """
    if not includ_vertex_specifc_features:
      kfp_compiler.Compiler().compile(
          pipeline_func=pipeline_func,
          package_path=package_path,
          pipeline_name=pipeline_name,
          pipeline_parameters=pipeline_parameters,
          type_check=type_check,
      )
      return

    local_temp_output_dir = path.join(path.dirname(package_path), "tmp.yaml")

    kfp_compiler.Compiler().compile(
        pipeline_func=pipeline_func,
        package_path=local_temp_output_dir,
        pipeline_name=pipeline_name,
        pipeline_parameters=pipeline_parameters,
        type_check=type_check,
    )

    original_pipeline_spec = self.parse_pipeline_spec_yaml(
        local_temp_output_dir
    )
    template_metadata = getattr(pipeline_func, "template_metadata", None)
    updated_pipeline_spec = self.merge_template_metadata_to_pipeline_spec_proto(
        template_metadata, original_pipeline_spec
    )
    updated_pipeline_spec_dict = json_format.MessageToDict(
        updated_pipeline_spec
    )

    with open(
        package_path,
        "w",
    ) as f:
      yaml.dump(updated_pipeline_spec_dict, f)

    os.remove(local_temp_output_dir)
