#!/usr/bin/env python3
# Copyright 2019 The Kubeflow Authors
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

import json
import os

import kfp
import kfp.onprem
import tfx.orchestration.kubeflow.kubeflow_dag_runner as kubeflow_dag_runner
import tensorflow_model_analysis as tfma
from tfx import v1 as tfx

# Define pipeline params used for pipeline execution.
# Path to the module file, should be a GCS path,
# or a module file baked in the docker image used by the pipeline.
_taxi_module_file_param = tfx.dsl.experimental.RuntimeParameter(
    name='module-file',
    default='/opt/conda/lib/python3.7/site-packages/tfx/examples/chicago_taxi_pipeline/taxi_utils_native_keras.py',
    ptype=str,
)

# Path to the CSV data file, under which their should be a data.csv file.
_data_root = '/opt/conda/lib/python3.7/site-packages/tfx/examples/chicago_taxi_pipeline/data/simple'

# Path of pipeline root, should be a GCS path.
_pipeline_root = os.path.join('gs://{{kfp-default-bucket}}', 'tfx_taxi_simple',
                              kfp.dsl.RUN_ID_PLACEHOLDER)

# Path that ML models are pushed, should be a GCS path.
_serving_model_dir = os.path.join('gs://your-bucket', 'serving_model',
                                  'tfx_taxi_simple')
_push_destination = tfx.dsl.experimental.RuntimeParameter(
    name='push_destination',
    default=json.dumps({'filesystem': {
        'base_directory': _serving_model_dir
    }}),
    ptype=str,
)


def _create_pipeline(
    pipeline_root: str,
    csv_input_location: str,
    taxi_module_file: tfx.dsl.experimental.RuntimeParameter,
    push_destination: tfx.dsl.experimental.RuntimeParameter,
    enable_cache: bool,
):
    """Creates a simple Kubeflow-based Chicago Taxi TFX pipeline.

  Args:
    pipeline_root: The root of the pipeline output.
    csv_input_location: The location of the input data directory.
    taxi_module_file: The location of the module file for Transform/Trainer.
    enable_cache: Whether to enable cache or not.

  Returns:
    A logical TFX pipeline.Pipeline object.
  """
    example_gen = tfx.components.CsvExampleGen(input_base=csv_input_location)
    statistics_gen = tfx.components.StatisticsGen(
        examples=example_gen.outputs['examples'])
    schema_gen = tfx.components.SchemaGen(
        statistics=statistics_gen.outputs['statistics'],
        infer_feature_shape=False,
    )
    example_validator = tfx.components.ExampleValidator(
        statistics=statistics_gen.outputs['statistics'],
        schema=schema_gen.outputs['schema'],
    )
    transform = tfx.components.Transform(
        examples=example_gen.outputs['examples'],
        schema=schema_gen.outputs['schema'],
        module_file=taxi_module_file,
    )
    trainer = tfx.components.Trainer(
        module_file=taxi_module_file,
        examples=transform.outputs['transformed_examples'],
        schema=schema_gen.outputs['schema'],
        transform_graph=transform.outputs['transform_graph'],
        train_args=tfx.proto.TrainArgs(num_steps=10),
        eval_args=tfx.proto.EvalArgs(num_steps=5),
    )
    # Set the TFMA config for Model Evaluation and Validation.
    eval_config = tfma.EvalConfig(
        model_specs=[
            tfma.ModelSpec(
                signature_name='serving_default',
                label_key='tips_xf',
                preprocessing_function_names=['transform_features'])
        ],
        metrics_specs=[
            tfma.MetricsSpec(
                # The metrics added here are in addition to those saved with the
                # model (assuming either a keras model or EvalSavedModel is used).
                # Any metrics added into the saved model (for example using
                # model.compile(..., metrics=[...]), etc) will be computed
                # automatically.
                metrics=[tfma.MetricConfig(class_name='ExampleCount')],
                # To add validation thresholds for metrics saved with the model,
                # add them keyed by metric name to the thresholds map.
                thresholds={
                    'binary_accuracy':
                        tfma.MetricThreshold(
                            value_threshold=tfma.GenericValueThreshold(
                                lower_bound={'value': 0.5}),
                            change_threshold=tfma.GenericChangeThreshold(
                                direction=tfma.MetricDirection.HIGHER_IS_BETTER,
                                absolute={'value': -1e-10}))
                })
        ],
        slicing_specs=[
            # An empty slice spec means the overall slice, i.e. the whole dataset.
            tfma.SlicingSpec(),
            # Data can be sliced along a feature column. In this case, data is
            # sliced along feature column trip_start_hour.
            tfma.SlicingSpec(feature_keys=['trip_start_hour'])
        ])

    evaluator = tfx.components.Evaluator(
        examples=example_gen.outputs['examples'],
        model=trainer.outputs['model'],
        eval_config=eval_config,
    )

    pusher = tfx.components.Pusher(
        model=trainer.outputs['model'],
        model_blessing=evaluator.outputs['blessing'],
        push_destination=push_destination,
    )

    return tfx.dsl.Pipeline(
        pipeline_name='parameterized_tfx_oss',
        pipeline_root=pipeline_root,
        components=[
            example_gen, statistics_gen, schema_gen, example_validator,
            transform, trainer, evaluator, pusher
        ],
        enable_cache=enable_cache,
    )


if __name__ == '__main__':
    enable_cache = True
    pipeline = _create_pipeline(
        _pipeline_root,
        _data_root,
        _taxi_module_file_param,
        _push_destination,
        enable_cache=enable_cache,
    )
    # Make sure the version of TFX image used is consistent with the version of
    # TFX SDK.
    config = tfx.orchestration.experimental.KubeflowDagRunnerConfig(
        kubeflow_metadata_config=tfx.orchestration.experimental
        .get_default_kubeflow_metadata_config(),
        tfx_image='gcr.io/tfx-oss-public/tfx:%s' % tfx.__version__,
        pipeline_operator_funcs=kubeflow_dag_runner
        .get_default_pipeline_operator_funcs(use_gcp_sa=False) + [
            kfp.onprem.add_default_resource_spec(
                memory_limit='2Gi', cpu_limit='1'),
        ],
    )
    kfp_runner = tfx.orchestration.experimental.KubeflowDagRunner(
        output_filename=__file__ + '.yaml',
        config=config,
    )

    kfp_runner.run(pipeline)
