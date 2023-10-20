# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
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

"""Optimization AI Inference and AutoSxS pipeline function."""

from typing import Optional

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components._implementation.llm import arbiter_preprocess
from google_cloud_pipeline_components._implementation.llm import autosxs_arbiter
from google_cloud_pipeline_components._implementation.llm import bulk_inferrer
from google_cloud_pipeline_components._implementation.llm import function_based as gcpc_function_based
from google_cloud_pipeline_components._implementation.llm import private_text_importer
from kfp import dsl

# Arbiter constants. Chosen as per
# https://docs.google.com/document/d/1rWDb6tsZhG_fuI98NaZtqB9V9a4ZJCmoV7ABU6_CZQI.
MAX_SUPPORTED_EXAMPLES = 500


@dsl.pipeline(name='predictions_pipeline')
def _get_predictions(
    name: str,
    project: str,
    location: str,
    prompt_dataset: str,
    prompt_column: str,
    large_model_reference: str,
    importer_machine_type: str,
    instruction: str,
    use_gpu_defaults: bool,
    accelerator_type_override: Optional[str],
    accelerator_count_override: Optional[int],
    model_checkpoint_override: str,
    prompt_sequence_length: int,
    target_sequence_length: int,
    max_num_input_examples: int,
    responses: str,
    response_column: str,
) -> str:
  """Preprocesses data and makes predictions for a given model."""
  # TODO: b/245451217 - Use fan-in if/else instead of an output file.
  predictions_output_path = (
      f'{dsl.PIPELINE_ROOT_PLACEHOLDER}{dsl.PIPELINE_TASK_ID_PLACEHOLDER}'
      f'/{name}_responses'
  )
  predictions_output_path = gcpc_function_based.identity(
      value=predictions_output_path,
  ).set_display_name('Response Directory')
  with dsl.If(responses == '', name='Inference Required'):  # pylint: disable=g-explicit-bool-comparison
    dataset_importer = private_text_importer.PrivateTextImporter(
        project=project,
        location=location,
        input_text=prompt_dataset,
        inputs_field_name=prompt_column,
        targets_field_name='',  # ignore targets_field_name
        large_model_reference=large_model_reference,
        machine_type=importer_machine_type,
        instruction=instruction,
        max_num_input_examples=max_num_input_examples,
    ).set_display_name(f'Dataset {name} Importer')

    # TODO: b/293360156 - Combine these two components for resolving specs
    model_bulk_inferrer_machine_spec = (
        gcpc_function_based.get_default_bulk_inference_machine_specs(
            large_model_reference=large_model_reference,
            use_gpu_defaults=use_gpu_defaults,
            accelerator_type_override=accelerator_type_override,
            accelerator_count_override=accelerator_count_override,
        ).set_display_name(f'Model {name} Bulk Inferrer Machine Spec Resolver')
    )
    bulk_inferrer_image = gcpc_function_based.resolve_private_image_uri(
        image_name='infer',
        accelerator_type=model_bulk_inferrer_machine_spec.outputs[
            'accelerator_type'
        ],
        accelerator_count=model_bulk_inferrer_machine_spec.outputs[
            'accelerator_count'
        ],
    ).set_display_name(f'Model {name} Bulk Inferrer Image URI Resolver')

    reference_model_metadata = (
        gcpc_function_based.resolve_reference_model_metadata(
            large_model_reference=large_model_reference,
            reference_model_path=model_checkpoint_override,
        ).set_display_name(f'Model {name} Metadata')
    )

    inferrer = (
        bulk_inferrer.BulkInferrer(
            project=project,
            location=location,
            inputs_sequence_length=prompt_sequence_length,
            targets_sequence_length=target_sequence_length,
            accelerator_type=model_bulk_inferrer_machine_spec.outputs[
                'accelerator_type'
            ],
            accelerator_count=model_bulk_inferrer_machine_spec.outputs[
                'accelerator_count'
            ],
            machine_type=(
                model_bulk_inferrer_machine_spec.outputs['machine_type']
            ),
            image_uri=bulk_inferrer_image.output,
            dataset_split='all',  # default of TextImporter
            large_model_reference=reference_model_metadata.outputs[
                'large_model_reference'
            ],
            input_model=(
                reference_model_metadata.outputs['reference_model_path']
            ),
            input_dataset_path=dataset_importer.outputs['imported_data_path'],
        )
        .set_retry(num_retries=5, backoff_duration='30s', backoff_factor=2)
        .set_display_name(f'Bulk Inferrer {name}')
    )
    gcpc_function_based.write_file(
        path=predictions_output_path.output,
        content=inferrer.outputs['output_prediction_gcs_path'],
    )
  with dsl.Else(name='Responses Provided'):
    arbiter_preprocess.arbiter_preprocess(
        # How to handle wildcard in gcs
        # how to set project id in direct runner
        prediction_uris=responses,
        output_path=predictions_output_path.output,
        prompt_column=prompt_column,
        response_column=response_column,
    ).set_display_name('Preprocess Predictions')
  return predictions_output_path.output


# pylint: disable=dangerous-default-value,g-bare-generic
@dsl.pipeline(
    name='autosxs',
    description='Determines the SxS winrate between two models.',
)
def autosxs_pipeline(
    prompt_dataset: str,
    prompt_column: str,
    task: str,
    large_model_a_reference: str = '',
    large_model_b_reference: str = '',
    instruction: str = '',
    prompt_sequence_length: int = 512,
    target_sequence_length: int = 64,
    importer_machine_type: str = 'e2-highmem-8',
    use_gpu_defaults: bool = False,
    model_a_responses: str = '',
    model_b_responses: str = '',
    response_column: str = '',
    model_a_checkpoint_override: str = '',
    model_b_checkpoint_override: str = '',
    accelerator_type_model_a_override: Optional[str] = None,
    accelerator_count_model_a_override: Optional[int] = None,
    accelerator_type_model_b_override: Optional[str] = None,
    accelerator_count_model_b_override: Optional[int] = None,
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
    location: str = _placeholders.LOCATION_PLACEHOLDER,
    judgments_format: str = 'jsonl',
    bigquery_destination_prefix: str = '',
):
  """Evaluates two models side-by-side using an arbiter model.

  Args:
    prompt_dataset: A GCS path to a JSONL dataset containing prompts to use for
      evaluation.
    prompt_column: The column containing prompts.
    task: Task to evaluate for.
    large_model_a_reference: The base instruction-tuned model used by Model A.
      Valid options are 'PALM_TINY', 'GECKO', 'OTTER', 'BISON', 'T5_SMALL',
      'T5_LARGE', 'T5_XL', and 'T5_XXL'. This parameter is optional if Model A
      responses are specified.
    large_model_b_reference: The base instruction-tuned model used by Model B.
      Valid options are 'PALM_TINY', 'GECKO', 'OTTER', 'BISON', 'T5_SMALL',
      'T5_LARGE', 'T5_XL', and 'T5_XXL'. This parameter is optional if Model B
      responses are specified.
    instruction: An instruction to prepend to all prompts. If no instruction is
      given, one will be generated based on the task.
    prompt_sequence_length: The maximum number of tokens allowed in a prompt.
    target_sequence_length: The maximum number of tokens allowed in a response.
    importer_machine_type: The machine type to use for preprocessing the data.
    use_gpu_defaults: If True, any components that use accelerators will default
      to using GPUs instead of TPUs.
    model_a_responses: A JSONL in GCS or BigQuery URI to a table containing
      pre-made responses from Model A for the prompt dataset.
    model_b_responses: A JSONL in GCS or BigQuery URI to a table containing
      pre-made responses from Model B for the prompt dataset.
    response_column: The column containing responses. Required if any model
      response tables are provided.
    model_a_checkpoint_override: The checkpoint to use for Model A if using a
      tuned model. If no checkpint is specified, the base image will be used.
    model_b_checkpoint_override: The checkpoint to use for Model B if using a
      tuned model. If no checkpint is specified, the base image will be used.
    accelerator_type_model_a_override: The accelerator type to use for Model A
      when accelerators are required. Valid options are: 'TPU_V2', 'TPU_V3',
      'NVIDIA_TESLA_A100', and 'NVIDIA_A100_80GB'.
    accelerator_count_model_a_override: The number of accelerators to use for
      Model A when accelerators are required. GPUs are counted by chip, TPUs are
      counted by core.
    accelerator_type_model_b_override: The accelerator type to use for Model B
      when accelerators are required. Valid options are: 'TPU_V2', 'TPU_V3',
      'NVIDIA_TESLA_A100', and 'NVIDIA_A100_80GB'.
    accelerator_count_model_b_override: The number of accelerators to use for
      Model B when accelerators are required. GPUs are counted by chip, TPUs are
      counted by core.
    project: Project used to run custom jobs. Default is the same project used
      to run the pipeline.
    location: Location used to run custom jobs. Default is the same location
      used to run the pipeline.
    judgments_format: The format to write judgments to. Can be either 'json' or
      'bigquery'.
    bigquery_destination_prefix: BigQuery table to write judgments to if the
      specified format is 'bigquery'.
  """
  instruction = gcpc_function_based.generate_default_instruction(
      task=task,
      target_sequence_length=target_sequence_length,
      instruction_override=instruction,
  ).set_display_name('Get Instruction')

  inferrer_a = _get_predictions(
      name='A',
      project=project,
      location=location,
      prompt_dataset=prompt_dataset,
      prompt_column=prompt_column,
      large_model_reference=large_model_a_reference,
      importer_machine_type=importer_machine_type,
      instruction=instruction.output,
      use_gpu_defaults=use_gpu_defaults,
      accelerator_type_override=accelerator_type_model_a_override,
      accelerator_count_override=accelerator_count_model_a_override,
      model_checkpoint_override=model_a_checkpoint_override,
      prompt_sequence_length=prompt_sequence_length,
      target_sequence_length=target_sequence_length,
      max_num_input_examples=MAX_SUPPORTED_EXAMPLES,
      responses=model_a_responses,
      response_column=response_column,
  ).set_display_name('Model A Responses')
  model_a_prediction_dir = gcpc_function_based.read_file(
      path=inferrer_a.output
  ).set_display_name('Read Model A Responses')

  inferrer_b = _get_predictions(
      name='B',
      project=project,
      location=location,
      prompt_dataset=prompt_dataset,
      prompt_column=prompt_column,
      large_model_reference=large_model_b_reference,
      importer_machine_type=importer_machine_type,
      instruction=instruction.output,
      use_gpu_defaults=use_gpu_defaults,
      accelerator_type_override=accelerator_type_model_b_override,
      accelerator_count_override=accelerator_count_model_b_override,
      model_checkpoint_override=model_b_checkpoint_override,
      prompt_sequence_length=prompt_sequence_length,
      target_sequence_length=target_sequence_length,
      max_num_input_examples=MAX_SUPPORTED_EXAMPLES,
      responses=model_b_responses,
      response_column=response_column,
  ).set_display_name('Model B Responses')
  model_b_prediction_dir = gcpc_function_based.read_file(
      path=inferrer_b.output
  ).set_display_name('Read Model B Responses')

  autosxs_arbiter.autosxs_arbiter(
      model_a_prediction_dir=model_a_prediction_dir.output,
      model_b_prediction_dir=model_b_prediction_dir.output,
      instruction=instruction.output,
      task=task,
      judgments_format=judgments_format,
      bigquery_destination_prefix=bigquery_destination_prefix,
  ).set_display_name('AutoSxS Arbiter')
