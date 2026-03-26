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
import { V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import { V2beta1Run, V2beta1RuntimeState } from 'src/apisv2beta1/run';
import { V2beta1RecurringRun } from 'src/apisv2beta1/recurringrun';
import { QUERY_PARAMS, RoutePage } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import NewRunSwitcher from 'src/pages/NewRunSwitcher';
import { PageProps } from 'src/pages/Page';
import React from 'react';
import { vi } from 'vitest';
import v2XGYamlTemplateString from 'src/data/test/xgboost_sample_pipeline.yaml?raw';
import {
  ORIGINAL_TEST_PIPELINE,
  ORIGINAL_TEST_PIPELINE_ID,
  ORIGINAL_TEST_PIPELINE_VERSION,
  ORIGINAL_TEST_PIPELINE_VERSION_ID,
  ORIGINAL_TEST_PIPELINE_VERSION_NAME,
  NEW_EXPERIMENT,
  V1_PIPELINE_VERSION,
  generatePropsNoPipelineDef,
  generatePropsNewRun,
} from './__tests__/newRunTestFixtures';

describe('NewRunSwitcher', () => {
  const TEST_RUN_ID = 'test-run-id';
  const TEST_RECURRING_RUN_ID = 'test-recurring-run-id';

  let getPipelineV1Spy: ReturnType<typeof vi.spyOn>;
  let getPipelineVersionTemplateSpy: ReturnType<typeof vi.spyOn>;

  function generatePropsCloneRun(runId = TEST_RUN_ID): PageProps {
    return {
      history: { push: vi.fn(), replace: vi.fn() } as any,
      location: {
        pathname: RoutePage.NEW_RUN,
        search: `?${QUERY_PARAMS.cloneFromRun}=${runId}`,
      } as any,
      match: '' as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: 'Clone a run' },
      updateBanner: vi.fn(),
      updateDialog: vi.fn(),
      updateSnackbar: vi.fn(),
      updateToolbar: vi.fn(),
    };
  }

  function generatePropsCloneRecurringRun(recurringRunId = TEST_RECURRING_RUN_ID): PageProps {
    return {
      history: { push: vi.fn(), replace: vi.fn() } as any,
      location: {
        pathname: RoutePage.NEW_RUN,
        search: `?${QUERY_PARAMS.cloneFromRecurringRun}=${recurringRunId}`,
      } as any,
      match: '' as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: 'Clone a recurring run' },
      updateBanner: vi.fn(),
      updateDialog: vi.fn(),
      updateSnackbar: vi.fn(),
      updateToolbar: vi.fn(),
    };
  }

  beforeEach(() => {
    vi.clearAllMocks();
    getPipelineV1Spy = vi.spyOn(Apis.pipelineServiceApi, 'getPipeline');
    getPipelineV1Spy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
    vi.spyOn(Apis.pipelineServiceApi, 'getPipelineVersion').mockResolvedValue(V1_PIPELINE_VERSION);
    getPipelineVersionTemplateSpy = vi.spyOn(Apis.pipelineServiceApi, 'getPipelineVersionTemplate');
    getPipelineVersionTemplateSpy.mockResolvedValue({ template: v2XGYamlTemplateString });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('routing to V1 or V2 new run page', () => {
    it('directs to new run v2 if no pipeline is selected (enter from run list)', () => {
      render(
        <CommonTestWrapper>
          <NewRunSwitcher {...generatePropsNoPipelineDef(null)} />
        </CommonTestWrapper>,
      );

      const chooseVersionBtn = screen.getAllByText('Choose')[1];
      expect(chooseVersionBtn.closest('button')?.disabled).toEqual(true);

      screen.getByText('Pipeline Root');
      screen.getByText('A pipeline must be selected');
    });

    it(
      'shows experiment name in new run v2 if experiment is selected' +
        '(enter from experiment details)',
      async () => {
        const getExperimentSpy = vi.spyOn(Apis.experimentServiceApiV2, 'getExperiment');
        getExperimentSpy.mockResolvedValue(NEW_EXPERIMENT);

        render(
          <CommonTestWrapper>
            <NewRunSwitcher {...generatePropsNoPipelineDef(NEW_EXPERIMENT.experiment_id)} />
          </CommonTestWrapper>,
        );

        await waitFor(() => {
          expect(getExperimentSpy).toHaveBeenCalled();
        });

        expect(await screen.findByDisplayValue(NEW_EXPERIMENT.display_name)).toBeInTheDocument();
        expect(await screen.findByText('Pipeline Root')).toBeInTheDocument();
      },
    );

    it('directs to new run v2 if it is v2 template (create run from pipeline)', async () => {
      vi.spyOn(features, 'isFeatureEnabled').mockImplementation(
        (featureKey) => featureKey === features.FeatureKey.V2_ALPHA,
      );
      const getPipelineSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
      getPipelineSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
      const getPipelineVersionSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
      getPipelineVersionSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE_VERSION);

      render(
        <CommonTestWrapper>
          <NewRunSwitcher {...generatePropsNewRun()} />
        </CommonTestWrapper>,
      );

      await waitFor(() => {
        expect(getPipelineSpy).toHaveBeenCalled();
        expect(getPipelineVersionSpy).toHaveBeenCalled();
      });

      expect(await screen.findByText('Pipeline Root')).toBeInTheDocument();
    });

    it('directs to new run v1 if it is not v2 template (create run from pipeline)', async () => {
      const TEST_PIPELINE_VERSION_NOT_V2SPEC: V2beta1PipelineVersion = {
        description: '',
        display_name: ORIGINAL_TEST_PIPELINE_VERSION_NAME,
        pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
        pipeline_version_id: 'test-not-v2-spec-version-id',
        pipeline_spec: { spec: { arguments: { parameters: [{ name: 'output' }] } } },
      };

      vi.spyOn(features, 'isFeatureEnabled').mockImplementation(
        (featureKey) => featureKey === features.FeatureKey.V2_ALPHA,
      );
      const getPipelineV2Spy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
      getPipelineV2Spy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
      const getPipelineVersionSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
      getPipelineVersionSpy.mockResolvedValue(TEST_PIPELINE_VERSION_NOT_V2SPEC);

      render(
        <CommonTestWrapper>
          <NewRunSwitcher
            {...generatePropsNewRun(ORIGINAL_TEST_PIPELINE_ID, 'test-not-v2-spec-version-id')}
          />
        </CommonTestWrapper>,
      );

      await waitFor(() => {
        expect(getPipelineV2Spy).toHaveBeenCalled();
        expect(getPipelineVersionSpy).toHaveBeenCalled();
      });

      await waitFor(() => {
        expect(getPipelineV1Spy).toHaveBeenCalled();
      });
    });

    it(
      'directs to new run v1 if pipeline_spec is not existing in pipeline_version ' +
        'and it is not v2 template in getPipelineVersionTemplate() response',
      async () => {
        const TEST_PIPELINE_VERSION_WITHOUT_SPEC: V2beta1PipelineVersion = {
          description: '',
          display_name: ORIGINAL_TEST_PIPELINE_VERSION_NAME,
          pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
          pipeline_version_id: 'test-no-spec-version-id',
          pipeline_spec: undefined,
        };

        vi.spyOn(features, 'isFeatureEnabled').mockImplementation(
          (featureKey) => featureKey === features.FeatureKey.V2_ALPHA,
        );
        const getPipelineV2Spy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
        getPipelineV2Spy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
        const getPipelineVersionSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
        getPipelineVersionSpy.mockResolvedValue(TEST_PIPELINE_VERSION_WITHOUT_SPEC);
        getPipelineVersionTemplateSpy.mockResolvedValueOnce({ template: 'test template' });

        render(
          <CommonTestWrapper>
            <NewRunSwitcher
              {...generatePropsNewRun(ORIGINAL_TEST_PIPELINE_ID, 'test-no-spec-version-id')}
            />
          </CommonTestWrapper>,
        );

        await waitFor(() => {
          expect(getPipelineV2Spy).toHaveBeenCalled();
          expect(getPipelineVersionSpy).toHaveBeenCalled();
        });

        await waitFor(() => {
          expect(getPipelineV1Spy).toHaveBeenCalled();
        });
      },
    );

    it(
      'directs to new run v2 if pipeline_spec is not existing in pipeline_version ' +
        'and it is v2 template in getPipelineVersionTemplate() response',
      async () => {
        const TEST_PIPELINE_VERSION_WITHOUT_SPEC: V2beta1PipelineVersion = {
          description: '',
          display_name: ORIGINAL_TEST_PIPELINE_VERSION_NAME,
          pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
          pipeline_version_id: 'test-no-spec-version-id',
          pipeline_spec: undefined,
        };

        vi.spyOn(features, 'isFeatureEnabled').mockImplementation(
          (featureKey) => featureKey === features.FeatureKey.V2_ALPHA,
        );
        const getPipelineV2Spy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
        getPipelineV2Spy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
        const getPipelineVersionSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
        getPipelineVersionSpy.mockResolvedValue(TEST_PIPELINE_VERSION_WITHOUT_SPEC);
        getPipelineVersionTemplateSpy.mockResolvedValueOnce({
          template: v2XGYamlTemplateString,
        });

        render(
          <CommonTestWrapper>
            <NewRunSwitcher
              {...generatePropsNewRun(ORIGINAL_TEST_PIPELINE_ID, 'test-no-spec-version-id')}
            />
          </CommonTestWrapper>,
        );

        await waitFor(() => {
          expect(getPipelineV2Spy).toHaveBeenCalled();
          expect(getPipelineVersionSpy).toHaveBeenCalled();
        });

        expect(await screen.findByText('Pipeline Root')).toBeInTheDocument();
      },
    );
  });

  describe('loading and clone scenarios', () => {
    it('shows loading message while queries are in-flight', () => {
      vi.spyOn(features, 'isFeatureEnabled').mockImplementation(
        (featureKey) => featureKey === features.FeatureKey.V2_ALPHA,
      );
      vi.spyOn(Apis.pipelineServiceApiV2, 'getPipeline').mockReturnValue(new Promise(() => {}));
      vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion').mockReturnValue(
        new Promise(() => {}),
      );

      render(
        <CommonTestWrapper>
          <NewRunSwitcher {...generatePropsNewRun()} />
        </CommonTestWrapper>,
      );

      expect(screen.getByText('Currently loading pipeline information')).toBeInTheDocument();
    });

    it('resolves template from cloned run.pipeline_spec', async () => {
      vi.spyOn(features, 'isFeatureEnabled').mockImplementation(
        (featureKey) => featureKey === features.FeatureKey.V2_ALPHA,
      );
      const sdkRun: V2beta1Run = {
        run_id: TEST_RUN_ID,
        display_name: 'SDK run',
        pipeline_spec: JsYaml.safeLoad(v2XGYamlTemplateString),
        state: V2beta1RuntimeState.SUCCEEDED,
      };
      vi.spyOn(Apis.runServiceApiV2, 'getRun').mockResolvedValue(sdkRun);

      render(
        <CommonTestWrapper>
          <NewRunSwitcher {...generatePropsCloneRun()} />
        </CommonTestWrapper>,
      );

      await waitFor(() => {
        expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledWith(TEST_RUN_ID);
      });

      expect(await screen.findByText('Pipeline Root')).toBeInTheDocument();
    });

    it('resolves template from cloned recurring_run.pipeline_spec', async () => {
      vi.spyOn(features, 'isFeatureEnabled').mockImplementation(
        (featureKey) => featureKey === features.FeatureKey.V2_ALPHA,
      );
      const sdkRecurringRun: V2beta1RecurringRun = {
        recurring_run_id: TEST_RECURRING_RUN_ID,
        display_name: 'SDK recurring run',
        pipeline_spec: JsYaml.safeLoad(v2XGYamlTemplateString),
      };
      vi.spyOn(Apis.recurringRunServiceApi, 'getRecurringRun').mockResolvedValue(sdkRecurringRun);

      render(
        <CommonTestWrapper>
          <NewRunSwitcher {...generatePropsCloneRecurringRun()} />
        </CommonTestWrapper>,
      );

      await waitFor(() => {
        expect(Apis.recurringRunServiceApi.getRecurringRun).toHaveBeenCalledWith(
          TEST_RECURRING_RUN_ID,
        );
      });

      expect(await screen.findByText('Pipeline Root')).toBeInTheDocument();
    });

    it('prefers inline pipeline_spec over version-derived template when cloning a run', async () => {
      vi.spyOn(features, 'isFeatureEnabled').mockImplementation(
        (featureKey) => featureKey === features.FeatureKey.V2_ALPHA,
      );
      const getPipelineVersionSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
      getPipelineVersionSpy.mockResolvedValue({
        pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
        pipeline_version_id: ORIGINAL_TEST_PIPELINE_VERSION_ID,
        pipeline_spec: {
          apiVersion: 'argoproj.io/v1alpha1',
          kind: 'Workflow',
          metadata: { name: 'from-version' },
          spec: { arguments: { parameters: [{ name: 'output' }] } },
        },
      } as V2beta1PipelineVersion);
      const sdkRun: V2beta1Run = {
        run_id: TEST_RUN_ID,
        display_name: 'SDK run',
        pipeline_spec: JsYaml.safeLoad(v2XGYamlTemplateString),
        pipeline_version_reference: {
          pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
          pipeline_version_id: ORIGINAL_TEST_PIPELINE_VERSION_ID,
        },
        state: V2beta1RuntimeState.SUCCEEDED,
      };
      vi.spyOn(Apis.runServiceApiV2, 'getRun').mockResolvedValue(sdkRun);

      render(
        <CommonTestWrapper>
          <NewRunSwitcher {...generatePropsCloneRun()} />
        </CommonTestWrapper>,
      );

      await waitFor(() => {
        expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledWith(TEST_RUN_ID);
        expect(getPipelineVersionSpy).toHaveBeenCalledWith(
          ORIGINAL_TEST_PIPELINE_ID,
          ORIGINAL_TEST_PIPELINE_VERSION_ID,
        );
        expect(screen.getByText('Pipeline Root')).toBeInTheDocument();
      });
    });

    it('prefers inline recurring_run.pipeline_spec over version-derived template when cloning', async () => {
      vi.spyOn(features, 'isFeatureEnabled').mockImplementation(
        (featureKey) => featureKey === features.FeatureKey.V2_ALPHA,
      );
      const getPipelineVersionSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
      getPipelineVersionSpy.mockResolvedValue({
        pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
        pipeline_version_id: ORIGINAL_TEST_PIPELINE_VERSION_ID,
        pipeline_spec: {
          apiVersion: 'argoproj.io/v1alpha1',
          kind: 'Workflow',
          metadata: { name: 'from-version' },
          spec: { arguments: { parameters: [{ name: 'output' }] } },
        },
      } as V2beta1PipelineVersion);
      const sdkRecurringRun: V2beta1RecurringRun = {
        recurring_run_id: TEST_RECURRING_RUN_ID,
        display_name: 'SDK recurring run',
        pipeline_spec: JsYaml.safeLoad(v2XGYamlTemplateString),
        pipeline_version_reference: {
          pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
          pipeline_version_id: ORIGINAL_TEST_PIPELINE_VERSION_ID,
        },
      };
      vi.spyOn(Apis.recurringRunServiceApi, 'getRecurringRun').mockResolvedValue(sdkRecurringRun);

      render(
        <CommonTestWrapper>
          <NewRunSwitcher {...generatePropsCloneRecurringRun()} />
        </CommonTestWrapper>,
      );

      await waitFor(() => {
        expect(Apis.recurringRunServiceApi.getRecurringRun).toHaveBeenCalledWith(
          TEST_RECURRING_RUN_ID,
        );
        expect(getPipelineVersionSpy).toHaveBeenCalledWith(
          ORIGINAL_TEST_PIPELINE_ID,
          ORIGINAL_TEST_PIPELINE_VERSION_ID,
        );
        expect(screen.getByText('Pipeline Root')).toBeInTheDocument();
      });
    });

    it('throws when both run and recurring run are non-null', async () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

      const run: V2beta1Run = {
        run_id: TEST_RUN_ID,
        display_name: 'test run',
      };
      const recurringRun: V2beta1RecurringRun = {
        recurring_run_id: TEST_RECURRING_RUN_ID,
        display_name: 'test recurring run',
      };
      vi.spyOn(Apis.runServiceApiV2, 'getRun').mockResolvedValue(run);
      vi.spyOn(Apis.recurringRunServiceApi, 'getRecurringRun').mockResolvedValue(recurringRun);

      class ErrorBoundary extends React.Component<
        { children: React.ReactNode },
        { hasError: boolean }
      > {
        state = { hasError: false };
        static getDerivedStateFromError() {
          return { hasError: true };
        }
        render() {
          if (this.state.hasError) {
            return <div data-testid='error-boundary'>Error caught</div>;
          }
          return this.props.children;
        }
      }

      const props: PageProps = {
        history: { push: vi.fn(), replace: vi.fn() } as any,
        location: {
          pathname: RoutePage.NEW_RUN,
          search: `?${QUERY_PARAMS.cloneFromRun}=${TEST_RUN_ID}&${QUERY_PARAMS.cloneFromRecurringRun}=${TEST_RECURRING_RUN_ID}`,
        } as any,
        match: '' as any,
        toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: 'Start a new run' },
        updateBanner: vi.fn(),
        updateDialog: vi.fn(),
        updateSnackbar: vi.fn(),
        updateToolbar: vi.fn(),
      };

      render(
        <CommonTestWrapper>
          <ErrorBoundary>
            <NewRunSwitcher {...props} />
          </ErrorBoundary>
        </CommonTestWrapper>,
      );

      await waitFor(() => {
        expect(screen.getByTestId('error-boundary')).toBeInTheDocument();
      });

      consoleSpy.mockRestore();
    });
  });
});
