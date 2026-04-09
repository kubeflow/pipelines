# Copyright 2023 The Kubeflow Authors
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
"""Utilitites for formatting, coloring, and controlling the output of logs."""
import builtins
import contextlib
import datetime
import logging
import shutil
import sys
import threading
from typing import Any, Dict, Generator, List

from kfp import dsl
from kfp.local import status


class Color:
    # color for task name
    CYAN = '\033[96m'
    # color for status success
    GREEN = '\033[92m'
    # color for status failure
    RED = '\033[91m'
    # color for pipeline name
    MAGENTA = '\033[95m'
    RESET = '\033[0m'


class MillisecondFormatter(logging.Formatter):

    def formatTime(
        self,
        record: logging.LogRecord,
        datefmt: str = None,
    ) -> str:
        created = datetime.datetime.fromtimestamp(record.created)
        s = created.strftime(datefmt)
        # truncate microseconds to milliseconds
        return s[:-3]


_ORIGINAL_PRINT = builtins.print
_ACTIVE_INDENT_CONTEXTS = 0
_INDENT_LOCK = threading.Lock()
_THREAD_LOCAL_STATE = threading.local()


def _thread_aware_indented_print(*args, **kwargs):
    num_spaces = getattr(_THREAD_LOCAL_STATE, 'num_spaces', 0)
    if num_spaces > 0:
        _ORIGINAL_PRINT(' ' * num_spaces, end='')
    return _ORIGINAL_PRINT(*args, **kwargs)


@contextlib.contextmanager
def local_logger_context() -> Generator[None, None, None]:
    """Context manager for creating and reseting the local execution logger."""

    logger = logging.getLogger()
    original_level = logger.level
    original_handlers = logger.handlers[:]
    formatter = MillisecondFormatter(
        fmt='%(asctime)s - %(levelname)s - %(message)s',
        datefmt='%H:%M:%S.%f',
    )
    # use sys.stdout so that both inner process and outer process logs
    # go to stdout
    # this is needed for logs to present sequentially in a colab notebook,
    # since stderr will print above stdout
    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(formatter)
    logger.handlers.clear()
    logger.addHandler(handler)
    logger.setLevel(logging.INFO)

    try:
        yield
    finally:
        logger.setLevel(original_level)
        logger.handlers.clear()
        for handler in original_handlers:
            logger.addHandler(handler)


@contextlib.contextmanager
def indented_print(num_spaces: int = 4) -> Generator[None, None, None]:
    """Context manager to indent all print statements in its scope by
    num_prints.

    Useful for visually separating a subprocess logs from the outer
    process logs.
    """
    global _ACTIVE_INDENT_CONTEXTS

    with _INDENT_LOCK:
        if _ACTIVE_INDENT_CONTEXTS == 0:
            builtins.print = _thread_aware_indented_print
        _ACTIVE_INDENT_CONTEXTS += 1

    current_num_spaces = getattr(_THREAD_LOCAL_STATE, 'num_spaces', 0)
    _THREAD_LOCAL_STATE.num_spaces = current_num_spaces + num_spaces

    try:
        yield
    finally:
        current_num_spaces = getattr(_THREAD_LOCAL_STATE, 'num_spaces', 0)
        _THREAD_LOCAL_STATE.num_spaces = max(current_num_spaces - num_spaces,
                                             0)

        with _INDENT_LOCK:
            _ACTIVE_INDENT_CONTEXTS = max(_ACTIVE_INDENT_CONTEXTS - 1, 0)
            if _ACTIVE_INDENT_CONTEXTS == 0:
                builtins.print = _ORIGINAL_PRINT


def color_text(text: str, color: Color) -> str:
    return f'{color}{text}{Color.RESET}'


def make_log_lines_for_artifact(artifact: dsl.Artifact,) -> List[str]:
    """Returns a list of log lines that represent a single artifact output."""
    artifact_class_name_and_paren = f'{artifact.__class__.__name__}( '
    # name
    artifact_lines = [f"{artifact_class_name_and_paren}name='{artifact.name}',"]
    newline_spaces = len(artifact_class_name_and_paren) * ' '
    # uri
    artifact_lines.append(f"{newline_spaces}uri='{artifact.uri}',")
    # metadata
    artifact_lines.append(f'{newline_spaces}metadata={artifact.metadata} )')
    return artifact_lines


def make_log_lines_for_outputs(outputs: Dict[str, Any]) -> List[str]:
    """Returns a list of log lines to repesent the outputs of a task."""
    INDENT = ' ' * 4
    SEPARATOR = ': '
    output_lines = []
    for key, value in outputs.items():
        key_chars = INDENT + key + SEPARATOR

        # present artifacts
        if isinstance(value, dsl.Artifact):
            artifact_lines = make_log_lines_for_artifact(value)

            first_artifact_line = artifact_lines[0]
            output_lines.append(f'{key_chars}{first_artifact_line}')

            remaining_artifact_lines = artifact_lines[1:]
            # indent to align with first char in artifact
            # to visually separate output keys
            remaining_artifact_lines = [
                len(key_chars) * ' ' + l for l in remaining_artifact_lines
            ]
            output_lines.extend(remaining_artifact_lines)

        # present params
        else:
            value = f"'{value}'" if isinstance(value, str) else value
            output_lines.append(f'{key_chars}{value}')

    return output_lines


def print_horizontal_line() -> None:
    columns, _ = shutil.get_terminal_size(fallback=(80, 24))
    print('-' * columns)


def format_task_name(task_name: str) -> str:
    return color_text(f'{task_name!r}', Color.CYAN)


def format_status(task_status: status.Status) -> str:
    if task_status == status.Status.SUCCESS:
        return color_text(task_status.name, Color.GREEN)
    elif task_status == status.Status.FAILURE:
        return color_text(task_status.name, Color.RED)
    else:
        raise ValueError(f'Got unknown status: {task_status}')


def format_pipeline_name(pipeline_name: str) -> str:
    return color_text(f'{pipeline_name!r}', Color.MAGENTA)
