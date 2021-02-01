# Copyright 2018-2019 Google LLC
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
from typing import List

import kfp
import kfp.compiler as compiler
import kfp.dsl as dsl
import json
import os
import shutil
import subprocess
import sys
import zipfile
import tarfile
import tempfile
import unittest
import yaml

from kfp.compiler import Compiler
from kfp.dsl._component import component
from kfp.dsl import ContainerOp, pipeline, PipelineParam
from kfp.dsl.types import Integer, InconsistentTypeException
from kubernetes.client import V1Toleration, V1Affinity, V1NodeSelector, V1NodeSelectorRequirement, V1NodeSelectorTerm, \
  V1NodeAffinity, V1PodDNSConfig, V1PodDNSConfigOption


def some_op():
  return dsl.ContainerOp(
      name='sleep',
      image='busybox',
      command=['sleep 1'],
  )


class TestCompiler(unittest.TestCase):
  # Define the places of samples covered by unit tests.
  core_sample_path = os.path.join(os.path.dirname(__file__), '..', '..', '..',
                                  '..', 'samples', 'core',)

  def test_operator_to_template(self):
    """Test converting operator to template"""

    from kubernetes import client as k8s_client

    def my_pipeline(msg1, json, kind, msg2='value2'):
      op = dsl.ContainerOp(name='echo', image='image', command=['sh', '-c'],
                           arguments=['echo %s %s | tee /tmp/message.txt' % (msg1, msg2)],
                           file_outputs={'merged': '/tmp/message.txt'}) \
        .add_volume_mount(k8s_client.V1VolumeMount(
          mount_path='/secret/gcp-credentials',
          name='gcp-credentials')) \
        .add_env_variable(k8s_client.V1EnvVar(
          name='GOOGLE_APPLICATION_CREDENTIALS',
          value='/secret/gcp-credentials/user-gcp-sa.json'))
      res = dsl.ResourceOp(
        name="test-resource",
        k8s_resource=k8s_client.V1PersistentVolumeClaim(
          api_version="v1",
          kind=kind,
          metadata=k8s_client.V1ObjectMeta(
            name="resource"
          )
        ),
        attribute_outputs={"out": json}
      )
      golden_output = {
        'container': {
          'image': 'image',
          'args': [
            'echo {{inputs.parameters.msg1}} {{inputs.parameters.msg2}} | tee /tmp/message.txt'
          ],
          'command': ['sh', '-c'],
          'env': [
            {
              'name': 'GOOGLE_APPLICATION_CREDENTIALS',
              'value': '/secret/gcp-credentials/user-gcp-sa.json'
            }
          ],
          'volumeMounts':[
            {
              'mountPath': '/secret/gcp-credentials',
              'name': 'gcp-credentials',
            }
          ]
        },
        'inputs': {'parameters':
          [
            {'name': 'msg1'},
            {'name': 'msg2'},
          ]},
        'name': 'echo',
        'outputs': {
          'artifacts': [
            {
              'name': 'echo-merged',
              'path': '/tmp/message.txt',
            },
          ],
          'parameters': [
            {'name': 'echo-merged',
            'valueFrom': {'path': '/tmp/message.txt'}
            }],
        }
      }
      res_output = {
        'inputs': {
          'parameters': [{
            'name': 'json'
          }, {
            'name': 'kind'
          }]
        },
        'name': 'test-resource',
        'outputs': {
          'parameters': [{
            'name': 'test-resource-manifest',
            'valueFrom': {
              'jsonPath': '{}'
            }
          }, {
            'name': 'test-resource-name',
            'valueFrom': {
              'jsonPath': '{.metadata.name}'
            }
          }, {
            'name': 'test-resource-out',
            'valueFrom': {
              'jsonPath': '{{inputs.parameters.json}}'
            }
          }]
        },
        'resource': {
          'action': 'create',
          'manifest': (
            "apiVersion: v1\n"
            "kind: '{{inputs.parameters.kind}}'\n"
            "metadata:\n"
            "  name: resource\n"
          )
        }
      }

      self.maxDiff = None
      self.assertEqual(golden_output, compiler._op_to_template._op_to_template(op))
      self.assertEqual(res_output, compiler._op_to_template._op_to_template(res))

    kfp.compiler.Compiler()._compile(my_pipeline)

  def _get_yaml_from_zip(self, zip_file):
    with zipfile.ZipFile(zip_file, 'r') as zip:
      with open(zip.extract(zip.namelist()[0]), 'r') as yaml_file:
        return yaml.safe_load(yaml_file)

  def _get_yaml_from_tar(self, tar_file):
    with tarfile.open(tar_file, 'r:gz') as tar:
      return yaml.safe_load(tar.extractfile(tar.getmembers()[0]))

  def test_basic_workflow(self):
    """Test compiling a basic workflow."""

    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    sys.path.append(test_data_dir)
    import basic
    tmpdir = tempfile.mkdtemp()
    package_path = os.path.join(tmpdir, 'workflow.zip')
    try:
      compiler.Compiler().compile(basic.save_most_frequent_word, package_path)
      with open(os.path.join(test_data_dir, 'basic.yaml'), 'r') as f:
        golden = yaml.safe_load(f)
      compiled = self._get_yaml_from_zip(package_path)

      for workflow in golden, compiled:
        del workflow['metadata']
        for template in workflow['spec']['templates']:
          template.pop('metadata', None)

      self.maxDiff = None
      # Comment next line for generating golden yaml.
      self.assertEqual(golden, compiled)
    finally:
      # Replace next line with commented line for gathering golden yaml.
      shutil.rmtree(tmpdir)
      # print(tmpdir)

  def test_basic_workflow_without_decorator(self):
    """Test compiling a workflow and appending pipeline params."""
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    sys.path.append(test_data_dir)
    import basic_no_decorator
    tmpdir = tempfile.mkdtemp()
    try:
      compiled_workflow = compiler.Compiler().create_workflow(
          basic_no_decorator.save_most_frequent_word,
          'Save Most Frequent',
          'Get Most Frequent Word and Save to GCS',
          [
            basic_no_decorator.message_param,
            basic_no_decorator.output_path_param
          ])
      with open(os.path.join(test_data_dir, 'basic_no_decorator.yaml'), 'r') as f:
        golden = yaml.safe_load(f)

      for workflow in golden, compiled_workflow:
        del workflow['metadata']

      self.assertEqual(golden, compiled_workflow)
    finally:
      shutil.rmtree(tmpdir)

  def test_composing_workflow(self):
    """Test compiling a simple workflow, and a bigger one composed from the simple one."""

    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    sys.path.append(test_data_dir)
    import compose
    tmpdir = tempfile.mkdtemp()
    try:
      # First make sure the simple pipeline can be compiled.
      simple_package_path = os.path.join(tmpdir, 'simple.zip')
      compiler.Compiler().compile(compose.save_most_frequent_word, simple_package_path)

      # Then make sure the composed pipeline can be compiled and also compare with golden.
      compose_package_path = os.path.join(tmpdir, 'compose.zip')
      compiler.Compiler().compile(compose.download_save_most_frequent_word, compose_package_path)
      with open(os.path.join(test_data_dir, 'compose.yaml'), 'r') as f:
        golden = yaml.safe_load(f)
      compiled = self._get_yaml_from_zip(compose_package_path)

      for workflow in golden, compiled:
        del workflow['metadata']
        for template in workflow['spec']['templates']:
          template.pop('metadata', None)

      self.maxDiff = None
      # Comment next line for generating golden yaml.
      self.assertEqual(golden, compiled)
    finally:
      # Replace next line with commented line for gathering golden yaml.
      shutil.rmtree(tmpdir)
      # print(tmpdir)

  def _test_py_compile_zip(self, file_base_name):
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    py_file = os.path.join(test_data_dir, file_base_name + '.py')
    tmpdir = tempfile.mkdtemp()
    try:
      target_zip = os.path.join(tmpdir, file_base_name + '.zip')
      subprocess.check_call([
          'dsl-compile', '--py', py_file, '--output', target_zip])
      with open(os.path.join(test_data_dir, file_base_name + '.yaml'), 'r') as f:
        golden = yaml.safe_load(f)
      compiled = self._get_yaml_from_zip(target_zip)

      for workflow in golden, compiled:
        del workflow['metadata']
        for template in workflow['spec']['templates']:
          template.pop('metadata', None)

      self.maxDiff = None
      self.assertEqual(golden, compiled)
    finally:
      shutil.rmtree(tmpdir)

  def _test_py_compile_targz(self, file_base_name):
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    py_file = os.path.join(test_data_dir, file_base_name + '.py')
    tmpdir = tempfile.mkdtemp()
    try:
      target_tar = os.path.join(tmpdir, file_base_name + '.tar.gz')
      subprocess.check_call([
          'dsl-compile', '--py', py_file, '--output', target_tar])
      with open(os.path.join(test_data_dir, file_base_name + '.yaml'), 'r') as f:
        golden = yaml.safe_load(f)
      compiled = self._get_yaml_from_tar(target_tar)

      for workflow in golden, compiled:
        del workflow['metadata']
        for template in workflow['spec']['templates']:
          template.pop('metadata', None)

      self.maxDiff = None
      self.assertEqual(golden, compiled)
    finally:
      shutil.rmtree(tmpdir)

  def _test_py_compile_yaml(self, file_base_name):
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    py_file = os.path.join(test_data_dir, file_base_name + '.py')
    tmpdir = tempfile.mkdtemp()
    try:
      target_yaml = os.path.join(tmpdir, file_base_name + '-pipeline.yaml')
      subprocess.check_call([
          'dsl-compile', '--py', py_file, '--output', target_yaml])
      with open(os.path.join(test_data_dir, file_base_name + '.yaml'), 'r') as f:
        golden = yaml.safe_load(f)

      with open(os.path.join(test_data_dir, target_yaml), 'r') as f:
        compiled = yaml.safe_load(f)

      for workflow in golden, compiled:
        del workflow['metadata']
        for template in workflow['spec']['templates']:
          template.pop('metadata', None)

      self.maxDiff = None
      self.assertEqual(golden, compiled)
    finally:
      shutil.rmtree(tmpdir)

  def _test_sample_py_compile_yaml(self, file_base_name):
    # Jump back to sample dir for sample python file.
    sample_data_dir = os.path.join(self.core_sample_path, file_base_name)
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    py_file = os.path.join(sample_data_dir, file_base_name + '.py')
    tmpdir = tempfile.mkdtemp()
    try:
      target_yaml = os.path.join(tmpdir, file_base_name + '-pipeline.yaml')
      subprocess.check_call(
          ['dsl-compile', '--py', py_file, '--output', target_yaml])
      with open(os.path.join(test_data_dir, file_base_name + '.yaml'),
                'r') as f:
        golden = yaml.safe_load(f)

      with open(os.path.join(test_data_dir, target_yaml), 'r') as f:
        compiled = yaml.safe_load(f)

      for workflow in golden, compiled:
        del workflow['metadata']
        for template in workflow['spec']['templates']:
          template.pop('metadata', None)

      self.maxDiff = None
      self.assertEqual(golden, compiled)
    finally:
      shutil.rmtree(tmpdir)

  def test_py_compile_basic(self):
    """Test basic sequential pipeline."""
    self._test_py_compile_zip('basic')

  def test_py_compile_with_sidecar(self):
    """Test pipeline with sidecar."""
    self._test_py_compile_yaml('sidecar')

  def test_py_compile_with_pipelineparams(self):
    """Test pipeline with multiple pipeline params."""
    self._test_py_compile_yaml('pipelineparams')

  def test_py_compile_with_opsgroups(self):
    """Test pipeline with multiple opsgroups."""
    self._test_py_compile_yaml('opsgroups')

  def test_py_compile_condition(self):
    """Test a pipeline with conditions."""
    self._test_py_compile_zip('coin')

  def test_py_compile_default_value(self):
    """Test a pipeline with a parameter with default value."""
    self._test_py_compile_targz('default_value')

  def test_py_volume(self):
    """Test a pipeline with a volume and volume mount."""
    self._test_py_compile_yaml('volume')

  def test_py_retry_policy(self):
      """Test retry policy is set."""

      policy = 'Always'

      def my_pipeline():
        some_op().set_retry(2, policy)

      workflow = kfp.compiler.Compiler()._compile(my_pipeline)
      name_to_template = {template['name']: template for template in workflow['spec']['templates']}
      main_dag_tasks = name_to_template[workflow['spec']['entrypoint']]['dag']['tasks']
      template = name_to_template[main_dag_tasks[0]['template']]

      self.assertEqual(template['retryStrategy']['retryPolicy'], policy)

  def test_py_retry_policy_invalid(self):
      def my_pipeline():
          some_op().set_retry(2, 'Invalid')

      with self.assertRaises(ValueError):
          kfp.compiler.Compiler()._compile(my_pipeline)

  def test_py_retry(self):
    """Test retry functionality."""
    number_of_retries = 137
    def my_pipeline():
      some_op().set_retry(number_of_retries)

    workflow = kfp.compiler.Compiler()._compile(my_pipeline)
    name_to_template = {template['name']: template for template in workflow['spec']['templates']}
    main_dag_tasks = name_to_template[workflow['spec']['entrypoint']]['dag']['tasks']
    template = name_to_template[main_dag_tasks[0]['template']]

    self.assertEqual(template['retryStrategy']['limit'], number_of_retries)

  def test_affinity(self):
    """Test affinity functionality."""
    exp_affinity = {
      'affinity': {
        'nodeAffinity': {
          'requiredDuringSchedulingIgnoredDuringExecution': {
            'nodeSelectorTerms': [
              {'matchExpressions': [
                {
                  'key': 'beta.kubernetes.io/instance-type',
                  'operator': 'In',
                  'values': ['p2.xlarge']}
              ]
              }]
          }}
      }
    }
    def my_pipeline():
      affinity = V1Affinity(
        node_affinity=V1NodeAffinity(
          required_during_scheduling_ignored_during_execution=V1NodeSelector(
            node_selector_terms=[V1NodeSelectorTerm(
              match_expressions=[V1NodeSelectorRequirement(
                key='beta.kubernetes.io/instance-type', operator='In', values=['p2.xlarge'])])])))
      some_op().add_affinity(affinity)

    workflow = kfp.compiler.Compiler()._compile(my_pipeline)

    self.assertEqual(workflow['spec']['templates'][1]['affinity'], exp_affinity['affinity'])

  def test_py_image_pull_secrets(self):
    """Test pipeline imagepullsecret."""
    self._test_sample_py_compile_yaml('imagepullsecrets')

  def test_py_timeout(self):
    """Test pipeline timeout."""
    self._test_py_compile_yaml('timeout')

  def test_py_recursive_do_while(self):
    """Test pipeline recursive."""
    self._test_py_compile_yaml('recursive_do_while')

  def test_py_recursive_while(self):
    """Test pipeline recursive."""
    self._test_py_compile_yaml('recursive_while')

  def test_py_resourceop_basic(self):
    """Test pipeline resourceop_basic."""
    self._test_py_compile_yaml('resourceop_basic')

  def test_py_volumeop_basic(self):
    """Test pipeline volumeop_basic."""
    self._test_py_compile_yaml('volumeop_basic')

  def test_py_volumeop_parallel(self):
    """Test pipeline volumeop_parallel."""
    self._test_py_compile_yaml('volumeop_parallel')

  def test_py_volumeop_dag(self):
    """Test pipeline volumeop_dag."""
    self._test_py_compile_yaml('volumeop_dag')

  def test_py_volume_snapshotop_sequential(self):
    """Test pipeline volume_snapshotop_sequential."""
    self._test_py_compile_yaml('volume_snapshotop_sequential')

  def test_py_volume_snapshotop_rokurl(self):
    """Test pipeline volumeop_sequential."""
    self._test_py_compile_yaml('volume_snapshotop_rokurl')

  def test_py_volumeop_sequential(self):
    """Test pipeline volumeop_sequential."""
    self._test_py_compile_yaml('volumeop_sequential')

  def test_py_param_substitutions(self):
    """Test pipeline param_substitutions."""
    self._test_py_compile_yaml('param_substitutions')

  def test_py_param_op_transform(self):
    """Test pipeline param_op_transform."""
    self._test_py_compile_yaml('param_op_transform')

  def test_py_preemptible_gpu(self):
    """Test preemptible GPU/TPU sample."""
    self._test_sample_py_compile_yaml('preemptible_tpu_gpu')

  def test_type_checking_with_consistent_types(self):
    """Test type check pipeline parameters against component metadata."""
    @component
    def a_op(field_m: {'GCSPath': {'path_type': 'file', 'file_type':'tsv'}}, field_o: Integer()):
      return ContainerOp(
          name = 'operator a',
          image = 'gcr.io/ml-pipeline/component-b',
          arguments = [
              '--field-l', field_m,
              '--field-o', field_o,
          ],
      )

    @pipeline(
        name='p1',
        description='description1'
    )
    def my_pipeline(a: {'GCSPath': {'path_type':'file', 'file_type': 'tsv'}}='good', b: Integer()=12):
      a_op(field_m=a, field_o=b)

    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    sys.path.append(test_data_dir)
    tmpdir = tempfile.mkdtemp()
    try:
      simple_package_path = os.path.join(tmpdir, 'simple.tar.gz')
      compiler.Compiler().compile(my_pipeline, simple_package_path, type_check=True)

    finally:
      shutil.rmtree(tmpdir)

  def test_type_checking_with_inconsistent_types(self):
    """Test type check pipeline parameters against component metadata."""
    @component
    def a_op(field_m: {'GCSPath': {'path_type': 'file', 'file_type':'tsv'}}, field_o: Integer()):
      return ContainerOp(
          name = 'operator a',
          image = 'gcr.io/ml-pipeline/component-b',
          arguments = [
              '--field-l', field_m,
              '--field-o', field_o,
          ],
      )

    @pipeline(
        name='p1',
        description='description1'
    )
    def my_pipeline(a: {'GCSPath': {'path_type':'file', 'file_type': 'csv'}}='good', b: Integer()=12):
      a_op(field_m=a, field_o=b)

    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    sys.path.append(test_data_dir)
    tmpdir = tempfile.mkdtemp()
    try:
      simple_package_path = os.path.join(tmpdir, 'simple.tar.gz')
      with self.assertRaises(InconsistentTypeException):
        compiler.Compiler().compile(my_pipeline, simple_package_path, type_check=True)
      compiler.Compiler().compile(my_pipeline, simple_package_path, type_check=False)

    finally:
      shutil.rmtree(tmpdir)

  def test_type_checking_with_json_schema(self):
    """Test type check pipeline parameters against the json schema."""
    @component
    def a_op(field_m: {'GCRPath': {'openapi_schema_validator': {"type": "string", "pattern": "^.*gcr\\.io/.*$"}}}, field_o: 'Integer'):
      return ContainerOp(
          name = 'operator a',
          image = 'gcr.io/ml-pipeline/component-b',
          arguments = [
              '--field-l', field_m,
              '--field-o', field_o,
          ],
      )

    @pipeline(
        name='p1',
        description='description1'
    )
    def my_pipeline(a: {'GCRPath': {'openapi_schema_validator': {"type": "string", "pattern": "^.*gcr\\.io/.*$"}}}='good', b: 'Integer'=12):
      a_op(field_m=a, field_o=b)

    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    sys.path.append(test_data_dir)
    tmpdir = tempfile.mkdtemp()
    try:
      simple_package_path = os.path.join(tmpdir, 'simple.tar.gz')
      import jsonschema
      with self.assertRaises(jsonschema.exceptions.ValidationError):
        compiler.Compiler().compile(my_pipeline, simple_package_path, type_check=True)

    finally:
      shutil.rmtree(tmpdir)

  def test_compile_pipeline_with_after(self):
    def op():
      return dsl.ContainerOp(
        name='Some component name',
        image='image'
      )

    @dsl.pipeline(name='Pipeline')
    def pipeline():
      task1 = op()
      task2 = op().after(task1)

    compiler.Compiler()._compile(pipeline)

  def _test_op_to_template_yaml(self, ops, file_base_name):
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    target_yaml = os.path.join(test_data_dir, file_base_name + '.yaml')
    with open(target_yaml, 'r') as f:
      expected = yaml.safe_load(f)['spec']['templates'][0]

    compiled_template = compiler._op_to_template._op_to_template(ops)

    del compiled_template['name'], expected['name']
    for output in compiled_template['outputs'].get('parameters', []) + compiled_template['outputs'].get('artifacts', []) + expected['outputs'].get('parameters', []) + expected['outputs'].get('artifacts', []):
      del output['name']
    assert compiled_template == expected

  def test_tolerations(self):
    """Test a pipeline with a tolerations."""
    op1 = dsl.ContainerOp(
      name='download',
      image='busybox',
      command=['sh', '-c'],
      arguments=['sleep 10; wget localhost:5678 -O /tmp/results.txt'],
      file_outputs={'downloaded': '/tmp/results.txt'}) \
      .add_toleration(V1Toleration(
      effect='NoSchedule',
      key='gpu',
      operator='Equal',
      value='run'))

    self._test_op_to_template_yaml(op1, file_base_name='tolerations')

  def test_set_display_name(self):
    """Test a pipeline with a customized task names."""

    import kfp
    op1 = kfp.components.load_component_from_text(
      '''
name: Component name
implementation:
  container:
    image: busybox
'''
    )

    @dsl.pipeline()
    def some_pipeline():
      op1().set_display_name('Custom name')

    workflow_dict = kfp.compiler.Compiler()._compile(some_pipeline)
    template = workflow_dict['spec']['templates'][0]
    self.assertEqual(template['metadata']['annotations']['pipelines.kubeflow.org/task_display_name'], 'Custom name')

  def test_set_dynamic_display_name(self):
    """Test a pipeline with a customized task names."""

    def some_pipeline(custom_name):
      some_op().set_display_name(custom_name)

    workflow_dict = kfp.compiler.Compiler()._compile(some_pipeline)
    template = [template for template in workflow_dict['spec']['templates'] if 'container' in template][0]
    self.assertNotIn('pipelineparam', template['metadata']['annotations']['pipelines.kubeflow.org/task_display_name'])

  def test_set_parallelism(self):
    """Test a pipeline with parallelism limits."""
    def some_op():
        return dsl.ContainerOp(
            name='sleep',
            image='busybox',
            command=['sleep 1'],
        )

    @dsl.pipeline()
    def some_pipeline():
      some_op()
      some_op()
      some_op()
      dsl.get_pipeline_conf().set_parallelism(1)

    workflow_dict = kfp.compiler.Compiler()._compile(some_pipeline)
    self.assertEqual(workflow_dict['spec']['parallelism'], 1)

  def test_set_ttl_seconds_after_finished(self):
    """Test a pipeline with ttl after finished."""
    def some_op():
        return dsl.ContainerOp(
            name='sleep',
            image='busybox',
            command=['sleep 1'],
        )

    @dsl.pipeline()
    def some_pipeline():
      some_op()
      dsl.get_pipeline_conf().set_ttl_seconds_after_finished(86400)

    workflow_dict = kfp.compiler.Compiler()._compile(some_pipeline)
    self.assertEqual(workflow_dict['spec']['ttlSecondsAfterFinished'], 86400)

  def test_pod_disruption_budget(self):
    """Test a pipeline with poddisruption budget."""
    def some_op():
        return dsl.ContainerOp(
            name='sleep',
            image='busybox',
            command=['sleep 1'],
        )

    @dsl.pipeline()
    def some_pipeline():
      some_op()
      dsl.get_pipeline_conf().set_pod_disruption_budget("100%")

    workflow_dict = kfp.compiler.Compiler()._compile(some_pipeline)
    self.assertEqual(workflow_dict['spec']["podDisruptionBudget"]['minAvailable'], "100%")

  def test_op_transformers(self):
    def some_op():
      return dsl.ContainerOp(
          name='sleep',
          image='busybox',
          command=['sleep 1'],
      )

    @dsl.pipeline(name='some_pipeline')
    def some_pipeline():
      task1 = some_op()
      task2 = some_op()
      task3 = some_op()

      dsl.get_pipeline_conf().op_transformers.append(lambda op: op.set_retry(5))

    workflow_dict = compiler.Compiler()._compile(some_pipeline)
    for template in workflow_dict['spec']['templates']:
      container = template.get('container', None)
      if container:
        self.assertEqual(template['retryStrategy']['limit'], 5)

  def test_image_pull_policy(self):
    def some_op():
      return dsl.ContainerOp(
          name='sleep',
          image='busybox',
          command=['sleep 1'],
      )

    @dsl.pipeline(name='some_pipeline')
    def some_pipeline():
      task1 = some_op()
      task2 = some_op()
      task3 = some_op()

      dsl.get_pipeline_conf().set_image_pull_policy(policy="Always")
    workflow_dict = compiler.Compiler()._compile(some_pipeline)
    for template in workflow_dict['spec']['templates']:
      container = template.get('container', None)
      if container:
        self.assertEqual(template['container']['imagePullPolicy'], "Always")


  def test_image_pull_policy_step_spec(self):
    def some_op():
      return dsl.ContainerOp(
          name='sleep',
          image='busybox',
          command=['sleep 1'],
      )

    def some_other_op():
      return dsl.ContainerOp(
          name='other',
          image='busybox',
          command=['sleep 1'],
      )

    @dsl.pipeline(name='some_pipeline')
    def some_pipeline():
      task1 = some_op()
      task2 = some_op()
      task3 = some_other_op().set_image_pull_policy("IfNotPresent")

      dsl.get_pipeline_conf().set_image_pull_policy(policy="Always")
    workflow_dict = compiler.Compiler()._compile(some_pipeline)
    for template in workflow_dict['spec']['templates']:
      container = template.get('container', None)
      if container:
        if template['name' ] == "other":
          self.assertEqual(template['container']['imagePullPolicy'], "IfNotPresent")
        elif template['name' ] == "sleep":
          self.assertEqual(template['container']['imagePullPolicy'], "Always")

  def test_image_pull_policy_invalid_setting(self):
    def some_op():
      return dsl.ContainerOp(
          name='sleep',
          image='busybox',
          command=['sleep 1'],
      )

    with self.assertRaises(ValueError):
      @dsl.pipeline(name='some_pipeline')
      def some_pipeline():
        task1 = some_op()
        task2 = some_op()
        dsl.get_pipeline_conf().set_image_pull_policy(policy="Alwayss")

      workflow_dict = compiler.Compiler()._compile(some_pipeline)

  def test_set_default_pod_node_selector(self):
    """Test a pipeline with node selector."""
    def some_op():
        return dsl.ContainerOp(
            name='sleep',
            image='busybox',
            command=['sleep 1'],
        )

    @dsl.pipeline()
    def some_pipeline():
      some_op()
      dsl.get_pipeline_conf().set_default_pod_node_selector(label_name="cloud.google.com/gke-accelerator", value="nvidia-tesla-p4")

    workflow_dict = kfp.compiler.Compiler()._compile(some_pipeline)
    self.assertEqual(workflow_dict['spec']['nodeSelector'], {"cloud.google.com/gke-accelerator":"nvidia-tesla-p4"})

  def test_set_dns_config(self):
    """Test a pipeline with node selector."""
    @dsl.pipeline()
    def some_pipeline():
      some_op()
      dsl.get_pipeline_conf().set_dns_config(V1PodDNSConfig(
        nameservers=["1.2.3.4"],
        options=[V1PodDNSConfigOption(name="ndots", value="2")]
      ))

    workflow_dict = kfp.compiler.Compiler()._compile(some_pipeline)
    self.assertEqual(
      workflow_dict['spec']['dnsConfig'],
      {"nameservers": ["1.2.3.4"], "options": [{"name": "ndots", "value": "2"}]}
    )

  def test_container_op_output_error_when_no_or_multiple_outputs(self):

    def no_outputs_pipeline():
      no_outputs_op = dsl.ContainerOp(name='dummy', image='dummy')
      dsl.ContainerOp(name='dummy', image='dummy', arguments=[no_outputs_op.output])

    def one_output_pipeline():
      one_output_op = dsl.ContainerOp(name='dummy', image='dummy', file_outputs={'out1': 'path1'})
      dsl.ContainerOp(name='dummy', image='dummy', arguments=[one_output_op.output])

    def two_outputs_pipeline():
      two_outputs_op = dsl.ContainerOp(name='dummy', image='dummy', file_outputs={'out1': 'path1', 'out2': 'path2'})
      dsl.ContainerOp(name='dummy', image='dummy', arguments=[two_outputs_op.output])

    with self.assertRaises(RuntimeError):
      compiler.Compiler()._compile(no_outputs_pipeline)

    compiler.Compiler()._compile(one_output_pipeline)

    with self.assertRaises(RuntimeError):
      compiler.Compiler()._compile(two_outputs_pipeline)

  def test_withitem_basic(self):
    self._test_py_compile_yaml('withitem_basic')

  def test_withitem_nested(self):
    self._test_py_compile_yaml('withitem_nested')

  def test_add_pod_env(self):
    self._test_py_compile_yaml('add_pod_env')

  def test_init_container(self):
    echo = dsl.UserContainer(
      name='echo',
      image='alpine:latest',
      command=['echo', 'bye'])

    @dsl.pipeline(name='InitContainer', description='A pipeline with init container.')
    def init_container_pipeline():
      dsl.ContainerOp(
        name='hello',
        image='alpine:latest',
        command=['echo', 'hello'],
        init_containers=[echo])

    workflow_dict = compiler.Compiler()._compile(init_container_pipeline)
    for template in workflow_dict['spec']['templates']:
      init_containers = template.get('initContainers', None)
      if init_containers:
        self.assertEqual(len(init_containers),1)
        init_container = init_containers[0]
        self.assertEqual(init_container, {'image':'alpine:latest', 'command': ['echo', 'bye'], 'name': 'echo'})

  def test_delete_resource_op(self):
      """Test a pipeline with a delete resource operation."""
      from kubernetes import client as k8s

      @dsl.pipeline()
      def some_pipeline():
        # create config map object with k6 load test script
        config_map = k8s.V1ConfigMap(
          api_version="v1",
          data={"foo": "bar"},
          kind="ConfigMap",
          metadata=k8s.V1ObjectMeta(
              name="foo-bar-cm",
              namespace="default"
          )
        )
        # delete the config map in k8s
        dsl.ResourceOp(
          name="delete-config-map",
          action="delete",
          k8s_resource=config_map
        )

      workflow_dict = kfp.compiler.Compiler()._compile(some_pipeline)
      delete_op_template = [template for template in workflow_dict['spec']['templates'] if template['name'] == 'delete-config-map'][0]

      # delete resource operation should not have success condition, failure condition or output parameters.
      # See https://github.com/argoproj/argo/blob/5331fc02e257266a4a5887dfe6277e5a0b42e7fc/cmd/argoexec/commands/resource.go#L30
      self.assertIsNone(delete_op_template.get("successCondition"))
      self.assertIsNone(delete_op_template.get("failureCondition"))
      self.assertDictEqual(delete_op_template.get("outputs", {}), {})

  def test_withparam_global(self):
    self._test_py_compile_yaml('withparam_global')

  def test_withparam_global_dict(self):
    self._test_py_compile_yaml('withparam_global_dict')

  def test_withparam_output(self):
    self._test_py_compile_yaml('withparam_output')

  def test_withparam_output_dict(self):
    self._test_py_compile_yaml('withparam_output_dict')

  def test_withparam_lightweight_out(self):
    self._test_py_compile_yaml('loop_over_lightweight_output')

  def test_parallelfor_pipeline_param_in_items_resolving(self):
    self._test_py_compile_yaml('parallelfor_pipeline_param_in_items_resolving')

  def test_parallelfor_item_argument_resolving(self):
    self._test_py_compile_yaml('parallelfor_item_argument_resolving')

  def test_py_input_artifact_raw_value(self):
    """Test pipeline input_artifact_raw_value."""
    self._test_py_compile_yaml('input_artifact_raw_value')

  def test_pipeline_name_same_as_task_name(self):
    def some_name():
      dsl.ContainerOp(
        name='some_name',
        image='alpine:latest',
      )

    workflow_dict = compiler.Compiler()._compile(some_name)
    template_names = set(template['name'] for template in workflow_dict['spec']['templates'])
    self.assertGreater(len(template_names), 1)
    self.assertEqual(template_names, {'some-name', 'some-name-2'})

  def test_set_execution_options_caching_strategy(self):
    def some_pipeline():
      task = some_op()
      task.execution_options.caching_strategy.max_cache_staleness = "P30D"

    workflow_dict = kfp.compiler.Compiler()._compile(some_pipeline)
    template = workflow_dict['spec']['templates'][0]
    self.assertEqual(template['metadata']['annotations']['pipelines.kubeflow.org/max_cache_staleness'], "P30D")

  def test_artifact_passing_using_volume(self):
    self._test_py_compile_yaml('artifact_passing_using_volume')

  def test_recursive_argument_mapping(self):
    # Verifying that the recursive call arguments are passed correctly when specified out of order
    component_2_in_0_out_op = kfp.components.load_component_from_text('''
inputs:
- name: in1
- name: in2
implementation:
  container:
    image: busybox
    command:
    - echo
    - inputValue: in1
    - inputValue: in2
    ''')

    @dsl.graph_component
    def subgraph(graph_in1, graph_in2):
      component_2_in_0_out_op(
        in1=graph_in1,
        in2=graph_in2,
      )
      subgraph(
        # Wrong order!
        graph_in2=graph_in2,
        graph_in1=graph_in1,
      )
    def some_pipeline(pipeline_in1, pipeline_in2):
      subgraph(pipeline_in1, pipeline_in2)

    workflow_dict = kfp.compiler.Compiler()._compile(some_pipeline)
    subgraph_template = [template for template in workflow_dict['spec']['templates'] if 'subgraph' in template['name']][0]
    recursive_subgraph_task = [task for task in subgraph_template['dag']['tasks'] if 'subgraph' in task['name']][0]
    for argument in recursive_subgraph_task['arguments']['parameters']:
      if argument['name'].endswith('in1'):
        self.assertTrue(
          argument['value'].endswith('in1}}'),
          'Wrong argument mapping: "{}" passed to "{}"'.format(argument['value'], argument['name']))
      elif argument['name'].endswith('in2'):
        self.assertTrue(
          argument['value'].endswith('in2}}'),
          'Wrong argument mapping: "{}" passed to "{}"'.format(argument['value'], argument['name']))
      else:
        self.fail('Unexpected input name: ' + argument['name'])

  def test_input_name_sanitization(self):
    component_2_in_1_out_op = kfp.components.load_component_from_text('''
inputs:
- name: Input 1
- name: Input 2
outputs:
- name: Output 1
implementation:
  container:
    image: busybox
    command:
    - echo
    - inputValue: Input 1
    - inputPath: Input 2
    - outputPath: Output 1
    ''')
    def some_pipeline():
      task1 = component_2_in_1_out_op('value 1', 'value 2')
      component_2_in_1_out_op(task1.output, task1.output)

    workflow_dict = kfp.compiler.Compiler()._compile(some_pipeline)
    container_templates = [template for template in workflow_dict['spec']['templates'] if 'container' in template]
    for template in container_templates:
      for argument in template['inputs'].get('parameters', []):
        self.assertNotIn(' ', argument['name'], 'The input name "{}" of template "{}" was not sanitized.'.format(argument['name'], template['name']))
      for argument in template['inputs']['artifacts']:
        self.assertNotIn(' ', argument['name'], 'The input name "{}" of template "{}" was not sanitized.'.format(argument['name'], template['name']))

  def test_container_op_with_arbitrary_name(self):
    def some_pipeline():
      dsl.ContainerOp(
        name=r''' !"#$%&'()*+,-./:;<=>?@[\]^_`''',
        image='alpine:latest',
      )
      dsl.ContainerOp(
        name=r''' !"#$%&'()*+,-./:;<=>?@[\]^_`''',
        image='alpine:latest',
      )
    workflow_dict = compiler.Compiler()._compile(some_pipeline)
    for template in workflow_dict['spec']['templates']:
      self.assertNotEqual(template['name'], '')

  def test_empty_string_pipeline_parameter_defaults(self):
    def some_pipeline(param1: str = ''):
      pass

    workflow_dict = kfp.compiler.Compiler()._compile(some_pipeline)
    self.assertEqual(workflow_dict['spec']['arguments']['parameters'][0].get('value'), '')

  def test_preserving_parameter_arguments_map(self):
    component_2_in_1_out_op = kfp.components.load_component_from_text('''
inputs:
- name: Input 1
- name: Input 2
outputs:
- name: Output 1
implementation:
  container:
    image: busybox
    command:
    - echo
    - inputValue: Input 1
    - inputPath: Input 2
    - outputPath: Output 1
    ''')
    def some_pipeline():
      task1 = component_2_in_1_out_op('value 1', 'value 2')
      component_2_in_1_out_op(task1.output, task1.output)

    workflow_dict = kfp.compiler.Compiler()._compile(some_pipeline)
    container_templates = [template for template in workflow_dict['spec']['templates'] if 'container' in template]
    for template in container_templates:
      parameter_arguments_json = template['metadata']['annotations']['pipelines.kubeflow.org/arguments.parameters']
      parameter_arguments = json.loads(parameter_arguments_json)
      self.assertEqual(set(parameter_arguments.keys()), {'Input 1'})

  def test__resolve_task_pipeline_param(self):
    p = PipelineParam(name='param2')
    resolved = Compiler._resolve_task_pipeline_param(p, group_type=None)
    self.assertEqual(resolved, "{{workflow.parameters.param2}}")

    p = PipelineParam(name='param1', op_name='op1')
    resolved = Compiler._resolve_task_pipeline_param(p, group_type=None)
    self.assertEqual(resolved, "{{tasks.op1.outputs.parameters.op1-param1}}")

    p = PipelineParam(name='param1', op_name='op1')
    resolved = Compiler._resolve_task_pipeline_param(p, group_type="subgraph")
    self.assertEqual(resolved, "{{inputs.parameters.op1-param1}}")

  def test_uri_artifact_passing(self):
    self._test_py_compile_yaml('uri_artifacts')

  def test_keyword_only_argument_for_pipeline_func(self):
    def some_pipeline(casual_argument: str, *, keyword_only_argument: str):
      pass
    kfp.compiler.Compiler()._create_workflow(some_pipeline)

  def test_keyword_only_argument_for_pipeline_func_identity(self):
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    sys.path.append(test_data_dir)

    # `@pipeline` is needed to make name the same for both functions

    @pipeline(name="pipeline_func")
    def pipeline_func_arg(foo_arg: str, bar_arg: str):
      dsl.ContainerOp(
        name='foo',
        image='foo',
        command=['bar'],
        arguments=[foo_arg, ' and ', bar_arg]
      )

    @pipeline(name="pipeline_func")
    def pipeline_func_kwarg(foo_arg: str, *, bar_arg: str):
      return pipeline_func_arg(foo_arg, bar_arg)

    pipeline_yaml_arg   = kfp.compiler.Compiler()._create_workflow(pipeline_func_arg)
    pipeline_yaml_kwarg = kfp.compiler.Compiler()._create_workflow(pipeline_func_kwarg)

    # the yamls may differ in metadata
    def remove_metadata(yaml) -> None:
      del yaml['metadata']
    remove_metadata(pipeline_yaml_arg)
    remove_metadata(pipeline_yaml_kwarg)

    # compare
    self.assertEqual(pipeline_yaml_arg, pipeline_yaml_kwarg)
