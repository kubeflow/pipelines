"""Tests for utils."""
from google_cloud_pipeline_components.experimental.automl.tabular import utils
import unittest


class UtilsTest(unittest.TestCase):

  def setUp(self):
    super(UtilsTest, self).setUp()
    self.parameter_values = {
        'project': 'project',
        'location': 'us-central1',
        'root_dir': 'gs://foo',
        'target_column_name': 'target',
        'prediction_type': 'classification',
        'optimization_objective': 'maximize-au-prc',
        'transformations': '{\\"auto\\": {\\"column_name\\": \\"feature_1\\"}}',
        'split_spec':
            '{\\"fraction_split\\": {\\"training_fraction\\": 0.8, '
            '\\"validation_fraction\\": 0.2, \\"test_fraction\\": 0.0}}',
        'data_source': '{\\"csv_data_source\\": {\\"csv_filenames\\": '
                       '[\\"gs://foo/bar.csv\\"]}}',
        'stage_1_deadline_hours': 0.7708333333333334,
        'stage_1_num_parallel_trials': 35,
        'stage_1_num_selected_trials': 7,
        'stage_1_single_run_max_secs': 634,
        'reduce_search_space_mode': 'minimal',
        'stage_2_deadline_hours': 0.22916666666666663,
        'stage_2_num_parallel_trials': 35,
        'stage_2_num_selected_trials': 5,
        'stage_2_single_run_max_secs': 634,
        'weight_column_name': '',
        'optimization_objective_recall_value': -1,
        'optimization_objective_precision_value': -1,
        'study_spec_override': '',
        'stage_1_tuner_worker_pool_specs_override': '',
        'cv_trainer_worker_pool_specs_override': '',
        'export_additional_model_without_custom_ops': False,
        'stats_and_example_gen_dataflow_machine_type': 'n1-standard-16',
        'stats_and_example_gen_dataflow_max_num_workers': 25,
        'stats_and_example_gen_dataflow_disk_size_gb': 40,
        'transform_dataflow_machine_type': 'n1-standard-16',
        'transform_dataflow_max_num_workers': 25,
        'transform_dataflow_disk_size_gb': 40,
        'encryption_spec_key_name': '',
        'dataflow_subnetwork': '',
        'dataflow_use_public_ips': True,
        'additional_experiments': ''
    }

  def test_input_dictionary_to_parameter_none(self):
    self.assertEqual(utils.input_dictionary_to_parameter(None), '')

  def test_input_dictionary_to_parameter_dict(self):
    self.assertEqual(
        utils.input_dictionary_to_parameter({'foo': 'bar'}),
        r'{\"foo\": \"bar\"}')

  def test_get_skip_evaluation_pipeline_and_parameters(self):
    _, parameter_values = utils.get_skip_evaluation_pipeline_and_parameters(
        'project', 'us-central1', 'gs://foo', 'target', 'classification',
        'maximize-au-prc', {'auto': {
            'column_name': 'feature_1'
        }}, {
            'fraction_split': {
                'training_fraction': 0.8,
                'validation_fraction': 0.2,
                'test_fraction': 0.0
            }
        }, {'csv_data_source': {
            'csv_filenames': ['gs://foo/bar.csv']
        }}, 1000)

    self.assertEqual(parameter_values, self.parameter_values)

  def test_get_feature_selection_skip_evaluation_pipeline_and_parameters(self):
    _, parameter_values = utils.get_feature_selection_skip_evaluation_pipeline_and_parameters(
        'project', 'us-central1', 'gs://foo', 'target', 'classification',
        'maximize-au-prc', {'auto': {
            'column_name': 'feature_1'
        }}, {
            'fraction_split': {
                'training_fraction': 0.8,
                'validation_fraction': 0.2,
                'test_fraction': 0.0
            }
        }, {'csv_data_source': {
            'csv_filenames': ['gs://foo/bar.csv']
        }}, 80, 1000)

    parameter_values = self.parameter_values
    parameter_values['max_selected_features'] = 80
    self.assertEqual(parameter_values, parameter_values)

  def test_get_distill_skip_evaluation_pipeline_and_parameters(self):
    _, parameter_values = utils.get_distill_skip_evaluation_pipeline_and_parameters(
        'project',
        'us-central1',
        'gs://foo',
        'target',
        'classification',
        'maximize-au-prc', {'auto': {
            'column_name': 'feature_1'
        }}, {
            'fraction_split': {
                'training_fraction': 0.8,
                'validation_fraction': 0.2,
                'test_fraction': 0.0
            }
        }, {'csv_data_source': {
            'csv_filenames': ['gs://foo/bar.csv']
        }},
        1000,
        distill_batch_predict_machine_type='n1-standard-32',
        distill_batch_predict_starting_replica_count=40,
        distill_batch_predict_max_replica_count=80)

    self.assertEqual(
        parameter_values, {
            'cv_trainer_worker_pool_specs_override':
                '',
            'data_source':
                '{\\"csv_data_source\\": {\\"csv_filenames\\": [\\"gs://foo/bar.csv\\"]}}',
            'dataflow_subnetwork':
                '',
            'dataflow_use_public_ips':
                True,
            'encryption_spec_key_name':
                '',
            'export_additional_model_without_custom_ops':
                False,
            'location':
                'us-central1',
            'optimization_objective':
                'maximize-au-prc',
            'optimization_objective_precision_value':
                -1,
            'optimization_objective_recall_value':
                -1,
            'prediction_type':
                'classification',
            'project':
                'project',
            'reduce_search_space_mode':
                'minimal',
            'root_dir':
                'gs://foo',
            'split_spec':
                '{\\"fraction_split\\": {\\"training_fraction\\": 0.8, \\"validation_fraction\\": 0.2, \\"test_fraction\\": 0.0}}',
            'stage_1_deadline_hours':
                0.7708333333333334,
            'stage_1_num_parallel_trials':
                35,
            'stage_1_num_selected_trials':
                7,
            'stage_1_single_run_max_secs':
                634,
            'stage_1_tuner_worker_pool_specs_override':
                '',
            'stage_2_deadline_hours':
                0.22916666666666663,
            'stage_2_num_parallel_trials':
                35,
            'stage_2_num_selected_trials':
                5,
            'stage_2_single_run_max_secs':
                634,
            'stats_and_example_gen_dataflow_disk_size_gb':
                40,
            'stats_and_example_gen_dataflow_machine_type':
                'n1-standard-16',
            'stats_and_example_gen_dataflow_max_num_workers':
                25,
            'study_spec_override':
                '',
            'target_column_name':
                'target',
            'transform_dataflow_disk_size_gb':
                40,
            'transform_dataflow_machine_type':
                'n1-standard-16',
            'transform_dataflow_max_num_workers':
                25,
            'transformations':
                '{\\"auto\\": {\\"column_name\\": \\"feature_1\\"}}',
            'weight_column_name':
                '',
            'distill_batch_predict_machine_type':
                'n1-standard-32',
            'distill_batch_predict_max_replica_count':
                80,
            'distill_batch_predict_starting_replica_count':
                40,
            'distill_stage_1_deadline_hours':
                3 * 634 * 1.3 / 3600,
            'additional_experiments':
                ''
        })

  def test_get_skip_architecture_search_pipeline_and_parameters(self):
    _, parameter_values = utils.get_skip_architecture_search_pipeline_and_parameters(
        'project', 'us-central1', 'gs://foo', 'target', 'classification',
        'maximize-au-prc', {'auto': {
            'column_name': 'feature_1'
        }}, {
            'fraction_split': {
                'training_fraction': 0.8,
                'validation_fraction': 0.2,
                'test_fraction': 0.0
            }
        }, {'csv_data_source': {
            'csv_filenames': ['gs://foo/bar.csv']
        }}, 1000, 'gs://bar')
    self.assertEqual(
        parameter_values, {
            'cv_trainer_worker_pool_specs_override':
                '',
            'data_source':
                '{\\"csv_data_source\\": {\\"csv_filenames\\": [\\"gs://foo/bar.csv\\"]}}',
            'encryption_spec_key_name':
                '',
            'export_additional_model_without_custom_ops':
                False,
            'location':
                'us-central1',
            'optimization_objective':
                'maximize-au-prc',
            'optimization_objective_precision_value':
                -1,
            'optimization_objective_recall_value':
                -1,
            'prediction_type':
                'classification',
            'project':
                'project',
            'root_dir':
                'gs://foo',
            'split_spec':
                '{\\"fraction_split\\": {\\"training_fraction\\": 0.8, \\"validation_fraction\\": 0.2, \\"test_fraction\\": 0.0}}',
            'stage_1_tuning_result_artifact_uri':
                'gs://bar',
            'stage_2_deadline_hours':
                1.0,
            'stage_2_num_parallel_trials':
                35,
            'stage_2_num_selected_trials':
                5,
            'stage_2_single_run_max_secs':
                2769,
            'stats_and_example_gen_dataflow_machine_type':
                'n1-standard-16',
            'stats_and_example_gen_dataflow_max_num_workers':
                25,
            'target_column_name':
                'target',
            'transform_dataflow_machine_type':
                'n1-standard-16',
            'transform_dataflow_max_num_workers':
                25,
            'transformations':
                '{\\"auto\\": {\\"column_name\\": \\"feature_1\\"}}',
            'weight_column_name':
                '',
            'dataflow_subnetwork':
                '',
            'dataflow_use_public_ips':
                True
        })

  def test_get_wide_and_deep_trainer_pipeline_and_parameters(self):
    _, parameter_values = utils.get_wide_and_deep_trainer_pipeline_and_parameters(
        'project', 'us-central1', 'gs://foo', 'target', 'classification',
        {'auto': {
            'column_name': 'feature_1'
        }}, {
            'fraction_split': {
                'training_fraction': 0.8,
                'validation_fraction': 0.2,
                'test_fraction': 0.0
            }
        }, {'csv_data_source': {
            'csv_filenames': ['gs://foo/bar.csv']
        }}, 0.01, 0.01)

    self.assertEqual(
        parameter_values, {
            'project':
                'project',
            'location':
                'us-central1',
            'root_dir':
                'gs://foo',
            'target_column':
                'target',
            'prediction_type':
                'classification',
            'transformations':
                '{\\"auto\\": {\\"column_name\\": \\"feature_1\\"}}',
            'split_spec':
                '{\\"fraction_split\\": {\\"training_fraction\\": 0.8, \\"validation_fraction\\": 0.2, \\"test_fraction\\": 0.0}}',
            'data_source':
                '{\\"csv_data_source\\": {\\"csv_filenames\\": [\\"gs://foo/bar.csv\\"]}}',
            'learning_rate':
                0.01,
            'dnn_learning_rate':
                0.01,
            'optimizer_type':
                'adam',
            'max_steps':
                -1,
            'max_train_secs':
                -1,
            'l1_regularization_strength':
                0,
            'l2_regularization_strength':
                0,
            'l2_shrinkage_regularization_strength':
                0,
            'beta_1':
                0.9,
            'beta_2':
                0.999,
            'hidden_units':
                '30,30,30',
            'use_wide':
                True,
            'embed_categories':
                True,
            'dnn_dropout':
                0,
            'dnn_optimizer_type':
                'ftrl',
            'dnn_l1_regularization_strength':
                0,
            'dnn_l2_regularization_strength':
                0,
            'dnn_l2_shrinkage_regularization_strength':
                0,
            'dnn_beta_1':
                0.9,
            'dnn_beta_2':
                0.999,
            'enable_profiler':
                False,
            'seed':
                1,
            'eval_steps':
                0,
            'batch_size':
                100,
            'eval_frequency_secs':
                600,
            'weight_column':
                '',
            'stats_and_example_gen_dataflow_machine_type':
                'n1-standard-16',
            'stats_and_example_gen_dataflow_max_num_workers':
                25,
            'stats_and_example_gen_dataflow_disk_size_gb':
                40,
            'transform_dataflow_machine_type':
                'n1-standard-16',
            'transform_dataflow_max_num_workers':
                25,
            'transform_dataflow_disk_size_gb':
                40,
            'training_machine_spec': {
                'machine_type': 'n1-standard-16'
            },
            'training_replica_count':
                1,
            'dataflow_subnetwork':
                '',
            'dataflow_use_public_ips':
                True,
            'encryption_spec_key_name':
                ''
        })

  def test_get_tabnet_hyperparameter_tuning_job_pipeline_and_parameters(self):
    _, parameter_values = utils.get_builtin_algorithm_hyperparameter_tuning_job_pipeline_and_parameters(
        'project',
        'us-central1',
        'gs://foo',
        'target',
        'classification', {'auto': {
            'column_name': 'feature_1'
        }}, {
            'fraction_split': {
                'training_fraction': 0.8,
                'validation_fraction': 0.2,
                'test_fraction': 0.0
            }
        }, {'csv_data_source': {
            'csv_filenames': ['gs://foo/bar.csv']
        }}, [{
            'metric_id': 'loss',
            'goal': 'MINIMIZE'
        }], [{
            'parameter_id': 'dnn_learning_rate',
            'double_value_spec': {
                'min_value': 0.0001,
                'max_value': 0.01
            },
            'scale_type': 'UNIT_LINEAR_SCALE'
        }, {
            'parameter_id': 'learning_rate',
            'double_value_spec': {
                'min_value': 0.001,
                'max_value': 0.01
            },
            'scale_type': 'UNIT_LINEAR_SCALE'
        }, {
            'parameter_id': 'max_steps',
            'discrete_value_spec': {
                'values': [2]
            }
        }],
        2,
        1,
        algorithm='tabnet')
    self.assertEqual(
        parameter_values, {
            'project':
                'project',
            'location':
                'us-central1',
            'root_dir':
                'gs://foo',
            'target_column':
                'target',
            'prediction_type':
                'classification',
            'transformations':
                '{\\"auto\\": {\\"column_name\\": \\"feature_1\\"}}',
            'split_spec':
                '{\\"fraction_split\\": {\\"training_fraction\\": 0.8, \\"validation_fraction\\": 0.2, \\"test_fraction\\": 0.0}}',
            'data_source':
                '{\\"csv_data_source\\": {\\"csv_filenames\\": [\\"gs://foo/bar.csv\\"]}}',
            'study_spec_metrics': [{
                'metric_id': 'loss',
                'goal': 'MINIMIZE'
            }],
            'study_spec_parameters': [{
                'parameter_id': 'dnn_learning_rate',
                'double_value_spec': {
                    'min_value': 0.0001,
                    'max_value': 0.01
                },
                'scale_type': 'UNIT_LINEAR_SCALE'
            }, {
                'parameter_id': 'learning_rate',
                'double_value_spec': {
                    'min_value': 0.001,
                    'max_value': 0.01
                },
                'scale_type': 'UNIT_LINEAR_SCALE'
            }, {
                'parameter_id': 'max_steps',
                'discrete_value_spec': {
                    'values': [2]
                }
            }],
            'max_trial_count':
                2,
            'parallel_trial_count':
                1,
            'tabnet':
                True,
            'enable_profiler':
                False,
            'seed':
                1,
            'eval_steps':
                0,
            'eval_frequency_secs':
                600,
            'weight_column':
                '',
            'max_failed_trial_count':
                0,
            'study_spec_algorithm':
                'ALGORITHM_UNSPECIFIED',
            'study_spec_measurement_selection_type':
                'BEST_MEASUREMENT',
            'stats_and_example_gen_dataflow_machine_type':
                'n1-standard-16',
            'stats_and_example_gen_dataflow_max_num_workers':
                25,
            'stats_and_example_gen_dataflow_disk_size_gb':
                40,
            'transform_dataflow_machine_type':
                'n1-standard-16',
            'transform_dataflow_max_num_workers':
                25,
            'transform_dataflow_disk_size_gb':
                40,
            'training_machine_spec': {
                'machine_type': 'n1-standard-16'
            },
            'training_replica_count':
                1,
            'dataflow_subnetwork':
                '',
            'dataflow_use_public_ips':
                True,
            'encryption_spec_key_name':
                ''
        })

  def test_get_wide_and_deep_hyperparameter_tuning_job_pipeline_and_parameters(
      self):
    _, parameter_values = utils.get_builtin_algorithm_hyperparameter_tuning_job_pipeline_and_parameters(
        'project',
        'us-central1',
        'gs://foo',
        'target',
        'classification', {'auto': {
            'column_name': 'feature_1'
        }}, {
            'fraction_split': {
                'training_fraction': 0.8,
                'validation_fraction': 0.2,
                'test_fraction': 0.0
            }
        }, {'csv_data_source': {
            'csv_filenames': ['gs://foo/bar.csv']
        }}, [{
            'metric_id': 'loss',
            'goal': 'MINIMIZE'
        }], [{
            'parameter_id': 'dnn_learning_rate',
            'double_value_spec': {
                'min_value': 0.0001,
                'max_value': 0.01
            },
            'scale_type': 'UNIT_LINEAR_SCALE'
        }, {
            'parameter_id': 'learning_rate',
            'double_value_spec': {
                'min_value': 0.001,
                'max_value': 0.01
            },
            'scale_type': 'UNIT_LINEAR_SCALE'
        }, {
            'parameter_id': 'max_steps',
            'discrete_value_spec': {
                'values': [2]
            }
        }],
        2,
        1,
        algorithm='wide_and_deep')
    self.assertEqual(
        parameter_values, {
            'project':
                'project',
            'location':
                'us-central1',
            'root_dir':
                'gs://foo',
            'target_column':
                'target',
            'prediction_type':
                'classification',
            'transformations':
                '{\\"auto\\": {\\"column_name\\": \\"feature_1\\"}}',
            'split_spec':
                '{\\"fraction_split\\": {\\"training_fraction\\": 0.8, \\"validation_fraction\\": 0.2, \\"test_fraction\\": 0.0}}',
            'data_source':
                '{\\"csv_data_source\\": {\\"csv_filenames\\": [\\"gs://foo/bar.csv\\"]}}',
            'study_spec_metrics': [{
                'metric_id': 'loss',
                'goal': 'MINIMIZE'
            }],
            'study_spec_parameters': [{
                'parameter_id': 'dnn_learning_rate',
                'double_value_spec': {
                    'min_value': 0.0001,
                    'max_value': 0.01
                },
                'scale_type': 'UNIT_LINEAR_SCALE'
            }, {
                'parameter_id': 'learning_rate',
                'double_value_spec': {
                    'min_value': 0.001,
                    'max_value': 0.01
                },
                'scale_type': 'UNIT_LINEAR_SCALE'
            }, {
                'parameter_id': 'max_steps',
                'discrete_value_spec': {
                    'values': [2]
                }
            }],
            'max_trial_count':
                2,
            'parallel_trial_count':
                1,
            'wide_and_deep':
                True,
            'enable_profiler':
                False,
            'seed':
                1,
            'eval_steps':
                0,
            'eval_frequency_secs':
                600,
            'weight_column':
                '',
            'max_failed_trial_count':
                0,
            'study_spec_algorithm':
                'ALGORITHM_UNSPECIFIED',
            'study_spec_measurement_selection_type':
                'BEST_MEASUREMENT',
            'stats_and_example_gen_dataflow_machine_type':
                'n1-standard-16',
            'stats_and_example_gen_dataflow_max_num_workers':
                25,
            'stats_and_example_gen_dataflow_disk_size_gb':
                40,
            'transform_dataflow_machine_type':
                'n1-standard-16',
            'transform_dataflow_max_num_workers':
                25,
            'transform_dataflow_disk_size_gb':
                40,
            'training_machine_spec': {
                'machine_type': 'n1-standard-16'
            },
            'training_replica_count':
                1,
            'dataflow_subnetwork':
                '',
            'dataflow_use_public_ips':
                True,
            'encryption_spec_key_name':
                ''
        })

  def test_get_tabnet_trainer_pipeline_and_parameters(self):
    _, parameter_values = utils.get_tabnet_trainer_pipeline_and_parameters(
        'project', 'us-central1', 'gs://foo', 'target', 'classification',
        {'auto': {
            'column_name': 'feature_1'
        }}, {
            'fraction_split': {
                'training_fraction': 0.8,
                'validation_fraction': 0.2,
                'test_fraction': 0.0
            }
        }, {'csv_data_source': {
            'csv_filenames': ['gs://foo/bar.csv']
        }}, 0.01)

    self.assertEqual(
        parameter_values, {
            'project':
                'project',
            'location':
                'us-central1',
            'root_dir':
                'gs://foo',
            'target_column':
                'target',
            'prediction_type':
                'classification',
            'transformations':
                '{\\"auto\\": {\\"column_name\\": \\"feature_1\\"}}',
            'split_spec':
                '{\\"fraction_split\\": {\\"training_fraction\\": 0.8, \\"validation_fraction\\": 0.2, \\"test_fraction\\": 0.0}}',
            'data_source':
                '{\\"csv_data_source\\": {\\"csv_filenames\\": [\\"gs://foo/bar.csv\\"]}}',
            'learning_rate':
                0.01,
            'max_steps':
                -1,
            'max_train_secs':
                -1,
            'large_category_dim':
                1,
            'large_category_thresh':
                300,
            'yeo_johnson_transform':
                True,
            'feature_dim':
                64,
            'feature_dim_ratio':
                0.5,
            'num_decision_steps':
                6,
            'relaxation_factor':
                1.5,
            'decay_every':
                100,
            'gradient_thresh':
                2000,
            'sparsity_loss_weight':
                1e-05,
            'batch_momentum':
                0.95,
            'batch_size_ratio':
                0.25,
            'num_transformer_layers':
                4,
            'num_transformer_layers_ratio':
                0.25,
            'class_weight':
                1.0,
            'loss_function_type':
                'default',
            'alpha_focal_loss':
                0.25,
            'gamma_focal_loss':
                2.0,
            'enable_profiler':
                False,
            'seed':
                1,
            'eval_steps':
                0,
            'batch_size':
                100,
            'eval_frequency_secs':
                600,
            'weight_column':
                '',
            'stats_and_example_gen_dataflow_machine_type':
                'n1-standard-16',
            'stats_and_example_gen_dataflow_max_num_workers':
                25,
            'stats_and_example_gen_dataflow_disk_size_gb':
                40,
            'transform_dataflow_machine_type':
                'n1-standard-16',
            'transform_dataflow_max_num_workers':
                25,
            'transform_dataflow_disk_size_gb':
                40,
            'training_machine_spec': {
                'machine_type': 'n1-standard-16'
            },
            'training_replica_count':
                1,
            'dataflow_subnetwork':
                '',
            'dataflow_use_public_ips':
                True,
            'encryption_spec_key_name':
                ''
        })


if __name__ == '__main__':
  unittest.main()
