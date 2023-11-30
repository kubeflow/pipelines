/*
 * Copyright 2022 The Kubeflow Authors
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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import fs from 'fs';
import 'jest';
import * as JsYaml from 'js-yaml';
import * as features from 'src/features';
import React from 'react';
import { testBestPractices } from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import {
  V2beta1Experiment,
  V2beta1ExperimentStorageState,
  V2beta1ListExperimentsResponse,
} from 'src/apisv2beta1/experiment';
import { V2beta1Filter, V2beta1PredicateOperation } from 'src/apisv2beta1/filter';
import {
  V2beta1Pipeline,
  V2beta1PipelineVersion,
  V2beta1ListPipelinesResponse,
  V2beta1ListPipelineVersionsResponse,
} from 'src/apisv2beta1/pipeline';
import { V2beta1Run, V2beta1RuntimeState } from 'src/apisv2beta1/run';
import { V2beta1RecurringRun, RecurringRunMode } from 'src/apisv2beta1/recurringrun';
import { QUERY_PARAMS, RoutePage } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import { convertYamlToV2PipelineSpec } from 'src/lib/v2/WorkflowUtils';
import NewRunV2 from 'src/pages/NewRunV2';
import NewRunSwitcher from 'src/pages/NewRunSwitcher';
import { PageProps } from 'src/Page';

const V2_XG_PIPELINESPEC_PATH = 'src/data/test/xgboost_sample_pipeline.yaml';
const v2XGYamlTemplateString = fs.readFileSync(V2_XG_PIPELINESPEC_PATH, 'utf8');
const v2XGPipelineSpec = convertYamlToV2PipelineSpec(v2XGYamlTemplateString);

const V2_LW_PIPELINESPEC_PATH = 'src/data/test/lightweight_python_functions_v2_pipeline_rev.yaml';
const v2LWYamlTemplateString = fs.readFileSync(V2_LW_PIPELINESPEC_PATH, 'utf8');
const v2LWPipelineSpec = convertYamlToV2PipelineSpec(v2LWYamlTemplateString);

testBestPractices();

describe('NewRunV2', () => {
  const TEST_RUN_ID = 'test-run-id';
  const TEST_RECURRING_RUN_ID = 'test-recurring-run-id';
  const ORIGINAL_TEST_PIPELINE_ID = 'test-pipeline-id';
  const ORIGINAL_TEST_PIPELINE_NAME = 'test pipeline';
  const ORIGINAL_TEST_PIPELINE_VERSION_ID = 'test-pipeline-version-id';
  const ORIGINAL_TEST_PIPELINE_VERSION_NAME = 'test pipeline version';
  const OTHER_TEST_PIPELINE_VERSION_ID = 'other-test-pipeline-version-id';
  const OTHER_TEST_PIPELINE_VERSION_NAME = 'other-test-pipeline-version';
  const ORIGINAL_TEST_PIPELINE: V2beta1Pipeline = {
    created_at: new Date(2018, 8, 5, 4, 3, 2),
    description: '',
    display_name: ORIGINAL_TEST_PIPELINE_NAME,
    pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
  };
  const ORIGINAL_TEST_PIPELINE_VERSION: V2beta1PipelineVersion = {
    description: '',
    display_name: ORIGINAL_TEST_PIPELINE_VERSION_NAME,
    pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
    pipeline_version_id: ORIGINAL_TEST_PIPELINE_VERSION_ID,
    pipeline_spec: JsYaml.safeLoad(v2XGYamlTemplateString),
  };
  const OTHER_TEST_PIPELINE_VERSION: V2beta1PipelineVersion = {
    description: '',
    display_name: OTHER_TEST_PIPELINE_VERSION_NAME,
    pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
    pipeline_version_id: OTHER_TEST_PIPELINE_VERSION_ID,
    pipeline_spec: v2LWPipelineSpec,
  };

  const NEW_TEST_PIPELINE_ID = 'new-test-pipeline-id';
  const NEW_TEST_PIPELINE_NAME = 'new-test-pipeline';
  const NEW_TEST_PIPELINE_VERSION_ID = 'new-test-pipeline-version-id';
  const NEW_TEST_PIPELINE_VERSION_NAME = 'new-test-pipeline-version';
  const NEW_TEST_PIPELINE: V2beta1Pipeline = {
    created_at: new Date(2018, 8, 7, 6, 5, 4),
    description: '',
    display_name: NEW_TEST_PIPELINE_NAME,
    pipeline_id: NEW_TEST_PIPELINE_ID,
  };

  const NEW_TEST_PIPELINE_VERSION: V2beta1PipelineVersion = {
    description: '',
    display_name: NEW_TEST_PIPELINE_VERSION_NAME,
    pipeline_id: NEW_TEST_PIPELINE_ID,
    pipeline_version_id: NEW_TEST_PIPELINE_VERSION_ID,
    pipeline_spec: v2LWPipelineSpec,
  };

  // Reponse from BE while POST a run for creating New UI-Run
  const API_UI_CREATED_NEW_RUN_DETAILS: V2beta1Run = {
    created_at: new Date('2021-05-17T20:58:23.000Z'),
    description: 'V2 xgboost',
    finished_at: new Date('2021-05-18T21:01:23.000Z'),
    run_id: TEST_RUN_ID,
    display_name: 'Run of v2-xgboost-ilbo',
    pipeline_version_reference: {
      pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
      pipeline_version_id: ORIGINAL_TEST_PIPELINE_VERSION_ID,
    },
    runtime_config: { parameters: { intParam: 123 }, pipeline_root: 'gs://dummy_pipeline_root' },
    scheduled_at: new Date('2021-05-17T20:58:23.000Z'),
    state: V2beta1RuntimeState.SUCCEEDED,
  };

  // Reponse from BE while POST a run for cloning UI-Run
  const API_UI_CREATED_CLONING_RUN_DETAILS: V2beta1Run = {
    created_at: new Date('2022-08-12T20:58:23.000Z'),
    description: 'V2 xgboost',
    finished_at: new Date('2022-08-12T21:01:23.000Z'),
    run_id: 'test-clone-ui-run-id',
    display_name: 'Clone of Run of v2-xgboost-ilbo',
    pipeline_version_reference: {
      pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
      pipeline_version_id: ORIGINAL_TEST_PIPELINE_VERSION_ID,
    },
    runtime_config: { parameters: { intParam: 123 }, pipeline_root: 'gs://dummy_pipeline_root' },
    scheduled_at: new Date('2022-08-12T20:58:23.000Z'),
    state: V2beta1RuntimeState.SUCCEEDED,
  };

  // Reponse from BE while SDK POST a new run for Creating run
  const API_SDK_CREATED_NEW_RUN_DETAILS: V2beta1Run = {
    created_at: new Date('2021-05-17T20:58:23.000Z'),
    description: 'V2 xgboost',
    finished_at: new Date('2021-05-17T21:01:23.000Z'),
    run_id: 'test-clone-sdk-run-id',
    display_name: 'Run of v2-xgboost-ilbo',
    pipeline_spec: v2XGPipelineSpec,
    runtime_config: { parameters: { intParam: 123 }, pipeline_root: 'gs://dummy_pipeline_root' },
    scheduled_at: new Date('2021-05-17T20:58:23.000Z'),
    state: V2beta1RuntimeState.SUCCEEDED,
  };

  // Reponse from BE while POST a run for cloning SDK-Run
  const API_SDK_CREATED_CLONING_RUN_DETAILS: V2beta1Run = {
    created_at: new Date('2022-08-12T20:58:23.000Z'),
    description: 'V2 xgboost',
    finished_at: new Date('2022-08-12T21:01:23.000Z'),
    run_id: 'test-clone-sdk-run-id',
    display_name: 'Clone of Run of v2-xgboost-ilbo',
    pipeline_spec: v2XGPipelineSpec,
    runtime_config: { parameters: { intParam: 123 }, pipeline_root: 'gs://dummy_pipeline_root' },
    scheduled_at: new Date('2022-08-12T20:58:23.000Z'),
    state: V2beta1RuntimeState.SUCCEEDED,
  };

  const API_UI_CREATED_NEW_RECURRING_RUN_DETAILS: V2beta1RecurringRun = {
    created_at: new Date('2021-05-17T20:58:23.000Z'),
    description: 'V2 xgboost',
    display_name: 'Run of v2-xgboost-ilbo',
    pipeline_version_reference: {
      pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
      pipeline_version_id: ORIGINAL_TEST_PIPELINE_VERSION_ID,
    },
    recurring_run_id: TEST_RECURRING_RUN_ID,
    runtime_config: { parameters: { intParam: 123 }, pipeline_root: 'gs://dummy_pipeline_root' },
    trigger: {
      periodic_schedule: { interval_second: '3600' },
    },
    max_concurrency: '10',
  };

  const API_UI_CREATED_CLONING_RECURRING_RUN_DETAILS: V2beta1RecurringRun = {
    created_at: new Date('2023-01-04T20:58:23.000Z'),
    description: 'V2 xgboost',
    display_name: 'Clone of Run of v2-xgboost-ilbo',
    pipeline_version_reference: {
      pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
      pipeline_version_id: ORIGINAL_TEST_PIPELINE_VERSION_ID,
    },
    recurring_run_id: 'test-clone-ui-recurring-run-id',
    runtime_config: { parameters: { intParam: 123 }, pipeline_root: 'gs://dummy_pipeline_root' },
    trigger: {
      periodic_schedule: { interval_second: '3600' },
    },
    max_concurrency: '10',
  };

  const API_SDK_CREATED_NEW_RECURRING_RUN_DETAILS: V2beta1RecurringRun = {
    created_at: new Date('2021-05-17T20:58:23.000Z'),
    description: 'V2 xgboost',
    display_name: 'Run of v2-xgboost-ilbo',
    pipeline_spec: v2XGPipelineSpec,
    recurring_run_id: TEST_RECURRING_RUN_ID,
    runtime_config: { parameters: { intParam: 123 }, pipeline_root: 'gs://dummy_pipeline_root' },
    trigger: {
      periodic_schedule: { interval_second: '3600' },
    },
    max_concurrency: '10',
  };

  const API_SDK_CREATED_CLONING_RECURRING_RUN_DETAILS: V2beta1RecurringRun = {
    created_at: new Date('2023-01-04T20:58:23.000Z'),
    description: 'V2 xgboost',
    display_name: 'Clone of Run of v2-xgboost-ilbo',
    pipeline_spec: v2XGPipelineSpec,
    recurring_run_id: 'test-clone-sdk-recurring-run-id',
    runtime_config: { parameters: { intParam: 123 }, pipeline_root: 'gs://dummy_pipeline_root' },
    trigger: {
      periodic_schedule: { interval_second: '3600' },
    },
    max_concurrency: '10',
  };

  const DEFAULT_EXPERIMENT: V2beta1Experiment = {
    created_at: new Date('2022-07-14T21:26:58Z'),
    experiment_id: 'default-experiment-id',
    display_name: 'Default',
    storage_state: V2beta1ExperimentStorageState.AVAILABLE,
  };

  const NEW_EXPERIMENT: V2beta1Experiment = {
    created_at: new Date('2022-07-26T17:44:28Z'),
    experiment_id: 'new-experiment-id',
    display_name: 'new-experiment',
    storage_state: V2beta1ExperimentStorageState.AVAILABLE,
  };

  const historyPushSpy = jest.fn();
  const historyReplaceSpy = jest.fn();
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();

  // For creating new run with no pipeline is selected (enter from run list)
  function generatePropsNoPipelineDef(eid: string | null): PageProps {
    return {
      history: { push: historyPushSpy, replace: historyReplaceSpy } as any,
      location: {
        pathname: RoutePage.NEW_RUN,
        search: eid ? `?${QUERY_PARAMS.experimentId}=${eid}` : `?${QUERY_PARAMS.experimentId}=`,
      } as any,
      match: '' as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: 'Start a new run' },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  // For creating new run with pipeine definition (enter from pipeline details)
  function generatePropsNewRun(
    pid = ORIGINAL_TEST_PIPELINE_ID,
    vid = ORIGINAL_TEST_PIPELINE_VERSION_ID,
  ): PageProps {
    return {
      history: { push: historyPushSpy, replace: historyReplaceSpy } as any,
      location: {
        pathname: RoutePage.NEW_RUN,
        search: `?${QUERY_PARAMS.pipelineId}=${pid}&${QUERY_PARAMS.pipelineVersionId}=${vid}`,
      } as any,
      match: '' as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: 'Start a new run' },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  // For clone run process
  function generatePropsClonedRun(): PageProps {
    return {
      history: { push: historyPushSpy, replace: historyReplaceSpy } as any,
      location: {
        pathname: RoutePage.NEW_RUN,
        search: `?${QUERY_PARAMS.cloneFromRun}=${TEST_RUN_ID}`,
      } as any,
      match: '' as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: 'Clone a run' },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  beforeEach(() => {});

  it('Fulfill default run value (start a new run)', async () => {
    const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
    getPipelineSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
    const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
    getPipelineVersionSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE_VERSION);

    render(
      <CommonTestWrapper>
        <NewRunV2
          {...generatePropsNewRun()}
          existingRunId={null}
          existingRun={undefined}
          existingRecurringRunId={null}
          existingRecurringRun={undefined}
          existingPipeline={ORIGINAL_TEST_PIPELINE}
          handlePipelineIdChange={jest.fn()}
          existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
          handlePipelineVersionIdChange={jest.fn()}
          templateString={v2XGYamlTemplateString}
          chosenExperiment={undefined}
        />
      </CommonTestWrapper>,
    );

    await screen.findByDisplayValue(ORIGINAL_TEST_PIPELINE_NAME);
    await screen.findByDisplayValue(ORIGINAL_TEST_PIPELINE_VERSION_NAME);
    await screen.findByDisplayValue((content, _) =>
      content.startsWith(`Run of ${ORIGINAL_TEST_PIPELINE_VERSION_NAME}`),
    );
  });

  it('Submit run ', async () => {
    const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
    getPipelineSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
    const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
    getPipelineVersionSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE_VERSION);

    render(
      <CommonTestWrapper>
        <NewRunV2
          {...generatePropsNewRun()}
          existingRunId={null}
          existingRun={undefined}
          existingRecurringRunId={null}
          existingRecurringRun={undefined}
          existingPipeline={ORIGINAL_TEST_PIPELINE}
          handlePipelineIdChange={jest.fn()}
          existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
          handlePipelineVersionIdChange={jest.fn()}
          templateString={v2XGYamlTemplateString}
          chosenExperiment={undefined}
        />
      </CommonTestWrapper>,
    );

    const startButton = await screen.findByText('Start');
    expect(startButton.closest('button')?.disabled).toEqual(false);
  });

  it('allows updating the run name (start a new run)', async () => {
    // TODO(jlyaoyuli): create a new test file for NewRunSwitcher and move the following test to it.
    const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
    getPipelineSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
    const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
    getPipelineVersionSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE_VERSION);

    render(
      <CommonTestWrapper>
        <NewRunV2
          {...generatePropsNewRun()}
          existingRunId={null}
          existingRun={undefined}
          existingRecurringRunId={null}
          existingRecurringRun={undefined}
          existingPipeline={ORIGINAL_TEST_PIPELINE}
          handlePipelineIdChange={jest.fn()}
          existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
          handlePipelineVersionIdChange={jest.fn()}
          templateString={v2XGYamlTemplateString}
          chosenExperiment={undefined}
        />
      </CommonTestWrapper>,
    );

    const runNameInput = await screen.findByDisplayValue((content, _) =>
      content.startsWith(`Run of ${ORIGINAL_TEST_PIPELINE_VERSION_NAME}`),
    );
    fireEvent.change(runNameInput, { target: { value: 'Run with custom name' } });
    expect(runNameInput.closest('input')?.value).toBe('Run with custom name');
  });

  describe('redirect to different new run page', () => {
    it('directs to new run v2 if no pipeline is selected (enter from run list)', () => {
      render(
        <CommonTestWrapper>
          <NewRunSwitcher {...generatePropsNoPipelineDef(null)} />
        </CommonTestWrapper>,
      );

      const chooseVersionBtn = screen.getAllByText('Choose')[1];
      // choose button for pipeline version is diabled if no pipeline is selected
      expect(chooseVersionBtn.closest('button')?.disabled).toEqual(true);

      screen.getByText('Pipeline Root'); // only v2 UI has 'Pipeline Root' section
      screen.getByText('A pipeline must be selected');
    });

    it(
      'shows experiment name in new run v2 if experiment is selected' +
        '(enter from experiment details)',
      async () => {
        const getExperimentSpy = jest.spyOn(Apis.experimentServiceApiV2, 'getExperiment');
        getExperimentSpy.mockImplementation(() => NEW_EXPERIMENT);

        render(
          <CommonTestWrapper>
            <NewRunSwitcher {...generatePropsNoPipelineDef(NEW_EXPERIMENT.experiment_id)} />
          </CommonTestWrapper>,
        );

        await waitFor(() => {
          expect(getExperimentSpy).toHaveBeenCalled();
        });

        screen.getByDisplayValue(NEW_EXPERIMENT.display_name);
        screen.getByText('Pipeline Root'); // only v2 UI has 'Pipeline Root' section
      },
    );

    it('directs to new run v2 if it is v2 template (create run from pipeline)', async () => {
      jest
        .spyOn(features, 'isFeatureEnabled')
        .mockImplementation(featureKey => featureKey === features.FeatureKey.V2_ALPHA);
      const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
      getPipelineSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
      const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
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

      screen.getByText('Pipeline Root'); // only v2 UI has 'Pipeline Root' section
    });

    it('directs to new run v1 if it is not v2 template (create run from pipeline)', async () => {
      const TEST_PIPELINE_VERSION_NOT_V2SPEC: V2beta1PipelineVersion = {
        description: '',
        display_name: ORIGINAL_TEST_PIPELINE_VERSION_NAME,
        pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
        pipeline_version_id: 'test-not-v2-spec-version-id',
        pipeline_spec: { spec: { arguments: { parameters: [{ name: 'output' }] } } },
      };

      jest
        .spyOn(features, 'isFeatureEnabled')
        .mockImplementation(featureKey => featureKey === features.FeatureKey.V2_ALPHA);
      const getPipelineV1Spy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
      const getPipelineV2Spy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
      getPipelineV2Spy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
      const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
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

      expect(getPipelineV1Spy).toHaveBeenCalled(); //calling v1 getPipeline() -> direct to new run v1 page
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

        jest
          .spyOn(features, 'isFeatureEnabled')
          .mockImplementation(featureKey => featureKey === features.FeatureKey.V2_ALPHA);
        const getPipelineV1Spy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
        const getPipelineV2Spy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
        getPipelineV2Spy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
        const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
        getPipelineVersionSpy.mockResolvedValue(TEST_PIPELINE_VERSION_WITHOUT_SPEC);
        const getV1PipelineVersionTemplateSpy = jest.spyOn(
          Apis.pipelineServiceApi,
          'getPipelineVersionTemplate',
        );
        getV1PipelineVersionTemplateSpy.mockResolvedValue({ template: 'test template' });

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

        expect(getPipelineV1Spy).toHaveBeenCalled(); //calling v1 getPipeline() -> direct to new run v1 page
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

        jest
          .spyOn(features, 'isFeatureEnabled')
          .mockImplementation(featureKey => featureKey === features.FeatureKey.V2_ALPHA);
        const getPipelineV1Spy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
        const getPipelineV2Spy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
        getPipelineV2Spy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
        const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
        getPipelineVersionSpy.mockResolvedValue(TEST_PIPELINE_VERSION_WITHOUT_SPEC);
        const getV1PipelineVersionTemplateSpy = jest.spyOn(
          Apis.pipelineServiceApi,
          'getPipelineVersionTemplate',
        );
        getV1PipelineVersionTemplateSpy.mockResolvedValue({ template: v2XGYamlTemplateString });

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

        screen.getByText('Pipeline Root'); // only v2 UI has 'Pipeline Root' section
      },
    );
  });

  describe('starting a new run', () => {
    it('disable start button if no run name (start a new run)', async () => {
      const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
      getPipelineSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
      const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
      getPipelineVersionSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE_VERSION);

      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsNewRun()}
            existingRunId={null}
            existingRun={undefined}
            existingRecurringRunId={null}
            existingRecurringRun={undefined}
            existingPipeline={ORIGINAL_TEST_PIPELINE}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2XGYamlTemplateString}
            chosenExperiment={undefined}
          />
        </CommonTestWrapper>,
      );

      const runNameInput = await screen.findByDisplayValue((content, _) =>
        content.startsWith(`Run of ${ORIGINAL_TEST_PIPELINE_VERSION_NAME}`),
      );
      fireEvent.change(runNameInput, { target: { value: '' } });

      const startButton = await screen.findByText('Start');
      expect(startButton.closest('button')?.disabled).toEqual(true);
      expect(await screen.findByText('Run name can not be empty.'));
    });

    it('submit a new run without parameter (create new run)', async () => {
      const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
      getPipelineSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
      const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
      getPipelineVersionSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE_VERSION);
      const createRunSpy = jest.spyOn(Apis.runServiceApiV2, 'createRun');
      createRunSpy.mockResolvedValue(API_UI_CREATED_NEW_RUN_DETAILS);

      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsNewRun()}
            existingRunId={null}
            existingRun={undefined}
            existingRecurringRunId={null}
            existingRecurringRun={undefined}
            existingPipeline={ORIGINAL_TEST_PIPELINE}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2XGYamlTemplateString}
            chosenExperiment={undefined}
          />
        </CommonTestWrapper>,
      );

      const startButton = await screen.findByText('Start');
      fireEvent.click(startButton);

      await waitFor(() => {
        expect(createRunSpy).toHaveBeenCalledWith(
          expect.objectContaining({
            description: '',
            pipeline_version_reference: {
              pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
              pipeline_version_id: ORIGINAL_TEST_PIPELINE_VERSION_ID,
            },
            runtime_config: { parameters: {}, pipeline_root: 'minio://dummy_root' },
            service_account: '',
          }),
        );
      });
    });
  });

  describe('choose a pipeline', () => {
    it('sets the pipeline from the selector modal when confirmed', async () => {
      const listPipelineSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'listPipelines');
      listPipelineSpy.mockImplementation(() => {
        const response: V2beta1ListPipelinesResponse = {
          pipelines: [ORIGINAL_TEST_PIPELINE, NEW_TEST_PIPELINE],
          total_size: 2,
        };
        return response;
      });
      const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
      getPipelineSpy.mockImplementation(() => NEW_TEST_PIPELINE);

      const listPipelineVersionsSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'listPipelineVersions');
      listPipelineVersionsSpy.mockImplementation(() => {
        const response: V2beta1ListPipelinesResponse = {
          pipeline_versions: [NEW_TEST_PIPELINE_VERSION],
          total_size: 1,
        };
        return response;
      });

      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsNewRun()}
            namespace='test-ns'
            existingRunId={null}
            existingRun={undefined}
            existingRecurringRunId={null}
            existingRecurringRun={undefined}
            existingPipeline={ORIGINAL_TEST_PIPELINE}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2XGYamlTemplateString}
            chosenExperiment={DEFAULT_EXPERIMENT}
          />
        </CommonTestWrapper>,
      );

      const choosePipelineButton = screen.getAllByText('Choose')[0];
      fireEvent.click(choosePipelineButton);

      const expectedPipeline = await screen.findByText(NEW_TEST_PIPELINE_NAME);
      fireEvent.click(expectedPipeline);

      await waitFor(() => {
        expect(getPipelineSpy).toHaveBeenCalled();
      });

      const usePipelineButton = screen.getByText('Use this pipeline');
      fireEvent.click(usePipelineButton);

      // After pipeline is selected, listPipelineVersions will be called to
      // retrieve the latest version.
      await waitFor(() => {
        expect(listPipelineVersionsSpy).toHaveBeenCalledWith(
          NEW_TEST_PIPELINE_ID,
          undefined,
          1,
          'created_at desc',
        );
      });

      screen.getByDisplayValue(NEW_TEST_PIPELINE_NAME);

      const chooseVersionBtn = screen.getAllByText('Choose')[1];
      // choose button for pipeline version is enabled after pipeline is selected
      expect(chooseVersionBtn.closest('button')?.disabled).toEqual(false);
    });
  });

  describe('choose a pipeline version', () => {
    it('sets the pipeline version from the selector modal when confirmed', async () => {
      const listPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'listPipelineVersions');
      listPipelineVersionSpy.mockImplementation(() => {
        const response: V2beta1ListPipelineVersionsResponse = {
          pipeline_versions: [ORIGINAL_TEST_PIPELINE_VERSION, OTHER_TEST_PIPELINE_VERSION],
          total_size: 2,
        };
        return response;
      });
      const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
      getPipelineVersionSpy.mockImplementation(() => OTHER_TEST_PIPELINE_VERSION);

      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsNewRun()}
            namespace='test-ns'
            existingRunId={null}
            existingRun={undefined}
            existingRecurringRunId={null}
            existingRecurringRun={undefined}
            existingPipeline={ORIGINAL_TEST_PIPELINE}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2XGYamlTemplateString}
            chosenExperiment={DEFAULT_EXPERIMENT}
          />
        </CommonTestWrapper>,
      );

      const choosePipelineVersionBtn = screen.getAllByText('Choose')[1];
      fireEvent.click(choosePipelineVersionBtn);

      const expectedPipelineVersion = await screen.findByText(OTHER_TEST_PIPELINE_VERSION_NAME);
      fireEvent.click(expectedPipelineVersion);

      await waitFor(() => {
        expect(getPipelineVersionSpy).toHaveBeenCalled();
      });

      const usePipelineVersionBtn = screen.getByText('Use this pipeline version');
      fireEvent.click(usePipelineVersionBtn);

      screen.getByDisplayValue(OTHER_TEST_PIPELINE_VERSION_NAME);
    });
  });

  describe('choose an experiment', () => {
    it('lists available experiments by namespace if available', async () => {
      const listExperimentSpy = jest.spyOn(Apis.experimentServiceApiV2, 'listExperiments');
      listExperimentSpy.mockImplementation(() => {
        const response: V2beta1ListPipelinesResponse = {
          experiments: [DEFAULT_EXPERIMENT, NEW_EXPERIMENT],
          total_size: 2,
        };
        return response;
      });

      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsNewRun()}
            namespace='test-ns'
            existingRunId={null}
            existingRun={undefined}
            existingRecurringRunId={null}
            existingRecurringRun={undefined}
            existingPipeline={ORIGINAL_TEST_PIPELINE}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2XGYamlTemplateString}
            chosenExperiment={DEFAULT_EXPERIMENT}
          />
        </CommonTestWrapper>,
      );

      const chooseExperimentButton = screen.getAllByText('Choose')[2];
      fireEvent.click(chooseExperimentButton);

      await waitFor(() => {
        expect(listExperimentSpy).toHaveBeenCalledWith(
          '',
          10,
          'created_at desc',
          encodeURIComponent(
            JSON.stringify({
              predicates: [
                {
                  key: 'storage_state',
                  operation: V2beta1PredicateOperation.NOTEQUALS,
                  string_value: V2beta1ExperimentStorageState.ARCHIVED.toString(),
                },
              ],
            } as V2beta1Filter),
          ),
          'test-ns',
        );
      });
    });

    it('sets the experiment from the selector modal when confirmed', async () => {
      const listExperimentSpy = jest.spyOn(Apis.experimentServiceApiV2, 'listExperiments');
      listExperimentSpy.mockImplementation(() => {
        const response: V2beta1ListExperimentsResponse = {
          experiments: [DEFAULT_EXPERIMENT, NEW_EXPERIMENT],
          total_size: 2,
        };
        return response;
      });
      const getExperimentSpy = jest.spyOn(Apis.experimentServiceApiV2, 'getExperiment');
      getExperimentSpy.mockImplementation(() => NEW_EXPERIMENT);

      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsNewRun()}
            existingRunId={null}
            existingRun={undefined}
            existingRecurringRunId={null}
            existingRecurringRun={undefined}
            existingPipeline={ORIGINAL_TEST_PIPELINE}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2XGYamlTemplateString}
            chosenExperiment={DEFAULT_EXPERIMENT}
          />
        </CommonTestWrapper>,
      );

      const chooseExperimentButton = screen.getAllByText('Choose')[2];
      fireEvent.click(chooseExperimentButton);

      const expectedExperiment = await screen.findByText('new-experiment');
      fireEvent.click(expectedExperiment);

      await waitFor(() => {
        expect(getExperimentSpy).toHaveBeenCalled();
      });

      const useExperimentButton = screen.getByText('Use this experiment');
      fireEvent.click(useExperimentButton);

      screen.getByDisplayValue('new-experiment');
    });
  });

  describe('creating a recurring run', () => {
    it('submits a new recurring run', async () => {
      const createRecurringRunSpy = jest.spyOn(Apis.recurringRunServiceApi, 'createRecurringRun');
      createRecurringRunSpy.mockResolvedValue(API_UI_CREATED_NEW_RECURRING_RUN_DETAILS);

      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsNewRun()}
            existingRunId={null}
            existingRun={undefined}
            existingRecurringRunId={null}
            existingRecurringRun={undefined}
            existingPipeline={ORIGINAL_TEST_PIPELINE}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2XGYamlTemplateString}
            chosenExperiment={DEFAULT_EXPERIMENT}
          />
        </CommonTestWrapper>,
      );

      const recurringSwitcher = screen.getByLabelText('Recurring');
      fireEvent.click(recurringSwitcher);
      screen.getByText('Run trigger');

      const startButton = await screen.findByText('Start');
      // Because start button is set false by default
      await waitFor(() => {
        expect(startButton.closest('button')?.disabled).toEqual(false);
      });
      fireEvent.click(startButton);

      await waitFor(() => {
        expect(createRecurringRunSpy).toHaveBeenCalledWith(
          expect.objectContaining({
            max_concurrency: '10',
            mode: RecurringRunMode.ENABLE,
            trigger: {
              periodic_schedule: {
                interval_second: '3600',
              },
            },
          }),
        );
      });

      await waitFor(() => {
        expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
          message: 'Successfully started new recurring Run: Run of v2-xgboost-ilbo',
          open: true,
        });
      });
    });

    it('enables to change the trigger parameters.', async () => {
      const createRecurringRunSpy = jest.spyOn(Apis.recurringRunServiceApi, 'createRecurringRun');

      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsNewRun()}
            existingRunId={null}
            existingRun={undefined}
            existingRecurringRunId={null}
            existingRecurringRun={undefined}
            existingPipeline={ORIGINAL_TEST_PIPELINE}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2XGYamlTemplateString}
            chosenExperiment={DEFAULT_EXPERIMENT}
          />
        </CommonTestWrapper>,
      );

      const recurringSwitcher = screen.getByLabelText('Recurring');
      fireEvent.click(recurringSwitcher);

      const maxConcurrenyParam = screen.getByDisplayValue('10');
      fireEvent.change(maxConcurrenyParam, { target: { value: '5' } });

      const timeCountParam = screen.getByDisplayValue('1');
      fireEvent.change(timeCountParam, { target: { value: '5' } });

      const timeUnitDropdown = screen.getAllByText('Hours')[0];
      fireEvent.click(timeUnitDropdown);
      const minutesItem = await screen.findByText('Minutes');
      fireEvent.click(minutesItem);

      const startButton = await screen.findByText('Start');
      // Because start button is set false by default
      await waitFor(() => {
        expect(startButton.closest('button')?.disabled).toEqual(false);
      });
      fireEvent.click(startButton);

      await waitFor(() => {
        expect(createRecurringRunSpy).toHaveBeenCalledWith(
          expect.objectContaining({
            max_concurrency: '5',
            mode: RecurringRunMode.ENABLE,
            trigger: {
              periodic_schedule: {
                interval_second: '300',
              },
            },
          }),
        );
      });
    });

    it('disables the start button if max concurrent run is invalid input (non-integer)', async () => {
      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsNewRun()}
            existingRunId={null}
            existingRun={undefined}
            existingRecurringRunId={null}
            existingRecurringRun={undefined}
            existingPipeline={ORIGINAL_TEST_PIPELINE}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2XGYamlTemplateString}
            chosenExperiment={DEFAULT_EXPERIMENT}
          />
        </CommonTestWrapper>,
      );

      const recurringSwitcher = screen.getByLabelText('Recurring');
      fireEvent.click(recurringSwitcher);

      const maxConcurrenyParam = screen.getByDisplayValue('10');
      fireEvent.change(maxConcurrenyParam, { target: { value: '10a' } });

      const startButton = await screen.findByText('Start');
      expect(startButton.closest('button')?.disabled).toEqual(true);
    });

    it('disables the start button if max concurrent run is invalid input (negative integer)', async () => {
      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsNewRun()}
            existingRunId={null}
            existingRun={undefined}
            existingRecurringRunId={null}
            existingRecurringRun={undefined}
            existingPipeline={ORIGINAL_TEST_PIPELINE}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2XGYamlTemplateString}
            chosenExperiment={DEFAULT_EXPERIMENT}
          />
        </CommonTestWrapper>,
      );

      const recurringSwitcher = screen.getByLabelText('Recurring');
      fireEvent.click(recurringSwitcher);

      const maxConcurrenyParam = screen.getByDisplayValue('10');
      fireEvent.change(maxConcurrenyParam, { target: { value: '-10' } });

      const startButton = await screen.findByText('Start');
      expect(startButton.closest('button')?.disabled).toEqual(true);
    });
  });

  describe('cloning an existing run', () => {
    it('only shows clone run name from original run', () => {
      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsClonedRun()}
            existingRunId={TEST_RUN_ID}
            existingRun={API_UI_CREATED_NEW_RUN_DETAILS}
            existingRecurringRunId={null}
            existingRecurringRun={undefined}
            existingPipeline={undefined}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={undefined}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2XGYamlTemplateString}
            chosenExperiment={undefined}
          />
        </CommonTestWrapper>,
      );
      screen.findByDisplayValue(`Clone of ${API_UI_CREATED_NEW_RUN_DETAILS.display_name}`);
    });

    it('submits a run (clone UI-created run)', async () => {
      const createRunSpy = jest.spyOn(Apis.runServiceApiV2, 'createRun');
      createRunSpy.mockResolvedValue(API_UI_CREATED_CLONING_RUN_DETAILS);

      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsClonedRun()}
            existingRunId={TEST_RUN_ID}
            existingRun={API_UI_CREATED_NEW_RUN_DETAILS}
            existingRecurringRunId={null}
            existingRecurringRun={undefined}
            existingPipeline={undefined}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={undefined}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2XGYamlTemplateString}
            chosenExperiment={undefined}
          />
        </CommonTestWrapper>,
      );

      const startButton = await screen.findByText('Start');
      // Because start button is set false by default
      await waitFor(() => {
        expect(startButton.closest('button')?.disabled).toEqual(false);
      });
      fireEvent.click(startButton);

      await waitFor(() => {
        expect(createRunSpy).toHaveBeenCalledWith(
          expect.objectContaining({
            description: '',
            display_name: 'Clone of Run of v2-xgboost-ilbo',
            pipeline_version_reference: {
              pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
              pipeline_version_id: ORIGINAL_TEST_PIPELINE_VERSION_ID,
            },
            runtime_config: {
              parameters: { intParam: 123 },
              pipeline_root: 'gs://dummy_pipeline_root',
            },
            service_account: '',
          }),
        );
      });
    });

    it('submits a run (clone SDK-created run)', async () => {
      const createRunSpy = jest.spyOn(Apis.runServiceApiV2, 'createRun');
      createRunSpy.mockResolvedValue(API_SDK_CREATED_CLONING_RUN_DETAILS);

      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsClonedRun()}
            existingRunId={TEST_RUN_ID}
            existingRun={API_SDK_CREATED_NEW_RUN_DETAILS}
            existingRecurringRunId={null}
            existingRecurringRun={undefined}
            existingPipeline={undefined}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={undefined}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2XGYamlTemplateString}
            chosenExperiment={undefined}
          />
        </CommonTestWrapper>,
      );

      const startButton = await screen.findByText('Start');
      // Because start button is set false by default
      await waitFor(() => {
        expect(startButton.closest('button')?.disabled).toEqual(false);
      });
      fireEvent.click(startButton);

      await waitFor(() => {
        expect(createRunSpy).toHaveBeenCalledWith(
          expect.objectContaining({
            description: '',
            display_name: 'Clone of Run of v2-xgboost-ilbo',
            pipeline_spec: JsYaml.safeLoad(v2XGYamlTemplateString),
            runtime_config: {
              parameters: { intParam: 123 },
              pipeline_root: 'gs://dummy_pipeline_root',
            },
            service_account: '',
          }),
        );
      });
    });
  });

  describe('clone an existing recurring run', () => {
    it('submits a recurring run with same runtimeConfig and trigger from clone UI-created recurring run', async () => {
      const createRecurringRunSpy = jest.spyOn(Apis.recurringRunServiceApi, 'createRecurringRun');
      createRecurringRunSpy.mockResolvedValue(API_UI_CREATED_CLONING_RECURRING_RUN_DETAILS);

      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsClonedRun()}
            existingRunId={null}
            existingRun={undefined}
            existingRecurringRunId={TEST_RECURRING_RUN_ID}
            existingRecurringRun={API_UI_CREATED_NEW_RECURRING_RUN_DETAILS}
            existingPipeline={undefined}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={undefined}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2XGYamlTemplateString}
            chosenExperiment={undefined}
          />
        </CommonTestWrapper>,
      );

      const startButton = await screen.findByText('Start');
      // Because start button is set false by default
      await waitFor(() => {
        expect(startButton.closest('button')?.disabled).toEqual(false);
      });
      fireEvent.click(startButton);

      await waitFor(() => {
        expect(createRecurringRunSpy).toHaveBeenCalledWith(
          expect.objectContaining({
            description: '',
            display_name: 'Clone of Run of v2-xgboost-ilbo',
            pipeline_version_reference: {
              pipeline_id: ORIGINAL_TEST_PIPELINE_ID,
              pipeline_version_id: ORIGINAL_TEST_PIPELINE_VERSION_ID,
            },
            runtime_config: {
              parameters: { intParam: 123 },
              pipeline_root: 'gs://dummy_pipeline_root',
            },
            trigger: {
              periodic_schedule: { interval_second: '3600' },
            },
            max_concurrency: '10',
          }),
        );
      });

      await waitFor(() => {
        expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
          message: 'Successfully started new recurring Run: Clone of Run of v2-xgboost-ilbo',
          open: true,
        });
      });
    });

    it('submits a recurring run with same runtimeConfig and trigger from clone SDK-created recurring run', async () => {
      const createRecurringRunSpy = jest.spyOn(Apis.recurringRunServiceApi, 'createRecurringRun');
      createRecurringRunSpy.mockResolvedValue(API_SDK_CREATED_CLONING_RECURRING_RUN_DETAILS);

      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsClonedRun()}
            existingRunId={null}
            existingRun={undefined}
            existingRecurringRunId={TEST_RECURRING_RUN_ID}
            existingRecurringRun={API_SDK_CREATED_NEW_RECURRING_RUN_DETAILS}
            existingPipeline={undefined}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={undefined}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2XGYamlTemplateString}
            chosenExperiment={undefined}
          />
        </CommonTestWrapper>,
      );

      const startButton = await screen.findByText('Start');
      // Because start button is set false by default
      await waitFor(() => {
        expect(startButton.closest('button')?.disabled).toEqual(false);
      });
      fireEvent.click(startButton);

      await waitFor(() => {
        expect(createRecurringRunSpy).toHaveBeenCalledWith(
          expect.objectContaining({
            description: '',
            display_name: 'Clone of Run of v2-xgboost-ilbo',
            pipeline_spec: JsYaml.safeLoad(v2XGYamlTemplateString),
            runtime_config: {
              parameters: { intParam: 123 },
              pipeline_root: 'gs://dummy_pipeline_root',
            },
            trigger: {
              periodic_schedule: { interval_second: '3600' },
            },
            max_concurrency: '10',
          }),
        );
      });

      await waitFor(() => {
        expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
          message: 'Successfully started new recurring Run: Clone of Run of v2-xgboost-ilbo',
          open: true,
        });
      });
    });
  });
});
