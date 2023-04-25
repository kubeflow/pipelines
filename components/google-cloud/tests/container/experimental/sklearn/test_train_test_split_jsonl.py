"""Unit test for preprocessing component function in the text classification."""

import json
import os

from absl.testing import absltest

from google_cloud_pipeline_components.container.experimental.sklearn import train_test_split_jsonl as component


class ComponentTest(absltest.TestCase):
  """Unit test for component.py."""

  def setUp(self):
    super(ComponentTest, self).setUp()

    self._input_dir = self.create_tempdir().full_path
    self._output_dir = self.create_tempdir().full_path

    with open(os.path.join(self._input_dir, 'data.json'), 'w') as f:
      f.writelines(
          [
              '{{ "text": "foo{0}", "label": "bar{1}" }}\n'.format(i, i % 2)
              for i in range(100)
          ]
      )

  def test_split_dataset_into_train_and_validation(self):
    """Test case to cover component function."""
    component.split_dataset_into_train_and_validation(
        training_data_path=os.path.join(self._output_dir, 'train.json'),
        validation_data_path=os.path.join(self._output_dir, 'validation.json'),
        input_data_path=os.path.join(self._input_dir, 'data.json'),
        validation_split=0.05,
        random_seed=0,
    )

    with open(os.path.join(self._output_dir, 'train.json'), 'r') as f:
      self.assertLen(
          [json.loads(line) for line in f.read().splitlines()], int(100 * 0.95)
      )

    with open(os.path.join(self._output_dir, 'validation.json'), 'r') as f:
      self.assertCountEqual(
          [json.loads(line) for line in f.read().splitlines()],
          [
              {'text': 'foo2', 'label': 'bar0'},
              {'text': 'foo26', 'label': 'bar0'},
              {'text': 'foo55', 'label': 'bar1'},
              {'text': 'foo75', 'label': 'bar1'},
              {'text': 'foo86', 'label': 'bar0'},
          ],
      )
