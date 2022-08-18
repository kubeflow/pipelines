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
import { ApiPipeline } from '../apis/pipeline';
import { ApiRelationship, ApiResourceType, ApiRunDetail } from '../apis/run';
import { QUERY_PARAMS, RoutePage } from '../components/Router';
import { Apis } from '../lib/Apis';
import NewRunV2 from './NewRunV2';
import { PageProps } from './Page';

const V2_PIPELINESPEC_PATH = 'src/data/test/xgboost_sample_pipeline.yaml';
const v2YamlTemplateString = fs.readFileSync(V2_PIPELINESPEC_PATH, 'utf8');

testBestPractices();

describe('NewRunV2', () => {
  const TEST_RUN_ID = 'test-run-id';
  const TEST_PIPELINE_ID = 'test-pipeline-id';
  const TEST_PIPELINE_NAME = 'test pipeline';
  const TEST_PIPELINE_VERSION_ID = 'test-pipeline-version-id';
  const TEST_PIPELINE_VERSION_NAME = 'test pipeline version';
  const TEST_PIPELINE: ApiPipeline = {
    created_at: new Date(2018, 8, 5, 4, 3, 2),
    description: '',
    id: 'test-pipeline-id',
    name: 'test pipeline',
    parameters: [{ name: 'param1', value: 'value1' }],
    default_version: {
      id: 'test-pipeline-version-id',
      description: '',
      name: TEST_PIPELINE_VERSION_NAME,
    },
  };
  const TEST_PIPELINE_VERSION = {
    id: 'test-pipeline-version-id',
    name: TEST_PIPELINE_VERSION_NAME,
    description: '',
  };

  // Reponse from BE while POST a run for creating New UI-Run
  const API_UI_CREATED_NEW_RUN_DETAILS: ApiRunDetail = {
    pipeline_runtime: {
      workflow_manifest: '',
    },
    run: {
      created_at: new Date('2021-05-17T20:58:23.000Z'),
      description: 'V2 xgboost',
      finished_at: new Date('2021-05-18T21:01:23.000Z'),
      id: TEST_RUN_ID,
      name: 'Run of v2-xgboost-ilbo',
      pipeline_spec: {
        pipeline_manifest: v2YamlTemplateString,
        runtime_config: { parameters: { intParam: 123 } },
      },
      resource_references: [
        {
          key: {
            id: '275ea11d-ac63-4ce3-bc33-ec81981ed56b',
            type: ApiResourceType.EXPERIMENT,
          },
          relationship: ApiRelationship.OWNER,
        },
        {
          key: {
            id: TEST_PIPELINE_VERSION_ID,
            type: ApiResourceType.PIPELINEVERSION,
          },
          relationship: ApiRelationship.CREATOR,
        },
      ],
      scheduled_at: new Date('2021-05-17T20:58:23.000Z'),
      status: 'Succeeded',
    },
  };

  // Reponse from BE while POST a run for cloning UI-Run
  const API_UI_CREATED_CLONING_RUN_DETAILS: ApiRunDetail = {
    pipeline_runtime: {
      workflow_manifest: '',
    },
    run: {
      created_at: new Date('2022-08-12T20:58:23.000Z'),
      description: 'V2 xgboost',
      finished_at: new Date('2022-08-12T21:01:23.000Z'),
      id: 'test-clone-ui-run-id',
      name: 'Clone of Run of v2-xgboost-ilbo',
      pipeline_spec: {
        pipeline_manifest: v2YamlTemplateString,
        runtime_config: { parameters: { intParam: 123 } },
      },
      resource_references: [
        {
          key: {
            id: '275ea11d-ac63-4ce3-bc33-ec81981ed56b',
            type: ApiResourceType.EXPERIMENT,
          },
          relationship: ApiRelationship.OWNER,
        },
        {
          key: {
            id: TEST_PIPELINE_VERSION_ID,
            type: ApiResourceType.PIPELINEVERSION,
          },
          relationship: ApiRelationship.CREATOR,
        },
      ],
      scheduled_at: new Date('2022-08-12T20:58:23.000Z'),
      status: 'Succeeded',
    },
  };

  // Reponse from BE while SDK POST a new run for Creating run
  const API_SDK_CREATED_NEW_RUN_DETAILS: ApiRunDetail = {
    pipeline_runtime: {
      workflow_manifest: '',
    },
    run: {
      created_at: new Date('2021-05-17T20:58:23.000Z'),
      description: 'V2 xgboost',
      finished_at: new Date('2021-05-17T21:01:23.000Z'),
      id: 'test-clone-sdk-run-id',
      name: 'Run of v2-xgboost-ilbo',
      pipeline_spec: {
        pipeline_manifest: v2YamlTemplateString,
        runtime_config: { parameters: { intParam: 123 } },
      },
      resource_references: [
        {
          key: {
            id: '275ea11d-ac63-4ce3-bc33-ec81981ed56b',
            type: ApiResourceType.EXPERIMENT,
          },
          relationship: ApiRelationship.OWNER,
        },
      ],
      scheduled_at: new Date('2021-05-17T20:58:23.000Z'),
      status: 'Succeeded',
    },
  };

  // Reponse from BE while POST a run for cloning SDK-Run
  const API_SDK_CREATED_CLONING_RUN_DETAILS: ApiRunDetail = {
    pipeline_runtime: {
      workflow_manifest: '',
    },
    run: {
      created_at: new Date('2022-08-12T20:58:23.000Z'),
      description: 'V2 xgboost',
      finished_at: new Date('2022-08-12T21:01:23.000Z'),
      id: 'test-clone-sdk-run-id',
      name: 'Clone of Run of v2-xgboost-ilbo',
      pipeline_spec: {
        pipeline_manifest: v2YamlTemplateString,
        runtime_config: { parameters: { intParam: 123 } },
      },
      resource_references: [
        {
          key: {
            id: '275ea11d-ac63-4ce3-bc33-ec81981ed56b',
            type: ApiResourceType.EXPERIMENT,
          },
          relationship: ApiRelationship.OWNER,
        },
      ],
      scheduled_at: new Date('2022-08-12T20:58:23.000Z'),
      status: 'Succeeded',
    },
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
        search: `?${QUERY_PARAMS.pipelineId}=${TEST_PIPELINE_ID}&${QUERY_PARAMS.pipelineVersionId}=${TEST_PIPELINE_VERSION_ID}`,
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
    getPipelineSpy.mockResolvedValue(TEST_PIPELINE);
    const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipelineVersion');
    getPipelineVersionSpy.mockResolvedValue(TEST_PIPELINE_VERSION);
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
          apiRun={undefined}
          apiPipeline={TEST_PIPELINE}
          apiPipelineVersion={TEST_PIPELINE_VERSION}
          templateString={v2YamlTemplateString}
        />
      </CommonTestWrapper>,
    );

    await screen.findByDisplayValue(TEST_PIPELINE_NAME);
    await screen.findByDisplayValue(TEST_PIPELINE_VERSION_NAME);
    await screen.findByDisplayValue((content, _) =>
      content.startsWith(`Run of ${TEST_PIPELINE_VERSION_NAME}`),
    );
  });

  it('Submit run ', async () => {
    const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
    getPipelineSpy.mockResolvedValue(TEST_PIPELINE);
    const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipelineVersion');
    getPipelineVersionSpy.mockResolvedValue(TEST_PIPELINE_VERSION);
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
          apiRun={undefined}
          apiPipeline={TEST_PIPELINE}
          apiPipelineVersion={TEST_PIPELINE_VERSION}
          templateString={v2YamlTemplateString}
        />
      </CommonTestWrapper>,
    );

    const startButton = await screen.findByText('Start');
    expect(startButton.closest('button')?.disabled).toEqual(false);
  });

  it('allows updating the run name (start a new run)', async () => {
    // TODO(jlyaoyuli): create a new test file for NewRunSwitcher and move the following test to it.
    const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
    getPipelineSpy.mockResolvedValue(TEST_PIPELINE);
    const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipelineVersion');
    getPipelineVersionSpy.mockResolvedValue(TEST_PIPELINE_VERSION);
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
          apiRun={undefined}
          apiPipeline={TEST_PIPELINE}
          apiPipelineVersion={TEST_PIPELINE_VERSION}
          templateString={v2YamlTemplateString}
        />
      </CommonTestWrapper>,
    );

    const runNameInput = await screen.findByDisplayValue((content, _) =>
      content.startsWith(`Run of ${TEST_PIPELINE_VERSION_NAME}`),
    );
    fireEvent.change(runNameInput, { target: { value: 'Run with custom name' } });
    expect(runNameInput.closest('input')?.value).toBe('Run with custom name');
  });

  describe('starting a new run', () => {
    it('disable start button if no run name (start a new run)', async () => {
      const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
      getPipelineSpy.mockResolvedValue(TEST_PIPELINE);
      const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipelineVersion');
      getPipelineVersionSpy.mockResolvedValue(TEST_PIPELINE_VERSION);
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
            apiRun={undefined}
            apiPipeline={TEST_PIPELINE}
            apiPipelineVersion={TEST_PIPELINE_VERSION}
            templateString={v2YamlTemplateString}
          />
        </CommonTestWrapper>,
      );

      const runNameInput = await screen.findByDisplayValue((content, _) =>
        content.startsWith(`Run of ${TEST_PIPELINE_VERSION_NAME}`),
      );
      fireEvent.change(runNameInput, { target: { value: '' } });

      const startButton = await screen.findByText('Start');
      expect(startButton.closest('button')?.disabled).toEqual(true);
      expect(await screen.findByText('Run name can not be empty.'));
    });

    it('submit a new run without parameter (create new run)', async () => {
      const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
      getPipelineSpy.mockResolvedValue(TEST_PIPELINE);
      const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipelineVersion');
      getPipelineVersionSpy.mockResolvedValue(TEST_PIPELINE_VERSION);
      const getPipelineVersionTemplateSpy = jest.spyOn(
        Apis.pipelineServiceApi,
        'getPipelineVersionTemplate',
      );
      getPipelineVersionTemplateSpy.mockImplementation(() =>
        Promise.resolve({ template: v2YamlTemplateString }),
      );
      const createRunSpy = jest.spyOn(Apis.runServiceApi, 'createRun');
      createRunSpy.mockResolvedValue(API_UI_CREATED_NEW_RUN_DETAILS);

      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsNewRun()}
            existingRunId={null}
            apiRun={undefined}
            apiPipeline={TEST_PIPELINE}
            apiPipelineVersion={TEST_PIPELINE_VERSION}
            templateString={v2YamlTemplateString}
          />
        </CommonTestWrapper>,
      );

      const startButton = await screen.findByText('Start');
      fireEvent.click(startButton);

      await waitFor(() => {
        expect(createRunSpy).toHaveBeenCalledWith(
          expect.objectContaining({
            description: '',
            pipeline_spec: {
              pipeline_manifest: undefined,
              runtime_config: { parameters: {}, pipeline_root: undefined },
            },
            resource_references: [
              {
                key: {
                  id: TEST_PIPELINE_VERSION_ID,
                  type: ApiResourceType.PIPELINEVERSION,
                },
                relationship: ApiRelationship.CREATOR,
              },
            ],
            service_account: '',
          }),
        );
      });
    });
  });

  describe('cloning a existing run', () => {
    it('only shows clone run name from original run', () => {
      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsClonedRun()}
            existingRunId='e0115ac1-0479-4194-a22d-01e65e09a32b'
            apiRun={API_UI_CREATED_NEW_RUN_DETAILS}
            apiPipeline={undefined}
            apiPipelineVersion={undefined}
            templateString={v2YamlTemplateString}
          />
        </CommonTestWrapper>,
      );
      screen.findByDisplayValue(`Clone of ${API_UI_CREATED_NEW_RUN_DETAILS.run?.name}`);
    });

    it('submits a run (clone UI-created run)', async () => {
      const createRunSpy = jest.spyOn(Apis.runServiceApi, 'createRun');
      createRunSpy.mockResolvedValue(API_UI_CREATED_CLONING_RUN_DETAILS);

      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsClonedRun()}
            existingRunId={TEST_RUN_ID}
            apiRun={API_UI_CREATED_NEW_RUN_DETAILS}
            apiPipeline={undefined}
            apiPipelineVersion={undefined}
            templateString={v2YamlTemplateString}
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
            name: 'Clone of Run of v2-xgboost-ilbo',
            pipeline_spec: {
              pipeline_manifest: undefined,
              runtime_config: { parameters: { intParam: 123 } },
            },
            resource_references: [
              {
                key: {
                  id: '275ea11d-ac63-4ce3-bc33-ec81981ed56b',
                  type: ApiResourceType.EXPERIMENT,
                },
                relationship: ApiRelationship.OWNER,
              },
              {
                key: {
                  id: TEST_PIPELINE_VERSION_ID,
                  type: ApiResourceType.PIPELINEVERSION,
                },
                relationship: ApiRelationship.CREATOR,
              },
            ],
            service_account: '',
          }),
        );
      });
    });

    it('submits a run (clone SDK-created run)', async () => {
      const createRunSpy = jest.spyOn(Apis.runServiceApi, 'createRun');
      createRunSpy.mockResolvedValue(API_SDK_CREATED_CLONING_RUN_DETAILS);

      render(
        <CommonTestWrapper>
          <NewRunV2
            {...generatePropsClonedRun()}
            existingRunId={TEST_RUN_ID}
            apiRun={API_SDK_CREATED_NEW_RUN_DETAILS}
            apiPipeline={undefined}
            apiPipelineVersion={undefined}
            templateString={v2YamlTemplateString}
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
            name: 'Clone of Run of v2-xgboost-ilbo',
            pipeline_spec: {
              pipeline_manifest: v2YamlTemplateString,
              runtime_config: { parameters: { intParam: 123 } },
            },
            resource_references: [
              {
                key: {
                  id: '275ea11d-ac63-4ce3-bc33-ec81981ed56b',
                  type: ApiResourceType.EXPERIMENT,
                },
                relationship: ApiRelationship.OWNER,
              },
            ],
            service_account: '',
          }),
        );
      });
    });
  });
});
