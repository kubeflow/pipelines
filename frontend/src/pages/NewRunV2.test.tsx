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
import React from 'react';
import { testBestPractices } from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import {
  ApiExperiment,
  ApiExperimentStorageState,
  ApiListExperimentsResponse,
} from 'src/apis/experiment';
import { ApiFilter, PredicateOp } from 'src/apis/filter';
import { ApiJob } from 'src/apis/job';
import {
  ApiListPipelinesResponse,
  ApiListPipelineVersionsResponse,
  ApiPipeline,
} from 'src/apis/pipeline';
import { ApiRelationship, ApiResourceType } from 'src/apis/run';
import { V2beta1Run, V2beta1RuntimeState } from 'src/apisv2beta1/run';
import { V2beta1RecurringRun, V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';
import { NameWithTooltip } from 'src/components/CustomTableNameColumn';
import { QUERY_PARAMS, RoutePage } from 'src/components/Router';
import { Apis, ExperimentSortKeys } from 'src/lib/Apis';
import { convertYamlToV2PipelineSpec } from 'src/lib/v2/WorkflowUtils';
import NewRunV2 from './NewRunV2';
import { PageProps } from './Page';
import * as JsYaml from 'js-yaml';

const V2_PIPELINESPEC_PATH = 'src/data/test/xgboost_sample_pipeline.yaml';
const v2YamlTemplateString = fs.readFileSync(V2_PIPELINESPEC_PATH, 'utf8');
const v2PipelineSpec = convertYamlToV2PipelineSpec(v2YamlTemplateString);

testBestPractices();

describe('NewRunV2', () => {
  const TEST_RUN_ID = 'test-run-id';
  const TEST_RECURRING_RUN_ID = 'test-recurring-run-id';
  const ORIGINAL_TEST_PIPELINE_ID = 'test-pipeline-id';
  const ORIGINAL_TEST_PIPELINE_NAME = 'test pipeline';
  const ORIGINAL_TEST_PIPELINE_VERSION_ID = 'test-pipeline-version-id';
  const ORIGINAL_TEST_PIPELINE_VERSION_NAME = 'test pipeline version';
  const ORIGINAL_TEST_PIPELINE: ApiPipeline = {
    created_at: new Date(2018, 8, 5, 4, 3, 2),
    description: '',
    id: 'test-pipeline-id',
    name: 'test pipeline',
    parameters: [{ name: 'param1', value: 'value1' }],
    default_version: {
      id: 'test-pipeline-version-id',
      description: '',
      name: ORIGINAL_TEST_PIPELINE_VERSION_NAME,
    },
  };
  const ORIGINAL_TEST_PIPELINE_VERSION = {
    id: 'test-pipeline-version-id',
    name: ORIGINAL_TEST_PIPELINE_VERSION_NAME,
    description: '',
  };

  const NEW_TEST_PIPELINE_ID = 'new-test-pipeline-id';
  const NEW_TEST_PIPELINE_NAME = 'new-test-pipeline';
  const NEW_TEST_PIPELINE_VERSION_ID = 'new-test-pipeline-version-id';
  const NEW_TEST_PIPELINE_VERSION_NAME = 'new-test-pipeline-version';
  const NEW_TEST_PIPELINE: ApiPipeline = {
    created_at: new Date(2018, 8, 7, 6, 5, 4),
    description: '',
    id: 'new-test-pipeline-id',
    name: 'new-test-pipeline',
    parameters: [{ name: 'param1', value: 'value1' }],
    default_version: {
      id: 'new-test-pipeline-version-id',
      description: '',
      name: NEW_TEST_PIPELINE_VERSION_NAME,
    },
  };
  const NEW_TEST_PIPELINE_VERSION = {
    id: 'new-test-pipeline-version-id',
    name: NEW_TEST_PIPELINE_VERSION_NAME,
    description: '',
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
    runtime_config: { parameters: { intParam: 123 } },
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
      pipeline_id: NEW_TEST_PIPELINE_ID,
      pipeline_version_id: NEW_TEST_PIPELINE_VERSION_ID,
    },
    runtime_config: { parameters: { intParam: 123 } },
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
    pipeline_spec: v2PipelineSpec,
    runtime_config: { parameters: { intParam: 123 } },
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
    pipeline_spec: v2PipelineSpec,
    runtime_config: { parameters: { intParam: 123 } },
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
    runtime_config: { parameters: { intParam: 123 } },
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
      pipeline_id: NEW_TEST_PIPELINE_ID,
      pipeline_version_id: NEW_TEST_PIPELINE_VERSION_ID,
    },
    recurring_run_id: 'test-clone-ui-recurring-run-id',
    runtime_config: { parameters: { intParam: 123 } },
    trigger: {
      periodic_schedule: { interval_second: '3600' },
    },
    max_concurrency: '10',
  };

  const API_SDK_CREATED_NEW_RECURRING_RUN_DETAILS: V2beta1RecurringRun = {
    created_at: new Date('2021-05-17T20:58:23.000Z'),
    description: 'V2 xgboost',
    display_name: 'Run of v2-xgboost-ilbo',
    pipeline_spec: v2PipelineSpec,
    recurring_run_id: TEST_RECURRING_RUN_ID,
    runtime_config: { parameters: { intParam: 123 } },
    trigger: {
      periodic_schedule: { interval_second: '3600' },
    },
    max_concurrency: '10',
  };

  const API_SDK_CREATED_CLONING_RECURRING_RUN_DETAILS: V2beta1RecurringRun = {
    created_at: new Date('2023-01-04T20:58:23.000Z'),
    description: 'V2 xgboost',
    display_name: 'Clone of Run of v2-xgboost-ilbo',
    pipeline_spec: v2PipelineSpec,
    recurring_run_id: 'test-clone-sdk-recurring-run-id',
    runtime_config: { parameters: { intParam: 123 } },
    trigger: {
      periodic_schedule: { interval_second: '3600' },
    },
    max_concurrency: '10',
  };

  const DEFAULT_EXPERIMENT: ApiExperiment = {
    created_at: new Date('2022-07-14T21:26:58Z'),
    id: '796eb126-dd76-44de-a21f-d70010c6a029',
    name: 'Default',
    storage_state: ApiExperimentStorageState.AVAILABLE,
  };

  const NEW_EXPERIMENT: ApiExperiment = {
    created_at: new Date('2022-07-26T17:44:28Z'),
    id: 'f66a1cee-b7cb-43f0-a3c2-6ec169c9a9b1',
    name: 'new-experiment',
    storage_state: ApiExperimentStorageState.AVAILABLE,
  };

  const historyPushSpy = jest.fn();
  const historyReplaceSpy = jest.fn();
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  function generatePropsNewRun(): PageProps {
    return {
      history: { push: historyPushSpy, replace: historyReplaceSpy } as any,
      location: {
        pathname: RoutePage.NEW_RUN,
        search: `?${QUERY_PARAMS.pipelineId}=${ORIGINAL_TEST_PIPELINE_ID}&${QUERY_PARAMS.pipelineVersionId}=${ORIGINAL_TEST_PIPELINE_VERSION_ID}`,
      } as any,
      match: '' as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: 'Start a new run' },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }
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
    const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
    getPipelineSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
    const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipelineVersion');
    getPipelineVersionSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE_VERSION);
    const getPipelineVersionTemplateSpy = jest.spyOn(
      Apis.pipelineServiceApi,
      'getPipelineVersionTemplate',
    );
    getPipelineVersionTemplateSpy.mockImplementation(() =>
      Promise.resolve({ template: v2YamlTemplateString }),
    );
    render(
      <CommonTestWrapper>
        <NewRunV2
          {...generatePropsNewRun()}
          existingRunId='e0115ac1-0479-4194-a22d-01e65e09a32b'
          existingRun={undefined}
          existingRecurringRunId={null}
          existingRecurringRun={undefined}
          existingPipeline={ORIGINAL_TEST_PIPELINE}
          handlePipelineIdChange={jest.fn()}
          existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
          handlePipelineVersionIdChange={jest.fn()}
          templateString={v2YamlTemplateString}
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
    const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
    getPipelineSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
    const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipelineVersion');
    getPipelineVersionSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE_VERSION);
    const getPipelineVersionTemplateSpy = jest.spyOn(
      Apis.pipelineServiceApi,
      'getPipelineVersionTemplate',
    );
    getPipelineVersionTemplateSpy.mockImplementation(() =>
      Promise.resolve({ template: v2YamlTemplateString }),
    );
    render(
      <CommonTestWrapper>
        <NewRunV2
          {...generatePropsNewRun()}
          existingRunId='e0115ac1-0479-4194-a22d-01e65e09a32b'
          existingRun={undefined}
          existingRecurringRunId={null}
          existingRecurringRun={undefined}
          existingPipeline={ORIGINAL_TEST_PIPELINE}
          handlePipelineIdChange={jest.fn()}
          existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
          handlePipelineVersionIdChange={jest.fn()}
          templateString={v2YamlTemplateString}
          chosenExperiment={undefined}
        />
      </CommonTestWrapper>,
    );

    const startButton = await screen.findByText('Start');
    expect(startButton.closest('button')?.disabled).toEqual(false);
  });

  it('allows updating the run name (start a new run)', async () => {
    // TODO(jlyaoyuli): create a new test file for NewRunSwitcher and move the following test to it.
    const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
    getPipelineSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
    const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipelineVersion');
    getPipelineVersionSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE_VERSION);
    const getPipelineVersionTemplateSpy = jest.spyOn(
      Apis.pipelineServiceApi,
      'getPipelineVersionTemplate',
    );
    getPipelineVersionTemplateSpy.mockImplementation(() =>
      Promise.resolve({ template: v2YamlTemplateString }),
    );
    render(
      <CommonTestWrapper>
        <NewRunV2
          {...generatePropsNewRun()}
          existingRunId='e0115ac1-0479-4194-a22d-01e65e09a32b'
          existingRun={undefined}
          existingRecurringRunId={null}
          existingRecurringRun={undefined}
          existingPipeline={ORIGINAL_TEST_PIPELINE}
          handlePipelineIdChange={jest.fn()}
          existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
          handlePipelineVersionIdChange={jest.fn()}
          templateString={v2YamlTemplateString}
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

  describe('starting a new run', () => {
    it('disable start button if no run name (start a new run)', async () => {
      const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
      getPipelineSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
      const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipelineVersion');
      getPipelineVersionSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE_VERSION);
      const getPipelineVersionTemplateSpy = jest.spyOn(
        Apis.pipelineServiceApi,
        'getPipelineVersionTemplate',
      );
      getPipelineVersionTemplateSpy.mockImplementation(() =>
        Promise.resolve({ template: v2YamlTemplateString }),
      );
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
            templateString={v2YamlTemplateString}
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
      const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
      getPipelineSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE);
      const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipelineVersion');
      getPipelineVersionSpy.mockResolvedValue(ORIGINAL_TEST_PIPELINE_VERSION);
      const getPipelineVersionTemplateSpy = jest.spyOn(
        Apis.pipelineServiceApi,
        'getPipelineVersionTemplate',
      );
      getPipelineVersionTemplateSpy.mockImplementation(() =>
        Promise.resolve({ template: v2YamlTemplateString }),
      );
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
            templateString={v2YamlTemplateString}
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
            runtime_config: { parameters: {}, pipeline_root: undefined },
            service_account: '',
          }),
        );
      });
    });
  });

  describe('choose a pipeline', () => {
    it('sets the pipeline from the selector modal when confirmed', async () => {
      const listPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'listPipelines');
      listPipelineSpy.mockImplementation(() => {
        const response: ApiListPipelinesResponse = {
          pipelines: [ORIGINAL_TEST_PIPELINE, NEW_TEST_PIPELINE],
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
            templateString={v2YamlTemplateString}
            chosenExperiment={DEFAULT_EXPERIMENT}
          />
        </CommonTestWrapper>,
      );

      const choosePipelineButton = screen.getAllByText('Choose')[0];
      fireEvent.click(choosePipelineButton);

      const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
      getPipelineSpy.mockImplementation(() => NEW_TEST_PIPELINE);

      const expectedPipeline = await screen.findByText(NEW_TEST_PIPELINE_NAME);
      fireEvent.click(expectedPipeline);

      const usePipelineButton = screen.getByText('Use this pipeline');
      fireEvent.click(usePipelineButton);

      screen.getByDisplayValue(NEW_TEST_PIPELINE_NAME);
    });
  });

  describe('choose a pipeline version', () => {
    it('sets the pipeline version from the selector modal when confirmed', async () => {
      const listPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'listPipelineVersions');
      listPipelineVersionSpy.mockImplementation(() => {
        const response: ApiListPipelineVersionsResponse = {
          versions: [ORIGINAL_TEST_PIPELINE_VERSION, NEW_TEST_PIPELINE_VERSION],
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
            templateString={v2YamlTemplateString}
            chosenExperiment={DEFAULT_EXPERIMENT}
          />
        </CommonTestWrapper>,
      );

      const choosePipelineVersionBtn = screen.getAllByText('Choose')[1];
      fireEvent.click(choosePipelineVersionBtn);

      const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipelineVersion');
      getPipelineVersionSpy.mockImplementation(() => NEW_TEST_PIPELINE_VERSION);

      const expectedPipelineVersion = await screen.findByText(NEW_TEST_PIPELINE_VERSION_NAME);
      fireEvent.click(expectedPipelineVersion);

      const usePipelineVersionBtn = screen.getByText('Use this pipeline version');
      fireEvent.click(usePipelineVersionBtn);

      screen.getByDisplayValue(NEW_TEST_PIPELINE_VERSION_NAME);
    });
  });

  describe('choose an experiment', () => {
    it('lists available experiments by namespace if available', async () => {
      const listExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'listExperiment');
      listExperimentSpy.mockImplementation(() => {
        const response: ApiListExperimentsResponse = {
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
            templateString={v2YamlTemplateString}
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
                  op: PredicateOp.NOTEQUALS,
                  string_value: ApiExperimentStorageState.ARCHIVED.toString(),
                },
              ],
            } as ApiFilter),
          ),
          'NAMESPACE',
          'test-ns',
        );
      });
    });

    it('sets the experiment from the selector modal when confirmed', async () => {
      const listExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'listExperiment');
      listExperimentSpy.mockImplementation(() => {
        const response: ApiListExperimentsResponse = {
          experiments: [DEFAULT_EXPERIMENT, NEW_EXPERIMENT],
          total_size: 2,
        };
        return response;
      });

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
            templateString={v2YamlTemplateString}
            chosenExperiment={DEFAULT_EXPERIMENT}
          />
        </CommonTestWrapper>,
      );

      const chooseExperimentButton = screen.getAllByText('Choose')[2];
      fireEvent.click(chooseExperimentButton);

      const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
      getExperimentSpy.mockImplementation(() => NEW_EXPERIMENT);

      const expectedExperiment = await screen.findByText('new-experiment');
      fireEvent.click(expectedExperiment);

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
            templateString={v2YamlTemplateString}
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
            status: V2beta1RecurringRunStatus.ENABLED,
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
            templateString={v2YamlTemplateString}
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
            status: V2beta1RecurringRunStatus.ENABLED,
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
            existingPipeline={ORIGINAL_TEST_PIPELINE}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2YamlTemplateString}
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
            existingPipeline={ORIGINAL_TEST_PIPELINE}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={ORIGINAL_TEST_PIPELINE_VERSION}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2YamlTemplateString}
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
            existingRunId='e0115ac1-0479-4194-a22d-01e65e09a32b'
            existingRun={API_UI_CREATED_NEW_RUN_DETAILS}
            existingRecurringRunId={null}
            existingRecurringRun={undefined}
            existingPipeline={undefined}
            handlePipelineIdChange={jest.fn()}
            existingPipelineVersion={undefined}
            handlePipelineVersionIdChange={jest.fn()}
            templateString={v2YamlTemplateString}
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
            templateString={v2YamlTemplateString}
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
            runtime_config: { parameters: { intParam: 123 } },
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
            templateString={v2YamlTemplateString}
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
            pipeline_spec: JsYaml.safeLoad(v2YamlTemplateString),
            runtime_config: { parameters: { intParam: 123 } },
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
            templateString={v2YamlTemplateString}
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
            runtime_config: { parameters: { intParam: 123 } },
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
            templateString={v2YamlTemplateString}
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
            pipeline_spec: JsYaml.safeLoad(v2YamlTemplateString),
            runtime_config: { parameters: { intParam: 123 } },
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
