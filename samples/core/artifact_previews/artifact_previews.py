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
"""Automatic previews for ordinary Parquet and XGBoost directory artifacts."""

import os
from pathlib import Path
from typing import List, NamedTuple

from kfp import compiler
from kfp import dsl

_PACKAGES = Path(__file__).with_name('requirements.txt').read_text(
    encoding='utf-8').splitlines()


@dsl.component(
    base_image='python:3.11',
    packages_to_install=_PACKAGES,
    kfp_package_path=os.getenv('KFP_PACKAGE_PATH'),
)
def create_preview_artifacts(
    dataset: dsl.Output[dsl.Dataset],
    model: dsl.Output[dsl.Model],
    seed: int = 42,
) -> None:
    """Write native files; the executor generates their previews by default.

    Args:
        dataset: Directory containing data.parquet.
        model: Directory containing model.ubj.
        seed: Random seed for the synthetic data and model.
    """
    from pathlib import Path

    import numpy as np
    import pyarrow as pa
    import pyarrow.parquet as pq
    import xgboost as xgb

    dataset_directory = Path(dataset.path)
    model_directory = Path(model.path)
    dataset_directory.mkdir(parents=True, exist_ok=True)
    model_directory.mkdir(parents=True, exist_ok=True)

    features = np.random.default_rng(seed).normal(size=(32, 24))
    labels = (features[:, 0] > np.median(features[:, 0])).astype(np.int32)
    data = {
        f'feature_{index:02d}': features[:, index]
        for index in range(features.shape[1])
    }
    data['label'] = labels
    pq.write_table(pa.table(data), dataset_directory / 'data.parquet')

    parameters = {
        'objective': 'binary:logistic',
        'n_estimators': 5,
        'max_depth': 2,
        'learning_rate': 0.1,
        'n_jobs': 1,
        'random_state': seed,
    }
    estimator = xgb.XGBClassifier(**parameters)
    estimator.fit(features, labels)
    estimator.save_model(model_directory / 'model.ubj')
    model.metadata['framework'] = 'xgboost'


@dsl.pipeline(name='artifact-previews')
def artifact_preview_pipeline() -> None:
    """Demonstrates previews without a separate preview-generating task."""
    create_preview_artifacts()


@dsl.pipeline(name='artifact-preview-lists')
def artifact_preview_list_pipeline() -> NamedTuple('Outputs', [(
    'datasets', List[dsl.Dataset]), ('models', List[dsl.Model])]):
    """Collect artifact references while retaining each member's own
    preview."""
    with dsl.ParallelFor([42, 43], parallelism=1) as seed:
        task = create_preview_artifacts(seed=seed)
    outputs = NamedTuple('Outputs', [('datasets', List[dsl.Dataset]),
                                     ('models', List[dsl.Model])])
    return outputs(
        dsl.Collected(task.outputs['dataset']),
        dsl.Collected(task.outputs['model']))


if __name__ == '__main__':
    compiler.Compiler().compile(
        artifact_preview_pipeline, package_path='artifact_previews.yaml')
