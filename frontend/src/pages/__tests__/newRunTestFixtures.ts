/*
 * Copyright 2026 The Kubeflow Authors
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

/**
 * Shared test fixtures for NewRunV2 and NewRunSwitcher test suites.
 *
 * Constants and prop-builder helpers that are identical in both suites are
 * centralised here to keep the two files in sync and reduce duplication.
 */

import fs from 'node:fs';
import * as JsYaml from 'js-yaml';
import { vi } from 'vitest';
import { V2beta1Experiment, V2beta1ExperimentStorageState } from 'src/apisv2beta1/experiment';
import { V2beta1Pipeline, V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import { QUERY_PARAMS, RoutePage } from 'src/components/Router';
import { PageProps } from 'src/pages/Page';

const v2XGYamlTemplateString = fs.readFileSync(
  'src/data/test/xgboost_sample_pipeline.yaml',
  'utf8',
);

export const ORIGINAL_TEST_PIPELINE_ID = 'test-pipeline-id';
export const ORIGINAL_TEST_PIPELINE_NAME = 'test pipeline';
export const ORIGINAL_TEST_PIPELINE_VERSION_ID = 'test-pipeline-version-id';
export const ORIGINAL_TEST_PIPELINE_VERSION_NAME = 'test pipeline version';

export const ORIGINAL_TEST_PIPELINE: V2beta1Pipeline = {
  created_at: new Date(2018, 8, 5, 4, 3, 2),
  description: '',
  display_name: ORIGINAL_TEST_PIPELINE_NAME,
  pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
};

export const ORIGINAL_TEST_PIPELINE_VERSION: V2beta1PipelineVersion = {
  description: '',
  display_name: ORIGINAL_TEST_PIPELINE_VERSION_NAME,
  pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
  pipeline_version_id: ORIGINAL_TEST_PIPELINE_VERSION_ID,
  pipeline_spec: JsYaml.safeLoad(v2XGYamlTemplateString),
};

export const V1_PIPELINE_VERSION = {
  id: ORIGINAL_TEST_PIPELINE_VERSION_ID,
  name: ORIGINAL_TEST_PIPELINE_VERSION_NAME,
  parameters: [],
  resource_references: [{ key: { id: ORIGINAL_TEST_PIPELINE_ID, type: 'PIPELINE' } }],
} as any;

export const NEW_EXPERIMENT: V2beta1Experiment = {
  created_at: new Date('2022-07-26T17:44:28Z'),
  experiment_id: 'new-experiment-id',
  display_name: 'new-experiment',
  storage_state: V2beta1ExperimentStorageState.AVAILABLE,
};

export function generatePropsNoPipelineDef(experimentId: string | null): PageProps {
  return {
    history: { push: vi.fn(), replace: vi.fn() } as any,
    location: {
      pathname: RoutePage.NEW_RUN,
      search: experimentId
        ? `?${QUERY_PARAMS.experimentId}=${experimentId}`
        : `?${QUERY_PARAMS.experimentId}=`,
    } as any,
    match: '' as any,
    toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: 'Start a new run' },
    updateBanner: vi.fn(),
    updateDialog: vi.fn(),
    updateSnackbar: vi.fn(),
    updateToolbar: vi.fn(),
  };
}

export function generatePropsNewRun(
  pipelineId = ORIGINAL_TEST_PIPELINE_ID,
  versionId = ORIGINAL_TEST_PIPELINE_VERSION_ID,
): PageProps {
  return {
    history: { push: vi.fn(), replace: vi.fn() } as any,
    location: {
      pathname: RoutePage.NEW_RUN,
      search: `?${QUERY_PARAMS.pipelineId}=${pipelineId}&${QUERY_PARAMS.pipelineVersionId}=${versionId}`,
    } as any,
    match: '' as any,
    toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: 'Start a new run' },
    updateBanner: vi.fn(),
    updateDialog: vi.fn(),
    updateSnackbar: vi.fn(),
    updateToolbar: vi.fn(),
  };
}
