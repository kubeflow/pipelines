"""Tests for utils."""
from google_cloud_pipeline_components.experimental.automl.tabular import utils
import unittest


class UtilsTest(unittest.TestCase):

  def test_input_dictionary_to_parameter_none(self):
    self.assertEqual(utils.input_dictionary_to_parameter(None), '')

  def test_input_dictionary_to_parameter_dict(self):
    self.assertEqual(
        utils.input_dictionary_to_parameter({'foo': 'bar'}),
        r'{\"foo\": \"bar\"}',
    )

  def test_get_skip_evaluation_pipeline_and_parameters(self):
    _, parameter_values = utils.get_skip_evaluation_pipeline_and_parameters(
        'project',
        'us-central1',
        'gs://foo',
        'target',
        'classification',
        'maximize-au-prc',
        {'auto': {'column_name': 'feature_1'}},
        {
            'fraction_split': {
                'training_fraction': 0.8,
                'validation_fraction': 0.2,
                'test_fraction': 0.0,
            }
        },
        {'csv_data_source': {'csv_filenames': ['gs://foo/bar.csv']}},
        1000,
    )

    expected_parameter_values = {
        'project': 'project',
        'location': 'us-central1',
        'root_dir': 'gs://foo',
        'target_column_name': 'target',
        'prediction_type': 'classification',
        'optimization_objective': 'maximize-au-prc',
        'transformations': '{\\"auto\\": {\\"column_name\\": \\"feature_1\\"}}',
        'split_spec': (
            '{\\"fraction_split\\": {\\"training_fraction\\": 0.8, '
            '\\"validation_fraction\\": 0.2, \\"test_fraction\\": 0.0}}'
        ),
        'data_source': (
            '{\\"csv_data_source\\": {\\"csv_filenames\\": '
            '[\\"gs://foo/bar.csv\\"]}}'
        ),
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
    }
    self.assertEqual(parameter_values, expected_parameter_values)

  def test_get_skip_evaluation_pipeline_and_parameters_with_additional_experiments(
      self,
  ):
    _, parameter_values = utils.get_skip_evaluation_pipeline_and_parameters(
        'project',
        'us-central1',
        'gs://foo',
        'target',
        'classification',
        'maximize-au-prc',
        {'auto': {'column_name': 'feature_1'}},
        {
            'fraction_split': {
                'training_fraction': 0.8,
                'validation_fraction': 0.2,
                'test_fraction': 0.0,
            }
        },
        {'csv_data_source': {'csv_filenames': ['gs://foo/bar.csv']}},
        1000,
        additional_experiments={
            'categorical_array_weights': [{
                'column_name': 'STRING_2unique_REPEATED',
                'weight_column_name': 'INTEGER_2unique_REPEATED',
            }]
        },
    )

    expected_parameter_values = {
        'project': 'project',
        'location': 'us-central1',
        'root_dir': 'gs://foo',
        'target_column_name': 'target',
        'prediction_type': 'classification',
        'optimization_objective': 'maximize-au-prc',
        'transformations': '{\\"auto\\": {\\"column_name\\": \\"feature_1\\"}}',
        'split_spec': (
            '{\\"fraction_split\\": {\\"training_fraction\\": 0.8, '
            '\\"validation_fraction\\": 0.2, \\"test_fraction\\": 0.0}}'
        ),
        'data_source': (
            '{\\"csv_data_source\\": {\\"csv_filenames\\": '
            '[\\"gs://foo/bar.csv\\"]}}'
        ),
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
        'additional_experiments': (
            '{\\"categorical_array_weights\\": [{\\"column_name\\": '
            '\\"STRING_2unique_REPEATED\\", \\"weight_column_name\\": '
            '\\"INTEGER_2unique_REPEATED\\"}]}'
        ),
    }
    self.assertEqual(parameter_values, expected_parameter_values)

  def test_get_distill_skip_evaluation_pipeline_and_parameters(self):
    _, parameter_values = utils.get_automl_tabular_pipeline_and_parameters(
        'project',
        'us-central1',
        'gs://foo',
        'target',
        'classification',
        'maximize-au-prc',
        {'auto': {'column_name': 'feature_1'}},
        training_fraction=0.8,
        validation_fraction=0.2,
        test_fraction=0.0,
        data_source_csv_filenames='gs://foo/bar.csv',
        train_budget_milli_node_hours=1000,
        distill_batch_predict_machine_type='n1-standard-32',
        distill_batch_predict_starting_replica_count=40,
        distill_batch_predict_max_replica_count=80,
        stage_1_tuning_result_artifact_uri='gs://bar',
    )

    self.assertEqual(
        parameter_values,
        {
            'additional_experiments': {},
            'cv_trainer_worker_pool_specs_override': [],
            'data_source_csv_filenames': 'gs://foo/bar.csv',
            'dataflow_use_public_ips': True,
            'export_additional_model_without_custom_ops': False,
            'location': 'us-central1',
            'optimization_objective': 'maximize-au-prc',
            'prediction_type': 'classification',
            'project': 'project',
            'root_dir': 'gs://foo',
            'run_evaluation': True,
            'stage_1_tuner_worker_pool_specs_override': [],
            'study_spec_parameters_override': [],
            'target_column': 'target',
            'test_fraction': 0.0,
            'train_budget_milli_node_hours': 1000,
            'training_fraction': 0.8,
            'transformations': {'auto': {'column_name': 'feature_1'}},
            'validation_fraction': 0.2,
            'stage_1_tuning_result_artifact_uri': 'gs://bar',
            'quantiles': [],
            'enable_probabilistic_inference': False,
        },
    )

  def test_get_default_pipeline_and_parameters_with_eval_and_distill(self):
    _, parameter_values = utils.get_default_pipeline_and_parameters(
        'project',
        'us-central1',
        'gs://foo',
        'target',
        'classification',
        'maximize-au-prc',
        {'auto': {'column_name': 'feature_1'}},
        {
            'fraction_split': {
                'training_fraction': 0.8,
                'validation_fraction': 0.2,
                'test_fraction': 0.0,
            }
        },
        {'csv_data_source': {'csv_filenames': ['gs://foo/bar.csv']}},
        1000,
        run_evaluation=True,
        run_distillation=True,
        distill_batch_predict_machine_type='n1-standard-32',
        distill_batch_predict_starting_replica_count=40,
        distill_batch_predict_max_replica_count=80,
    )
    self.assertEqual(
        parameter_values,
        {
            'project': 'project',
            'location': 'us-central1',
            'root_dir': 'gs://foo',
            'target_column_name': 'target',
            'prediction_type': 'classification',
            'optimization_objective': 'maximize-au-prc',
            'transformations': (
                '{\\"auto\\": {\\"column_name\\": \\"feature_1\\"}}'
            ),
            'split_spec': (
                '{\\"fraction_split\\": {\\"training_fraction\\": 0.8,'
                ' \\"validation_fraction\\": 0.2, \\"test_fraction\\": 0.0}}'
            ),
            'data_source': (
                '{\\"csv_data_source\\": {\\"csv_filenames\\":'
                ' [\\"gs://foo/bar.csv\\"]}}'
            ),
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
            'dataflow_subnetwork': '',
            'dataflow_use_public_ips': True,
            'encryption_spec_key_name': '',
            'dataflow_service_account': '',
            'evaluation_batch_predict_machine_type': 'n1-highmem-8',
            'evaluation_batch_predict_starting_replica_count': 20,
            'evaluation_batch_predict_max_replica_count': 20,
            'evaluation_dataflow_machine_type': 'n1-standard-4',
            'evaluation_dataflow_max_num_workers': 100,
            'evaluation_dataflow_disk_size_gb': 50,
            'run_evaluation': True,
            'distill_stage_1_deadline_hours': 0.6868333333333333,
            'distill_batch_predict_machine_type': 'n1-standard-32',
            'distill_batch_predict_starting_replica_count': 40,
            'distill_batch_predict_max_replica_count': 80,
            'run_distillation': True,
        },
    )

  def test_get_skip_architecture_search_pipeline_and_parameters(self):
    _, parameter_values = (
        utils.get_skip_architecture_search_pipeline_and_parameters(
            'project',
            'us-central1',
            'gs://foo',
            'target',
            'classification',
            'maximize-au-prc',
            'gs://foo',
            1000,
            'gs://bar',
            training_fraction=0.8,
            validation_fraction=0.2,
            test_fraction=0.0,
            data_source_csv_filenames='gs://foo/bar.csv',
        )
    )

    self.assertEqual(
        parameter_values,
        {
            'project': 'project',
            'location': 'us-central1',
            'root_dir': 'gs://foo',
            'stage_1_tuner_worker_pool_specs_override': [],
            'study_spec_parameters_override': [],
            'target_column': 'target',
            'prediction_type': 'classification',
            'data_source_csv_filenames': 'gs://foo/bar.csv',
            'training_fraction': 0.8,
            'validation_fraction': 0.2,
            'test_fraction': 0.0,
            'optimization_objective': 'maximize-au-prc',
            'transformations': 'gs://foo',
            'train_budget_milli_node_hours': 1000,
            'stage_1_tuning_result_artifact_uri': 'gs://bar',
            'cv_trainer_worker_pool_specs_override': [],
            'export_additional_model_without_custom_ops': False,
            'dataflow_use_public_ips': True,
            'additional_experiments': {},
            'run_evaluation': True,
            'quantiles': [],
            'enable_probabilistic_inference': False,
        },
    )

  def test_get_wide_and_deep_trainer_pipeline_and_parameters(self):
    _, parameter_values = (
        utils.get_wide_and_deep_trainer_pipeline_and_parameters(
            'project',
            'us-central1',
            'gs://foo',
            'target',
            'classification',
            0.01,
            0.01,
            data_source_csv_filenames='gs://foo/bar.csv',
            dataflow_service_account='service-account',
        )
    )

    self.assertEqual(
        parameter_values,
        {
            'project': 'project',
            'location': 'us-central1',
            'root_dir': 'gs://foo',
            'target_column': 'target',
            'prediction_type': 'classification',
            'dataset_level_custom_transformation_definitions': [],
            'dataset_level_transformations': [],
            'tf_auto_transform_features': {},
            'tf_custom_transformation_definitions': [],
            'materialized_examples_format': 'tfrecords_gzip',
            'tf_transform_execution_engine': 'dataflow',
            'data_source_csv_filenames': 'gs://foo/bar.csv',
            'learning_rate': 0.01,
            'dnn_learning_rate': 0.01,
            'optimizer_type': 'adam',
            'max_steps': -1,
            'max_train_secs': -1,
            'l1_regularization_strength': 0,
            'l2_regularization_strength': 0,
            'l2_shrinkage_regularization_strength': 0,
            'beta_1': 0.9,
            'beta_2': 0.999,
            'hidden_units': '30,30,30',
            'use_wide': True,
            'embed_categories': True,
            'dnn_dropout': 0,
            'dnn_optimizer_type': 'adam',
            'dnn_l1_regularization_strength': 0,
            'dnn_l2_regularization_strength': 0,
            'dnn_l2_shrinkage_regularization_strength': 0,
            'dnn_beta_1': 0.9,
            'dnn_beta_2': 0.999,
            'enable_profiler': False,
            'cache_data': 'auto',
            'seed': 1,
            'eval_steps': 0,
            'batch_size': 100,
            'eval_frequency_secs': 600,
            'weight_column': '',
            'transform_dataflow_machine_type': 'n1-standard-16',
            'transform_dataflow_max_num_workers': 25,
            'transform_dataflow_disk_size_gb': 40,
            'worker_pool_specs_override': [],
            'run_evaluation': True,
            'evaluation_batch_predict_machine_type': 'n1-highmem-8',
            'evaluation_batch_predict_starting_replica_count': 20,
            'evaluation_batch_predict_max_replica_count': 20,
            'evaluation_dataflow_machine_type': 'n1-standard-4',
            'evaluation_dataflow_starting_num_workers': 10,
            'evaluation_dataflow_max_num_workers': 100,
            'evaluation_dataflow_disk_size_gb': 50,
            'dataflow_service_account': 'service-account',
            'dataflow_subnetwork': '',
            'dataflow_use_public_ips': True,
            'encryption_spec_key_name': '',
            'run_feature_selection': False,
        },
    )

  def test_get_tabnet_hyperparameter_tuning_job_pipeline_and_parameters(self):
    _, parameter_values = (
        utils.get_tabnet_hyperparameter_tuning_job_pipeline_and_parameters(
            'project',
            'us-central1',
            'gs://foo',
            'target',
            'classification',
            'loss',
            'MINIMIZE',
            [
                {
                    'parameter_id': 'dnn_learning_rate',
                    'double_value_spec': {
                        'min_value': 0.0001,
                        'max_value': 0.01,
                    },
                    'scale_type': 'UNIT_LINEAR_SCALE',
                },
                {
                    'parameter_id': 'learning_rate',
                    'double_value_spec': {
                        'min_value': 0.001,
                        'max_value': 0.01,
                    },
                    'scale_type': 'UNIT_LINEAR_SCALE',
                },
                {
                    'parameter_id': 'max_steps',
                    'discrete_value_spec': {'values': [2]},
                },
            ],
            2,
            1,
            data_source_csv_filenames='gs://foo/bar.csv',
            dataflow_service_account='service-account',
        )
    )
    self.assertEqual(
        parameter_values,
        {
            'project': 'project',
            'location': 'us-central1',
            'root_dir': 'gs://foo',
            'target_column': 'target',
            'prediction_type': 'classification',
            'dataset_level_custom_transformation_definitions': [],
            'dataset_level_transformations': [],
            'tf_auto_transform_features': {},
            'tf_custom_transformation_definitions': [],
            'materialized_examples_format': 'tfrecords_gzip',
            'tf_transform_execution_engine': 'dataflow',
            'data_source_csv_filenames': 'gs://foo/bar.csv',
            'study_spec_metric_id': 'loss',
            'study_spec_metric_goal': 'MINIMIZE',
            'study_spec_parameters_override': [
                {
                    'parameter_id': 'dnn_learning_rate',
                    'double_value_spec': {
                        'min_value': 0.0001,
                        'max_value': 0.01,
                    },
                    'scale_type': 'UNIT_LINEAR_SCALE',
                },
                {
                    'parameter_id': 'learning_rate',
                    'double_value_spec': {
                        'min_value': 0.001,
                        'max_value': 0.01,
                    },
                    'scale_type': 'UNIT_LINEAR_SCALE',
                },
                {
                    'parameter_id': 'max_steps',
                    'discrete_value_spec': {'values': [2]},
                },
            ],
            'max_trial_count': 2,
            'parallel_trial_count': 1,
            'enable_profiler': False,
            'cache_data': 'auto',
            'seed': 1,
            'eval_steps': 0,
            'eval_frequency_secs': 600,
            'weight_column': '',
            'max_failed_trial_count': 0,
            'study_spec_algorithm': 'ALGORITHM_UNSPECIFIED',
            'study_spec_measurement_selection_type': 'BEST_MEASUREMENT',
            'transform_dataflow_machine_type': 'n1-standard-16',
            'transform_dataflow_max_num_workers': 25,
            'transform_dataflow_disk_size_gb': 40,
            'worker_pool_specs_override': [],
            'run_evaluation': True,
            'evaluation_batch_predict_machine_type': 'n1-highmem-8',
            'evaluation_batch_predict_starting_replica_count': 20,
            'evaluation_batch_predict_max_replica_count': 20,
            'evaluation_dataflow_machine_type': 'n1-standard-4',
            'evaluation_dataflow_starting_num_workers': 10,
            'evaluation_dataflow_max_num_workers': 100,
            'evaluation_dataflow_disk_size_gb': 50,
            'dataflow_service_account': 'service-account',
            'dataflow_subnetwork': '',
            'dataflow_use_public_ips': True,
            'encryption_spec_key_name': '',
            'run_feature_selection': False,
        },
    )

  def test_get_wide_and_deep_hyperparameter_tuning_job_pipeline_and_parameters(
      self,
  ):
    _, parameter_values = (
        utils.get_wide_and_deep_hyperparameter_tuning_job_pipeline_and_parameters(
            'project',
            'us-central1',
            'gs://foo',
            'target',
            'classification',
            'loss',
            'MINIMIZE',
            [
                {
                    'parameter_id': 'dnn_learning_rate',
                    'double_value_spec': {
                        'min_value': 0.0001,
                        'max_value': 0.01,
                    },
                    'scale_type': 'UNIT_LINEAR_SCALE',
                },
                {
                    'parameter_id': 'learning_rate',
                    'double_value_spec': {
                        'min_value': 0.001,
                        'max_value': 0.01,
                    },
                    'scale_type': 'UNIT_LINEAR_SCALE',
                },
                {
                    'parameter_id': 'max_steps',
                    'discrete_value_spec': {'values': [2]},
                },
            ],
            2,
            1,
            data_source_csv_filenames='gs://foo/bar.csv',
            dataflow_service_account='service-account',
        )
    )
    self.assertEqual(
        parameter_values,
        {
            'project': 'project',
            'location': 'us-central1',
            'root_dir': 'gs://foo',
            'target_column': 'target',
            'prediction_type': 'classification',
            'dataset_level_custom_transformation_definitions': [],
            'dataset_level_transformations': [],
            'tf_auto_transform_features': {},
            'tf_custom_transformation_definitions': [],
            'materialized_examples_format': 'tfrecords_gzip',
            'tf_transform_execution_engine': 'dataflow',
            'data_source_csv_filenames': 'gs://foo/bar.csv',
            'study_spec_metric_id': 'loss',
            'study_spec_metric_goal': 'MINIMIZE',
            'study_spec_parameters_override': [
                {
                    'parameter_id': 'dnn_learning_rate',
                    'double_value_spec': {
                        'min_value': 0.0001,
                        'max_value': 0.01,
                    },
                    'scale_type': 'UNIT_LINEAR_SCALE',
                },
                {
                    'parameter_id': 'learning_rate',
                    'double_value_spec': {
                        'min_value': 0.001,
                        'max_value': 0.01,
                    },
                    'scale_type': 'UNIT_LINEAR_SCALE',
                },
                {
                    'parameter_id': 'max_steps',
                    'discrete_value_spec': {'values': [2]},
                },
            ],
            'max_trial_count': 2,
            'parallel_trial_count': 1,
            'enable_profiler': False,
            'cache_data': 'auto',
            'seed': 1,
            'eval_steps': 0,
            'eval_frequency_secs': 600,
            'weight_column': '',
            'max_failed_trial_count': 0,
            'study_spec_algorithm': 'ALGORITHM_UNSPECIFIED',
            'study_spec_measurement_selection_type': 'BEST_MEASUREMENT',
            'transform_dataflow_machine_type': 'n1-standard-16',
            'transform_dataflow_max_num_workers': 25,
            'transform_dataflow_disk_size_gb': 40,
            'worker_pool_specs_override': [],
            'run_evaluation': True,
            'evaluation_batch_predict_machine_type': 'n1-highmem-8',
            'evaluation_batch_predict_starting_replica_count': 20,
            'evaluation_batch_predict_max_replica_count': 20,
            'evaluation_dataflow_machine_type': 'n1-standard-4',
            'evaluation_dataflow_starting_num_workers': 10,
            'evaluation_dataflow_max_num_workers': 100,
            'evaluation_dataflow_disk_size_gb': 50,
            'dataflow_service_account': 'service-account',
            'dataflow_subnetwork': '',
            'dataflow_use_public_ips': True,
            'encryption_spec_key_name': '',
            'run_feature_selection': False,
        },
    )

  def test_get_tabnet_builtin_algorithm_hyperparameter_tuning_job_pipeline_and_parameters(
      self,
  ):
    _, parameter_values = (
        utils.get_builtin_algorithm_hyperparameter_tuning_job_pipeline_and_parameters(
            'project',
            'us-central1',
            'gs://foo',
            'target',
            'classification',
            'loss',
            'MINIMIZE',
            [
                {
                    'parameter_id': 'dnn_learning_rate',
                    'double_value_spec': {
                        'min_value': 0.0001,
                        'max_value': 0.01,
                    },
                    'scale_type': 'UNIT_LINEAR_SCALE',
                },
                {
                    'parameter_id': 'learning_rate',
                    'double_value_spec': {
                        'min_value': 0.001,
                        'max_value': 0.01,
                    },
                    'scale_type': 'UNIT_LINEAR_SCALE',
                },
                {
                    'parameter_id': 'max_steps',
                    'discrete_value_spec': {'values': [2]},
                },
            ],
            2,
            1,
            algorithm='tabnet',
            data_source_csv_filenames='gs://foo/bar.csv',
            dataflow_service_account='service-account',
        )
    )
    self.assertEqual(
        parameter_values,
        {
            'project': 'project',
            'location': 'us-central1',
            'root_dir': 'gs://foo',
            'target_column': 'target',
            'prediction_type': 'classification',
            'dataset_level_custom_transformation_definitions': [],
            'dataset_level_transformations': [],
            'tf_auto_transform_features': {},
            'tf_custom_transformation_definitions': [],
            'materialized_examples_format': 'tfrecords_gzip',
            'tf_transform_execution_engine': 'dataflow',
            'data_source_csv_filenames': 'gs://foo/bar.csv',
            'study_spec_metric_id': 'loss',
            'study_spec_metric_goal': 'MINIMIZE',
            'study_spec_parameters_override': [
                {
                    'parameter_id': 'dnn_learning_rate',
                    'double_value_spec': {
                        'min_value': 0.0001,
                        'max_value': 0.01,
                    },
                    'scale_type': 'UNIT_LINEAR_SCALE',
                },
                {
                    'parameter_id': 'learning_rate',
                    'double_value_spec': {
                        'min_value': 0.001,
                        'max_value': 0.01,
                    },
                    'scale_type': 'UNIT_LINEAR_SCALE',
                },
                {
                    'parameter_id': 'max_steps',
                    'discrete_value_spec': {'values': [2]},
                },
            ],
            'max_trial_count': 2,
            'parallel_trial_count': 1,
            'enable_profiler': False,
            'cache_data': 'auto',
            'seed': 1,
            'eval_steps': 0,
            'eval_frequency_secs': 600,
            'weight_column': '',
            'max_failed_trial_count': 0,
            'study_spec_algorithm': 'ALGORITHM_UNSPECIFIED',
            'study_spec_measurement_selection_type': 'BEST_MEASUREMENT',
            'transform_dataflow_machine_type': 'n1-standard-16',
            'transform_dataflow_max_num_workers': 25,
            'transform_dataflow_disk_size_gb': 40,
            'worker_pool_specs_override': [],
            'run_evaluation': True,
            'evaluation_batch_predict_machine_type': 'n1-highmem-8',
            'evaluation_batch_predict_starting_replica_count': 20,
            'evaluation_batch_predict_max_replica_count': 20,
            'evaluation_dataflow_machine_type': 'n1-standard-4',
            'evaluation_dataflow_starting_num_workers': 10,
            'evaluation_dataflow_max_num_workers': 100,
            'evaluation_dataflow_disk_size_gb': 50,
            'dataflow_service_account': 'service-account',
            'dataflow_subnetwork': '',
            'dataflow_use_public_ips': True,
            'encryption_spec_key_name': '',
            'run_feature_selection': False,
        },
    )

  def test_get_wide_and_deep_builtin_algorithm_hyperparameter_tuning_job_pipeline_and_parameters(
      self,
  ):
    _, parameter_values = (
        utils.get_builtin_algorithm_hyperparameter_tuning_job_pipeline_and_parameters(
            'project',
            'us-central1',
            'gs://foo',
            'target',
            'classification',
            'loss',
            'MINIMIZE',
            [
                {
                    'parameter_id': 'dnn_learning_rate',
                    'double_value_spec': {
                        'min_value': 0.0001,
                        'max_value': 0.01,
                    },
                    'scale_type': 'UNIT_LINEAR_SCALE',
                },
                {
                    'parameter_id': 'learning_rate',
                    'double_value_spec': {
                        'min_value': 0.001,
                        'max_value': 0.01,
                    },
                    'scale_type': 'UNIT_LINEAR_SCALE',
                },
                {
                    'parameter_id': 'max_steps',
                    'discrete_value_spec': {'values': [2]},
                },
            ],
            2,
            1,
            algorithm='wide_and_deep',
            data_source_csv_filenames='gs://foo/bar.csv',
            dataflow_service_account='service-account',
        )
    )
    self.assertEqual(
        parameter_values,
        {
            'project': 'project',
            'location': 'us-central1',
            'root_dir': 'gs://foo',
            'target_column': 'target',
            'prediction_type': 'classification',
            'dataset_level_custom_transformation_definitions': [],
            'dataset_level_transformations': [],
            'tf_auto_transform_features': {},
            'tf_custom_transformation_definitions': [],
            'materialized_examples_format': 'tfrecords_gzip',
            'tf_transform_execution_engine': 'dataflow',
            'data_source_csv_filenames': 'gs://foo/bar.csv',
            'study_spec_metric_id': 'loss',
            'study_spec_metric_goal': 'MINIMIZE',
            'study_spec_parameters_override': [
                {
                    'parameter_id': 'dnn_learning_rate',
                    'double_value_spec': {
                        'min_value': 0.0001,
                        'max_value': 0.01,
                    },
                    'scale_type': 'UNIT_LINEAR_SCALE',
                },
                {
                    'parameter_id': 'learning_rate',
                    'double_value_spec': {
                        'min_value': 0.001,
                        'max_value': 0.01,
                    },
                    'scale_type': 'UNIT_LINEAR_SCALE',
                },
                {
                    'parameter_id': 'max_steps',
                    'discrete_value_spec': {'values': [2]},
                },
            ],
            'max_trial_count': 2,
            'parallel_trial_count': 1,
            'enable_profiler': False,
            'cache_data': 'auto',
            'seed': 1,
            'eval_steps': 0,
            'eval_frequency_secs': 600,
            'weight_column': '',
            'max_failed_trial_count': 0,
            'study_spec_algorithm': 'ALGORITHM_UNSPECIFIED',
            'study_spec_measurement_selection_type': 'BEST_MEASUREMENT',
            'transform_dataflow_machine_type': 'n1-standard-16',
            'transform_dataflow_max_num_workers': 25,
            'transform_dataflow_disk_size_gb': 40,
            'worker_pool_specs_override': [],
            'run_evaluation': True,
            'evaluation_batch_predict_machine_type': 'n1-highmem-8',
            'evaluation_batch_predict_starting_replica_count': 20,
            'evaluation_batch_predict_max_replica_count': 20,
            'evaluation_dataflow_machine_type': 'n1-standard-4',
            'evaluation_dataflow_starting_num_workers': 10,
            'evaluation_dataflow_max_num_workers': 100,
            'evaluation_dataflow_disk_size_gb': 50,
            'dataflow_service_account': 'service-account',
            'dataflow_subnetwork': '',
            'dataflow_use_public_ips': True,
            'encryption_spec_key_name': '',
            'run_feature_selection': False,
        },
    )

  def test_get_tabnet_trainer_pipeline_and_parameters(self):
    _, parameter_values = utils.get_tabnet_trainer_pipeline_and_parameters(
        'project',
        'us-central1',
        'gs://foo',
        'target',
        'classification',
        0.01,
        data_source_csv_filenames='gs://foo/bar.csv',
        dataflow_service_account='service-account',
    )

    self.assertEqual(
        parameter_values,
        {
            'project': 'project',
            'location': 'us-central1',
            'root_dir': 'gs://foo',
            'target_column': 'target',
            'prediction_type': 'classification',
            'dataset_level_custom_transformation_definitions': [],
            'dataset_level_transformations': [],
            'tf_auto_transform_features': {},
            'tf_custom_transformation_definitions': [],
            'materialized_examples_format': 'tfrecords_gzip',
            'tf_transform_execution_engine': 'dataflow',
            'data_source_csv_filenames': 'gs://foo/bar.csv',
            'learning_rate': 0.01,
            'max_steps': -1,
            'max_train_secs': -1,
            'large_category_dim': 1,
            'large_category_thresh': 300,
            'yeo_johnson_transform': True,
            'feature_dim': 64,
            'feature_dim_ratio': 0.5,
            'num_decision_steps': 6,
            'relaxation_factor': 1.5,
            'decay_every': 100,
            'decay_rate': 0.95,
            'gradient_thresh': 2000,
            'sparsity_loss_weight': 1e-05,
            'batch_momentum': 0.95,
            'batch_size_ratio': 0.25,
            'num_transformer_layers': 4,
            'num_transformer_layers_ratio': 0.25,
            'class_weight': 1.0,
            'loss_function_type': 'default',
            'alpha_focal_loss': 0.25,
            'gamma_focal_loss': 2.0,
            'enable_profiler': False,
            'cache_data': 'auto',
            'seed': 1,
            'eval_steps': 0,
            'batch_size': 100,
            'eval_frequency_secs': 600,
            'weight_column': '',
            'transform_dataflow_machine_type': 'n1-standard-16',
            'transform_dataflow_max_num_workers': 25,
            'transform_dataflow_disk_size_gb': 40,
            'worker_pool_specs_override': [],
            'run_evaluation': True,
            'evaluation_batch_predict_machine_type': 'n1-highmem-8',
            'evaluation_batch_predict_starting_replica_count': 20,
            'evaluation_batch_predict_max_replica_count': 20,
            'evaluation_dataflow_machine_type': 'n1-standard-4',
            'evaluation_dataflow_starting_num_workers': 10,
            'evaluation_dataflow_max_num_workers': 100,
            'evaluation_dataflow_disk_size_gb': 50,
            'dataflow_service_account': 'service-account',
            'dataflow_subnetwork': '',
            'dataflow_use_public_ips': True,
            'encryption_spec_key_name': '',
            'run_feature_selection': False,
        },
    )

  def test_get_tabnet_study_spec_parameters_override_classification(self):
    study_spec_parameters_override = (
        utils.get_tabnet_study_spec_parameters_override(
            'medium', 'classification', 'medium'
        )
    )

    self.assertEqual(
        study_spec_parameters_override,
        [
            {
                'parameter_id': 'max_steps',
                'discrete_value_spec': {
                    'values': [5000, 10000, 20000, 30000, 40000, 50000]
                },
            },
            {
                'parameter_id': 'max_train_secs',
                'discrete_value_spec': {'values': [-1]},
            },
            {
                'parameter_id': 'batch_size',
                'discrete_value_spec': {
                    'values': [1024, 2048, 4096, 8192, 16384]
                },
            },
            {
                'parameter_id': 'learning_rate',
                'double_value_spec': {'min_value': 0.00007, 'max_value': 0.02},
                'scale_type': 'UNIT_LOG_SCALE',
            },
            {
                'parameter_id': 'large_category_dim',
                'discrete_value_spec': {'values': [5]},
            },
            {
                'parameter_id': 'large_category_thresh',
                'discrete_value_spec': {'values': [10]},
            },
            {
                'parameter_id': 'feature_dim',
                'integer_value_spec': {'min_value': 50, 'max_value': 400},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'feature_dim_ratio',
                'double_value_spec': {'min_value': 0.2, 'max_value': 0.8},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'num_decision_steps',
                'integer_value_spec': {'min_value': 2, 'max_value': 6},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'relaxation_factor',
                'double_value_spec': {'min_value': 1.2, 'max_value': 2.5},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'decay_rate',
                'double_value_spec': {'min_value': 0.5, 'max_value': 0.999},
                'scale_type': 'UNIT_LOG_SCALE',
            },
            {
                'parameter_id': 'decay_every',
                'integer_value_spec': {'min_value': 10000, 'max_value': 50000},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'sparsity_loss_weight',
                'double_value_spec': {
                    'min_value': 0.0000001,
                    'max_value': 0.001,
                },
                'scale_type': 'UNIT_LOG_SCALE',
            },
            {
                'parameter_id': 'batch_momentum',
                'double_value_spec': {'min_value': 0.5, 'max_value': 0.95},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'batch_size_ratio',
                'discrete_value_spec': {'values': [0.0625, 0.125, 0.25, 0.5]},
            },
            {
                'parameter_id': 'num_transformer_layers',
                'integer_value_spec': {'min_value': 4, 'max_value': 10},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'num_transformer_layers_ratio',
                'double_value_spec': {'min_value': 0.2, 'max_value': 0.8},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'class_weight',
                'double_value_spec': {'min_value': 1.0, 'max_value': 100.0},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'loss_function_type',
                'categorical_value_spec': {
                    'values': ['weighted_cross_entropy', 'focal_loss']
                },
            },
            {
                'parameter_id': 'alpha_focal_loss',
                'discrete_value_spec': {
                    'values': [0.1, 0.25, 0.5, 0.75, 0.9, 0.99]
                },
            },
            {
                'parameter_id': 'gamma_focal_loss',
                'discrete_value_spec': {
                    'values': [0.0, 0.5, 1.0, 2.0, 3.0, 4.0]
                },
            },
            {
                'parameter_id': 'yeo_johnson_transform',
                'categorical_value_spec': {'values': ['false']},
            },
        ],
    )

  def test_get_tabnet_study_spec_parameters_override_regression(self):
    study_spec_parameters_override = (
        utils.get_tabnet_study_spec_parameters_override(
            'medium', 'regression', 'large'
        )
    )

    self.assertEqual(
        study_spec_parameters_override,
        [
            {
                'parameter_id': 'max_steps',
                'discrete_value_spec': {
                    'values': [50000, 60000, 70000, 80000, 90000, 100000]
                },
            },
            {
                'parameter_id': 'max_train_secs',
                'discrete_value_spec': {'values': [-1]},
            },
            {
                'parameter_id': 'batch_size',
                'discrete_value_spec': {
                    'values': [1024, 2048, 4096, 8192, 16384]
                },
            },
            {
                'parameter_id': 'learning_rate',
                'double_value_spec': {'min_value': 0.00007, 'max_value': 0.03},
                'scale_type': 'UNIT_LOG_SCALE',
            },
            {
                'parameter_id': 'large_category_dim',
                'discrete_value_spec': {'values': [3, 5, 10]},
            },
            {
                'parameter_id': 'large_category_thresh',
                'discrete_value_spec': {'values': [5, 10]},
            },
            {
                'parameter_id': 'feature_dim',
                'integer_value_spec': {'min_value': 50, 'max_value': 500},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'feature_dim_ratio',
                'double_value_spec': {'min_value': 0.2, 'max_value': 0.8},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'num_decision_steps',
                'integer_value_spec': {'min_value': 2, 'max_value': 8},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'relaxation_factor',
                'double_value_spec': {'min_value': 1.05, 'max_value': 3.2},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'decay_rate',
                'double_value_spec': {'min_value': 0.5, 'max_value': 0.999},
                'scale_type': 'UNIT_LOG_SCALE',
            },
            {
                'parameter_id': 'decay_every',
                'integer_value_spec': {'min_value': 10000, 'max_value': 50000},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'sparsity_loss_weight',
                'double_value_spec': {'min_value': 0.0000001, 'max_value': 100},
                'scale_type': 'UNIT_LOG_SCALE',
            },
            {
                'parameter_id': 'batch_momentum',
                'double_value_spec': {'min_value': 0.5, 'max_value': 0.95},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'batch_size_ratio',
                'discrete_value_spec': {'values': [0.0625, 0.125, 0.25, 0.5]},
            },
            {
                'parameter_id': 'num_transformer_layers',
                'integer_value_spec': {'min_value': 4, 'max_value': 10},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'num_transformer_layers_ratio',
                'double_value_spec': {'min_value': 0.2, 'max_value': 0.8},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'loss_function_type',
                'categorical_value_spec': {'values': ['mae', 'mse']},
            },
            {
                'parameter_id': 'yeo_johnson_transform',
                'categorical_value_spec': {'values': ['false', 'true']},
            },
        ],
    )

  def test_get_wide_and_deep_study_spec_parameters_override(self):
    study_spec_parameters_override = (
        utils.get_wide_and_deep_study_spec_parameters_override()
    )

    self.assertEqual(
        study_spec_parameters_override,
        [
            {
                'parameter_id': 'max_steps',
                'discrete_value_spec': {
                    'values': [5000, 10000, 20000, 30000, 40000, 50000]
                },
            },
            {
                'parameter_id': 'max_train_secs',
                'discrete_value_spec': {'values': [-1]},
            },
            {
                'parameter_id': 'learning_rate',
                'double_value_spec': {'min_value': 0.0001, 'max_value': 0.0005},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'optimizer_type',
                'categorical_value_spec': {'values': ['adam', 'ftrl', 'sgd']},
            },
            {
                'parameter_id': 'l1_regularization_strength',
                'discrete_value_spec': {'values': [0, 0.01, 0.02]},
            },
            {
                'parameter_id': 'l2_regularization_strength',
                'discrete_value_spec': {'values': [0, 0.01, 0.02]},
            },
            {
                'parameter_id': 'l2_shrinkage_regularization_strength',
                'discrete_value_spec': {'values': [0, 0.01, 0.02]},
            },
            {
                'parameter_id': 'beta_1',
                'discrete_value_spec': {'values': [0.7, 0.8, 0.9]},
            },
            {
                'parameter_id': 'beta_2',
                'discrete_value_spec': {'values': [0.8, 0.9, 0.999]},
            },
            {
                'parameter_id': 'hidden_units',
                'categorical_value_spec': {'values': ['30,30,30']},
            },
            {
                'parameter_id': 'use_wide',
                'categorical_value_spec': {'values': ['true', 'false']},
            },
            {
                'parameter_id': 'embed_categories',
                'categorical_value_spec': {'values': ['true', 'false']},
            },
            {
                'parameter_id': 'dnn_dropout',
                'discrete_value_spec': {'values': [0, 0.1, 0.2]},
            },
            {
                'parameter_id': 'dnn_learning_rate',
                'double_value_spec': {'min_value': 0.0001, 'max_value': 0.0005},
                'scale_type': 'UNIT_LINEAR_SCALE',
            },
            {
                'parameter_id': 'dnn_optimizer_type',
                'categorical_value_spec': {'values': ['adam', 'ftrl', 'sgd']},
            },
            {
                'parameter_id': 'dnn_l1_regularization_strength',
                'discrete_value_spec': {'values': [0, 0.01, 0.02]},
            },
            {
                'parameter_id': 'dnn_l2_regularization_strength',
                'discrete_value_spec': {'values': [0, 0.01, 0.02]},
            },
            {
                'parameter_id': 'dnn_l2_shrinkage_regularization_strength',
                'discrete_value_spec': {'values': [0, 0.01, 0.02]},
            },
            {
                'parameter_id': 'dnn_beta_1',
                'discrete_value_spec': {'values': [0.7, 0.8, 0.9]},
            },
            {
                'parameter_id': 'dnn_beta_2',
                'discrete_value_spec': {'values': [0.8, 0.9, 0.999]},
            },
            {
                'parameter_id': 'batch_size',
                'discrete_value_spec': {
                    'values': [1024, 2048, 4096, 8192, 16384]
                },
            },
        ],
    )

  def test_get_xgboost_study_spec_parameters_override(self):
    study_spec_parameters_override = (
        utils.get_xgboost_study_spec_parameters_override()
    )

    self.assertEqual(
        study_spec_parameters_override,
        [
            {
                'parameter_id': 'num_boost_round',
                'discrete_value_spec': {'values': [1, 5, 10, 15, 20]},
            },
            {
                'parameter_id': 'early_stopping_rounds',
                'discrete_value_spec': {'values': [3, 5, 10]},
            },
            {
                'parameter_id': 'base_score',
                'discrete_value_spec': {'values': [0.5]},
            },
            {
                'parameter_id': 'booster',
                'categorical_value_spec': {
                    'values': ['gbtree', 'gblinear', 'dart']
                },
                'conditional_parameter_specs': [
                    {
                        'parameter_spec': {
                            'parameter_id': 'eta',
                            'double_value_spec': {
                                'min_value': 0.0001,
                                'max_value': 1.0,
                            },
                            'scale_type': 'UNIT_LOG_SCALE',
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'gamma',
                            'discrete_value_spec': {
                                'values': [0, 10, 50, 100, 500, 1000]
                            },
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'max_depth',
                            'integer_value_spec': {
                                'min_value': 6,
                                'max_value': 10,
                            },
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'min_child_weight',
                            'double_value_spec': {
                                'min_value': 0.0,
                                'max_value': 10.0,
                            },
                            'scale_type': 'UNIT_LINEAR_SCALE',
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'max_delta_step',
                            'discrete_value_spec': {
                                'values': [0.0, 1.0, 3.0, 5.0, 7.0, 9.0]
                            },
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'subsample',
                            'double_value_spec': {
                                'min_value': 0.0001,
                                'max_value': 1.0,
                            },
                            'scale_type': 'UNIT_LINEAR_SCALE',
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'colsample_bytree',
                            'double_value_spec': {
                                'min_value': 0.0001,
                                'max_value': 1.0,
                            },
                            'scale_type': 'UNIT_LINEAR_SCALE',
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'colsample_bylevel',
                            'double_value_spec': {
                                'min_value': 0.0001,
                                'max_value': 1.0,
                            },
                            'scale_type': 'UNIT_LINEAR_SCALE',
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'colsample_bynode',
                            'double_value_spec': {
                                'min_value': 0.0001,
                                'max_value': 1.0,
                            },
                            'scale_type': 'UNIT_LINEAR_SCALE',
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'lambda',
                            'double_value_spec': {
                                'min_value': 0.0001,
                                'max_value': 1.0,
                            },
                            'scale_type': 'UNIT_REVERSE_LOG_SCALE',
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart', 'gblinear']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'alpha',
                            'double_value_spec': {
                                'min_value': 0.0001,
                                'max_value': 1.0,
                            },
                            'scale_type': 'UNIT_LOG_SCALE',
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart', 'gblinear']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'tree_method',
                            'categorical_value_spec': {'values': ['auto']},
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'scale_pos_weight',
                            'discrete_value_spec': {'values': [1.0]},
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'refresh_leaf',
                            'discrete_value_spec': {'values': [1]},
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'process_type',
                            'categorical_value_spec': {'values': ['default']},
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'grow_policy',
                            'categorical_value_spec': {'values': ['depthwise']},
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'sampling_method',
                            'categorical_value_spec': {'values': ['uniform']},
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'sample_type',
                            'categorical_value_spec': {'values': ['uniform']},
                        },
                        'parent_categorical_values': {'values': ['dart']},
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'normalize_type',
                            'categorical_value_spec': {'values': ['tree']},
                        },
                        'parent_categorical_values': {'values': ['dart']},
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'rate_drop',
                            'discrete_value_spec': {'values': [0.0]},
                        },
                        'parent_categorical_values': {'values': ['dart']},
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'one_drop',
                            'discrete_value_spec': {'values': [0]},
                        },
                        'parent_categorical_values': {'values': ['dart']},
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'skip_drop',
                            'discrete_value_spec': {'values': [0.0]},
                        },
                        'parent_categorical_values': {'values': ['dart']},
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'num_parallel_tree',
                            'discrete_value_spec': {'values': [1]},
                        },
                        'parent_categorical_values': {'values': ['gblinear']},
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'feature_selector',
                            'categorical_value_spec': {'values': ['cyclic']},
                        },
                        'parent_categorical_values': {'values': ['gblinear']},
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'top_k',
                            'discrete_value_spec': {'values': [0]},
                        },
                        'parent_categorical_values': {'values': ['gblinear']},
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'max_leaves',
                            'discrete_value_spec': {'values': [0]},
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart']
                        },
                    },
                    {
                        'parameter_spec': {
                            'parameter_id': 'max_bin',
                            'discrete_value_spec': {'values': [256]},
                        },
                        'parent_categorical_values': {
                            'values': ['gbtree', 'dart']
                        },
                    },
                ],
            },
        ],
    )

  def test_get_xgboost_trainer_pipeline_and_parameters(self):
    _, parameter_values = utils.get_xgboost_trainer_pipeline_and_parameters(
        'project',
        'us-central1',
        'gs://foo',
        'target',
        'multi:softprob',
        data_source_csv_filenames='gs://foo/bar.csv',
        dataflow_service_account='service-account',
    )

    self.assertEqual(
        parameter_values,
        {
            'data_source_csv_filenames': 'gs://foo/bar.csv',
            'dataflow_service_account': 'service-account',
            'dataset_level_custom_transformation_definitions': [],
            'dataset_level_transformations': [],
            'location': 'us-central1',
            'objective': 'multi:softprob',
            'project': 'project',
            'root_dir': 'gs://foo',
            'target_column': 'target',
            'tf_auto_transform_features': {},
            'tf_custom_transformation_definitions': [],
        },
    )

  def test_get_xgboost_hyperparameter_tuning_job_pipeline_and_parameters(self):
    _, parameter_values = (
        utils.get_xgboost_hyperparameter_tuning_job_pipeline_and_parameters(
            'project',
            'us-central1',
            'gs://foo',
            'target',
            'multi:softprob',
            'logloss',
            'MINIMIZE',
            10,
            5,
            data_source_csv_filenames='gs://foo/bar.csv',
            dataflow_service_account='service-account',
        )
    )

    self.assertEqual(
        parameter_values,
        {
            'data_source_csv_filenames': 'gs://foo/bar.csv',
            'dataflow_service_account': 'service-account',
            'dataset_level_custom_transformation_definitions': [],
            'dataset_level_transformations': [],
            'location': 'us-central1',
            'objective': 'multi:softprob',
            'study_spec_metric_id': 'logloss',
            'study_spec_metric_goal': 'MINIMIZE',
            'max_trial_count': 10,
            'parallel_trial_count': 5,
            'project': 'project',
            'root_dir': 'gs://foo',
            'study_spec_parameters_override': [],
            'target_column': 'target',
            'tf_auto_transform_features': {},
            'tf_custom_transformation_definitions': [],
        },
    )


if __name__ == '__main__':
  unittest.main()
