# Copyright 2018-2022 The Kubeflow Authors
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
"""Helpers for recognizing pagination compatibility errors."""

import json

import kfp.server_api


def is_pagination_restart_required(error: Exception) -> bool:
    """Return whether an API error requires restarting a paginated listing.

    Matches the structured server error, never its human-readable message.
    Works with errors from both ``kfp.Client`` and ``PipelinesClient``.
    It does not retry requests or modify the exception.

    When this returns ``True``, discard the page token and all results collected
    for this traversal before requesting page one with the same list criteria.
    If earlier results have already produced side effects, reconcile those
    effects before restarting to avoid duplicate processing. Bound any retry;
    mixed server versions may continue to reject tokens during a rollout.

    Args:
        error: The exception raised by a list request.

    Returns:
        ``True`` only for a structured ``PAGINATION_RESTART_REQUIRED`` error.
        Malformed responses and unrelated failures return ``False``.
    """
    if not isinstance(error,
                      kfp.server_api.ApiException) or error.status != 400:
        return False
    if not isinstance(error.body, (str, bytes)):
        return False
    try:
        body = json.loads(error.body)
    except (ValueError, UnicodeError, RecursionError):
        return False
    if not isinstance(body, dict) or body.get('code') != 9:
        return False
    details = body.get('details')
    if not isinstance(details, list):
        return False
    return any(
        isinstance(detail, dict) and detail.get('@type') ==
        'type.googleapis.com/google.rpc.ErrorInfo' and detail.get('reason') ==
        'PAGINATION_RESTART_REQUIRED' and detail.get('domain') == 'kubeflow.org'
        for detail in details)
