# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Bounded recovery for explicitly rejected schedule mutations."""

import json
import logging
import time
from typing import Callable, TypeVar

import kfp_server_api

_Result = TypeVar('_Result')


def _is_schedule_handoff(error: Exception) -> bool:
    if not isinstance(error,
                      kfp_server_api.ApiException) or error.status != 503:
        return False
    if not isinstance(error.body, (str, bytes)):
        return False
    try:
        body = json.loads(error.body)
    except (ValueError, UnicodeError, RecursionError):
        return False
    if not isinstance(body, dict) or body.get('code') != 14:
        return False
    details = body.get('details')
    return isinstance(details, list) and any(
        isinstance(detail, dict) and
        detail.get('@type') == 'type.googleapis.com/google.rpc.ErrorInfo' and
        detail.get('reason') == 'SCHEDULE_HANDOFF_IN_PROGRESS' and
        detail.get('domain') == 'kubeflow.org' for detail in details)


def retry_schedule_handoff(request: Callable[[], _Result]) -> _Result:
    """Retry only a server guarantee that no schedule mutation was applied.

    Five attempts wait 1, 2, 4, and 8 seconds between explicit handoff
    rejections. Generic outages and transport failures are never replayed:
    creation may already have succeeded. Preserve the final API exception.
    """
    for attempt in range(5):
        try:
            return request()
        except kfp_server_api.ApiException as error:
            if not _is_schedule_handoff(error):
                raise
            if attempt == 4:
                logging.warning(
                    'Schedule writer handoff is still in progress after 5 '
                    'attempts. The request was not applied; existing schedules '
                    'retain their enabled state. Wait for the rollout to '
                    'complete, retry the operation, and verify the schedule '
                    'state.')
                raise
            time.sleep(2**attempt)
    raise AssertionError('unreachable')
