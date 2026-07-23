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

from kfp import compiler, dsl

_SKLEARN_PACKAGES = ['scikit-learn', 'numpy', 'joblib']


@dsl.component(packages_to_install=_SKLEARN_PACKAGES)
def download_mnist_dataset(
    dataset: dsl.Output[dsl.Dataset],
):
    """Downloads the MNIST dataset from OpenML and persists it as an artifact."""
    import joblib
    from sklearn.datasets import fetch_openml

    # Fetch MNIST dataset (784 features, 28x28 images)
    # Using as_frame=False to get numpy arrays directly
    mnist = fetch_openml('mnist_784', version=1, as_frame=False, parser='liac-arff')

    joblib.dump(
        {
            'data': mnist.data,
            'target': mnist.target,
        },
        dataset.path,
    )


@dsl.component(packages_to_install=_SKLEARN_PACKAGES)
def preprocess_mnist_dataset(
    dataset: dsl.Input[dsl.Dataset],
    train_dataset: dsl.Output[dsl.Dataset],
    test_dataset: dsl.Output[dsl.Dataset],
    test_size: float = 0.2,
    random_state: int = 42,
):
    """Normalizes the image data to [0, 1], flattens them, and splits into train and test sets."""
    import joblib
    from sklearn.model_selection import train_test_split

    loaded = joblib.load(dataset.path)
    data = loaded['data']
    target = loaded['target']

    # Normalize image pixel values from [0, 255] to [0, 1]
    normalized_data = data / 255.0

    # Ensure the data is flattened (MNIST is already flat 784, but we reshape to be explicit)
    num_samples = normalized_data.shape[0]
    flattened_data = normalized_data.reshape((num_samples, -1))

    X_train, X_test, y_train, y_test = train_test_split(
        flattened_data,
        target,
        test_size=test_size,
        random_state=random_state,
        stratify=target,
    )

    joblib.dump({'X': X_train, 'y': y_train}, train_dataset.path)
    joblib.dump({'X': X_test, 'y': y_test}, test_dataset.path)


@dsl.component(packages_to_install=_SKLEARN_PACKAGES)
def train_mnist_classifier(
    train_dataset: dsl.Input[dsl.Dataset],
    model: dsl.Output[dsl.Model],
    n_estimators: int = 50,
    random_state: int = 42,
):
    """Trains a Random Forest classifier on the training MNIST dataset."""
    import joblib
    from sklearn.ensemble import RandomForestClassifier

    train_data = joblib.load(train_dataset.path)
    X_train = train_data['X']
    y_train = train_data['y']

    clf = RandomForestClassifier(
        n_estimators=n_estimators,
        random_state=random_state,
        n_jobs=-1,
    )
    clf.fit(X_train, y_train)

    joblib.dump(clf, model.path)


@dsl.component(packages_to_install=_SKLEARN_PACKAGES)
def evaluate_mnist_classifier(
    test_dataset: dsl.Input[dsl.Dataset],
    model: dsl.Input[dsl.Model],
    metrics: dsl.Output[dsl.Metrics],
) -> str:
    """Evaluates the trained model on test data, logging metrics and returning a report string."""
    import joblib
    from sklearn.metrics import accuracy_score, precision_score, recall_score, classification_report

    test_data = joblib.load(test_dataset.path)
    X_test = test_data['X']
    y_test = test_data['y']

    clf = joblib.load(model.path)
    y_pred = clf.predict(X_test)

    # Compute metrics
    accuracy = accuracy_score(y_test, y_pred)
    precision = precision_score(y_test, y_pred, average='macro')
    recall = recall_score(y_test, y_pred, average='macro')

    # Log metrics to KFP Metrics artifact
    metrics.log_metric('accuracy', float(accuracy))
    metrics.log_metric('precision', float(precision))
    metrics.log_metric('recall', float(recall))

    # Generate classification report
    report_str = classification_report(y_test, y_pred)

    evaluation_report = (
        "MNIST Classification Evaluation Report\n"
        "======================================\n"
        f"Accuracy:  {accuracy:.4f}\n"
        f"Precision (macro): {precision:.4f}\n"
        f"Recall (macro):    {recall:.4f}\n\n"
        "Detailed Classification Report:\n"
        f"{report_str}"
    )

    return evaluation_report


@dsl.pipeline(
    name='mnist-classification-pipeline',
    description='A pipeline that downloads MNIST, preprocesses the data, trains a Random Forest model, and evaluates it.',
)
def mnist_classification_pipeline(
    test_size: float = 0.2,
    n_estimators: int = 50,
    random_state: int = 42,
):
    """MNIST classification pipeline definition."""
    download_task = download_mnist_dataset()

    preprocess_task = preprocess_mnist_dataset(
        dataset=download_task.outputs['dataset'],
        test_size=test_size,
        random_state=random_state,
    )

    train_task = train_mnist_classifier(
        train_dataset=preprocess_task.outputs['train_dataset'],
        n_estimators=n_estimators,
        random_state=random_state,
    )

    evaluate_mnist_classifier(
        test_dataset=preprocess_task.outputs['test_dataset'],
        model=train_task.outputs['model'],
    )


if __name__ == '__main__':
    compiler.Compiler().compile(mnist_classification_pipeline, __file__ + '.yaml')
