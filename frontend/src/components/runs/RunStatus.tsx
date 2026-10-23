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

import {
  CircleCheck,
  CircleX,
  CircleHelp,
  Clock3,
  LoaderCircle,
  Ban,
  Pause,
  SkipForward,
} from 'lucide-react';
import { V2RuntimeState } from 'src/apisv2/run';
import { getRunStatus } from 'src/lib/StatusUtils';
import '../tables/RunsTable.css';

export function RunStatus({ state }: { state?: V2RuntimeState }) {
  const status = getRunStatus(state);
  const Icon =
    {
      SUCCEEDED: CircleCheck,
      FAILED: CircleX,
      RUNNING: LoaderCircle,
      CANCELING: LoaderCircle,
      PENDING: Clock3,
      CANCELED: Ban,
      PAUSED: Pause,
      SKIPPED: SkipForward,
      RUNTIME_STATE_UNSPECIFIED: CircleHelp,
    }[state || V2RuntimeState.RUNTIME_STATE_UNSPECIFIED] || CircleHelp;
  return (
    <span className='kfp-run-status' data-tone={status.tone}>
      <Icon className='kfp-run-status-icon' size={14} aria-hidden='true' />
      {status.label}
    </span>
  );
}
