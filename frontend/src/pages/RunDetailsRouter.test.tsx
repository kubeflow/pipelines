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

import { render, screen, waitFor } from '@testing-library/react';
import * as JsYaml from 'js-yaml';
import * as features from 'src/features';
import { CommonTestWrapper } from 'src/TestWrapper';
import { RouteParams } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import { V2beta1Run } from 'src/apisv2beta1/run';
import { V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import RunDetailsRouter from './RunDetailsRouter';
import v2YamlTemplateString from 'src/data/test/lightweight_python_functions_v2_pipeline_rev.yaml?raw';
import { vi } from 'vitest';

vi.mock('src/pages/RunDetailsV2', () => ({
  RunDetailsV2: (props: any) => (
    <div data-testid='run-details-v2' data-pipeline-job={props.pipeline_job} />
  ),
}));

vi.mock('src/pages/RunDetails', () => ({
  __esModule: true,
  default: (props: any) => (
    <div data-testid='enhanced-run-details' data-is-loading={String(props.isLoading)} />
  ),
}));

const TEST_RUN_ID = 'test-run-id';
const TEST_PIPELINE_ID = 'test-pipeline-id';
const TEST_PIPELINE_VERSION_ID = 'test-pipeline-version-id';

const v2PipelineSpec = JsYaml.safeLoad(v2YamlTemplateString);

function generateProps(runId = TEST_RUN_ID) {
  return {
    history: { push: vi.fn(), replace: vi.fn() } as any,
    location: { pathname: `/runs/details/${runId}` } as any,
    match: {
      isExact: true,
      params: { [RouteParams.runId]: runId },
      path: '',
      url: '',
    },
    toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
    updateBanner: vi.fn(),
    updateDialog: vi.fn(),
    updateSnackbar: vi.fn(),
    updateToolbar: vi.fn(),
  } as any;
}

describe('RunDetailsRouter', () => {
  let getRunSpy: ReturnType<typeof vi.spyOn>;
  let getPipelineVersionSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    vi.clearAllMocks();
    getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    getPipelineVersionSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
    vi.spyOn(features, 'isFeatureEnabled').mockImplementation(
      (featureKey) => featureKey === features.FeatureKey.V2_ALPHA,
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders EnhancedRunDetails with isLoading=true while run is fetching', () => {
    getRunSpy.mockReturnValue(new Promise(() => {}));

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    const element = screen.getByTestId('enhanced-run-details');
    expect(element).toBeInTheDocument();
    expect(element.dataset.isLoading).toBe('true');
  });

  it('renders RunDetailsV2 when template is a v2 pipeline spec', async () => {
    const v2Run: V2beta1Run = {
      run_id: TEST_RUN_ID,
      pipeline_spec: v2PipelineSpec,
    };
    getRunSpy.mockResolvedValue(v2Run);

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(screen.getByTestId('run-details-v2')).toBeInTheDocument();
    });
  });

  it('renders EnhancedRunDetails (V1) when template is not a v2 pipeline spec', async () => {
    const argoWorkflow = {
      apiVersion: 'argoproj.io/v1alpha1',
      kind: 'Workflow',
      metadata: { name: 'test' },
      spec: { arguments: { parameters: [{ name: 'output' }] } },
    };
    const v1Run: V2beta1Run = {
      run_id: TEST_RUN_ID,
      pipeline_spec: argoWorkflow,
    };
    getRunSpy.mockResolvedValue(v1Run);

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getRunSpy).toHaveBeenCalledWith(TEST_RUN_ID);
    });

    const element = screen.getByTestId('enhanced-run-details');
    expect(element).toBeInTheDocument();
  });

  it('renders EnhancedRunDetails with isLoading=true while pipeline version template is fetching', async () => {
    const runWithVersionRef: V2beta1Run = {
      run_id: TEST_RUN_ID,
      pipeline_version_reference: {
        pipeline_id: TEST_PIPELINE_ID,
        pipeline_version_id: TEST_PIPELINE_VERSION_ID,
      },
    };
    getRunSpy.mockResolvedValue(runWithVersionRef);
    getPipelineVersionSpy.mockReturnValue(new Promise(() => {}));

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getRunSpy).toHaveBeenCalledWith(TEST_RUN_ID);
    });

    const element = screen.getByTestId('enhanced-run-details');
    expect(element).toBeInTheDocument();
    expect(element.dataset.isLoading).toBe('true');
  });

  it('resolves template from run.pipeline_spec directly when present', async () => {
    const v2Run: V2beta1Run = {
      run_id: TEST_RUN_ID,
      pipeline_spec: v2PipelineSpec,
      pipeline_version_reference: {
        pipeline_id: TEST_PIPELINE_ID,
        pipeline_version_id: TEST_PIPELINE_VERSION_ID,
      },
    };
    getRunSpy.mockResolvedValue(v2Run);

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(screen.getByTestId('run-details-v2')).toBeInTheDocument();
    });
  });

  it('fetches template from pipeline version when run has no inline spec', async () => {
    const runWithVersionRef: V2beta1Run = {
      run_id: TEST_RUN_ID,
      pipeline_version_reference: {
        pipeline_id: TEST_PIPELINE_ID,
        pipeline_version_id: TEST_PIPELINE_VERSION_ID,
      },
    };
    const pipelineVersion: V2beta1PipelineVersion = {
      pipeline_id: TEST_PIPELINE_ID,
      pipeline_version_id: TEST_PIPELINE_VERSION_ID,
      pipeline_spec: v2PipelineSpec,
    };
    getRunSpy.mockResolvedValue(runWithVersionRef);
    getPipelineVersionSpy.mockResolvedValue(pipelineVersion);

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getPipelineVersionSpy).toHaveBeenCalledWith(
        TEST_PIPELINE_ID,
        TEST_PIPELINE_VERSION_ID,
      );
    });

    await waitFor(() => {
      expect(screen.getByTestId('run-details-v2')).toBeInTheDocument();
    });
  });
});
