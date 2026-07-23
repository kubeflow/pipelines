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
def download_iris_dataset(
    dataset_path: dsl.OutputPath(str),
) -> int:
    """Download the Iris dataset from scikit-learn and persist it to disk."""
    import joblib
    from sklearn.datasets import load_iris

    iris = load_iris()
    joblib.dump(
        {
            'features': iris.data,
            'target': iris.target,
            'feature_names': iris.feature_names,
            'target_names': iris.target_names,
        },
        dataset_path,
    )
    return len(iris.target)


@dsl.component(packages_to_install=_SKLEARN_PACKAGES)
def preprocess_iris_dataset(
    input_dataset_path: dsl.InputPath(str),
    train_features_path: dsl.OutputPath(str),
    test_features_path: dsl.OutputPath(str),
    train_labels_path: dsl.OutputPath(str),
    test_labels_path: dsl.OutputPath(str),
    test_size: float = 0.2,
    random_state: int = 42,
) -> int:
    """Standardize features and split the Iris dataset into train and test sets."""
    import joblib
    from sklearn.model_selection import train_test_split
    from sklearn.preprocessing import StandardScaler

    dataset = joblib.load(input_dataset_path)
    features = dataset['features']
    labels = dataset['target']

    scaler = StandardScaler()
    scaled_features = scaler.fit_transform(features)

    train_features, test_features, train_labels, test_labels = train_test_split(
        scaled_features,
        labels,
        test_size=test_size,
        random_state=random_state,
        stratify=labels,
    )

    joblib.dump({'features': train_features, 'scaler': scaler}, train_features_path)
    joblib.dump({'features': test_features, 'scaler': scaler}, test_features_path)
    joblib.dump(train_labels, train_labels_path)
    joblib.dump(test_labels, test_labels_path)

    return len(train_labels)


@dsl.component(packages_to_install=_SKLEARN_PACKAGES)
def train_iris_model(
    train_features_path: dsl.InputPath(str),
    train_labels_path: dsl.InputPath(str),
    model_path: dsl.OutputPath(str),
) -> str:
    """Train a logistic regression classifier on the preprocessed training data."""
    import joblib
    from sklearn.linear_model import LogisticRegression

    train_features_bundle = joblib.load(train_features_path)
    train_labels = joblib.load(train_labels_path)

    model = LogisticRegression(max_iter=200, random_state=42)
    model.fit(train_features_bundle['features'], train_labels)

    joblib.dump(model, model_path)
    return model.__class__.__name__


@dsl.component(packages_to_install=_SKLEARN_PACKAGES)
def evaluate_iris_model(
    test_features_path: dsl.InputPath(str),
    test_labels_path: dsl.InputPath(str),
    model_path: dsl.InputPath(str),
) -> float:
    """Evaluate the trained model and return classification accuracy."""
    import joblib
    from sklearn.metrics import accuracy_score

    test_features_bundle = joblib.load(test_features_path)
    test_labels = joblib.load(test_labels_path)
    model = joblib.load(model_path)

    predictions = model.predict(test_features_bundle['features'])
    return float(accuracy_score(test_labels, predictions))


@dsl.pipeline(
    name='iris-training-pipeline',
    description='Download Iris data, preprocess, train a classifier, and evaluate accuracy.',
)
def iris_training_pipeline(test_size: float = 0.2, random_state: int = 42):
    """End-to-end Iris classification pipeline with ETL, training, and evaluation."""
    download_task = download_iris_dataset()

    preprocess_task = preprocess_iris_dataset(
        input_dataset_path=download_task.outputs['dataset_path'],
        test_size=test_size,
        random_state=random_state,
    )

    train_task = train_iris_model(
        train_features_path=preprocess_task.outputs['train_features_path'],
        train_labels_path=preprocess_task.outputs['train_labels_path'],
    )

    evaluate_iris_model(
        test_features_path=preprocess_task.outputs['test_features_path'],
        test_labels_path=preprocess_task.outputs['test_labels_path'],
        model_path=train_task.outputs['model_path'],
    )


if __name__ == '__main__':
    compiler.Compiler().compile(iris_training_pipeline, __file__ + '.yaml')
