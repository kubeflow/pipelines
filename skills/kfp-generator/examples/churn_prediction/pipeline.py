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

from kfp import dsl, compiler
from kfp.dsl import Input, Output, Dataset, Model, Metrics


@dsl.component(
    base_image='python:3.9',
    packages_to_install=['pandas', 'scikit-learn']
)
def load_churn_data(
    num_samples: int,
    train_dataset: Output[Dataset],
    test_dataset: Output[Dataset]
):
    """Ingests raw tabular customer churn data, scales continuous features dynamically,
    and performs a stratified train/test split.

    Args:
        num_samples: Number of synthetic customer records to generate.
        train_dataset: Output path for scaled training dataset.
        test_dataset: Output path for scaled testing dataset.
    """
    import pandas as pd
    import numpy as np
    from sklearn.model_selection import train_test_split
    from sklearn.preprocessing import StandardScaler

    # Generate synthetic tabular customer churn data
    np.random.seed(42)
    data = {
        'tenure': np.random.randint(1, 72, size=num_samples),
        'monthly_charges': np.random.uniform(20.0, 120.0, size=num_samples),
        'total_charges': np.random.uniform(20.0, 8000.0, size=num_samples),
        'support_calls': np.random.randint(0, 10, size=num_samples),
        'payment_delay': np.random.randint(0, 30, size=num_samples),
    }
    df = pd.DataFrame(data)
    
    # Generate target 'churn' based on features to simulate relationship
    # Higher support calls, payment delays, and monthly charges increase churn risk
    churn_prob = (
        0.1 * df['support_calls'] + 
        0.05 * df['payment_delay'] + 
        0.005 * df['monthly_charges'] - 
        0.02 * df['tenure']
    )
    # Map probability to binary labels using sigmoid and threshold
    probs = 1 / (1 + np.exp(-churn_prob))
    df['churn'] = (probs > 0.5).astype(int)

    # Continuous features to scale
    continuous_cols = ['tenure', 'monthly_charges', 'total_charges', 'support_calls', 'payment_delay']
    
    # Stratified train/test split
    train_df, test_df = train_test_split(
        df, test_size=0.2, stratify=df['churn'], random_state=42
    )
    
    # Scale continuous features using StandardScaler
    scaler = StandardScaler()
    train_df[continuous_cols] = scaler.fit_transform(train_df[continuous_cols])
    test_df[continuous_cols] = scaler.transform(test_df[continuous_cols])

    # Write out to output dataset paths
    train_df.to_csv(train_dataset.path, index=False)
    test_df.to_csv(test_dataset.path, index=False)


@dsl.component(
    base_image='python:3.9',
    packages_to_install=['pandas', 'scikit-learn', 'xgboost']
)
def train_xgboost_model(
    train_dataset: Input[Dataset],
    model: Output[Model],
    n_estimators: int = 100,
    max_depth: int = 6,
    learning_rate: float = 0.3
):
    """Fits an XGBoost classifier on the preprocessed training split, tracking tuning parameters.

    Args:
        train_dataset: Input training dataset.
        model: Output model artifact path.
        n_estimators: Number of boosting rounds.
        max_depth: Maximum tree depth.
        learning_rate: Boosting learning rate.
    """
    import pickle
    import pandas as pd
    from xgboost import XGBClassifier

    # Load the preprocessed training dataset
    train_df = pd.read_csv(train_dataset.path)
    
    # Split features and label
    X_train = train_df.drop(columns=['churn'])
    y_train = train_df['churn']

    # Initialize and fit XGBoost classifier
    clf = XGBClassifier(
        n_estimators=n_estimators,
        max_depth=max_depth,
        learning_rate=learning_rate,
        random_state=42,
        use_label_encoder=False,
        eval_metric='logloss'
    )
    clf.fit(X_train, y_train)

    # Save the optimized model artifact using a serialized format (pickle)
    with open(model.path, 'wb') as f:
        pickle.dump(clf, f)

    # Track tuning parameters in model metadata
    model.metadata['n_estimators'] = n_estimators
    model.metadata['max_depth'] = max_depth
    model.metadata['learning_rate'] = learning_rate
    model.metadata['framework'] = 'xgboost'


@dsl.component(
    base_image='python:3.9',
    packages_to_install=['pandas', 'scikit-learn', 'xgboost']
)
def evaluate_and_validate(
    test_dataset: Input[Dataset],
    model: Input[Model],
    metrics: Output[Metrics]
):
    """Computes ROC-AUC, Precision, and Recall metrics, exporting them into a native KFP Metrics artifact.

    Args:
        test_dataset: Input testing dataset.
        model: Input model artifact.
        metrics: Output KFP metrics artifact to record evaluations.
    """
    import pickle
    import pandas as pd
    from sklearn.metrics import roc_auc_score, precision_score, recall_score

    # Load test dataset
    test_df = pd.read_csv(test_dataset.path)
    X_test = test_df.drop(columns=['churn'])
    y_test = test_df['churn']

    # Load model
    with open(model.path, 'rb') as f:
        clf = pickle.load(f)

    # Predictions and probabilities
    y_pred = clf.predict(X_test)
    y_prob = clf.predict_proba(X_test)[:, 1]

    # Compute metrics
    roc_auc = float(roc_auc_score(y_test, y_prob))
    precision = float(precision_score(y_test, y_pred))
    recall = float(recall_score(y_test, y_pred))

    # Log metrics to native KFP Metrics artifact
    metrics.log_metric('roc_auc', roc_auc)
    metrics.log_metric('precision', precision)
    metrics.log_metric('recall', recall)


@dsl.pipeline(
    name='customer-churn-prediction-pipeline',
    description='A production-grade v2 customer churn prediction pipeline.'
)
def churn_prediction_pipeline(
    num_samples: int = 1000,
    n_estimators: int = 100,
    max_depth: int = 6,
    learning_rate: float = 0.3
):
    """A customer churn prediction pipeline with train split, scaling, training and evaluation."""
    
    # Load and scale data
    load_data_task = load_churn_data(num_samples=num_samples)
    
    # Train the XGBoost model
    train_task = train_xgboost_model(
        train_dataset=load_data_task.outputs['train_dataset'],
        n_estimators=n_estimators,
        max_depth=max_depth,
        learning_rate=learning_rate
    )
    
    # Evaluate model performance
    evaluate_task = evaluate_and_validate(
        test_dataset=load_data_task.outputs['test_dataset'],
        model=train_task.outputs['model']
    )


if __name__ == '__main__':
    compiler.Compiler().compile(churn_prediction_pipeline, __file__ + '.yaml')
