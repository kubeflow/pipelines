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
"""Internal automatic previews for directory-style output artifacts."""

from __future__ import annotations

import codecs
import csv
import datetime
import decimal
import functools
import importlib
import io
import itertools
import json
import logging
import math
import tempfile
import threading
from collections.abc import Mapping
from dataclasses import dataclass
from pathlib import Path
from types import ModuleType
from typing import Any, Callable

from kfp.dsl.types.artifact_types import Artifact

MAX_BYTES = 64 * 1024
# Stored beside, not inside, the artifact directory so directory readers such
# as Polars never encounter a non-data file.
PREVIEW_SIBLING_SUFFIX = '.kfp-preview.json'
MAX_ROWS = 10
MAX_COLUMNS = 20
MAX_STRING_LENGTH = 256
MAX_SAFE_INTEGER = 2**53 - 1
MAX_TEXT_SOURCE_BYTES = 1024 * 1024

_LOGGER = logging.getLogger(__name__)
_HOOK_LOCK = threading.RLock()
_SESSIONS: list[ArtifactPreviewSession] = []
_SAVE_METHODS: list[
    tuple[type | ModuleType, str, Callable[..., Any], Callable[..., Any]]
] = []


class PreviewGenerationError(ValueError):
    """An optional preview cannot be generated without changing the
    artifact."""


def _encode(preview: dict[str, Any]) -> bytes:
    """Encode only standards-compliant UTF-8 JSON."""
    return json.dumps(
        preview, ensure_ascii=False, allow_nan=False, separators=(',', ':')
    ).encode('utf-8')


class _Values:
    """Bound preview values without traversing arbitrary model objects."""

    def __init__(self) -> None:
        self.shortened = False
        self.remaining = 512

    def text(self, value: str) -> str:
        """Limit text by Unicode code points, not encoded bytes."""
        if len(value) > MAX_STRING_LENGTH:
            self.shortened = True
            return value[: MAX_STRING_LENGTH - 3] + '...'
        return value

    def value(self, value: Any, depth: int = 0) -> Any:
        """Represent bounded JSON values and label unsupported Python
        values."""
        self.remaining -= 1
        if self.remaining < 0:
            self.shortened = True
            return '<value omitted>'
        if value is None or isinstance(value, bool):
            return value
        if isinstance(value, str):
            return self.text(value)
        if isinstance(value, int):
            return (
                value
                if abs(value) <= MAX_SAFE_INTEGER
                else self.text(str(value))
            )
        if isinstance(value, float):
            if not math.isfinite(value):
                return (
                    'NaN'
                    if math.isnan(value)
                    else ('Infinity' if value > 0 else '-Infinity')
                )
            if value.is_integer() and abs(value) > MAX_SAFE_INTEGER:
                return str(value)
            return value
        if isinstance(
            value,
            (datetime.date, datetime.time, datetime.timedelta, decimal.Decimal),
        ):
            return self.text(str(value))
        if isinstance(value, (list, tuple, dict)):
            if depth >= 4 or self.remaining <= 0:
                self.shortened = True
                return '<nested value omitted>'
            count = min(len(value), 100, self.remaining)
            self.shortened |= count < len(value)
            if isinstance(value, dict):
                result = {}
                for key, item in itertools.islice(value.items(), count):
                    if not isinstance(key, str) or len(key) > MAX_STRING_LENGTH:
                        self.shortened = True
                        continue
                    result[key] = self.value(item, depth + 1)
                return result
            return [self.value(item, depth + 1) for item in value[:count]]
        self.shortened = True
        return self.text(
            f'<{type(value).__module__}.{type(value).__qualname__}>'
        )


def _parquet_preview(directory: Path, include_rows: bool) -> dict[str, Any]:
    """Read the footer and one projected batch, never a complete Parquet
    table."""
    candidates = list(
        itertools.islice(
            (
                path
                for path in directory.iterdir()
                if path.suffix.lower() == '.parquet'
            ),
            2,
        )
    )
    if (
        len(candidates) != 1
        or candidates[0].is_symlink()
        or not candidates[0].is_file()
    ):
        raise PreviewGenerationError(
            'Store exactly one Parquet file directly inside each Dataset directory.'
        )
    try:
        parquet_module = importlib.import_module('pyarrow.parquet')
        arrow = importlib.import_module('pyarrow')
    except (ImportError, OSError) as error:
        raise PreviewGenerationError(
            'Install PyArrow in the component image to enable Parquet previews.'
        ) from error

    try:
        with parquet_module.ParquetFile(candidates[0]) as parquet:
            return _parquet_sample(parquet, arrow, include_rows)
    except (arrow.ArrowException, OverflowError) as error:
        raise PreviewGenerationError(
            f'PyArrow could not read a bounded Parquet sample: {error}'
        ) from error


def _dataset_preview(directory: Path, include_rows: bool) -> dict[str, Any]:
    """Select one supported data file without guessing among multiple
    outputs."""
    extensions = {
        '.parquet',
        '.csv',
        '.json',
        '.jsonl',
        '.ndjson',
        '.arrow',
        '.ipc',
        '.feather',
    }
    candidates = list(
        itertools.islice(
            (
                path
                for path in directory.iterdir()
                if path.suffix.lower() in extensions
                and path.name != '_kfp_preview.json'
            ),
            2,
        )
    )
    if (
        len(candidates) != 1
        or candidates[0].is_symlink()
        or not candidates[0].is_file()
    ):
        raise PreviewGenerationError(
            'Store exactly one supported data file directly inside each Dataset directory.'
        )
    path = candidates[0]
    if path.suffix.lower() == '.parquet':
        return _parquet_preview(directory, include_rows)
    if path.suffix.lower() in {'.arrow', '.ipc', '.feather'}:
        return _arrow_preview(path, include_rows)
    try:
        return _text_dataset_preview(path, include_rows)
    except (csv.Error, RecursionError) as error:
        raise PreviewGenerationError(
            f'The data file could not be sampled within the preview limits: {error}'
        ) from error


def _dataset_payload(
    format_name: str,
    columns: list[tuple[str, str]],
    rows: list[list[Any]],
    values: _Values,
    include_rows: bool,
    total_rows: int | None,
    total_columns: int | None,
    more_rows: bool,
    more_columns: bool,
) -> dict[str, Any]:
    """Apply the shared dataset contract and encoded-byte budget."""
    preview: dict[str, Any] = {
        'version': 1,
        'kind': 'dataset',
        'format': format_name,
        'columns': [
            {'name': values.text(name), 'dtype': values.text(dtype)}
            for name, dtype in columns
        ],
        'rows': rows if include_rows else [],
        'total_rows': total_rows,
        'total_columns': total_columns,
        'sample': {
            'method': 'head',
            'rows_truncated': include_rows and more_rows,
            'columns_truncated': more_columns,
            'values_truncated': values.shortened,
            'redacted': False,
            'rows_omitted': not include_rows,
        },
    }
    if preview['rows'] and not columns:
        raise PreviewGenerationError(
            'A row sample requires at least one column.'
        )
    while preview['rows'] and len(_encode(preview)) > MAX_BYTES:
        preview['rows'].pop()
        preview['sample']['rows_truncated'] = True
    return preview


def _text_prefix(path: Path) -> tuple[str, bool]:
    """Bound text input as well as output; never read an entire large file."""
    with path.open('rb') as source:
        content = source.read(MAX_TEXT_SOURCE_BYTES + 1)
    complete = len(content) <= MAX_TEXT_SOURCE_BYTES
    decoder = codecs.getincrementaldecoder('utf-8-sig')()
    return decoder.decode(
        content[:MAX_TEXT_SOURCE_BYTES], final=complete
    ), complete


def _json_object(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    """Reject duplicate fields rather than silently displaying a different
    row."""
    result = {}
    for name, value in pairs:
        if name in result:
            raise PreviewGenerationError(
                'Duplicate JSON field names are not supported.'
            )
        result[name] = value
    return result


def _json_records(
    text: str, complete: bool, lines: bool
) -> tuple[list[dict[str, Any]], bool]:
    """Decode at most ten records from JSON arrays or JSON Lines prefixes."""
    decoder = json.JSONDecoder(object_pairs_hook=_json_object)
    records: list[dict[str, Any]] = []
    offset = 0
    after_comma = False
    text = text.lstrip()
    if not lines:
        if not text.startswith('['):
            raise PreviewGenerationError('Expected a JSON records array.')
        offset = 1
    while True:
        while offset < len(text) and text[offset].isspace():
            offset += 1
        if not lines and offset < len(text) and text[offset] == ']':
            if after_comma or text[offset + 1 :].strip():
                raise PreviewGenerationError('Invalid JSON records array.')
            return records, True
        if offset == len(text):
            if lines and complete:
                return records, True
            raise PreviewGenerationError(
                'A complete JSON sample exceeds the 1 MiB read limit.'
            )
        if len(records) == MAX_ROWS:
            return records, False
        record, end = decoder.raw_decode(text, offset)
        if not isinstance(record, dict):
            raise PreviewGenerationError(
                'JSON rows must be objects with column names.'
            )
        records.append(record)
        after_comma = False
        offset = end
        if lines:
            if offset < len(text) and not text[offset].isspace():
                raise PreviewGenerationError(
                    'JSON Lines records must be separated by whitespace.'
                )
        else:
            while offset < len(text) and text[offset].isspace():
                offset += 1
            if offset < len(text) and text[offset] == ',':
                offset += 1
                after_comma = True
            elif offset >= len(text) or text[offset] != ']':
                raise PreviewGenerationError(
                    'JSON records must be separated by commas.'
                )


def _json_columns(document: Any) -> tuple[list[str], list[list[Any]], int]:
    """Support bounded Pandas column-oriented and split-oriented JSON files."""
    if not isinstance(document, dict):
        raise PreviewGenerationError(
            'Expected JSON records, columns, or split orientation.'
        )
    if isinstance(document.get('columns'), list) and isinstance(
        document.get('data'), list
    ):
        names, data = document['columns'], document['data']
        if not all(isinstance(name, str) for name in names) or any(
            not isinstance(row, list) or len(row) != len(names) for row in data
        ):
            raise PreviewGenerationError(
                'JSON split columns and rows do not match.'
            )
        return names, [row[:MAX_COLUMNS] for row in data[:MAX_ROWS]], len(data)
    if not all(isinstance(column, dict) for column in document.values()):
        raise PreviewGenerationError(
            'Use JSON records, columns, split, or JSON Lines.'
        )
    names = list(document)
    indexes = list(
        dict.fromkeys(index for column in document.values() for index in column)
    )
    rows = [
        [document[name].get(index) for name in names[:MAX_COLUMNS]]
        for index in indexes[:MAX_ROWS]
    ]
    return names, rows, len(indexes)


def _json_dtype(values: list[Any]) -> str:
    """Describe only observed JSON types, without guessing missing file
    schema."""
    names = {
        (
            'null'
            if value is None
            else 'boolean'
            if isinstance(value, bool)
            else 'integer'
            if isinstance(value, int)
            else 'number'
            if isinstance(value, float)
            else 'string'
            if isinstance(value, str)
            else 'array'
            if isinstance(value, list)
            else 'object'
        )
        for value in values
    }
    return f'{" | ".join(sorted(names))} (sample)' if names else 'unknown'


def _text_dataset_preview(path: Path, include_rows: bool) -> dict[str, Any]:
    """Sample CSV or JSON without installing a dataframe library."""
    text, complete = _text_prefix(path)
    values = _Values()
    extension = path.suffix.lower()
    total_columns: int | None
    raw_rows: list[list[Any]]
    rows: list[list[Any]]
    if extension == '.csv':
        if not complete:
            text = text.rsplit('\n', 1)[0] + '\n' if '\n' in text else ''
        reader = csv.reader(io.StringIO(text, newline=''), strict=True)
        names = next(reader, None)
        if names is None:
            raise PreviewGenerationError(
                'CSV previews require a complete header row.'
            )
        raw_rows = (
            list(itertools.islice(reader, MAX_ROWS + 1)) if include_rows else []
        )
        if any(len(row) != len(names) for row in raw_rows):
            raise PreviewGenerationError(
                'CSV rows do not match the header width.'
            )
        total_rows = (
            len(raw_rows)
            if include_rows and complete and len(raw_rows) <= MAX_ROWS
            else None
        )
        total_columns = len(names)
        more_rows = not complete or len(raw_rows) > MAX_ROWS
        rows = [
            [values.value(cell) for cell in row[:MAX_COLUMNS]]
            for row in raw_rows[:MAX_ROWS]
        ]
        columns = [(name, 'string') for name in names[:MAX_COLUMNS]]
        format_name = 'csv'
    else:
        lines = extension in {'.jsonl', '.ndjson'}
        if lines or text.lstrip().startswith('['):
            records, exhausted = _json_records(text, complete, lines)
            names = list(
                dict.fromkeys(name for record in records for name in record)
            )
            raw_rows = [
                [record.get(name) for name in names[:MAX_COLUMNS]]
                for record in records
            ]
            total_rows = len(records) if exhausted else None
            total_columns = len(names) if exhausted else None
            more_rows = not exhausted
        else:
            if not complete:
                raise PreviewGenerationError(
                    'Column/split-oriented JSON exceeds 1 MiB; use records or JSON Lines for large files.'
                )
            names, raw_rows, total_rows = _json_columns(
                json.loads(text, object_pairs_hook=_json_object)
            )
            total_columns = len(names)
            more_rows = total_rows > len(raw_rows)
        columns = [
            (name, _json_dtype([row[index] for row in raw_rows]))
            for index, name in enumerate(names[:MAX_COLUMNS])
        ]
        rows = []
        if include_rows:
            for row in raw_rows:
                cells = []
                for cell in row:
                    if isinstance(cell, (dict, list)):
                        values.shortened = True
                        cells.append('<nested JSON value omitted>')
                    else:
                        cells.append(values.value(cell))
                rows.append(cells)
        format_name = 'jsonl' if lines else 'json'
    return _dataset_payload(
        format_name,
        columns,
        rows,
        values,
        include_rows,
        total_rows,
        total_columns,
        more_rows,
        len(names) > MAX_COLUMNS,
    )


def _arrow_preview(path: Path, include_rows: bool) -> dict[str, Any]:
    """Read projected IPC batches, including Feather v2, without reading all
    rows."""
    try:
        arrow = importlib.import_module('pyarrow')
        ipc = importlib.import_module('pyarrow.ipc')
    except (ImportError, OSError) as error:
        raise PreviewGenerationError(
            'Install PyArrow for Arrow/Feather previews.'
        ) from error
    values = _Values()
    try:
        with arrow.memory_map(str(path), 'r') as source:
            is_file = source.read(6) == b'ARROW1'
            source.seek(0)
            open_reader = ipc.open_file if is_file else ipc.open_stream
            schema = open_reader(source).schema
            fields = list(itertools.islice(schema, MAX_COLUMNS))
            rows: list[list[Any]] = []
            total_rows = None
            more_rows = False
            if include_rows:
                source.seek(0)
                reader = open_reader(
                    source,
                    options=ipc.IpcReadOptions(
                        included_fields=list(range(len(fields)))
                    ),
                )
                batches = (
                    (
                        reader.get_batch(index)
                        for index in range(reader.num_record_batches)
                    )
                    if is_file
                    else iter(reader)
                )
                exhausted = True
                seen = 0
                for index, batch in enumerate(batches):
                    seen += batch.num_rows
                    for row_index in range(
                        min(batch.num_rows, MAX_ROWS + 1 - len(rows))
                    ):
                        rows.append(
                            [
                                _parquet_cell(column[row_index], arrow, values)
                                for column in batch.columns
                            ]
                        )
                    if len(rows) > MAX_ROWS or index == 63:
                        exhausted = False
                        break
                more_rows = not exhausted
                if exhausted or (is_file and reader.num_record_batches == 1):
                    total_rows = seen
                rows = rows[:MAX_ROWS]
            return _dataset_payload(
                'feather' if path.suffix.lower() == '.feather' else 'arrow',
                [(field.name, str(field.type)) for field in fields],
                rows,
                values,
                include_rows,
                total_rows,
                len(schema),
                more_rows,
                len(schema) > len(fields),
            )
    except (arrow.ArrowException, OverflowError) as error:
        raise PreviewGenerationError(
            f'Could not read a bounded Arrow/Feather sample: {error}'
        ) from error


def _parquet_cell(scalar: Any, arrow: Any, values: _Values) -> Any:
    """Decode scalar cells only; do not materialize nested or binary
    payloads."""
    if not scalar.is_valid:
        return None
    data_type = scalar.type
    if arrow.types.is_dictionary(data_type):
        scalar = scalar.value
        return _parquet_cell(scalar, arrow, values)
    string_types: list[Callable[[Any], bool]] = [
        arrow.types.is_string,
        arrow.types.is_large_string,
    ]
    if hasattr(arrow.types, 'is_string_view'):
        string_types.append(arrow.types.is_string_view)
    if any(supported(data_type) for supported in string_types):
        buffer = scalar.as_buffer()
        byte_limit = MAX_STRING_LENGTH * 4 + 4
        decoder = codecs.getincrementaldecoder('utf-8')()
        text = decoder.decode(
            buffer.slice(0, min(buffer.size, byte_limit)).to_pybytes(),
            final=buffer.size <= byte_limit,
        )
        return values.text(text)
    supported_types = (
        arrow.types.is_boolean,
        arrow.types.is_integer,
        arrow.types.is_floating,
        arrow.types.is_decimal,
        arrow.types.is_date,
        arrow.types.is_time,
        arrow.types.is_timestamp,
        arrow.types.is_duration,
    )
    if any(supported(data_type) for supported in supported_types):
        return values.value(scalar.as_py())
    values.shortened = True
    return values.text(f'<{data_type} value omitted>')


def _parquet_sample(
    parquet: Any, arrow: Any, include_rows: bool
) -> dict[str, Any]:
    """Build a bounded sample with footer dimensions and explicit omissions."""
    values = _Values()
    fields = list(itertools.islice(parquet.schema_arrow, MAX_COLUMNS))
    names = parquet.schema_arrow.names
    if len(set(names)) != len(names):
        raise PreviewGenerationError(
            'Automatic Parquet sampling requires unique column names.'
        )
    rows = []
    if include_rows and fields:
        selected_names = [field.name for field in fields]
        batch = next(
            parquet.iter_batches(batch_size=MAX_ROWS, columns=selected_names),
            None,
        )
        if batch is not None:
            if batch.schema.names != selected_names:
                raise PreviewGenerationError(
                    'Parquet column names are ambiguous; a bounded projection was not possible.'
                )
            rows = [
                [
                    _parquet_cell(column[row_index], arrow, values)
                    for column in batch.columns
                ]
                for row_index in range(batch.num_rows)
            ]
    return _dataset_payload(
        'parquet',
        [(field.name, str(field.type)) for field in fields],
        rows,
        values,
        include_rows,
        parquet.metadata.num_rows,
        len(parquet.schema_arrow),
        parquet.metadata.num_rows > len(rows),
        len(parquet.schema_arrow) > len(fields),
    )


def _xgboost_preview(model: Any, framework: Any) -> dict[str, Any]:
    """Capture live configuration without reloading weights or dumping
    trees."""
    sklearn_model = not isinstance(model, framework.Booster)
    booster = model.get_booster() if sklearn_model else model
    config = json.loads(booster.save_config())['learner']
    gradient_booster = config['gradient_booster']
    tree_config = gradient_booster.get('gbtree', gradient_booster)
    if sklearn_model:
        parameters = {
            name: value
            for name, value in model.get_params(deep=False).items()
            if value is not None
        }
    else:
        parameters = {
            'training': config['learner_train_param'],
            'tree_parameters': tree_config.get('tree_train_param', {}),
        }
    structure = {
        'feature_count': booster.num_features(),
        'boosting_rounds': booster.num_boosted_rounds(),
    }
    tree_count = tree_config.get('gbtree_model_param', {}).get('num_trees')
    if tree_count is not None:
        structure['tree_count'] = int(tree_count)
    return _model_preview(
        'xgboost',
        framework.__version__,
        f'xgboost.{type(model).__name__}',
        parameters,
        structure,
        [
            'Current configuration at save time, not a complete training record. '
            'Unset parameters and fit-time arguments not retained by the estimator are omitted.',
        ],
    )


def _torch_preview(model: Any, framework: Any) -> dict[str, Any]:
    """Summarize live tensors without reading their values or loading a
    checkpoint."""
    structure: dict[str, Any]
    if isinstance(model, framework.nn.Module):
        tensors = dict(model.named_parameters())
        structure = {
            'parameter_count': sum(
                parameter.numel() for parameter in tensors.values()
            ),
            'trainable_parameter_count': sum(
                parameter.numel()
                for parameter in tensors.values()
                if parameter.requires_grad
            ),
            'buffer_count': sum(1 for _ in model.buffers()),
        }
        estimator_class = f'{type(model).__module__}.{type(model).__qualname__}'
        note = 'Module structure at save time; no weights or tensor values are included.'
    else:
        state = model
        if isinstance(state, Mapping):
            for key in ('model_state_dict', 'state_dict'):
                if isinstance(state.get(key), Mapping):
                    state = state[key]
                    break
        if not isinstance(state, Mapping):
            raise PreviewGenerationError(
                'PyTorch previews support modules and state dictionaries.'
            )
        tensors = {
            name: value
            for name, value in state.items()
            if isinstance(name, str) and isinstance(value, framework.Tensor)
        }
        if not tensors:
            raise PreviewGenerationError(
                'No model tensors were found in this PyTorch checkpoint.'
            )
        structure = {
            'tensor_count': len(tensors),
            'tensor_element_count': sum(
                tensor.numel() for tensor in tensors.values()
            ),
        }
        estimator_class = 'PyTorch state_dict'
        note = (
            'State dictionaries may contain parameters and buffers. Architecture, training '
            'hyperparameters, and trainability cannot be recovered from them; tensor values are omitted.'
        )
    structure['tensors'] = {
        name: {'shape': list(tensor.shape), 'dtype': str(tensor.dtype)}
        for name, tensor in itertools.islice(tensors.items(), 100)
    }
    notes = [note]
    if len(tensors) > 100:
        notes.append('Only the first 100 tensor descriptions are included.')
    return _model_preview(
        'pytorch', framework.__version__, estimator_class, {}, structure, notes
    )


def _transformers_preview(directory: Path) -> dict[str, Any] | None:
    """Read native Transformers configuration JSON, never model weights."""
    path = directory / 'config.json'
    if not path.exists():
        return None
    if path.is_symlink() or not path.is_file():
        raise PreviewGenerationError(
            'Transformers config.json must be a regular file.'
        )
    with path.open('rb') as source:
        content = source.read(MAX_BYTES + 1)
    if len(content) > MAX_BYTES:
        raise PreviewGenerationError(
            'Transformers config.json exceeds the 64 KiB read limit.'
        )
    config = json.loads(content, object_pairs_hook=_json_object)
    if (
        not isinstance(config, dict)
        or not isinstance(config.get('model_type'), str)
        or not isinstance(config.get('transformers_version'), str)
    ):
        return None
    architectures = config.get('architectures')
    estimator_class = (
        f'transformers.{architectures[0]}'
        if isinstance(architectures, list)
        and architectures
        and isinstance(architectures[0], str)
        else 'Architecture not recorded'
    )
    parameters = {
        name: value
        for name, value in config.items()
        if not name.startswith('_')
        and name not in {'architectures', 'model_type', 'transformers_version'}
    }
    return _model_preview(
        'transformers',
        config['transformers_version'],
        estimator_class,
        parameters,
        {'model_type': config['model_type']},
        [
            'Configuration from the native save_pretrained config.json file. '
            'Model weights were not opened or validated; this is not a training history.',
        ],
    )


def _model_preview(
    framework_name: str,
    framework_version: str,
    estimator_class: str,
    parameters: dict[str, Any],
    structure: dict[str, Any],
    notes: list[str],
) -> dict[str, Any]:
    """Apply one bounded, JSON-safe model contract across supported
    frameworks."""
    values = _Values()
    bounded_structure = {
        name: values.value(value)
        for name, value in itertools.islice(structure.items(), 100)
    }
    hyperparameters = {}
    for name, value in itertools.islice(
        parameters.items(), 100 - len(bounded_structure)
    ):
        if not isinstance(name, str) or len(name) > MAX_STRING_LENGTH:
            values.shortened = True
            continue
        hyperparameters[name] = values.value(value)
    values.shortened |= len(parameters) > len(hyperparameters)
    preview: dict[str, Any] = {
        'version': 1,
        'kind': 'model',
        'framework': {
            'name': framework_name,
            'version': values.text(framework_version),
        },
        'estimator_class': values.text(estimator_class),
        'hyperparameters': hyperparameters,
        'structure': bounded_structure,
        'notes': list(notes),
    }
    if values.shortened:
        preview['notes'].append(
            'Some parameter values were shortened or omitted.'
        )
    budget_note = (
        'Some summary entries were omitted to keep the preview within 64 KiB.'
    )
    while len(_encode(preview)) > MAX_BYTES and (
        hyperparameters or bounded_structure
    ):
        section, largest = max(
            (
                (section, key)
                for section in (hyperparameters, bounded_structure)
                for key in section
            ),
            key=lambda item: len(_encode({item[1]: item[0][item[1]]})),
        )
        value = section[largest]
        if isinstance(value, dict) and value:
            value.popitem()
        elif isinstance(value, list) and value:
            value.pop()
        else:
            del section[largest]
        if budget_note not in preview['notes']:
            preview['notes'].append(budget_note)
    return preview


def _fingerprint(path: Path) -> tuple[int, int, int, int]:
    """Detect a payload replaced or modified after its configuration was
    captured."""
    status = path.stat()
    return status.st_dev, status.st_ino, status.st_size, status.st_mtime_ns


@dataclass(frozen=True)
class _SavedModel:
    path: Path
    fingerprint: tuple[int, int, int, int]
    preview: dict[str, Any]


def _save_wrapper(
    original: Callable[..., Any],
    framework: Any,
    preview_fn: Callable[[Any, Any], dict[str, Any]],
    model_arg: str,
    filename_arg: str,
) -> Callable[..., Any]:
    """Observe a successful native save without changing its arguments or
    result."""

    @functools.wraps(original)
    def save(*args: Any, **kwargs: Any) -> Any:
        result = original(*args, **kwargs)
        model = args[0] if args else kwargs[model_arg]
        fname = args[1] if len(args) > 1 else kwargs[filename_arg]
        with _HOOK_LOCK:
            sessions = tuple(_SESSIONS)
        for session in sessions:
            session.capture_model(model, fname, framework, preview_fn)
        return result

    return save


def _restore_save_hooks(start: int = 0) -> None:
    """Restore installed hooks without overwriting a later framework
    integration."""
    for owner, method, original, wrapper in reversed(_SAVE_METHODS[start:]):
        if getattr(owner, method, None) is wrapper:
            setattr(owner, method, original)
        else:
            _LOGGER.warning(
                'A framework integration replaced %s.%s during execution; '
                'its replacement was left unchanged.',
                owner.__name__,
                method,
            )
    del _SAVE_METHODS[start:]


def _install_save_hooks() -> None:
    """Install available adapters independently; a broken optional one must not
    disable others."""
    for name in ('xgboost', 'torch'):
        try:
            framework = importlib.import_module(name)
        except ImportError as error:
            _LOGGER.debug(
                'Optional %s preview adapter unavailable: %s', name, error
            )
            continue
        except Exception as error:
            # Optional framework initializers can raise non-import errors,
            # including binary-compatibility failures from transitive libraries.
            _LOGGER.warning(
                'Could not initialize optional %s preview adapter: %s',
                name,
                error,
            )
            continue
        start = len(_SAVE_METHODS)
        try:
            if name == 'xgboost':
                methods = [
                    (
                        framework.Booster,
                        'save_model',
                        _xgboost_preview,
                        'self',
                        'fname',
                    )
                ]
                if getattr(framework, 'XGBModel', None) is not None:
                    methods.append(
                        (
                            framework.XGBModel,
                            'save_model',
                            _xgboost_preview,
                            'self',
                            'fname',
                        )
                    )
            else:
                methods = [
                    (framework, 'save', _torch_preview, 'obj', 'f'),
                    (framework.jit, 'save', _torch_preview, 'm', 'f'),
                    (
                        framework.jit.ScriptModule,
                        'save',
                        _torch_preview,
                        'self',
                        'f',
                    ),
                ]
            for owner, method, preview_fn, model_arg, filename_arg in methods:
                original = getattr(owner, method)
                wrapper = _save_wrapper(
                    original, framework, preview_fn, model_arg, filename_arg
                )
                setattr(owner, method, wrapper)
                _SAVE_METHODS.append((owner, method, original, wrapper))
        except (AttributeError, TypeError) as error:
            _restore_save_hooks(start)
            _LOGGER.warning(
                'Could not install automatic %s preview hooks: %s', name, error
            )


class ArtifactPreviewSession:
    """Scope automatic previews to one Python component execution."""

    def __init__(
        self, artifacts: dict[str, Artifact], include_rows: bool = True
    ) -> None:
        self.artifacts = artifacts
        self.include_rows = include_rows
        self.models: dict[str, _SavedModel] = {}
        self.ambiguous_models: set[str] = set()
        self._hooks_enabled = False

    def __enter__(self) -> ArtifactPreviewSession:
        """Install only the supported framework hooks needed by declared
        outputs."""
        if not any(
            artifact.schema_title == 'system.Model'
            and '_kfp_preview' not in artifact.metadata
            for artifact in self.artifacts.values()
        ):
            return self
        with _HOOK_LOCK:
            if not _SESSIONS:
                try:
                    _install_save_hooks()
                except BaseException:
                    # __exit__ is not called when __enter__ is interrupted.
                    _restore_save_hooks()
                    raise
            if not _SAVE_METHODS:
                return self
            _SESSIONS.append(self)
            self._hooks_enabled = True
        return self

    def __exit__(self, *_exception: Any) -> None:
        """Restore native methods even if the component or its save operation
        fails."""
        if not self._hooks_enabled:
            return
        with _HOOK_LOCK:
            _SESSIONS.remove(self)
            self._hooks_enabled = False
            if not _SESSIONS:
                _restore_save_hooks()
        self.models.clear()

    def capture_model(
        self,
        model: Any,
        filename: Any,
        framework: Any,
        preview_fn: Callable[[Any, Any], dict[str, Any]],
    ) -> None:
        """Capture only saves inside this execution's declared Model
        directories."""
        try:
            path = Path(filename).resolve()
            for name, artifact in self.artifacts.items():
                if (
                    artifact.schema_title != 'system.Model'
                    or not artifact.path
                    or '_kfp_preview' in artifact.metadata
                ):
                    continue
                directory = Path(artifact.path).resolve()
                if not directory.is_dir() or not path.is_relative_to(directory):
                    continue
                previous = self.models.get(name)
                if previous is not None and previous.path != path:
                    self.ambiguous_models.add(name)
                if name not in self.ambiguous_models:
                    self.models.pop(name, None)
                    self.models[name] = _SavedModel(
                        path, _fingerprint(path), preview_fn(model, framework)
                    )
        except Exception as error:
            # The native save already succeeded; an optional preview must not
            # turn it into a component failure.
            _LOGGER.warning(
                'Could not capture an automatic model preview; the saved model is unchanged: %s',
                error,
            )

    def generate(self) -> None:
        """Publish optional previews only after successful execution on the
        chief worker."""
        for name, artifact in self.artifacts.items():
            if artifact.schema_title not in ('system.Dataset', 'system.Model'):
                continue
            if '_kfp_preview' in artifact.metadata:
                continue
            try:
                if not artifact.path or not Path(artifact.path).is_dir():
                    raise PreviewGenerationError(
                        'Automatic previews require directory artifacts; single-file outputs '
                        'are left unchanged.'
                    )
                if artifact.metadata.get(
                    '_kfp_workspace'
                ) or artifact.uri.startswith('oci://'):
                    raise PreviewGenerationError(
                        'Automatic previews require uploaded S3, MinIO, or GCS artifacts, '
                        'not workspace or OCI outputs.'
                    )
                directory = Path(artifact.path)
                if artifact.schema_title == 'system.Dataset':
                    preview = _dataset_preview(directory, self.include_rows)
                else:
                    transformer_preview = _transformers_preview(directory)
                    if transformer_preview is not None:
                        self._publish(directory, artifact, transformer_preview)
                        continue
                    saved = self.models.get(name)
                    if name in self.ambiguous_models:
                        raise PreviewGenerationError(
                            'Store one native model per Model artifact directory.'
                        )
                    if saved is None:
                        raise PreviewGenerationError(
                            'No supported XGBoost or PyTorch save was captured, and no native '
                            'Transformers configuration was found. Model weights are never deserialized.'
                        )
                    if not saved.path.is_relative_to(directory.resolve()):
                        raise PreviewGenerationError(
                            'The output artifact path changed after the model was saved.'
                        )
                    if _fingerprint(saved.path) != saved.fingerprint:
                        raise PreviewGenerationError(
                            'The saved model changed after its configuration was captured. '
                            'Save the final model inside its artifact directory using a supported native API.'
                        )
                    preview = saved.preview
                self._publish(directory, artifact, preview)
            except Exception as error:
                # Library conversions can raise unexpected errors, such as an
                # unresolvable Arrow time zone. Skip only this optional preview
                # so executor output is still written.
                _LOGGER.warning(
                    'Automatic preview unavailable for output %r; its artifact is unchanged: %s',
                    name,
                    error,
                )

    @staticmethod
    def _publish(
        directory: Path, artifact: Artifact, preview: dict[str, Any]
    ) -> None:
        """Write the bounded sibling object atomically before advertising it in
        artifact metadata.

        The launcher uploads ``<artifact path>.kfp-preview.json`` to
        ``<artifact URI>.kfp-preview.json`` only when the descriptor is
        present.
        """
        content = _encode(preview)
        if len(content) > MAX_BYTES:
            raise PreviewGenerationError(
                'Preview exceeds 64 KiB; no preview was published.'
            )
        destination = directory.with_name(
            directory.name + PREVIEW_SIBLING_SUFFIX
        )
        if destination.exists() or destination.is_symlink():
            raise PreviewGenerationError(
                f'{destination.name} already exists without a descriptor; it was not overwritten.'
            )
        temporary = None
        try:
            with tempfile.NamedTemporaryFile(
                dir=destination.parent,
                prefix='.kfp-preview-',
                suffix='.tmp',
                delete=False,
            ) as stream:
                temporary = Path(stream.name)
                stream.write(content)
            temporary.replace(destination)
            artifact.metadata['_kfp_preview'] = {
                'version': 2,
                'storage': 'sibling',
            }
        finally:
            if temporary is not None and temporary.exists():
                temporary.unlink()
