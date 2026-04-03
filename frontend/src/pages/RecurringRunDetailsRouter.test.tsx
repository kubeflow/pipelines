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
import { V2beta1RecurringRun } from 'src/apisv2beta1/recurringrun';
import { V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import RecurringRunDetailsRouter from './RecurringRunDetailsRouter';
import { PageProps } from 'src/pages/Page';
import v2YamlTemplateString from 'src/data/test/lightweight_python_functions_v2_pipeline_rev.yaml?raw';
import { vi } from 'vitest';

vi.mock('src/pages/RecurringRunDetails', () => ({
  __esModule: true,
  default: () => <div data-testid='recurring-run-details-v1' />,
}));

vi.mock('src/pages/RecurringRunDetailsV2', () => ({
  __esModule: true,
  default: () => <div data-testid='recurring-run-details-v2' />,
}));

vi.mock('src/pages/functional_components/RecurringRunDetailsV2FC', () => ({
  RecurringRunDetailsV2FC: () => <div data-testid='recurring-run-details-v2-fc' />,
}));

const TEST_RECURRING_RUN_ID = 'test-recurring-run-id';
const TEST_PIPELINE_ID = 'test-pipeline-id';
const TEST_PIPELINE_VERSION_ID = 'test-pipeline-version-id';

const v2PipelineSpec = JsYaml.safeLoad(v2YamlTemplateString);

function generateProps(recurringRunId = TEST_RECURRING_RUN_ID): PageProps {
  return {
    history: { push: vi.fn(), replace: vi.fn() } as any,
    location: { pathname: `/recurringrun/details/${recurringRunId}` } as any,
    match: {
      isExact: true,
      params: { [RouteParams.recurringRunId]: recurringRunId },
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

describe('RecurringRunDetailsRouter', () => {
  let getRecurringRunSpy: ReturnType<typeof vi.spyOn>;
  let getPipelineVersionSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    vi.clearAllMocks();
    getRecurringRunSpy = vi.spyOn(Apis.recurringRunServiceApi, 'getRecurringRun');
    getPipelineVersionSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders RecurringRunDetailsV2 when template is a v2 pipeline spec', async () => {
    // Required by WorkflowUtils.isPipelineSpec, which gates v2 spec parsing on V2_ALPHA.
    vi.spyOn(features, 'isFeatureEnabled').mockImplementation(
      (featureKey) => featureKey === features.FeatureKey.V2_ALPHA,
    );
    const recurringRun: V2beta1RecurringRun = {
      recurring_run_id: TEST_RECURRING_RUN_ID,
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

    getRecurringRunSpy.mockResolvedValue(recurringRun);
    getPipelineVersionSpy.mockResolvedValue(pipelineVersion);

    render(
      <CommonTestWrapper>
        <RecurringRunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(screen.getByTestId('recurring-run-details-v2')).toBeInTheDocument();
    });
  });

  it('prefers inline recurring run pipeline_spec over pipeline version template', async () => {
    vi.spyOn(features, 'isFeatureEnabled').mockImplementation(
      (featureKey) => featureKey === features.FeatureKey.V2_ALPHA,
    );
    const recurringRun: V2beta1RecurringRun = {
      recurring_run_id: TEST_RECURRING_RUN_ID,
      pipeline_spec: v2PipelineSpec,
      pipeline_version_reference: {
        pipeline_id: TEST_PIPELINE_ID,
        pipeline_version_id: TEST_PIPELINE_VERSION_ID,
      },
    };
    const pipelineVersion: V2beta1PipelineVersion = {
      pipeline_id: TEST_PIPELINE_ID,
      pipeline_version_id: TEST_PIPELINE_VERSION_ID,
      pipeline_spec: {
        apiVersion: 'argoproj.io/v1alpha1',
        kind: 'Workflow',
        metadata: { name: 'from-version' },
        spec: { arguments: { parameters: [{ name: 'output' }] } },
      },
    };

    getRecurringRunSpy.mockResolvedValue(recurringRun);
    getPipelineVersionSpy.mockResolvedValue(pipelineVersion);

    render(
      <CommonTestWrapper>
        <RecurringRunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getPipelineVersionSpy).toHaveBeenCalledWith(
        TEST_PIPELINE_ID,
        TEST_PIPELINE_VERSION_ID,
      );
      expect(screen.getByTestId('recurring-run-details-v2')).toBeInTheDocument();
    });
  });

  it('renders RecurringRunDetailsV2FC when FUNCTIONAL_COMPONENT flag is enabled', async () => {
    // Enables both V2_ALPHA (required by isPipelineSpec) and FUNCTIONAL_COMPONENT (router branch).
    vi.spyOn(features, 'isFeatureEnabled').mockImplementation(
      (featureKey) =>
        featureKey === features.FeatureKey.V2_ALPHA ||
        featureKey === features.FeatureKey.FUNCTIONAL_COMPONENT,
    );
    const recurringRun: V2beta1RecurringRun = {
      recurring_run_id: TEST_RECURRING_RUN_ID,
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

    getRecurringRunSpy.mockResolvedValue(recurringRun);
    getPipelineVersionSpy.mockResolvedValue(pipelineVersion);

    render(
      <CommonTestWrapper>
        <RecurringRunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(screen.getByTestId('recurring-run-details-v2-fc')).toBeInTheDocument();
    });
  });

  it('renders RecurringRunDetails (V1) when template is not a v2 pipeline spec', async () => {
    // V2_ALPHA must be enabled so isPipelineSpec evaluates the template structure.
    // Without it, isPipelineSpec returns false for any template regardless of content.
    vi.spyOn(features, 'isFeatureEnabled').mockImplementation(
      (featureKey) => featureKey === features.FeatureKey.V2_ALPHA,
    );
    const argoWorkflow = {
      apiVersion: 'argoproj.io/v1alpha1',
      kind: 'Workflow',
      metadata: { name: 'test' },
      spec: { arguments: { parameters: [{ name: 'output' }] } },
    };
    const recurringRun: V2beta1RecurringRun = {
      recurring_run_id: TEST_RECURRING_RUN_ID,
      pipeline_version_reference: {
        pipeline_id: TEST_PIPELINE_ID,
        pipeline_version_id: TEST_PIPELINE_VERSION_ID,
      },
    };
    const pipelineVersion: V2beta1PipelineVersion = {
      pipeline_id: TEST_PIPELINE_ID,
      pipeline_version_id: TEST_PIPELINE_VERSION_ID,
      pipeline_spec: argoWorkflow,
    };

    getRecurringRunSpy.mockResolvedValue(recurringRun);
    getPipelineVersionSpy.mockResolvedValue(pipelineVersion);

    render(
      <CommonTestWrapper>
        <RecurringRunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getRecurringRunSpy).toHaveBeenCalledWith(TEST_RECURRING_RUN_ID);
      expect(getPipelineVersionSpy).toHaveBeenCalledWith(
        TEST_PIPELINE_ID,
        TEST_PIPELINE_VERSION_ID,
      );
    });

    await waitFor(() => {
      expect(screen.getByTestId('recurring-run-details-v1')).toBeInTheDocument();
    });
  });

  it('returns null when fetch fails and no template is available', async () => {
    getRecurringRunSpy.mockRejectedValue(new Error('Not found'));

    const { container } = render(
      <CommonTestWrapper>
        <RecurringRunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getRecurringRunSpy).toHaveBeenCalledWith(TEST_RECURRING_RUN_ID);
    });

    await waitFor(() => {
      expect(container.querySelector('[data-testid]')).toBeNull();
      expect(screen.queryByText('Currently loading recurring run information')).toBeNull();
    });
  });

  it('shows loading message while recurring run data is being fetched', () => {
    getRecurringRunSpy.mockReturnValue(new Promise(() => {}));

    render(
      <CommonTestWrapper>
        <RecurringRunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    expect(screen.getByText('Currently loading recurring run information')).toBeInTheDocument();
  });
});
