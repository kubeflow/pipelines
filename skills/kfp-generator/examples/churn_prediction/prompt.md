@kfp-generator Create a production-grade KFP v2 customer churn prediction pipeline named 'pipeline.py' under a self-contained evaluation directory.

The pipeline must orchestrate the following operational steps:
1. load_churn_data: Ingests raw tabular user interaction records, scales continuous features dynamically using a standard scaler, and performs a stratified train/test split.
2. train_xgboost_model: Fits an XGBoost or Random Forest classifier on the preprocessed training split, tracking tuning parameters.
3. evaluate_and_validate: Computes ROC-AUC, Precision, and Recall metrics, exporting them into a native KFP Metrics artifact. It must save the optimized model artifact using a serialized format.

Ensure the pipeline script includes proper Apache 2.0 headers, explicit dsl.InputPath/dsl.OutputPath type annotations for clean component data linkage, and the standard compiler block. Run our local evaluation script on it to guarantee a valid KFP v2 YAML intermediate representation spec is produced.