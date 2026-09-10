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

"""Production-grade KFP v2 Customer Churn Prediction Pipeline."""

from kfp import compiler, dsl


@dsl.component(
    packages_to_install=['pandas', 'numpy', 'scikit-learn']
)
def load_churn_data(
    dataset_train: dsl.OutputPath(dsl.Dataset),
    dataset_test: dsl.OutputPath(dsl.Dataset),
    num_samples: int = 1000,
    random_state: int = 42,
):
    """Ingests raw tabular user interaction records, scales continuous features dynamically using a standard scaler, and performs a stratified train/test split."""
    import pandas as pd
    import numpy as np
    from sklearn.model_selection import train_test_split
    from sklearn.preprocessing import StandardScaler

    np.random.seed(random_state)

    # Generate synthetic tabular user interaction and churn records
    tenure = np.random.randint(1, 72, size=num_samples)
    monthly_charges = np.random.uniform(18.0, 120.0, size=num_samples)
    total_charges = tenure * monthly_charges + np.random.normal(0, 10, size=num_samples)
    num_support_tickets = np.random.poisson(lam=2, size=num_samples)
    contract_type = np.random.choice([0, 1, 2], size=num_samples)

    churn_logit = (
        -0.05 * tenure
        + 0.02 * monthly_charges
        + 0.3 * num_support_tickets
        - 0.8 * contract_type
    )
    churn_prob = 1 / (1 + np.exp(-churn_logit))
    churn = (churn_prob > np.median(churn_prob)).astype(int)

    df = pd.DataFrame({
        'tenure': tenure,
        'monthly_charges': monthly_charges,
        'total_charges': total_charges,
        'num_support_tickets': num_support_tickets,
        'contract_type': contract_type,
        'churn': churn,
    })

    # Scale continuous features dynamically using StandardScaler
    continuous_cols = ['tenure', 'monthly_charges', 'total_charges', 'num_support_tickets']
    scaler = StandardScaler()
    df[continuous_cols] = scaler.fit_transform(df[continuous_cols])

    # Stratified train/test split
    train_df, test_df = train_test_split(
        df,
        test_size=0.2,
        random_state=random_state,
        stratify=df['churn'],
    )

    train_df.to_csv(dataset_train, index=False)
    test_df.to_csv(dataset_test, index=False)


@dsl.component(
    packages_to_install=['pandas', 'numpy', 'scikit-learn', 'xgboost', 'joblib']
)
def train_xgboost_model(
    dataset_train: dsl.InputPath(dsl.Dataset),
    model: dsl.OutputPath(dsl.Model),
    n_estimators: int = 100,
    max_depth: int = 4,
    learning_rate: float = 0.1,
):
    """Fits an XGBoost classifier on the preprocessed training split, tracking tuning parameters."""
    import pandas as pd
    import xgboost as xgb
    import joblib

    train_df = pd.read_csv(dataset_train)
    X_train = train_df.drop(columns=['churn'])
    y_train = train_df['churn']

    clf = xgb.XGBClassifier(
        n_estimators=n_estimators,
        max_depth=max_depth,
        learning_rate=float(learning_rate),
        random_state=42,
        eval_metric='logloss',
    )
    clf.fit(X_train, y_train)

    joblib.dump(clf, model)


@dsl.component(
    packages_to_install=['pandas', 'numpy', 'scikit-learn', 'xgboost', 'joblib']
)
def evaluate_and_validate(
    dataset_test: dsl.InputPath(dsl.Dataset),
    model_path: dsl.InputPath(dsl.Model),
    metrics: dsl.Output[dsl.Metrics],
    model_artifact: dsl.OutputPath(dsl.Model),
):
    """Computes ROC-AUC, Precision, and Recall metrics, exporting them to native KFP Metrics artifact, and saves the serialized model."""
    import pandas as pd
    import joblib
    from sklearn.metrics import roc_auc_score, precision_score, recall_score

    test_df = pd.read_csv(dataset_test)
    X_test = test_df.drop(columns=['churn'])
    y_test = test_df['churn']

    clf = joblib.load(model_path)

    y_pred_proba = clf.predict_proba(X_test)[:, 1]
    y_pred = clf.predict(X_test)

    roc_auc = float(roc_auc_score(y_test, y_pred_proba))
    precision = float(precision_score(y_test, y_pred, zero_division=0))
    recall = float(recall_score(y_test, y_pred, zero_division=0))

    metrics.log_metric('roc_auc', roc_auc)
    metrics.log_metric('precision', precision)
    metrics.log_metric('recall', recall)

    joblib.dump(clf, model_artifact)


@dsl.pipeline(
    name='customer-churn-prediction-pipeline',
    description='Production-grade KFP v2 customer churn prediction pipeline.',
)
def customer_churn_pipeline(
    num_samples: int = 1000,
    random_state: int = 42,
    n_estimators: int = 100,
    max_depth: int = 4,
    learning_rate: float = 0.1,
):
    """Customer churn prediction pipeline orchestrating data loading, training, and evaluation."""
    load_task = load_churn_data(
        num_samples=num_samples,
        random_state=random_state,
    )

    train_task = train_xgboost_model(
        dataset_train=load_task.outputs['dataset_train'],
        n_estimators=n_estimators,
        max_depth=max_depth,
        learning_rate=learning_rate,
    )

    evaluate_and_validate(
        dataset_test=load_task.outputs['dataset_test'],
        model_path=train_task.outputs['model'],
    )


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=customer_churn_pipeline,
        package_path=__file__ + '.yaml',
    )