# Copyright 2024 The Kubeflow Authors
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
"""Pipeline demonstrating Pydantic-based input validation.

This example shows how to use Pydantic models for robust input validation
in KFP components, improving reliability and providing clear error messages.
"""

from kfp import compiler
from kfp import dsl


@dsl.component(
    base_image="public.ecr.aws/docker/library/python:3.12",
    packages_to_install=["pydantic>=2.0,<3"]
)
def validate_training_config(
    learning_rate: float,
    epochs: int,
    batch_size: int,
    model_type: str
) -> str:
    """Validate training configuration using Pydantic.
    
    Args:
        learning_rate: Learning rate for training (0 < lr <= 1)
        epochs: Number of training epochs (1-1000)
        batch_size: Batch size (must be power of 2)
        model_type: Type of model (cnn, rnn, or transformer)
    
    Returns:
        Validation status message
    """
    from pydantic import BaseModel, Field, field_validator
    
    class TrainingConfig(BaseModel):
        """Validated training configuration."""
        learning_rate: float = Field(gt=0, le=1, description="Learning rate")
        epochs: int = Field(gt=0, le=1000, description="Number of epochs")
        batch_size: int = Field(gt=0, description="Batch size")
        model_type: str = Field(pattern="^(cnn|rnn|transformer)$", description="Model architecture")
        
        @field_validator('batch_size')
        @classmethod
        def batch_size_power_of_two(cls, v):
            """Ensure batch_size is a power of two."""
            if v & (v - 1) != 0:
                raise ValueError(f'batch_size must be power of 2, got {v}')
            return v
    
    # Validate configuration
    config = TrainingConfig(
        learning_rate=learning_rate,
        epochs=epochs,
        batch_size=batch_size,
        model_type=model_type
    )
    
    status = f"✓ Configuration valid: lr={config.learning_rate}, epochs={config.epochs}, batch_size={config.batch_size}, model={config.model_type}"
    print(status)
    return status


@dsl.component(
    base_image="public.ecr.aws/docker/library/python:3.12",
    packages_to_install=["pydantic>=2.0,<3"]
)
def validate_data_paths(
    train_path: str,
    valid_path: str,
    test_path: str
) -> str:
    """Validate data file paths.
    
    Args:
        train_path: Path to training data
        valid_path: Path to validation data
        test_path: Path to test data
    
    Returns:
        Validation status
    """
    from pydantic import BaseModel, Field
    
    class DataPaths(BaseModel):
        """Validated data paths."""
        train_path: str = Field(min_length=1, description="Training data path")
        valid_path: str = Field(min_length=1, description="Validation data path")
        test_path: str = Field(min_length=1, description="Test data path")
    
    paths = DataPaths(
        train_path=train_path,
        valid_path=valid_path,
        test_path=test_path
    )
    
    status = f"✓ Paths validated: train={paths.train_path}, valid={paths.valid_path}, test={paths.test_path}"
    print(status)
    return status


@dsl.component(base_image="public.ecr.aws/docker/library/python:3.12")
def mock_training(config_status: str, paths_status: str) -> str:
    """Mock training task that uses validated inputs.
    
    Args:
        config_status: Configuration validation status
        paths_status: Paths validation status
    
    Returns:
        Training completion message
    """
    print("Validation results:")
    print(config_status)
    print(paths_status)
    print("Starting model training with validated configuration...")
    return "Training completed successfully!"


@dsl.pipeline(
    name='pipeline-with-pydantic-validation',
    description='Demonstrates Pydantic validation for pipeline inputs'
)
def validated_pipeline(
    learning_rate: float = 0.001,
    epochs: int = 10,
    batch_size: int = 32,
    model_type: str = "cnn",
    train_data_path: str = "/data/train.csv",
    valid_data_path: str = "/data/valid.csv",
    test_data_path: str = "/data/test.csv"
) -> str:
    """Pipeline with Pydantic-validated inputs.
    
    Args:
        learning_rate: Learning rate (0 < lr <= 1)
        epochs: Number of epochs (1-1000)
        batch_size: Batch size (power of 2)
        model_type: Model type (cnn/rnn/transformer)
        train_data_path: Path to training data
        valid_data_path: Path to validation data
        test_data_path: Path to test data
    
    Returns:
        Training result message
    """
    # Validate training configuration
    config_validation = validate_training_config(
        learning_rate=learning_rate,
        epochs=epochs,
        batch_size=batch_size,
        model_type=model_type
    )
    
    # Validate data paths
    paths_validation = validate_data_paths(
        train_path=train_data_path,
        valid_path=valid_data_path,
        test_path=test_data_path
    )
    
    # Run training with validated inputs
    training_result = mock_training(
        config_status=config_validation.output,
        paths_status=paths_validation.output
    )
    
    return training_result.output


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=validated_pipeline,
        package_path=__file__.replace('.py', '.yaml')
    )
