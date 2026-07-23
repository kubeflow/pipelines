# Copyright 2026 The Kubeflow Authors
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

from typing import NamedTuple
from kfp import dsl, compiler
from kfp.dsl import Input, Output, Dataset, Model, Metrics


@dsl.component(
    base_image='python:3.9',
    packages_to_install=['pandas', 'numpy']
)
def extract_telemetry_metrics(
    num_readings: int,
    preprocessed_data: Output[Dataset]
):
    """Loads a historical sequence of numerical readings, handles missing values
    using forward-filling imputation logic, and creates rolling lag features.

    Args:
        num_readings: Number of telemetry metrics sequence steps to generate.
        preprocessed_data: Output dataset path to write the preprocessed features.
    """
    import pandas as pd
    import numpy as np

    # Generate synthetic time-series demand/telemetry data
    np.random.seed(42)
    time_index = pd.date_range(start='2026-01-01', periods=num_readings, freq='H')
    
    # Generate demand with a daily seasonality and random noise
    base_demand = 100 + 30 * np.sin(2 * np.pi * time_index.hour / 24.0)
    noise = np.random.normal(0, 10, size=num_readings)
    demand = base_demand + noise
    
    # Inject missing values (NaNs) to demonstrate forward-filling imputation
    mask = np.random.choice([True, False], size=num_readings, p=[0.05, 0.95])
    demand[mask] = np.nan
    
    df = pd.DataFrame({'timestamp': time_index, 'demand': demand})
    
    # Impute missing values using forward-filling (ffill)
    # For robust imputation, fill any remaining NaNs at the beginning with backward-filling (bfill)
    df['demand'] = df['demand'].ffill().bfill()
    
    # Create rolling lag features (lag_1, lag_2, lag_3)
    df['lag_1'] = df['demand'].shift(1)
    df['lag_2'] = df['demand'].shift(2)
    df['lag_3'] = df['demand'].shift(3)
    
    # Drop rows with NaN values created by lag operations
    df = df.dropna().reset_index(drop=True)
    
    # Save preprocessed telemetry dataset
    df.to_csv(preprocessed_data.path, index=False)


@dsl.component(
    base_image='python:3.9',
    packages_to_install=['pandas', 'scikit-learn']
)
def train_regressor(
    preprocessed_data: Input[Dataset],
    model: Output[Model],
    alpha: float = 1.0
):
    """Trains a Ridge regression model to forecast future demand steps based on lag features.

    Args:
        preprocessed_data: Input preprocessed dataset containing features and target.
        model: Output model artifact path.
        alpha: Ridge regression regularization strength.
    """
    import pickle
    import pandas as pd
    from sklearn.linear_model import Ridge

    # Load dataset
    df = pd.read_csv(preprocessed_data.path)
    
    # Features: lag features; Target: current demand
    feature_cols = ['lag_1', 'lag_2', 'lag_3']
    X = df[feature_cols]
    y = df['demand']

    # Train-test split (80% train, 20% test for internal evaluation / validation)
    # Since it's a time-series, perform a sequential split instead of random split
    split_idx = int(len(df) * 0.8)
    X_train, y_train = X.iloc[:split_idx], y.iloc[:split_idx]

    # Train Ridge regression model
    reg = Ridge(alpha=alpha)
    reg.fit(X_train, y_train)

    # Save the serialized model artifact
    with open(model.path, 'wb') as f:
        pickle.dump(reg, f)

    # Record hyperparameters in model metadata
    model.metadata['alpha'] = alpha
    model.metadata['algorithm'] = 'RidgeRegression'
    model.metadata['framework'] = 'scikit-learn'


@dsl.component(
    base_image='python:3.9',
    packages_to_install=['pandas', 'scikit-learn']
)
def evaluate_forecast(
    preprocessed_data: Input[Dataset],
    model: Input[Model],
    metrics: Output[Metrics]
) -> NamedTuple('Outputs', report=str):
    """Computes Mean Absolute Error (MAE) and R-squared (R2) metrics, exports them
    into a clean KFP Metrics structure, and returns a summary report.

    Args:
        preprocessed_data: Input preprocessed dataset.
        model: Input trained model.
        metrics: Output metrics artifact.

    Returns:
        Outputs: NamedTuple containing the report text.
    """
    import pickle
    import pandas as pd
    from typing import NamedTuple
    from sklearn.metrics import mean_absolute_error, r2_score

    # Load dataset
    df = pd.read_csv(preprocessed_data.path)
    feature_cols = ['lag_1', 'lag_2', 'lag_3']
    X = df[feature_cols]
    y = df['demand']

    # Load model
    with open(model.path, 'rb') as f:
        reg = pickle.load(f)

    # Perform predictions on the testing portion (the last 20% of sequence)
    split_idx = int(len(df) * 0.8)
    X_test, y_test = X.iloc[split_idx:], y.iloc[split_idx:]
    
    y_pred = reg.predict(X_test)

    # Compute metrics
    mae = float(mean_absolute_error(y_test, y_pred))
    r2 = float(r2_score(y_test, y_pred))

    # Log metrics to native KFP Metrics artifact
    metrics.log_metric('mae', mae)
    metrics.log_metric('r2_score', r2)

    # Generate string-formatted text summary report
    report_text = (
        f"=== Demand Forecasting Evaluation Report ===\n"
        f"Model Algorithm: {model.metadata.get('algorithm', 'Unknown')}\n"
        f"Regularization Alpha: {model.metadata.get('alpha', 'N/A')}\n"
        f"Total Test Samples Evaluated: {len(y_test)}\n"
        f"Mean Absolute Error (MAE): {mae:.4f}\n"
        f"R-squared (R^2): {r2:.4f}\n"
        f"============================================"
    )

    outputs = NamedTuple('Outputs', report=str)
    return outputs(report_text)


@dsl.pipeline(
    name='time-series-demand-forecasting-pipeline',
    description='A production-grade v2 time-series demand forecasting pipeline.'
)
def demand_forecasting_pipeline(
    num_readings: int = 500,
    alpha: float = 1.0
):
    """A pipeline to simulate telemetry data preprocessing, regression training, and forecast evaluation."""
    
    # 1. Load telemetry metrics and preprocess lag features
    extract_task = extract_telemetry_metrics(num_readings=num_readings)
    
    # 2. Train forecasting regressor
    train_task = train_regressor(
        preprocessed_data=extract_task.outputs['preprocessed_data'],
        alpha=alpha
    )
    
    # 3. Evaluate regressor performance and output report
    evaluate_task = evaluate_forecast(
        preprocessed_data=extract_task.outputs['preprocessed_data'],
        model=train_task.outputs['model']
    )


if __name__ == '__main__':
    compiler.Compiler().compile(demand_forecasting_pipeline, __file__ + '.yaml')
