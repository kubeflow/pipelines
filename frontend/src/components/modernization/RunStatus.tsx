/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { V2beta1RuntimeState } from 'src/apisv2beta1/run';
import './RunsTable.css';

const states: Record<V2beta1RuntimeState, { label: string; tone: string }> = {
  SUCCEEDED: { label: 'Succeeded', tone: 'succeeded' },
  RUNNING: { label: 'Running', tone: 'running' },
  FAILED: { label: 'Failed', tone: 'failed' },
  PENDING: { label: 'Pending', tone: 'neutral' },
  CANCELING: { label: 'Canceling', tone: 'running' },
  CANCELED: { label: 'Canceled', tone: 'neutral' },
  PAUSED: { label: 'Paused', tone: 'warning' },
  SKIPPED: { label: 'Skipped', tone: 'neutral' },
  RUNTIME_STATE_UNSPECIFIED: { label: 'Unknown', tone: 'neutral' },
};

export function RunStatus({ state }: { state?: V2beta1RuntimeState }) {
  const status = (state && states[state]) || states.RUNTIME_STATE_UNSPECIFIED;
  return (
    <span className='kfp-run-status' data-tone={status.tone}>
      <span className='kfp-run-status-dot' aria-hidden='true' />
      {status.label}
    </span>
  );
}
