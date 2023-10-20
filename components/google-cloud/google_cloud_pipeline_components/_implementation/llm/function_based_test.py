"""Tests for function-based components."""

from absl.testing import parameterized
import re
from typing import NamedTuple

from google_cloud_pipeline_components._implementation.llm import function_based
import unittest

MachineSpec = NamedTuple(
    'MachineSpec',
    accelerator_type=str,
    accelerator_count=int,
    machine_type=str,
)


class FunctionBasedTest(parameterized.TestCase):

  @parameterized.named_parameters(
      {
          'testcase_name': 'gpu_default',
          'large_model_reference': 'OTTER',
          'use_gpu_defaults': True,
          'accelerator_type_override': None,
          'accelerator_count_override': None,
          'expected_machine_spec': MachineSpec(
              accelerator_type='NVIDIA_TESLA_A100',
              accelerator_count=2,
              machine_type='a2-highgpu-2g',
          ),
      },
      {
          'testcase_name': 'gpu_default_non_none',
          'large_model_reference': 'OTTER',
          'use_gpu_defaults': True,
          'accelerator_type_override': '',
          'accelerator_count_override': 0,
          'expected_machine_spec': MachineSpec(
              accelerator_type='NVIDIA_TESLA_A100',
              accelerator_count=2,
              machine_type='a2-highgpu-2g',
          ),
      },
      {
          'testcase_name': 'tpu_default',
          'large_model_reference': 'T5_SMALL',
          'use_gpu_defaults': False,
          'accelerator_type_override': None,
          'accelerator_count_override': None,
          'expected_machine_spec': MachineSpec(
              accelerator_type='TPU_V3',
              accelerator_count=32,
              machine_type='cloud-tpu',
          ),
      },
      {
          'testcase_name': 'accelerator_type_and_count_override',
          'large_model_reference': 'GECKO',
          'use_gpu_defaults': True,
          'accelerator_type_override': 'NVIDIA_A100_80GB',
          'accelerator_count_override': 8,
          'expected_machine_spec': MachineSpec(
              accelerator_type='NVIDIA_A100_80GB',
              accelerator_count=8,
              machine_type='a2-ultragpu-8g',
          ),
      },
  )
  def test_get_default_bulk_inference_machine_specs_good_params(
      self,
      large_model_reference,
      use_gpu_defaults,
      accelerator_type_override,
      accelerator_count_override,
      expected_machine_spec,
  ):
    self.assertEqual(
        function_based.get_default_bulk_inference_machine_specs.python_func(
            large_model_reference=large_model_reference,
            use_gpu_defaults=use_gpu_defaults,
            accelerator_type_override=accelerator_type_override,
            accelerator_count_override=accelerator_count_override,
        ),
        expected_machine_spec,
    )

  @parameterized.named_parameters(
      {
          'testcase_name': 'bad_model_name',
          'large_model_reference': 'not_a_model',
          'accelerator_type_override': None,
          'accelerator_count_override': None,
          'expected_error_regex': 'large_model_reference must be one of',
      },
      {
          'testcase_name': 'bad_accelerator_type',
          'large_model_reference': 'GECKO',
          'accelerator_type_override': 'NVIDIA_A100_40GB',
          'accelerator_count_override': 8,
          'expected_error_regex': 'accelerator_type_override must be one of',
      },
      {
          'testcase_name': 'too_many_a100_80gb',
          'large_model_reference': 'GECKO',
          'accelerator_type_override': 'NVIDIA_A100_80GB',
          'accelerator_count_override': 9,
          'expected_error_regex': (
              'Too many NVIDIA_A100_80GB requested. Must be <= 8'
          ),
      },
      {
          'testcase_name': 'too_few_a100_80gb',
          'large_model_reference': 'GECKO',
          'accelerator_type_override': 'NVIDIA_A100_80GB',
          'accelerator_count_override': -1,
          'expected_error_regex': 'accelerator_count must be at least 1',
      },
      {
          'testcase_name': 'override_missing',
          'large_model_reference': 'GECKO',
          'accelerator_type_override': 'NVIDIA_A100_80GB',
          'accelerator_count_override': None,
          'expected_error_regex': 'Accelerator type and count must both be set',
      },
  )
  def test_get_default_bulk_inference_machine_specs_bad_params(
      self,
      large_model_reference,
      accelerator_type_override,
      accelerator_count_override,
      expected_error_regex,
  ):
    with self.assertRaisesRegex(ValueError, re.escape(expected_error_regex)):
      function_based.get_default_bulk_inference_machine_specs.python_func(
          large_model_reference=large_model_reference,
          accelerator_type_override=accelerator_type_override,
          accelerator_count_override=accelerator_count_override,
      )

  def test_read_file(self):
    file = self.create_tempfile()
    with open(file, 'w') as f:
      f.write('foo')
    self.assertEqual(
        'foo', function_based.read_file.python_func(file.full_path)
    )

  def test_write_file(self):
    file = self.create_tempfile()
    function_based.write_file.python_func(file.full_path, 'foo')
    self.assertEqual(
        'foo', function_based.read_file.python_func(file.full_path)
    )

  def test_identity(self):
    self.assertEqual('foo', function_based.identity.python_func('foo'))


if __name__ == '__main__':
  unittest.main()
