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

import { Link } from 'react-router';
import type { V2RecurringRun } from 'src/apisv2/recurringrun';
import { QUERY_PARAMS, RoutePageFactory } from 'src/components/Router';

/** Link the saved reference without fetching a version just to render schedule metadata. */
export function RecurringRunPipeline({ run }: { run: V2RecurringRun }) {
  const reference = run.pipeline_version_reference;
  const pipelineId = reference?.pipeline_id;
  const versionId = reference?.pipeline_version_id;
  const inlineHref =
    run.pipeline_spec && run.recurring_run_id
      ? `${RoutePageFactory.pipelineDetails('')}?${new URLSearchParams({ [QUERY_PARAMS.fromRecurringRunId]: run.recurring_run_id })}`
      : undefined;
  return (
    <section className='kfp-inspection-fields' aria-label='Pipeline definition'>
      <h3>Pipeline definition</h3>
      <dl>
        {pipelineId ? (
          <>
            <div className='kfp-inspection-field'>
              <dt>Pipeline</dt>
              <dd>
                <Link to={RoutePageFactory.pipelineDetails(pipelineId)}>{pipelineId}</Link>
              </dd>
            </div>
            <div className='kfp-inspection-field'>
              <dt>Version</dt>
              <dd>
                {versionId ? (
                  <Link to={RoutePageFactory.pipelineVersionDetails(pipelineId, versionId)}>
                    {versionId}
                  </Link>
                ) : (
                  'Latest version at each scheduled run'
                )}
              </dd>
            </div>
          </>
        ) : (
          <div className='kfp-inspection-field'>
            <dt>Definition</dt>
            <dd>
              {inlineHref ? (
                <Link to={inlineHref}>View inline pipeline definition</Link>
              ) : (
                'No pipeline definition available'
              )}
            </dd>
          </div>
        )}
      </dl>
    </section>
  );
}
