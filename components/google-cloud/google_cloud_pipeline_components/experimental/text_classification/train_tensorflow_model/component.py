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

from kfp import dsl
from typing import Dict, Optional
from google_cloud_pipeline_components.types.artifact_types import VertexDataset
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp.dsl import Output
from kfp.dsl import Input
from kfp.dsl import Metrics
from kfp.dsl import Artifact


@dsl.container_component
def train_tensorflow_text_classification_model(
    preprocessed_training_data_path: Input[Artifact],
    preprocessed_validation_data_path: Input[Artifact],
    class_names: list,
    trained_model_path: Output[Artifact],
    model_name: Optional[str] = 'small_bert/bert_en_uncased_L-2_H-128_A-2',
    batch_size: Optional[int] = 256,
    num_epochs: Optional[int] = 10,
    learning_rate: Optional[float] = 0.0003,
    random_seed: Optional[int] = 0,
):
  # fmt: off
  """
  Creates a trained text classification TensorFlow model.
  Args:
    preprocessed_training_data_path (str):
        Path for the training data for text classification in jsonl format. The data must contain
        'text' (input) and 'label' (output for a prediction) fields.
    preprocessed_validation_data_path (str):
        Path for the validation data (same format as the preprocessed_training_data).
    trained_model_path (str):
        Output path for a trained TensorFlow SavedModel compliant with
        https://github.com/tensorflow/tensorflow/blob/v2.11.0/tensorflow/python/keras/saving/save.py
    class_names (Sequence[str]):
        Sequence of strings of categories for classification.
    model_name (Optional[str]):
        Name of pre-trained BERT encoder model (https://tfhub.dev/google/collections/bert/1) to be used.
        Eligible model_name:
        - bert_en_uncased_L-12_H-768_A-12
        - bert_en_cased_L-12_H-768_A-12
        - bert_multi_cased_L-12_H-768_A-12
        - small_bert/bert_en_uncased_L-2_H-128_A-2 (default)
        - small_bert/bert_en_uncased_L-2_H-256_A-4
        - small_bert/bert_en_uncased_L-2_H-512_A-8
        - small_bert/bert_en_uncased_L-2_H-768_A-12
        - small_bert/bert_en_uncased_L-4_H-128_A-2
        - small_bert/bert_en_uncased_L-4_H-256_A-4
        - small_bert/bert_en_uncased_L-4_H-512_A-8
        - small_bert/bert_en_uncased_L-4_H-768_A-12
        - small_bert/bert_en_uncased_L-6_H-128_A-2
        - small_bert/bert_en_uncased_L-6_H-256_A-4
        - small_bert/bert_en_uncased_L-6_H-512_A-8
        - small_bert/bert_en_uncased_L-6_H-768_A-12
        - small_bert/bert_en_uncased_L-8_H-128_A-2
        - small_bert/bert_en_uncased_L-8_H-256_A-4
        - small_bert/bert_en_uncased_L-8_H-512_A-8
        - small_bert/bert_en_uncased_L-8_H-768_A-12
        - small_bert/bert_en_uncased_L-10_H-128_A-2
        - small_bert/bert_en_uncased_L-10_H-256_A-4
        - small_bert/bert_en_uncased_L-10_H-512_A-8
        - small_bert/bert_en_uncased_L-10_H-768_A-12
        - small_bert/bert_en_uncased_L-12_H-128_A-2
        - small_bert/bert_en_uncased_L-12_H-256_A-4
        - small_bert/bert_en_uncased_L-12_H-512_A-8
        - small_bert/bert_en_uncased_L-12_H-768_A-12
        - albert_en_base
        - electra_small
        - electra_base
        - experts_pubmed
        - experts_wiki_books
        - talking-heads_base
    batch_size (Optional[int]):
        A number of samples processed before the model is updated (must be >= 1 and <= number of
        samples in the dataset).
    num_epochs (Optional[int]):
        Number of training iterations over data.
    learning_rate (Optional[float]): Learning rate controls how quickly the model is adapted to
        the problem, often in the range between 0.0 and 1.0.
    random_seed (Optional[int]):
        The global random seed to ensure the system gets a unique random sequence that is
        deterministic (https://www.tensorflow.org/api_docs/python/tf/random/set_seed).
  """
  # fmt: on
  return dsl.ContainerSpec(
      image='us-docker.pkg.dev/vertex-ai/ready-to-go-text-classification/training:v0.1',
      command=[
          'python3',
          '/pipelines/component/src/component.py',
          '--preprocessed-training-data-path',
          preprocessed_training_data_path.path,
          '--preprocessed-validation-data-path',
          preprocessed_validation_data_path.path,
          '--class-names',
          class_names,
          '--model-name',
          model_name,
          '--batch-size',
          batch_size,
          '--num-epochs',
          num_epochs,
          '--learning-rate',
          learning_rate,
          '--random-seed',
          random_seed,
          '--trained-model-path',
          trained_model_path.path,
      ],
      args=[],
  )
