/*
 * Copyright 2018 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { ApiJob } from '../apis/job';
import { ApiRun, ApiResourceType, ApiResourceReference } from '../apis/run';
import { orderBy } from 'lodash';

export interface MetricMetadata {
  count: number;
  maxValue: number;
  minValue: number;
  name: string;
}

function getPipelineId(run?: ApiRun | ApiJob): string | null {
  return run && run.pipeline_spec && run.pipeline_spec.pipeline_id || null;
}

function getFirstExperimentReferenceId(run?: ApiRun | ApiJob): string | null {
  if (run) {
    const reference = getAllExperimentReferences(run)[0];
    return reference && reference.key && reference.key.id || null;
  }
  return null;
}

function getFirstExperimentReference(run?: ApiRun | ApiJob): ApiResourceReference | null {
  return run && getAllExperimentReferences(run)[0] || null;
}

function getAllExperimentReferences(run?: ApiRun | ApiJob): ApiResourceReference[] {
  return (run && run.resource_references || [])
    .filter((ref) => ref.key && ref.key.type && ref.key.type === ApiResourceType.EXPERIMENT || false);
}

function extractMetricMetadata(runs: ApiRun[]) {
  const metrics = Array.from(
    runs.reduce((metricMetadatas, run) => {
      if (!run || !run.metrics) {
        return metricMetadatas;
      }
      run.metrics.forEach((metric) => {
        if (!metric.name || metric.number_value === undefined || isNaN(metric.number_value)) {
          return;
        }

        let metricMetadata = metricMetadatas.get(metric.name);
        if (!metricMetadata) {
          metricMetadata = {
            count: 0,
            maxValue: Number.MIN_VALUE,
            minValue: Number.MAX_VALUE,
            name: metric.name,
          };
          metricMetadatas.set(metricMetadata.name, metricMetadata);
        }
        metricMetadata.count++;
        metricMetadata.minValue = Math.min(metricMetadata.minValue, metric.number_value);
        metricMetadata.maxValue = Math.max(metricMetadata.maxValue, metric.number_value);
      });
      return metricMetadatas;
    }, new Map<string, MetricMetadata>()).values()
  );
  return orderBy(metrics, ['count', 'name'], ['desc', 'asc']);
}

export default {
  extractMetricMetadata,
  getAllExperimentReferences,
  getFirstExperimentReference,
  getFirstExperimentReferenceId,
  getPipelineId,
};
