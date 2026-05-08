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
"""File-based local task output cache for kfp.local."""
import hashlib
import json
import logging
import os
import tempfile
import threading
from typing import Any, Dict, List, Optional

from kfp import dsl
from kfp.dsl import executor

_ARTIFACT_MARKER = '__kfp_artifact__'
_ARTIFACT_LIST_MARKER = '__kfp_artifact_list__'


class LocalCache:
    """Thread-safe, file-backed cache for local task outputs.

    Cache entries are keyed by a SHA256 of (component_name, image,
    full_command, arguments). Each entry is a small JSON file on disk.
    Artifact outputs are referenced by URI — on retrieval, if the
    referenced file no longer exists, the entry is treated as a cache
    miss.
    """

    def __init__(self, cache_root: str) -> None:
        self.cache_root = cache_root
        os.makedirs(self.cache_root, exist_ok=True)
        self._lock = threading.Lock()

    # ----- key computation -----

    @staticmethod
    def _serialize_value_for_key(value: Any) -> Any:
        """Serializes a Python value to a JSON-friendly form for key
        computation."""
        if isinstance(value, dsl.Artifact):
            return {
                _ARTIFACT_MARKER: True,
                'schema_title': type(value).schema_title,
                'name': value.name,
                'uri': value.uri,
                'metadata': value.metadata,
            }
        if isinstance(value, list):
            return [LocalCache._serialize_value_for_key(v) for v in value]
        if isinstance(value, dict):
            return {
                k: LocalCache._serialize_value_for_key(v)
                for k, v in value.items()
            }
        return value

    def compute_key(
        self,
        component_name: str,
        image: str,
        full_command: List[str],
        arguments: Dict[str, Any],
        env_vars: Optional[Dict[str, str]] = None,
        custom_cache_key: Optional[str] = None,
    ) -> str:
        """Compute the cache key for a task invocation.

        Env vars participate in the key because they materially affect the
        component's execution (e.g. the same image + command + inputs but
        different `MYVAR` can produce different outputs); caching them out
        would serve stale results.

        If `custom_cache_key` is provided, it is still combined with the
        component name so that keys from different components do not
        collide.
        """
        h = hashlib.sha256()
        h.update(component_name.encode('utf-8'))
        h.update(b'\0')
        if custom_cache_key:
            h.update(custom_cache_key.encode('utf-8'))
            return h.hexdigest()
        h.update(image.encode('utf-8'))
        h.update(b'\0')
        h.update(json.dumps(list(full_command), sort_keys=True).encode('utf-8'))
        h.update(b'\0')
        h.update(
            json.dumps(
                self._serialize_value_for_key(arguments),
                sort_keys=True,
                default=str,
            ).encode('utf-8'))
        h.update(b'\0')
        h.update(
            json.dumps(dict(env_vars or {}), sort_keys=True).encode('utf-8'))
        return h.hexdigest()

    # ----- read/write -----

    def _entry_path(self, key: str) -> str:
        return os.path.join(self.cache_root, f'{key}.json')

    def get(self, key: str) -> Optional[Dict[str, Any]]:
        """Retrieves cached outputs for `key`, or None on cache miss.

        Returns None if the cache entry references an artifact file that
        no longer exists on disk (to avoid returning stale/broken URIs).
        """
        entry_path = self._entry_path(key)
        with self._lock:
            if not os.path.exists(entry_path):
                return None
            try:
                with open(entry_path) as f:
                    serialized = json.load(f)
            except (OSError, json.JSONDecodeError) as e:
                logging.warning(f'Failed to read cache entry {entry_path}: {e}')
                return None

        try:
            return self._deserialize_outputs(serialized)
        except _MissingArtifactError as e:
            logging.info(
                f'Cache entry for key {key} references missing artifact '
                f'{e.uri}. Treating as cache miss.')
            return None

    def put(self, key: str, outputs: Dict[str, Any]) -> None:
        """Persists `outputs` under `key`, atomically."""
        entry_path = self._entry_path(key)
        serialized = self._serialize_outputs(outputs)
        with self._lock:
            # Atomic write: write to a tempfile in the same directory, rename.
            fd, tmp_path = tempfile.mkstemp(
                prefix='.kfp_cache_', suffix='.tmp', dir=self.cache_root)
            try:
                with os.fdopen(fd, 'w') as f:
                    json.dump(serialized, f)
                os.replace(tmp_path, entry_path)
            except Exception:
                if os.path.exists(tmp_path):
                    os.remove(tmp_path)
                raise

    # ----- (de)serialization of outputs -----

    @staticmethod
    def _serialize_outputs(outputs: Dict[str, Any]) -> Dict[str, Any]:
        return {k: LocalCache._serialize_output(v) for k, v in outputs.items()}

    @staticmethod
    def _serialize_output(value: Any) -> Any:
        if isinstance(value, dsl.Artifact):
            return LocalCache._serialize_artifact(value)
        if isinstance(value, list) and value and isinstance(
                value[0], dsl.Artifact):
            return {
                _ARTIFACT_LIST_MARKER: True,
                'artifacts': [LocalCache._serialize_artifact(a) for a in value],
            }
        return value

    @staticmethod
    def _serialize_artifact(artifact: dsl.Artifact) -> Dict[str, Any]:
        return {
            _ARTIFACT_MARKER: True,
            'schema_title': type(artifact).schema_title,
            'schema_version': type(artifact).schema_version,
            'name': artifact.name,
            'uri': artifact.uri,
            'metadata': artifact.metadata,
        }

    @staticmethod
    def _deserialize_outputs(serialized: Dict[str, Any]) -> Dict[str, Any]:
        return {
            k: LocalCache._deserialize_output(v) for k, v in serialized.items()
        }

    @staticmethod
    def _deserialize_output(value: Any) -> Any:
        if isinstance(value, dict) and value.get(_ARTIFACT_MARKER):
            return LocalCache._deserialize_artifact(value)
        if isinstance(value, dict) and value.get(_ARTIFACT_LIST_MARKER):
            return [
                LocalCache._deserialize_artifact(a)
                for a in value.get('artifacts', [])
            ]
        return value

    @staticmethod
    def _deserialize_artifact(data: Dict[str, Any]) -> dsl.Artifact:
        uri = data.get('uri', '')
        # Treat missing local-file artifacts as a cache miss. Remote
        # artifacts (gs://, s3://, etc.) we have to trust.
        if uri and not _is_remote_uri(uri) and not os.path.exists(uri):
            raise _MissingArtifactError(uri)
        runtime_artifact = {
            'name': data.get('name', ''),
            'uri': uri,
            'metadata': data.get('metadata', {}) or {},
            'type': {
                'schemaTitle': data.get('schema_title', 'system.Artifact'),
            },
        }
        return executor.create_artifact_instance(runtime_artifact)


class _MissingArtifactError(Exception):

    def __init__(self, uri: str) -> None:
        super().__init__(f'Artifact file missing: {uri}')
        self.uri = uri


_REMOTE_URI_PREFIXES = ('gs://', 's3://', 'minio://', 'oci://', 'http://',
                        'https://')


def _is_remote_uri(uri: str) -> bool:
    return uri.startswith(_REMOTE_URI_PREFIXES)


# ----- config helpers -----

_cache_singleton_lock = threading.Lock()
_cache_singleton: Optional[LocalCache] = None
_cache_singleton_root: Optional[str] = None


def get_local_cache() -> Optional[LocalCache]:
    """Returns the shared LocalCache instance if local caching is enabled, else
    None."""
    # Lazy import to avoid circular import at module load time.
    from kfp.local import config

    cfg = config.LocalExecutionConfig.instance
    if cfg is None:
        return None
    if not getattr(cfg, 'enable_caching', False):
        return None

    cache_root = getattr(cfg, 'cache_root', None) or os.path.join(
        cfg.pipeline_root, '.kfp_cache')

    global _cache_singleton, _cache_singleton_root
    with _cache_singleton_lock:
        if (_cache_singleton is None or _cache_singleton_root != cache_root):
            _cache_singleton = LocalCache(cache_root=cache_root)
            _cache_singleton_root = cache_root
    return _cache_singleton


def reset_local_cache_singleton() -> None:
    """Test hook: clears the module-level cache singleton."""
    global _cache_singleton, _cache_singleton_root
    with _cache_singleton_lock:
        _cache_singleton = None
        _cache_singleton_root = None
