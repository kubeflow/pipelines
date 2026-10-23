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

import type { V2RecurringRun } from 'src/apisv2/recurringrun';
import { InspectionFields } from 'src/components/inspection/InspectionFields';
import { RecurringRunPipeline } from './RecurringRunPipeline';
import { formatDateString, enabledDisplayStringV2 } from 'src/lib/Utils';
import { scheduleDescription } from 'src/lib/ScheduleDisplay';
import type { KeyValue } from 'src/lib/DetailsTableTypes';

export function RecurringRunOverview({ run }: { run: V2RecurringRun }) {
  const schedule = run.trigger?.cron_schedule || run.trigger?.periodic_schedule;
  const scheduleFields: Array<KeyValue<string>> = [
    ['Enabled', enabledDisplayStringV2(run.trigger, run.status!)],
    ['Frequency', scheduleDescription(run.trigger)],
  ];
  if (run.trigger?.cron_schedule?.cron) {
    scheduleFields.push(['Cron expression', run.trigger.cron_schedule.cron]);
  }
  scheduleFields.push(
    ['Starts', schedule?.start_time ? formatDateString(schedule.start_time) : 'Immediately'],
    ['Ends', schedule?.end_time ? formatDateString(schedule.end_time) : 'No end date'],
  );
  const parameters: Array<KeyValue<string>> = Object.entries(
    run.runtime_config?.parameters || {},
  ).map(([key, value]) => [
    key,
    value == null ? '' : typeof value === 'string' ? value : JSON.stringify(value),
  ]);
  return (
    <div className='kfp-detail-page kfp-recurring-detail'>
      <div className='kfp-detail-sections kfp-detail-sections-stacked'>
        <InspectionFields title='Schedule' fields={scheduleFields} />
        <RecurringRunPipeline run={run} />
        <InspectionFields
          title='Execution settings'
          fields={[
            ['Max. concurrent runs', run.max_concurrency ?? 'Not specified'],
            ['Catch up missed runs', run.no_catchup ? 'Disabled' : 'Enabled'],
          ]}
        />
        {parameters.length > 0 ? (
          <InspectionFields title='Run parameters' fields={parameters} />
        ) : (
          <section className='kfp-inspection-fields' aria-label='Run parameters'>
            <h3>Run parameters</h3>
            <p>No run parameters configured.</p>
          </section>
        )}
        <InspectionFields
          title='Recurring run details'
          fields={[
            ['Description', run.description || 'No description'],
            ['Created at', formatDateString(run.created_at)],
          ]}
        />
      </div>
    </div>
  );
}
