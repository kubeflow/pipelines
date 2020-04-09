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

import { orderBy } from 'lodash';
import { ApiParameter, ApiPipelineVersion } from '../apis/pipeline';
import { Workflow } from '../../third_party/argo-ui/argo_template';
import { ApiJob } from '../apis/job';
import {
  ApiPipelineRuntime,
  ApiResourceReference,
  ApiResourceType,
  ApiRun,
  ApiRunDetail,
  ApiRelationship,
} from '../apis/run';
import { logger } from './Utils';
import WorkflowParser from './WorkflowParser';
import { ApiExperiment } from 'src/apis/experiment';

export interface MetricMetadata {
  count: number;
  maxValue: number;
  minValue: number;
  name: string;
}

export interface ExperimentInfo {
  displayName?: string;
  id: string;
}

function getParametersFromRun(run: ApiRunDetail): ApiParameter[] {
  return getParametersFromRuntime(run.pipeline_runtime);
}

function getParametersFromRuntime(runtime?: ApiPipelineRuntime): ApiParameter[] {
  if (!runtime) {
    return [];
  }

  try {
    const workflow = JSON.parse(runtime.workflow_manifest!) as Workflow;
    return WorkflowParser.getParameters(workflow);
  } catch (err) {
    logger.error('Failed to parse runtime workflow manifest', err);
    return [];
  }
}

function getPipelineId(run?: ApiRun | ApiJob): string | null {
  return (run && run.pipeline_spec && run.pipeline_spec.pipeline_id) || null;
}

function getPipelineName(run?: ApiRun | ApiJob): string | null {
  return (run && run.pipeline_spec && run.pipeline_spec.pipeline_name) || null;
}

function getPipelineVersionId(run?: ApiRun | ApiJob): string | null {
  return run &&
    run.resource_references &&
    run.resource_references.some(
      ref => ref.key && ref.key.type && ref.key.type === ApiResourceType.PIPELINEVERSION,
    )
    ? run.resource_references.find(
        ref => ref.key && ref.key.type && ref.key.type === ApiResourceType.PIPELINEVERSION,
      )!.key!.id!
    : null;
}

function getPipelineIdFromApiPipelineVersion(
  pipelineVersion?: ApiPipelineVersion,
): string | undefined {
  return pipelineVersion &&
    pipelineVersion.resource_references &&
    pipelineVersion.resource_references.some(
      ref => ref.key && ref.key.type && ref.key.id && ref.key.type === ApiResourceType.PIPELINE,
    )
    ? pipelineVersion.resource_references.find(
        ref => ref.key && ref.key.type && ref.key.id && ref.key.type === ApiResourceType.PIPELINE,
      )!.key!.id!
    : undefined;
}

function getWorkflowManifest(run?: ApiRun | ApiJob): string | null {
  return (run && run.pipeline_spec && run.pipeline_spec.workflow_manifest) || null;
}

function getFirstExperimentReference(run?: ApiRun | ApiJob): ApiResourceReference | null {
  return getAllExperimentReferences(run)[0] || null;
}

function getFirstExperimentReferenceId(run?: ApiRun | ApiJob): string | null {
  const reference = getFirstExperimentReference(run);
  return (reference && reference.key && reference.key.id) || null;
}

function getFirstExperimentReferenceName(run?: ApiRun | ApiJob): string | null {
  const reference = getFirstExperimentReference(run);
  return (reference && reference.name) || null;
}

function getAllExperimentReferences(run?: ApiRun | ApiJob): ApiResourceReference[] {
  return ((run && run.resource_references) || []).filter(
    ref => (ref.key && ref.key.type && ref.key.type === ApiResourceType.EXPERIMENT) || false,
  );
}

function getNamespaceReferenceName(run?: ApiExperiment): string | undefined {
  // There should be only one namespace reference.
  const namespaceRef =
    run &&
    run.resource_references &&
    run.resource_references.find(
      ref =>
        ref.relationship === ApiRelationship.OWNER &&
        ref.key &&
        ref.key.type === ApiResourceType.NAMESPACE,
    );
  return namespaceRef && namespaceRef.key && namespaceRef.key.id;
}

/**
 * Takes an array of Runs and returns a map where each key represents a single metric, and its value
 * contains the name again, how many of that metric were collected across all supplied Runs, and the
 * max and min values encountered for that metric.
 */
function runsToMetricMetadataMap(runs: ApiRun[]): Map<string, MetricMetadata> {
  return runs.reduce((metricMetadatas, run) => {
    if (!run || !run.metrics) {
      return metricMetadatas;
    }
    run.metrics.forEach(metric => {
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
  }, new Map<string, MetricMetadata>());
}

function extractMetricMetadata(runs: ApiRun[]): MetricMetadata[] {
  const metrics = Array.from(runsToMetricMetadataMap(runs).values());
  return orderBy(metrics, ['count', 'name'], ['desc', 'asc']);
}

function getRecurringRunId(run?: ApiRun): string {
  if (!run) {
    return '';
  }

  for (const ref of run.resource_references || []) {
    if (ref.key && ref.key.type === ApiResourceType.JOB) {
      return ref.key.id || '';
    }
  }
  return '';
}

// TODO: This file needs tests
export default {
  extractMetricMetadata,
  getAllExperimentReferences,
  getFirstExperimentReference,
  getFirstExperimentReferenceId,
  getFirstExperimentReferenceName,
  getNamespaceReferenceName,
  getParametersFromRun,
  getParametersFromRuntime,
  getPipelineId,
  getPipelineIdFromApiPipelineVersion,
  getPipelineName,
  getPipelineVersionId,
  getRecurringRunId,
  getWorkflowManifest,
  runsToMetricMetadataMap,
};
