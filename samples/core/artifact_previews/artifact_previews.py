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
"""Native dataframe/model outputs with default-on, pipeline-controlled
previews."""

import os
from pathlib import Path
from typing import List, NamedTuple

from kfp import compiler, dsl

_REQUIREMENTS = (
    Path(__file__)
    .with_name('requirements.txt')
    .read_text(encoding='utf-8')
    .splitlines()
)
_PACKAGES = [line for line in _REQUIREMENTS if not line.startswith('--')]
_PACKAGE_INDEXES = ['https://pypi.org/simple'] + [
    line.split(maxsplit=1)[1]
    for line in _REQUIREMENTS
    if line.startswith('--extra-index-url ')
]


@dsl.component(
    base_image='python:3.11',
    packages_to_install=_PACKAGES,
    pip_index_urls=_PACKAGE_INDEXES,
    kfp_package_path=os.getenv('KFP_PACKAGE_PATH'),
)
def create_preview_artifacts(
    dataset: dsl.Output[dsl.Dataset],
    model: dsl.Output[dsl.Model],
    seed: int = 42,
) -> None:
    """Write native files; the SDK generates their previews by default.

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


@dsl.component(
    base_image='python:3.11',
    packages_to_install=_PACKAGES,
    pip_index_urls=_PACKAGE_INDEXES,
    kfp_package_path=os.getenv('KFP_PACKAGE_PATH'),
)
def create_dataframe_artifact(
    dataset: dsl.Output[dsl.Dataset],
    library: str = 'pandas',
    file_format: str = 'parquet',
) -> None:
    """Write synthetic dataframe data using native Pandas or Polars file APIs.

    Args:
        dataset: Directory containing one native data file.
        library: Pandas or Polars; neither needs preview-specific calls.
        file_format: Parquet, CSV, JSON records/columns/split, JSONL, Arrow,
            Arrow stream, or Feather. Columns/split JSON requires Pandas.
    """
    from pathlib import Path

    import pyarrow as pa

    directory = Path(dataset.path)
    directory.mkdir(parents=True, exist_ok=True)
    data = {
        'record_id': list(range(32)),
        'category': [
            'group_a' if row % 2 == 0 else 'group_b' for row in range(32)
        ],
        'score': [row / 8 for row in range(32)],
        'accepted': [row % 2 == 0 for row in range(32)],
        **{
            f'feature_{column:02d}': [column * 10 + row for row in range(32)]
            for column in range(21)
        },
    }
    if library == 'pandas':
        import pandas as pd

        frame = pd.DataFrame(data)
    elif library == 'polars':
        import polars as pl

        frame = pl.DataFrame(data)
    else:
        raise ValueError(f'Unsupported dataframe library: {library}')

    if file_format in ('arrow', 'arrow-stream'):
        table = (
            pa.Table.from_pandas(frame, preserve_index=False)
            if library == 'pandas'
            else frame.to_arrow()
        )
        writer = (
            pa.ipc.new_stream
            if file_format == 'arrow-stream'
            else pa.ipc.new_file
        )
        with pa.OSFile(str(directory / 'data.arrow'), 'wb') as output:
            with writer(output, table.schema) as stream:
                stream.write_table(table, max_chunksize=8)
    elif file_format == 'parquet':
        if library == 'pandas':
            frame.to_parquet(directory / 'data.parquet', index=False)
        else:
            frame.write_parquet(directory / 'data.parquet')
    elif file_format == 'csv':
        if library == 'pandas':
            frame.to_csv(directory / 'data.csv', index=False)
        else:
            frame.write_csv(directory / 'data.csv')
    elif file_format == 'feather':
        if library == 'pandas':
            frame.to_feather(directory / 'data.feather')
        else:
            frame.write_ipc(directory / 'data.feather', compression='lz4')
    elif file_format in ('json', 'jsonl', 'json-columns', 'json-split'):
        path = directory / (
            'data.jsonl' if file_format == 'jsonl' else 'data.json'
        )
        if library == 'pandas':
            orientation = (
                file_format.removeprefix('json-')
                if '-' in file_format
                else 'records'
            )
            frame.to_json(
                path, orient=orientation, lines=file_format == 'jsonl'
            )
        elif file_format == 'json':
            frame.write_json(path)
        elif file_format == 'jsonl':
            frame.write_ndjson(path)
        else:
            raise ValueError(
                'Columns/split JSON is a Pandas format in this sample.'
            )
    else:
        raise ValueError(f'Unsupported dataframe format: {file_format}')
    dataset.metadata['dataframe_library'] = library
    dataset.metadata['file_format'] = file_format


@dsl.component(
    base_image='python:3.11',
    packages_to_install=_PACKAGES,
    pip_index_urls=_PACKAGE_INDEXES,
    kfp_package_path=os.getenv('KFP_PACKAGE_PATH'),
)
def create_model_artifact(
    model: dsl.Output[dsl.Model],
    framework: str = 'pytorch',
    serialization: str = 'state_dict',
) -> None:
    """Save a tiny local model without downloading pretrained weights.

    Args:
        model: Directory for the framework's unchanged native files.
        framework: PyTorch or Hugging Face Transformers.
        serialization: PyTorch state_dict, checkpoint, module, or torchscript;
            Transformers uses save_pretrained with safetensors.
    """
    from pathlib import Path

    import torch

    directory = Path(model.path)
    directory.mkdir(parents=True, exist_ok=True)
    torch.manual_seed(42)
    torch.set_num_threads(1)
    if framework == 'pytorch':
        network = torch.nn.Sequential(
            torch.nn.Linear(4, 8), torch.nn.ReLU(), torch.nn.Linear(8, 2)
        )
        network.eval()
        if serialization == 'state_dict':
            torch.save(network.state_dict(), directory / 'model.pt')
        elif serialization == 'checkpoint':
            torch.save(
                {'model_state_dict': network.state_dict(), 'epoch': 0},
                directory / 'model.pt',
            )
        elif serialization == 'module':
            torch.save(network, directory / 'model.pt')
        elif serialization == 'torchscript':
            torch.jit.save(torch.jit.script(network), directory / 'model.pt')
        else:
            raise ValueError(
                f'Unsupported PyTorch serialization: {serialization}'
            )
    elif framework == 'transformers':
        from transformers import BertConfig, BertForSequenceClassification

        configuration = BertConfig(
            vocab_size=32,
            hidden_size=16,
            num_hidden_layers=1,
            num_attention_heads=2,
            intermediate_size=32,
            max_position_embeddings=32,
            num_labels=2,
        )
        network = BertForSequenceClassification(configuration)
        network.eval()
        network.save_pretrained(directory)
    else:
        raise ValueError(f'Unsupported model framework: {framework}')
    model.metadata['framework'] = framework


@dsl.pipeline(name='artifact-previews')
def artifact_preview_pipeline() -> None:
    """Demonstrates previews without a separate preview-generating task."""
    create_preview_artifacts()


@dsl.pipeline(
    name='artifact-previews-disabled',
    pipeline_config=dsl.PipelineConfig(artifact_previews=False),
)
def artifact_preview_disabled_pipeline() -> None:
    """Write the same native artifacts without generating previews."""
    create_preview_artifacts()


@dsl.pipeline(name='artifact-preview-lists')
def artifact_preview_list_pipeline() -> NamedTuple(
    'Outputs', [('datasets', List[dsl.Dataset]), ('models', List[dsl.Model])]
):
    """Collect artifact references while retaining each member's own
    preview."""
    with dsl.ParallelFor([42, 43], parallelism=1) as seed:
        task = create_preview_artifacts(seed=seed)
    outputs = NamedTuple(
        'Outputs',
        [('datasets', List[dsl.Dataset]), ('models', List[dsl.Model])],
    )
    return outputs(
        dsl.Collected(task.outputs['dataset']),
        dsl.Collected(task.outputs['model']),
    )


@dsl.pipeline(name='artifact-preview-formats')
def artifact_preview_formats_pipeline() -> NamedTuple(
    'Outputs',
    [('dataframes', List[dsl.Dataset]), ('pytorch_models', List[dsl.Model])],
):
    """Exercise file formats and per-member previews without extra preview
    tasks."""
    with dsl.ParallelFor(
        ['parquet', 'csv', 'json', 'jsonl', 'arrow', 'feather'], parallelism=1
    ) as file_format:
        dataframe = create_dataframe_artifact(
            library='polars', file_format=file_format
        )
    with dsl.ParallelFor(
        ['state_dict', 'checkpoint', 'module', 'torchscript'], parallelism=1
    ) as serialization:
        pytorch_model = create_model_artifact(serialization=serialization)
    create_model_artifact(framework='transformers')
    outputs = NamedTuple(
        'Outputs',
        [
            ('dataframes', List[dsl.Dataset]),
            ('pytorch_models', List[dsl.Model]),
        ],
    )
    return outputs(
        dsl.Collected(dataframe.output), dsl.Collected(pytorch_model.output)
    )


if __name__ == '__main__':
    compiler.Compiler().compile(
        artifact_preview_pipeline, package_path='artifact_previews.yaml'
    )
