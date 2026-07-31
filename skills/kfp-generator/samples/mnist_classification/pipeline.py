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

"""KFP v2 MNIST Classification Training Pipeline."""

from kfp import compiler, dsl


@dsl.component(
    base_image='python:3.11',
    packages_to_install=['scikit-learn', 'numpy', 'pandas'],
)
def download_mnist(
    dataset: dsl.OutputPath(dsl.Dataset),
    num_samples: int = 2000,
):
    """Downloads the MNIST dataset from OpenML and saves raw arrays."""
    import numpy as np
    from sklearn.datasets import fetch_openml

    X, y = fetch_openml(
        'mnist_784',
        version=1,
        return_X_y=True,
        as_frame=False,
        parser='auto',
    )
    # Subset dataset for fast training/evaluation if specified
    if num_samples > 0 and num_samples < len(X):
        X = X[:num_samples]
        y = y[:num_samples]

    X = X.astype(np.float32)
    y = np.array(y, dtype=np.int64)

    with open(dataset, 'wb') as f:
        np.savez_compressed(f, X=X, y=y)


@dsl.component(
    base_image='python:3.11',
    packages_to_install=['scikit-learn', 'numpy'],
)
def preprocess_and_normalize(
    dataset_raw: dsl.InputPath(dsl.Dataset),
    dataset_train: dsl.OutputPath(dsl.Dataset),
    dataset_test: dsl.OutputPath(dsl.Dataset),
    test_size: float = 0.2,
    random_state: int = 42,
):
    """Normalizes image pixel values to [0, 1], flattens them, and performs train/test split."""
    import numpy as np
    from sklearn.model_selection import train_test_split

    with open(dataset_raw, 'rb') as f:
        data = np.load(f)
        X = data['X'].astype(np.float32) / 255.0
        y = data['y'].astype(np.int64)

    X_train, X_test, y_train, y_test = train_test_split(
        X,
        y,
        test_size=test_size,
        random_state=random_state,
        stratify=y,
    )

    with open(dataset_train, 'wb') as f:
        np.savez_compressed(f, X=X_train, y=y_train)

    with open(dataset_test, 'wb') as f:
        np.savez_compressed(f, X=X_test, y=y_test)


@dsl.component(
    base_image='python:3.11',
    packages_to_install=['scikit-learn', 'numpy', 'joblib'],
)
def train_classifier(
    dataset_train: dsl.InputPath(dsl.Dataset),
    model: dsl.OutputPath(dsl.Model),
    n_estimators: int = 50,
    random_state: int = 42,
):
    """Trains a Random Forest classifier on the normalized training split."""
    import joblib
    import numpy as np
    from sklearn.ensemble import RandomForestClassifier

    with open(dataset_train, 'rb') as f:
        train_data = np.load(f)
        X_train = train_data['X']
        y_train = train_data['y']

    classifier = RandomForestClassifier(
        n_estimators=n_estimators,
        random_state=random_state,
    )
    classifier.fit(X_train, y_train)

    joblib.dump(classifier, model)


@dsl.component(
    base_image='python:3.11',
    packages_to_install=['scikit-learn', 'numpy', 'joblib'],
)
def evaluate_model(
    dataset_test: dsl.InputPath(dsl.Dataset),
    model_path: dsl.InputPath(dsl.Model),
    metrics: dsl.Output[dsl.Metrics],
    report: dsl.OutputPath(str),
):
    """Evaluates the classifier on test data, logs accuracy, precision, recall, and saves report string."""
    import joblib
    import numpy as np
    from sklearn.metrics import (
        accuracy_score,
        classification_report,
        precision_score,
        recall_score,
    )

    with open(dataset_test, 'rb') as f:
        test_data = np.load(f)
        X_test = test_data['X']
        y_test = test_data['y']

    classifier = joblib.load(model_path)
    y_pred = classifier.predict(X_test)

    acc = float(accuracy_score(y_test, y_pred))
    prec = float(precision_score(y_test, y_pred, average='macro', zero_division=0))
    rec = float(recall_score(y_test, y_pred, average='macro', zero_division=0))

    metrics.log_metric('accuracy', acc)
    metrics.log_metric('precision', prec)
    metrics.log_metric('recall', rec)

    report_str = (
        f"=== MNIST Classification Evaluation Report ===\n"
        f"Accuracy:  {acc:.4f}\n"
        f"Precision: {prec:.4f}\n"
        f"Recall:    {rec:.4f}\n\n"
        f"Detailed Classification Report:\n"
        f"{classification_report(y_test, y_pred, zero_division=0)}"
    )

    with open(report, 'w') as f:
        f.write(report_str)


@dsl.pipeline(
    name='mnist-classification-pipeline',
    description='KFP v2 training pipeline for MNIST classification.',
)
def mnist_classification_pipeline(
    num_samples: int = 2000,
    test_size: float = 0.2,
    n_estimators: int = 50,
    random_state: int = 42,
):
    """MNIST classification pipeline orchestrating dataset download, preprocessing, training, and evaluation."""
    download_task = download_mnist(
        num_samples=num_samples,
    )

    preprocess_task = preprocess_and_normalize(
        dataset_raw=download_task.outputs['dataset'],
        test_size=test_size,
        random_state=random_state,
    )

    train_task = train_classifier(
        dataset_train=preprocess_task.outputs['dataset_train'],
        n_estimators=n_estimators,
        random_state=random_state,
    )

    evaluate_model(
        dataset_test=preprocess_task.outputs['dataset_test'],
        model_path=train_task.outputs['model'],
    )


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=mnist_classification_pipeline,
        package_path=__file__ + '.yaml',
    )