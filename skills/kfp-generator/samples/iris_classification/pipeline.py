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

import os

from kfp import compiler, dsl
from kfp.dsl import Dataset, Input, Metrics, Model, Output

_KFP_PACKAGE_PATH = os.getenv('KFP_PACKAGE_PATH')


@dsl.component(
    kfp_package_path=_KFP_PACKAGE_PATH,
    packages_to_install=['scikit-learn', 'pandas'],
)
def download_iris_dataset(dataset: Output[Dataset]):
    """Downloads the Iris dataset using sklearn and saves it as a CSV file."""
    import os
    import pandas as pd
    from sklearn.datasets import load_iris

    iris = load_iris(as_frame=True)
    df = iris.frame

    os.makedirs(os.path.dirname(dataset.path), exist_ok=True)
    df.to_csv(dataset.path, index=False)


@dsl.component(
    kfp_package_path=_KFP_PACKAGE_PATH,
    packages_to_install=['scikit-learn', 'pandas'],
)
def preprocess_and_split_data(
    dataset: Input[Dataset],
    train_dataset: Output[Dataset],
    test_dataset: Output[Dataset],
    test_size: float = 0.2,
    random_state: int = 42,
):
    """Preprocesses the Iris dataset and splits it into train and test sets."""
    import os
    import pandas as pd
    from sklearn.model_selection import train_test_split

    df = pd.read_csv(dataset.path)
    train_df, test_df = train_test_split(
        df, test_size=test_size, random_state=random_state, stratify=df['target']
    )

    os.makedirs(os.path.dirname(train_dataset.path), exist_ok=True)
    os.makedirs(os.path.dirname(test_dataset.path), exist_ok=True)

    train_df.to_csv(train_dataset.path, index=False)
    test_df.to_csv(test_dataset.path, index=False)


@dsl.component(
    kfp_package_path=_KFP_PACKAGE_PATH,
    packages_to_install=['scikit-learn', 'pandas', 'joblib'],
)
def train_iris_model(
    train_dataset: Input[Dataset],
    model: Output[Model],
    n_estimators: int = 100,
    max_depth: int = 3,
    random_state: int = 42,
):
    """Trains a Random Forest classifier model on the Iris training dataset."""
    import os
    import joblib
    import pandas as pd
    from sklearn.ensemble import RandomForestClassifier

    train_df = pd.read_csv(train_dataset.path)
    x_train = train_df.drop(columns=['target'])
    y_train = train_df['target']

    classifier = RandomForestClassifier(
        n_estimators=n_estimators,
        max_depth=max_depth,
        random_state=random_state,
    )
    classifier.fit(x_train, y_train)

    os.makedirs(os.path.dirname(model.path), exist_ok=True)
    joblib.dump(classifier, model.path)


@dsl.component(
    kfp_package_path=_KFP_PACKAGE_PATH,
    packages_to_install=['scikit-learn', 'pandas', 'joblib'],
)
def evaluate_iris_model(
    test_dataset: Input[Dataset],
    model: Input[Model],
    metrics: Output[Metrics],
):
    """Evaluates the trained model on the test dataset and logs evaluation metrics."""
    import joblib
    import pandas as pd
    from sklearn.metrics import accuracy_score, f1_score, precision_score, recall_score

    test_df = pd.read_csv(test_dataset.path)
    x_test = test_df.drop(columns=['target'])
    y_test = test_df['target']

    classifier = joblib.load(model.path)
    y_pred = classifier.predict(x_test)

    accuracy = float(accuracy_score(y_test, y_pred))
    precision = float(precision_score(y_test, y_pred, average='macro'))
    recall = float(recall_score(y_test, y_pred, average='macro'))
    f1 = float(f1_score(y_test, y_pred, average='macro'))

    metrics.log_metric('accuracy', accuracy)
    metrics.log_metric('precision', precision)
    metrics.log_metric('recall', recall)
    metrics.log_metric('f1_score', f1)

    print(f'Accuracy: {accuracy:.4f}')
    print(f'Precision: {precision:.4f}')
    print(f'Recall: {recall:.4f}')
    print(f'F1 Score: {f1:.4f}')


@dsl.pipeline(
    name='iris-training-pipeline',
    description='A complete end-to-end Iris dataset training and evaluation pipeline.',
)
def iris_training_pipeline(
    test_size: float = 0.2,
    random_state: int = 42,
    n_estimators: int = 100,
    max_depth: int = 3,
):
    download_task = download_iris_dataset()
    preprocess_task = preprocess_and_split_data(
        dataset=download_task.outputs['dataset'],
        test_size=test_size,
        random_state=random_state,
    )
    train_task = train_iris_model(
        train_dataset=preprocess_task.outputs['train_dataset'],
        n_estimators=n_estimators,
        max_depth=max_depth,
        random_state=random_state,
    )
    evaluate_task = evaluate_iris_model(
        test_dataset=preprocess_task.outputs['test_dataset'],
        model=train_task.outputs['model'],
    )


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=iris_training_pipeline,
        package_path=__file__ + '.yaml',
    )