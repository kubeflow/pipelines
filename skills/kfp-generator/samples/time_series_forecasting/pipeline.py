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

"""Production-Grade KFP v2 Time-Series Demand Forecasting Pipeline."""

import os

from kfp import compiler, dsl

_KFP_PACKAGE_PATH = os.getenv('KFP_PACKAGE_PATH')


@dsl.component(
    kfp_package_path=_KFP_PACKAGE_PATH,
    packages_to_install=['pandas', 'numpy', 'scikit-learn'],
)
def extract_telemetry_metrics(
    processed_dataset: dsl.OutputPath(dsl.Dataset),
    num_steps: int = 500,
    num_lags: int = 3,
    random_state: int = 42,
):
    """Loads historical sequence of numerical readings, forward-fills missing values, and creates rolling lag features."""
    import os
    import numpy as np
    import pandas as pd

    np.random.seed(random_state)
    time_idx = pd.date_range(start='2026-01-01', periods=num_steps, freq='h')

    trend = np.linspace(10, 50, num_steps)
    seasonality = 10 * np.sin(np.pi * np.arange(num_steps) / 12.0)
    noise = np.random.normal(0, 2, num_steps)
    readings = trend + seasonality + noise

    nan_mask = np.random.rand(num_steps) < 0.05
    readings[nan_mask] = np.nan

    df = pd.DataFrame({'timestamp': time_idx, 'demand': readings})
    df['demand'] = df['demand'].ffill().bfill()

    for lag in range(1, num_lags + 1):
        df[f'lag_{lag}'] = df['demand'].shift(lag)

    df = df.dropna().reset_index(drop=True)

    os.makedirs(os.path.dirname(processed_dataset), exist_ok=True)
    df.to_csv(processed_dataset, index=False)


@dsl.component(
    kfp_package_path=_KFP_PACKAGE_PATH,
    packages_to_install=['pandas', 'scikit-learn', 'joblib'],
)
def train_regressor(
    processed_dataset: dsl.InputPath(dsl.Dataset),
    train_dataset: dsl.OutputPath(dsl.Dataset),
    test_dataset: dsl.OutputPath(dsl.Dataset),
    model: dsl.OutputPath(dsl.Model),
    alpha: float = 1.0,
    test_size: float = 0.2,
    random_state: int = 42,
):
    """Trains a Ridge Regression model on historical lag sequences and splits dataset into train/test."""
    import os
    import joblib
    import pandas as pd
    from sklearn.linear_model import Ridge
    from sklearn.model_selection import train_test_split

    df = pd.read_csv(processed_dataset)
    feature_cols = [c for c in df.columns if c.startswith('lag_')]
    target_col = 'demand'

    X = df[feature_cols]
    y = df[target_col]

    X_train, X_test, y_train, y_test = train_test_split(
        X, y, test_size=test_size, random_state=random_state, shuffle=False
    )

    reg = Ridge(alpha=float(alpha), random_state=random_state)
    reg.fit(X_train, y_train)

    train_df = pd.concat([X_train, y_train], axis=1)
    test_df = pd.concat([X_test, y_test], axis=1)

    os.makedirs(os.path.dirname(train_dataset), exist_ok=True)
    os.makedirs(os.path.dirname(test_dataset), exist_ok=True)
    os.makedirs(os.path.dirname(model), exist_ok=True)

    train_df.to_csv(train_dataset, index=False)
    test_df.to_csv(test_dataset, index=False)
    joblib.dump(reg, model)


@dsl.component(
    kfp_package_path=_KFP_PACKAGE_PATH,
    packages_to_install=['pandas', 'scikit-learn', 'joblib'],
)
def evaluate_forecast(
    test_dataset: dsl.InputPath(dsl.Dataset),
    model_path: dsl.InputPath(dsl.Model),
    metrics: dsl.Output[dsl.Metrics],
    eval_report: dsl.OutputPath(str),
):
    """Computes MAE and R-squared metrics, logs to KFP Metrics artifact, and outputs text summary report."""
    import os
    import joblib
    import pandas as pd
    from sklearn.metrics import mean_absolute_error, r2_score

    test_df = pd.read_csv(test_dataset)
    X_test = test_df.drop(columns=['demand'])
    y_test = test_df['demand']

    reg = joblib.load(model_path)
    y_pred = reg.predict(X_test)

    mae = float(mean_absolute_error(y_test, y_pred))
    r2 = float(r2_score(y_test, y_pred))

    metrics.log_metric('mae', mae)
    metrics.log_metric('r2_score', r2)

    report_text = f"Time-Series Forecast Evaluation Report\n"
    report_text += f"====================================\n"
    report_text += f"Mean Absolute Error (MAE): {mae:.4f}\n"
    report_text += f"R-squared (R2 Score):      {r2:.4f}\n"

    os.makedirs(os.path.dirname(eval_report), exist_ok=True)
    with open(eval_report, 'w') as f:
        f.write(report_text)

    print(report_text)


@dsl.pipeline(
    name='time-series-forecasting-pipeline',
    description='Production-grade KFP v2 time-series demand forecasting pipeline.',
)
def time_series_forecasting_pipeline(
    num_steps: int = 500,
    num_lags: int = 3,
    alpha: float = 1.0,
    test_size: float = 0.2,
    random_state: int = 42,
):
    """Orchestrates telemetry metrics extraction, regressor training, and forecast evaluation."""
    extract_task = extract_telemetry_metrics(
        num_steps=num_steps,
        num_lags=num_lags,
        random_state=random_state,
    )

    train_task = train_regressor(
        processed_dataset=extract_task.outputs['processed_dataset'],
        alpha=alpha,
        test_size=test_size,
        random_state=random_state,
    )

    evaluate_forecast(
        test_dataset=train_task.outputs['test_dataset'],
        model_path=train_task.outputs['model'],
    )


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=time_series_forecasting_pipeline,
        package_path=__file__ + '.yaml',
    )