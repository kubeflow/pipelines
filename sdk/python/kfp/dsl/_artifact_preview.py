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

from dataclasses import dataclass
import datetime
import decimal
import functools
import importlib
import itertools
import json
import logging
import math
from pathlib import Path
import tempfile
import threading
from typing import Any, Callable

from kfp.dsl.types.artifact_types import Artifact

MAX_BYTES = 64 * 1024
MAX_ROWS = 10
MAX_COLUMNS = 20
MAX_STRING_LENGTH = 256
MAX_SAFE_INTEGER = 2**53 - 1

_LOGGER = logging.getLogger(__name__)
_DEPLOYMENT_CONFIG_PATH = Path(
    '/etc/kfp/launcher-config/artifactPreviewsEnabled')
_HOOK_LOCK = threading.RLock()
_SESSIONS: list[ArtifactPreviewSession] = []
_SAVE_METHODS: list[tuple[type, Callable[..., Any], Callable[..., Any]]] = []


class PreviewGenerationError(ValueError):
    """An optional preview cannot be generated without changing the
    artifact."""


def _deployment_allows_previews() -> bool:
    """Read the mounted launcher policy, disabling previews on invalid
    settings."""
    try:
        value = _DEPLOYMENT_CONFIG_PATH.read_text(
            encoding='utf-8').strip().lower()
    except FileNotFoundError:
        return True
    except (OSError, UnicodeError) as error:
        _LOGGER.warning(
            'Automatic artifact previews disabled: cannot read %s. '
            'Check the deployment launcher configuration: %s',
            _DEPLOYMENT_CONFIG_PATH, error)
        return False
    if value == 'true':
        return True
    if value == 'false':
        _LOGGER.info(
            'Automatic artifact previews disabled by the deployment configuration.'
        )
        return False
    _LOGGER.warning(
        'Automatic artifact previews disabled: set artifactPreviewsEnabled '
        'in the deployment kfp-launcher ConfigMap to "true" or "false".')
    return False


def _encode(preview: dict[str, Any]) -> bytes:
    """Encode only standards-compliant UTF-8 JSON."""
    return json.dumps(
        preview, ensure_ascii=False, allow_nan=False,
        separators=(',', ':')).encode('utf-8')


class _Values:
    """Bound preview values without traversing arbitrary model objects."""

    def __init__(self) -> None:
        self.shortened = False
        self.remaining = 512

    def text(self, value: str) -> str:
        """Limit text by Unicode code points, not encoded bytes."""
        if len(value) > MAX_STRING_LENGTH:
            self.shortened = True
            return value[:MAX_STRING_LENGTH - 3] + '...'
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
            return value if abs(value) <= MAX_SAFE_INTEGER else self.text(
                str(value))
        if isinstance(value, float):
            if not math.isfinite(value):
                return 'NaN' if math.isnan(value) else (
                    'Infinity' if value > 0 else '-Infinity')
            if value.is_integer() and abs(value) > MAX_SAFE_INTEGER:
                return str(value)
            return value
        if isinstance(value, (datetime.date, datetime.time, datetime.timedelta,
                              decimal.Decimal)):
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
            f'<{type(value).__module__}.{type(value).__qualname__}>')


def _parquet_preview(directory: Path, include_rows: bool) -> dict[str, Any]:
    """Read the footer and one projected batch, never a complete Parquet
    table."""
    candidates = list(itertools.islice(directory.glob('*.parquet'), 2))
    if len(candidates) != 1 or candidates[0].is_symlink():
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


def _parquet_cell(scalar: Any, arrow: Any, values: _Values) -> Any:
    """Decode scalar cells only; do not materialize nested or binary
    payloads."""
    if not scalar.is_valid:
        return None
    data_type = scalar.type
    if arrow.types.is_dictionary(data_type):
        data_type = data_type.value_type
    supported_types = (
        arrow.types.is_boolean,
        arrow.types.is_integer,
        arrow.types.is_floating,
        arrow.types.is_decimal,
        arrow.types.is_date,
        arrow.types.is_time,
        arrow.types.is_timestamp,
        arrow.types.is_duration,
        arrow.types.is_string,
        arrow.types.is_large_string,
    )
    if any(supported(data_type) for supported in supported_types):
        return values.value(scalar.as_py())
    values.shortened = True
    return values.text(f'<{data_type} value omitted>')


def _parquet_sample(parquet: Any, arrow: Any,
                    include_rows: bool) -> dict[str, Any]:
    """Build a bounded sample with footer dimensions and explicit omissions."""
    values = _Values()
    fields = list(itertools.islice(parquet.schema_arrow, MAX_COLUMNS))
    names = parquet.schema_arrow.names
    if len(set(names)) != len(names):
        raise PreviewGenerationError(
            'Automatic Parquet sampling requires unique column names.')
    rows = []
    if include_rows and fields:
        selected_names = [field.name for field in fields]
        batch = next(
            parquet.iter_batches(batch_size=MAX_ROWS, columns=selected_names),
            None)
        if batch is not None:
            if batch.schema.names != selected_names:
                raise PreviewGenerationError(
                    'Parquet column names are ambiguous; a bounded projection was not possible.'
                )
            rows = [[
                _parquet_cell(column[row_index], arrow, values)
                for column in batch.columns
            ]
                    for row_index in range(batch.num_rows)]
    preview: dict[str, Any] = {
        'version': 1,
        'kind': 'dataset',
        'format': 'parquet',
        'columns': [{
            'name': values.text(field.name),
            'dtype': values.text(str(field.type)),
        } for field in fields],
        'rows': rows,
        'total_rows': parquet.metadata.num_rows,
        'total_columns': len(parquet.schema_arrow),
        'sample': {
            'method':
                'head',
            'rows_truncated':
                include_rows and parquet.metadata.num_rows > len(rows),
            'columns_truncated':
                len(parquet.schema_arrow) > len(fields),
            'values_truncated':
                values.shortened,
            'redacted':
                False,
            'rows_omitted':
                not include_rows,
        },
    }
    while rows and len(_encode(preview)) > MAX_BYTES:
        rows.pop()
        preview['sample']['rows_truncated'] = True
    return preview


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
    values = _Values()
    hyperparameters = {}
    for name, value in itertools.islice(parameters.items(),
                                        100 - len(structure)):
        if not isinstance(name, str) or len(name) > MAX_STRING_LENGTH:
            values.shortened = True
            continue
        hyperparameters[name] = values.value(value)
    values.shortened |= len(parameters) > len(hyperparameters)
    preview: dict[str, Any] = {
        'version':
            1,
        'kind':
            'model',
        'framework': {
            'name': 'xgboost',
            'version': values.text(framework.__version__)
        },
        'estimator_class':
            values.text(f'xgboost.{type(model).__name__}'),
        'hyperparameters':
            hyperparameters,
        'structure':
            structure,
        'notes': [
            'Current configuration at save time, not a complete training record. '
            'Unset parameters and fit-time arguments not retained by the estimator are omitted.',
        ],
    }
    if values.shortened:
        preview['notes'].append(
            'Some parameter values were shortened or omitted.')
    while hyperparameters and len(_encode(preview)) > MAX_BYTES:
        largest = max(
            hyperparameters,
            key=lambda key: len(_encode({key: hyperparameters[key]})))
        del hyperparameters[largest]
        if 'Some parameters were omitted to keep the preview within 64 KiB.' not in preview[
                'notes']:
            preview['notes'].append(
                'Some parameters were omitted to keep the preview within 64 KiB.'
            )
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


def _save_wrapper(original: Callable[..., Any],
                  framework: Any) -> Callable[..., Any]:
    """Observe a successful native save without changing its arguments or
    result."""

    @functools.wraps(original)
    def save(model: Any, fname: Any, *args: Any, **kwargs: Any) -> Any:
        result = original(model, fname, *args, **kwargs)
        with _HOOK_LOCK:
            sessions = tuple(_SESSIONS)
        for session in sessions:
            session.capture_model(model, fname, framework)
        return result

    return save


def _restore_save_hooks() -> None:
    """Restore installed hooks without overwriting a later framework
    integration."""
    for model_class, original, wrapper in reversed(_SAVE_METHODS):
        if getattr(model_class, 'save_model', None) is wrapper:
            setattr(model_class, 'save_model', original)
        else:
            _LOGGER.warning(
                'A framework integration replaced %s.save_model during execution; '
                'its replacement was left unchanged.', model_class.__name__)
    _SAVE_METHODS.clear()


class ArtifactPreviewSession:
    """Scope automatic previews to one Python component execution."""

    def __init__(self,
                 artifacts: dict[str, Artifact],
                 include_rows: bool = True) -> None:
        self.artifacts = artifacts
        self.include_rows = include_rows
        self.models: dict[str, _SavedModel] = {}
        self.ambiguous_models: set[str] = set()
        self._hooks_enabled = False

    def __enter__(self) -> ArtifactPreviewSession:
        """Install only the supported framework hooks needed by declared
        outputs."""
        if not _deployment_allows_previews():
            return self
        if not any(artifact.schema_title == 'system.Model' and
                   '_kfp_preview' not in artifact.metadata
                   for artifact in self.artifacts.values()):
            return self
        try:
            framework = importlib.import_module('xgboost')
        except (ImportError, OSError) as error:
            _LOGGER.warning(
                'Automatic model previews require XGBoost installed in the component image: %s',
                error)
            return self
        with _HOOK_LOCK:
            if not _SESSIONS:
                try:
                    model_classes = [framework.Booster]
                    estimator_class = getattr(framework, 'XGBModel', None)
                    if estimator_class is not None:
                        model_classes.append(estimator_class)
                    for model_class in model_classes:
                        original = model_class.save_model
                        wrapper = _save_wrapper(original, framework)
                        model_class.save_model = wrapper
                        _SAVE_METHODS.append((model_class, original, wrapper))
                except (AttributeError, TypeError) as error:
                    _restore_save_hooks()
                    _LOGGER.warning(
                        'Could not install automatic preview hooks for this XGBoost version: %s',
                        error)
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

    def capture_model(self, model: Any, filename: Any, framework: Any) -> None:
        """Capture only saves inside this execution's declared Model
        directories."""
        try:
            path = Path(filename).resolve()
            for name, artifact in self.artifacts.items():
                if (artifact.schema_title != 'system.Model' or
                        not artifact.path or
                        '_kfp_preview' in artifact.metadata):
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
                        path, _fingerprint(path),
                        _xgboost_preview(model, framework))
        except (OSError, ValueError, TypeError, AttributeError, KeyError,
                OverflowError, framework.core.XGBoostError) as error:
            _LOGGER.warning(
                'Could not capture an automatic model preview; the saved model is unchanged: %s',
                error)

    def generate(self) -> None:
        """Publish optional previews only after successful execution on the
        chief worker."""
        # Recheck because the mounted policy can change while a component runs.
        if not _deployment_allows_previews():
            return
        for name, artifact in self.artifacts.items():
            if artifact.schema_title not in ('system.Dataset', 'system.Model'):
                continue
            if '_kfp_preview' in artifact.metadata:
                continue
            try:
                if not artifact.path or not Path(artifact.path).is_dir():
                    raise PreviewGenerationError(
                        'Automatic previews require directory artifacts; single-file outputs '
                        'are left unchanged.')
                if artifact.metadata.get(
                        '_kfp_workspace') or artifact.uri.startswith('oci://'):
                    raise PreviewGenerationError(
                        'Automatic previews require uploaded S3, MinIO, or GCS artifacts, '
                        'not workspace or OCI outputs.')
                directory = Path(artifact.path)
                if artifact.schema_title == 'system.Dataset':
                    preview = _parquet_preview(directory, self.include_rows)
                else:
                    saved = self.models.get(name)
                    if name in self.ambiguous_models:
                        raise PreviewGenerationError(
                            'Store one native model per Model artifact directory.'
                        )
                    if saved is None:
                        raise PreviewGenerationError(
                            'No supported XGBoost save_model call was captured in this output '
                            'directory. Other model formats are not inspected or deserialized.'
                        )
                    if not saved.path.is_relative_to(directory.resolve()):
                        raise PreviewGenerationError(
                            'The output artifact path changed after the model was saved.'
                        )
                    if _fingerprint(saved.path) != saved.fingerprint:
                        raise PreviewGenerationError(
                            'The saved model changed after its configuration was captured. '
                            'Save the final model with save_model inside its artifact directory.'
                        )
                    preview = saved.preview
                self._publish(directory, artifact, preview)
            except (OSError, ValueError, TypeError) as error:
                _LOGGER.warning(
                    'Automatic preview unavailable for output %r; its artifact is unchanged: %s',
                    name, error)

    @staticmethod
    def _publish(directory: Path, artifact: Artifact,
                 preview: dict[str, Any]) -> None:
        """Write the bounded file atomically before advertising it in artifact
        metadata."""
        content = _encode(preview)
        if len(content) > MAX_BYTES:
            raise PreviewGenerationError(
                'Preview exceeds 64 KiB; no preview was published.')
        destination = directory / 'preview.json'
        if destination.exists() or destination.is_symlink():
            raise PreviewGenerationError(
                'preview.json already exists without a descriptor; it was not overwritten.'
            )
        temporary = None
        try:
            with tempfile.NamedTemporaryFile(
                    dir=directory,
                    prefix='.kfp-preview-',
                    suffix='.tmp',
                    delete=False) as stream:
                temporary = Path(stream.name)
                stream.write(content)
            temporary.replace(destination)
            artifact.metadata['_kfp_preview'] = {
                'version': 1,
                'path': 'preview.json'
            }
        finally:
            if temporary is not None and temporary.exists():
                temporary.unlink()
