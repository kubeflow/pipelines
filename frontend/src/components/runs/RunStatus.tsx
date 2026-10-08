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
import { getRunStatus } from 'src/lib/StatusUtils';
import '../tables/RunsTable.css';

export function RunStatus({ state }: { state?: V2beta1RuntimeState }) {
  const status = getRunStatus(state);
  return (
    <span className='kfp-run-status' data-tone={status.tone}>
      <span className='kfp-run-status-dot' aria-hidden='true' />
      {status.label}
    </span>
  );
}
