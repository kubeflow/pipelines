from TrainingJob.src.TrainingJob_spec import SageMakerTrainingJobSpec
import unittest


class TrainingSpecTestCase(unittest.TestCase):

    REQUIRED_ARGS = [
        "--region",
        "us-west-1",
        "--algorithm_specification",
        "{'trainingImage': '746614075791.dkr.ecr.us-west-1.amazonaws.com/sagemaker-xgboost:1.2-1', 'trainingInputMode': 'File'}",
        "--hyper_parameters",
        "{'max_depth': '2', 'gamma': '10', 'eta': '0.3', 'min_child_weight': '6', 'objective': 'multi:softmax', 'num_class': '10', 'num_round': '10'}",
        "--input_data_config",
        "[]",
        "--output_data_config",
        "{'s3OutputPath': 's3://ack-sagemaker-bucket-123456789012'}",
        "--resource_config",
        "{'instanceCount': 1, 'instanceType': 'ml.m4.xlarge', 'volumeSizeInGB': 5}",
        "--role_arn",
        "arn:aws:iam::123456789012:role/ack-sagemaker-execution-role",
        "--stopping_condition",
        "{'maxRuntimeInSeconds': 86400}",
        "--training_job_name",
        "kfp-ack-training-job-99999",
    ]
    INCORRECT_ARGS = [
        "--empty"
    ]
    


    def test_minimum_required_args(self):
        # Will raise an exception if the inputs are incorrect
        spec = SageMakerTrainingJobSpec(self.REQUIRED_ARGS)
    
    def test_incorrect_args(self):
        # Will raise an exception if the inputs are incorrect
        
        with self.assertRaises(SystemExit):
            spec = SageMakerTrainingJobSpec(self.INCORRECT_ARGS)


if __name__ == "__main__":
    unittest.main()
