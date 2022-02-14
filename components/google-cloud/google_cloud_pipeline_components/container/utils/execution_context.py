# Copyright 2022 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Execution context for launcher clients to support cancellation propagation."""
import logging
import os
import signal


class ExecutionContext:
  """Execution context for running inside Google Cloud Pipeline Components.

    The base class is aware of the GCPC environment and can cascade
    a pipeline cancel event to the operation through ``on_cancel`` handler.
    Args:
        on_cancel: optional, function to handle KFP cancel event.
  """

  def __init__(self, on_cancel=None):
    logging.info('Starting GCPC context')
    self._on_cancel = on_cancel
    self._original_sigterm_handler = None

  def __enter__(self):
    logging.info('Adding signal handler')
    self._original_sigterm_handler = signal.signal(signal.SIGTERM,
                                                   self._exit_gracefully)
    return self

  def __exit__(self, *_):
    logging.info('Exiting GCPC context')
    signal.signal(signal.SIGTERM, self._original_sigterm_handler)

  def _exit_gracefully(self, *_):
    logging.info('SIGTERM signal received.')
    if self._on_cancel:
      logging.info('Cancelling...')
      self._on_cancel()

    # Exiting here to prevent any downstream errors due to the cancelled job
    logging.info('Exiting GCPC context due to cancellation.')
    os._exit(0)
