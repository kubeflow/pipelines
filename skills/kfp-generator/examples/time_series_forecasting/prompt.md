@kfp-generator Create a production-grade KFP v2 time-series demand forecasting pipeline named 'pipeline.py' under a self-contained evaluation directory.

The pipeline must string together these explicit tasks:
1. extract_telemetry_metrics: Loads a historical sequence of numerical readings, handles missing values using forward-filling imputation logic, and creates rolling lag features.
2. train_regressor: Trains a Ridge Regression or Gradient Boosting model to forecast future data steps based on the engineered lag sequences.
3. evaluate_forecast: Computes Mean Absolute Error (MAE) and R-squared ($R^2$) metrics, mapping them into a clean KFP Metrics component structure, and outputs a string-formatted text summary report.

The python script must enforce strict v2 syntax rules (no legacy v1 components), include proper Apache 2.0 licenses, and append the core compiler execution block. Run the local evaluation runner script to guarantee the pipeline compiles flawlessly to a valid YAML target on disk.