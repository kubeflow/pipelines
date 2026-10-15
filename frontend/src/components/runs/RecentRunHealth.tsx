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

import { V2beta1Run } from 'src/apisv2beta1/run';
import { getRunStatus } from 'src/lib/StatusUtils';
import './ExperimentWorkflows.css';

export function RecentRunHealth({ runs }: { runs: V2beta1Run[] }) {
  if (!runs.length) return <span className='kfp-workflow-muted'>No recent runs</span>;
  const summary = `Last ${runs.length} ${runs.length === 1 ? 'run' : 'runs'}: ${runs.map((run) => run.state || 'Unknown').join(', ')}`;
  return (
    <div className='kfp-recent-runs'>
      <div className='kfp-recent-runs-bar' role='img' aria-label={summary} title={summary}>
        {runs.map((run, index) => (
          <span key={run.run_id || index} data-tone={getRunStatus(run.state).tone} />
        ))}
      </div>
      <span className='kfp-workflow-muted'>
        {runs.length} recent {runs.length === 1 ? 'run' : 'runs'}
      </span>
    </div>
  );
}
