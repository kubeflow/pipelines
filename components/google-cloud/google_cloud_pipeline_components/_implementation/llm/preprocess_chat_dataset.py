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
"""KFP Component the preprocesses chat dataset before tokenization."""

from google_cloud_pipeline_components import _image
from kfp import dsl


@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def preprocess_chat_dataset(
    large_model_reference: str,
    input_dataset_uri: str,
    dataset_type: str,
    processed_dataset: dsl.OutputPath(dsl.Artifact),  # pytype: disable=invalid-annotation
    processed_dataset_uri: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    default_context: str = '',
    allow_local_files: bool = False,
):  # pylint: disable=g-doc-args
  # fmt: off
  """Preprocesses datasets before tokenization.

  For text datasets, this is a no-op.

  Args:
    large_model_reference: Name of the base model. Supported values are `text-bison@001`, `chat-bison@001`, `t5-small`, `t5-large`, `t5-xl` and `t5-xxl`. `text-bison@001`, `chat-bison@001` and `t5-small` are supported in ``us-central1` and `europe-west4`. `t5-large`, `t5-xl` and `t5-xxl` are only supported in `europe-west4`.
    input_dataset_uri: Path to an unprocessed JSONL dataset.
    default_context: Default context to apply to each example if a chat model is specified.
    allow_local_files: Whether input URIs can specify local file paths.
    is_prompt_dataset: Whether the input dataset contains prompts for inference. In this case, the last author in `messages` should be the `user`, and the output dataset will only contain `input_text`.

  Returns:
    processed_dataset: Processed chat dataset. Each example will contain fields `input_text`, and if the input dataset is not a prompt dataset example will also contain `output_text`.
    processed_dataset_uri: String pattern that can be used to find the processed dataset in downstream components.

  """
  # fmt: on
  # pylint: disable=g-import-not-at-top
  import dataclasses
  import json
  import os
  from typing import Any, Callable, List, Mapping
  import apache_beam as beam
  # pylint: enable=g-import-not-at-top

  # [ Define helper methods and classes for preprocessing
  # pylint: disable=invalid-name
  INPUT_TEXT_KEY = 'input_text'
  OUTPUT_TEXT_KEY = 'output_text'
  CONTEXT_KEY = 'context'
  MESSAGES_KEY = 'messages'
  CANDIDATE_0_KEY = 'candidate_0'
  CANDIDATE_1_KEY = 'candidate_1'
  CHOICE_KEY = 'choice'
  AUTHOR_KEY = 'author'
  CONTENT_KEY = 'content'
  AUTHOR_USER = 'user'
  AUTHOR_ASSISTANT = 'assistant'
  VALID_AUTHORS = {AUTHOR_USER, AUTHOR_ASSISTANT}
  SUPERVISED_DATASET = 'supervised'
  PROMPT_DATASET = 'prompt'
  PREFERENCE_DATASET = 'preference'
  VALID_DATASETS = {SUPERVISED_DATASET, PROMPT_DATASET, PREFERENCE_DATASET}
  # pylint: enable=invalid-name

  @dataclasses.dataclass
  class PromptSchema:
    global_prefix: str
    user_prefix: str
    user_postfix: str
    assistant_prefix: str
    assistant_postfix: str
    get_system_message: Callable[[str], str]  # pytype: disable=invalid-annotation

  def _get_chat_bison_001_system_message(context: str) -> str:
    return f'[SYSTEM]:{context}\n\n' if context else ''

  chat_bison_001_schema = PromptSchema(
      global_prefix=(
          'Only answer after [assistant] and never reply as [user]:\n'
      ),
      get_system_message=_get_chat_bison_001_system_message,
      user_prefix='[user]:',
      user_postfix='\n',
      assistant_prefix='[assistant]:',
      assistant_postfix='\n',
  )

  def _get_chat_llama_system_message(context: str) -> str:
    return f'<<SYS>>\n{context}\n<</SYS>>\n\n' if context else ''

  chat_llama_schema = PromptSchema(
      global_prefix='<s>[INST] ',
      get_system_message=_get_chat_llama_system_message,
      user_prefix='',
      user_postfix=' [/INST]',
      assistant_prefix=' ',
      assistant_postfix='</s><s>[INST] ',
  )

  MODEL_TO_SCHEMA_MAPPING = {  # pylint: disable=invalid-name
      'chat-bison@001': chat_bison_001_schema,
      'llama-2-7b-chat': chat_llama_schema,
      'llama-2-13b-chat': chat_llama_schema,
  }

  def get_gcs_path(input_path: str, allow_local_files: bool) -> str:
    """Gets the /gcs/ path for a given URI."""
    if input_path.startswith('gs://'):
      return input_path.replace('gs://', '/gcs/', 1)
    elif input_path.startswith('/gcs/') or allow_local_files:
      return input_path
    else:
      raise ValueError(
          f'Invalid Cloud storage URI {input_path}. '
          'Must start with `gs://` or `/gcs/`.'
      )

  def get_gs_path(input_path: str, allow_local_files: bool) -> str:
    """Gets the gs:// path for a given URI."""
    if input_path.startswith('/gcs/'):
      return input_path.replace('/gcs/', 'gs://', 1)
    elif input_path.startswith('gs://') or allow_local_files:
      return input_path
    else:
      raise ValueError(
          f'Invalid Cloud storage URI {input_path}. '
          'Must start with `gs://` or `/gcs/`.'
      )

  class JsonCoder(beam.coders.Coder):
    """A coder that encodes/decodes lines as JSON strings."""

    def encode(self, x):
      return json.dumps(x).encode('utf-8')

    def decode(self, x):
      return json.loads(x)

  class ChatDatasetProcessor(beam.DoFn):
    """Converts chat data from input format to the format expected by the model."""

    def __init__(
        self,
        default_context: str,
        prompt_schema: PromptSchema,
        dataset_type: str,
    ):
      self._default_context = default_context
      self._schema = prompt_schema
      self._dataset_type = dataset_type

    def _get_messages_or_fail(
        self, element: Mapping[str, Any]
    ) -> List[Mapping[str, str]]:
      messages = element.get(MESSAGES_KEY)
      if not messages:
        raise ValueError(
            'No messages present. Please include a non-empty '
            f'`messages` field in each line of dataset: {element}.'
        )
      elif messages[0].get(AUTHOR_KEY) != AUTHOR_USER:
        raise ValueError(f'First author must be the {AUTHOR_USER}: {element}')
      elif (
          self._dataset_type in {PROMPT_DATASET, PREFERENCE_DATASET}
          and messages[-1].get(AUTHOR_KEY) != AUTHOR_USER
      ):
        raise ValueError(
            f'Last author in the {self._dataset_type} dataset must be the'
            f' {AUTHOR_USER}: {element}'
        )
      elif (
          self._dataset_type == SUPERVISED_DATASET
          and messages[-1].get(AUTHOR_KEY) != AUTHOR_ASSISTANT
      ):
        raise ValueError(
            f'Last author in the {self._dataset_type} dataset must be the'
            f' {AUTHOR_ASSISTANT}: {element}'
        )
      return messages

    def _get_or_fail(self, message: Mapping[str, str], key: str) -> str:
      value = message.get(key)
      if not value and value != 0:
        raise ValueError(
            f'Each message must contain non-empty value for {key}. '
            f'Invalid message: {message}'
        )
      return value

    def _get_author_or_fail(self, message: Mapping[str, str]) -> str:
      author = self._get_or_fail(message, AUTHOR_KEY)
      if author not in VALID_AUTHORS:
        raise ValueError(
            'The `author` of each message needs to be from one of'
            f' {VALID_AUTHORS}. Got author = {author}.'
        )
      return author

    def process(self, element):
      context = element.get(CONTEXT_KEY, self._default_context)
      messages = self._get_messages_or_fail(element)

      message_history = [
          self._schema.global_prefix,
          self._schema.get_system_message(context),
      ]
      for message in messages:
        author = self._get_author_or_fail(message)
        content = self._get_or_fail(message, CONTENT_KEY)
        if author == AUTHOR_USER:
          message_history.append(
              f'{self._schema.user_prefix}{content}{self._schema.user_postfix}'
          )
        elif author == AUTHOR_ASSISTANT:
          message_history.append(self._schema.assistant_prefix)
          input_text = ''.join(message_history)
          # For training datasets yield an example for each user/assistant
          # exchange:
          if self._dataset_type == SUPERVISED_DATASET:
            yield {
                INPUT_TEXT_KEY: input_text.rstrip(),
                OUTPUT_TEXT_KEY: content,
            }
          message_history = [
              input_text,
              f'{content}{self._schema.assistant_postfix}',
          ]
        else:
          raise ValueError(
              f'Unknown author {author}. Must be one of {VALID_AUTHORS}.'
          )
      # For prompt and preference datasets, only yield an example after the
      # final user message:
      if self._dataset_type == PROMPT_DATASET:
        message_history.append(self._schema.assistant_prefix)
        input_text = ''.join(message_history)
        yield {INPUT_TEXT_KEY: input_text.rstrip()}
      elif self._dataset_type == PREFERENCE_DATASET:
        message_history.append(self._schema.assistant_prefix)
        input_text = ''.join(message_history)
        yield {
            INPUT_TEXT_KEY: input_text.rstrip(),
            CANDIDATE_0_KEY: self._get_or_fail(element, CANDIDATE_0_KEY),
            CANDIDATE_1_KEY: self._get_or_fail(element, CANDIDATE_1_KEY),
            CHOICE_KEY: self._get_or_fail(element, CHOICE_KEY),
        }

  # ]

  processed_dataset_uri = get_gcs_path(processed_dataset_uri, allow_local_files)

  # Reuse the input dataset if no preprocessing is needed.
  if large_model_reference.lower() not in MODEL_TO_SCHEMA_MAPPING:
    with open(processed_dataset_uri, 'w') as f:
      f.write(input_dataset_uri)
    return

  prompt_schema = MODEL_TO_SCHEMA_MAPPING[large_model_reference]

  # Provide gs:// paths for datasets processed by Beam.
  input_dataset_uri = get_gs_path(input_dataset_uri, allow_local_files)
  processed_dataset = get_gs_path(processed_dataset, allow_local_files)
  os.makedirs(processed_dataset, exist_ok=True)
  processed_dataset_prefix = os.path.join(processed_dataset, 'shard')
  dataset_type = dataset_type.lower()
  if dataset_type not in VALID_DATASETS:
    raise ValueError(
        f'Unknown dataset type {dataset_type}. Must be one of {VALID_DATASETS}.'
    )

  pipeline_options = (
      beam.options.pipeline_options.PipelineOptions.from_dictionary({
          'runner': 'DirectRunner',
      })
  )
  with beam.Pipeline(options=pipeline_options) as pipeline:
    _ = (
        pipeline
        | 'Read JSON from input dataset'
        >> beam.io.ReadFromText(input_dataset_uri, coder=JsonCoder())
        | 'Process chat dataset'
        >> beam.ParDo(
            ChatDatasetProcessor(
                default_context=default_context,
                prompt_schema=prompt_schema,
                dataset_type=dataset_type,
            )
        )
        | 'Write processed JSON to output file'
        >> beam.io.WriteToText(
            file_path_prefix=processed_dataset_prefix,
            file_name_suffix='.jsonl',
            coder=JsonCoder(),
        )
    )

  # Write file pattern that the tokenizer can use to find all processed files.
  with open(processed_dataset_uri, 'w') as f:
    processed_dataset_pattern = os.path.join(processed_dataset, '*.jsonl')
    f.write(processed_dataset_pattern)
