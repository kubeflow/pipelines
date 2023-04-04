/*
 * Copyright 2018 The Kubeflow Authors
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

import { render, waitFor } from '@testing-library/react';
import { graphlib } from 'dagre';
import { ReactWrapper, shallow, ShallowWrapper } from 'enzyme';
import * as React from 'react';
import { ApiExperiment } from '../apis/experiment';
import { ApiPipeline, ApiPipelineVersion } from '../apis/pipeline';
import { ApiResourceType, ApiRunDetail } from '../apis/run';
import { QUERY_PARAMS, RoutePage, RouteParams } from '../components/Router';
import { Apis } from '../lib/Apis';
import { ButtonKeys } from '../lib/Buttons';
import * as StaticGraphParser from '../lib/StaticGraphParser';
import TestUtils from '../TestUtils';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';
import * as features from 'src/features';
import { PageProps } from './Page';
import PipelineDetails from './PipelineDetails';
import { ApiJob } from '../apis/job';
import { V2beta1Run } from 'src/apisv2beta1/run';
import { V2beta1RecurringRun } from 'src/apisv2beta1/recurringrun';

describe('PipelineDetails', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
  const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipelineVersion');
  const listPipelineVersionsSpy = jest.spyOn(Apis.pipelineServiceApi, 'listPipelineVersions');
  const getV1RunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
  const getV1RecurringRunSpy = jest.spyOn(Apis.jobServiceApi, 'getJob');
  const getV2RunSpy = jest.spyOn(Apis.runServiceApiV2, 'getRun');
  const getV2RecurringRunSpy = jest.spyOn(Apis.recurringRunServiceApi, 'getRecurringRun');
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
  const deletePipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'deletePipelineVersion');
  const getPipelineVersionTemplateSpy = jest.spyOn(
    Apis.pipelineServiceApi,
    'getPipelineVersionTemplate',
  );
  const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');

  let tree: ShallowWrapper | ReactWrapper;
  let testPipeline: ApiPipeline = {};
  let testPipelineVersion: ApiPipelineVersion = {};
  let testV1Run: ApiRunDetail = {};
  let testV1RecurringRun: ApiJob = {};
  let testV2Run: V2beta1Run = {};
  let testV2RecurringRun: V2beta1RecurringRun = {};

  function generateProps(fromRunSpec = false, fromRecurringRunSpec = false): PageProps {
    let params = {};
    // If no fromXXX parameter is provided, it means KFP UI expects to
    // show Pipeline detail with pipeline version ID
    if (!fromRunSpec && !fromRecurringRunSpec) {
      params = {
        [RouteParams.pipelineId]: testPipeline.id,
        [RouteParams.pipelineVersionId]:
          (testPipeline.default_version && testPipeline.default_version!.id) || '',
      };
    }

    let search = '';
    if (fromRunSpec) {
      search = `?${QUERY_PARAMS.fromRunId}=test-run-id`;
    } else if (fromRecurringRunSpec) {
      search = `?${QUERY_PARAMS.fromRecurringRunId}=test-recurring-run-id`;
    }

    const match = {
      isExact: true,
      params: params,
      path: '',
      url: '',
    };
    const location = { search } as any;
    const pageProps = TestUtils.generatePageProps(
      PipelineDetails,
      location,
      match,
      historyPushSpy,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
    );
    return pageProps;
  }

  beforeAll(() => jest.spyOn(console, 'error').mockImplementation());

  beforeEach(() => {
    jest.clearAllMocks();

    testPipeline = {
      created_at: new Date(2018, 8, 5, 4, 3, 2),
      description: 'test pipeline description',
      id: 'test-pipeline-id',
      name: 'test pipeline',
      parameters: [{ name: 'param1', value: 'value1' }],
      default_version: {
        id: 'test-pipeline-version-id',
        name: 'test-pipeline-version',
      },
    };

    testPipelineVersion = {
      id: 'test-pipeline-version-id',
      name: 'test-pipeline-version',
    };

    testV1Run = {
      run: {
        id: 'test-run-id',
        name: 'test run',
        pipeline_spec: {
          pipeline_id: 'run-pipeline-id',
        },
      },
    };

    testV1RecurringRun = {
      id: 'test-recurring-run-id',
      name: 'test recurring run',
      pipeline_spec: {
        pipeline_id: 'run-pipeline-id',
      },
    };

    testV2Run = {
      run_id: 'test-run-id',
      display_name: 'test run',
      pipeline_version_reference: {},
    };

    testV2RecurringRun = {
      recurring_run_id: 'test-recurring-run-id',
      display_name: 'test recurring run',
      pipeline_version_reference: {},
    };

    getPipelineSpy.mockImplementation(() => Promise.resolve(testPipeline));
    getPipelineVersionSpy.mockImplementation(() => Promise.resolve(testPipelineVersion));
    listPipelineVersionsSpy.mockImplementation(() =>
      Promise.resolve({ versions: [testPipelineVersion] }),
    );
    getV1RunSpy.mockImplementation(() => Promise.resolve(testV1Run));
    getV1RecurringRunSpy.mockImplementation(() => Promise.resolve(testV1RecurringRun));
    getV2RunSpy.mockImplementation(() => Promise.resolve(testV2Run));
    getV2RecurringRunSpy.mockImplementation(() => Promise.resolve(testV2RecurringRun));
    getExperimentSpy.mockImplementation(() =>
      Promise.resolve({ id: 'test-experiment-id', name: 'test experiment' } as ApiExperiment),
    );
    // getTemplateSpy.mockImplementation(() => Promise.resolve({ template: 'test template' }));
    getPipelineVersionTemplateSpy.mockImplementation(() =>
      Promise.resolve({ template: 'test template' }),
    );
    createGraphSpy.mockImplementation(() => new graphlib.Graph());
  });

  afterEach(async () => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    await tree.unmount();
    jest.resetAllMocks();
  });

  it('shows pipeline name in page name, and breadcrumb to go back to pipelines', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        breadcrumbs: [{ displayName: 'Pipelines', href: RoutePage.PIPELINES }],
        pageTitle: testPipeline.name + ' (' + testPipelineVersion.name + ')',
      }),
    );
  });

  it(
    'shows all runs breadcrumbs, and "Pipeline details" as page title when the pipeline ' +
      'comes from a run spec that does not have an experiment',
    async () => {
      tree = shallow(<PipelineDetails {...generateProps(true)} />);
      await getV1RunSpy;
      await getV2RunSpy;
      await createGraphSpy;
      await getPipelineVersionTemplateSpy;
      await TestUtils.flushPromises();
      expect(updateToolbarSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          breadcrumbs: [
            { displayName: 'All runs', href: RoutePage.RUNS },
            {
              displayName: testV1Run.run!.name,
              href: RoutePage.RUN_DETAILS.replace(':' + RouteParams.runId, testV1Run.run!.id!),
            },
          ],
          pageTitle: 'Pipeline details',
        }),
      );
    },
  );

  it(
    'shows all runs breadcrumbs, and "Pipeline details" as page title when the pipeline ' +
      'comes from a recurring run spec that does not have an experiment',
    async () => {
      tree = shallow(<PipelineDetails {...generateProps(false, true)} />);
      await getV1RecurringRunSpy;
      await getV2RecurringRunSpy;
      await getPipelineVersionTemplateSpy;
      await TestUtils.flushPromises();
      expect(updateToolbarSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          breadcrumbs: [
            { displayName: 'All recurring runs', href: RoutePage.RECURRING_RUNS },
            {
              displayName: testV1RecurringRun.name,
              href: RoutePage.RECURRING_RUN_DETAILS.replace(
                ':' + RouteParams.recurringRunId,
                testV1RecurringRun.id!,
              ),
            },
          ],
          pageTitle: 'Pipeline details',
        }),
      );
    },
  );

  it(
    'shows all runs breadcrumbs, and "Pipeline details" as page title when the pipeline ' +
      'comes from a run spec that has an experiment',
    async () => {
      testV2Run.experiment_id = 'test-experiment-id';
      tree = shallow(<PipelineDetails {...generateProps(true)} />);
      await getV1RunSpy;
      await getV2RunSpy;
      await getExperimentSpy;
      await getPipelineVersionTemplateSpy;
      await TestUtils.flushPromises();
      expect(updateToolbarSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          breadcrumbs: [
            { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
            {
              displayName: 'test experiment',
              href: RoutePage.EXPERIMENT_DETAILS.replace(
                ':' + RouteParams.experimentId,
                'test-experiment-id',
              ),
            },
            {
              displayName: testV2Run.display_name,
              href: RoutePage.RUN_DETAILS.replace(':' + RouteParams.runId, testV2Run.run_id!),
            },
          ],
          pageTitle: 'Pipeline details',
        }),
      );
    },
  );

  it(
    'shows all runs breadcrumbs, and "Pipeline details" as page title when the pipeline ' +
      'comes from a recurring run spec that has an experiment',
    async () => {
      testV2RecurringRun.experiment_id = 'test-experiment-id';
      tree = shallow(<PipelineDetails {...generateProps(false, true)} />);
      await getV1RecurringRunSpy;
      await getV2RecurringRunSpy;
      await getExperimentSpy;
      await getPipelineVersionTemplateSpy;
      await TestUtils.flushPromises();
      expect(updateToolbarSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          breadcrumbs: [
            { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
            {
              displayName: 'test experiment',
              href: RoutePage.EXPERIMENT_DETAILS.replace(
                ':' + RouteParams.experimentId,
                'test-experiment-id',
              ),
            },
            {
              displayName: testV1RecurringRun.name,
              href: RoutePage.RECURRING_RUN_DETAILS.replace(
                ':' + RouteParams.recurringRunId,
                testV1RecurringRun.id!,
              ),
            },
          ],
          pageTitle: 'Pipeline details',
        }),
      );
    },
  );

  it('parses the workflow source in embedded pipeline spec as JSON and then converts it to YAML (v1)', async () => {
    testV1Run.run!.pipeline_spec = {
      pipeline_id: 'run-pipeline-id',
      workflow_manifest: '{"spec": {"arguments": {"parameters": [{"name": "output"}]}}}',
    };

    tree = shallow(<PipelineDetails {...generateProps(true)} />);
    await getV1RunSpy;
    await getV2RunSpy;
    await TestUtils.flushPromises();

    expect(tree.state('templateString')).toBe(
      'spec:\n  arguments:\n    parameters:\n      - name: output\n',
    );
  });

  it('directly use pipeline_manifest dumped from pipeline_spec in run as template string (v2)', async () => {
    jest.spyOn(features, 'isFeatureEnabled').mockReturnValue(true);
    testV2Run.pipeline_spec = { spec: { arguments: { parameters: [{ name: 'output' }] } } };

    tree = shallow(<PipelineDetails {...generateProps(true)} />);
    await getV1RunSpy;
    await getV2RunSpy;
    await TestUtils.flushPromises();

    expect(tree.state('templateString')).toBe(
      'spec:\n  arguments:\n    parameters:\n      - name: output\n',
    );
  });

  it('directly use pipeline_manifest dumped from pipeline_spec in recurring run as template string (v2)', async () => {
    jest.spyOn(features, 'isFeatureEnabled').mockReturnValue(true);
    testV2RecurringRun.pipeline_spec = {
      spec: { arguments: { parameters: [{ name: 'output' }] } },
    };

    tree = shallow(<PipelineDetails {...generateProps(false, true)} />);
    await getV1RecurringRunSpy;
    await getV2RecurringRunSpy;
    await TestUtils.flushPromises();

    expect(tree.state('templateString')).toBe(
      'spec:\n  arguments:\n    parameters:\n      - name: output\n',
    );
  });

  it('use pipeline_version_id in run to get pipeline template string (v2)', async () => {
    jest.spyOn(features, 'isFeatureEnabled').mockReturnValue(true);
    testV2Run.pipeline_version_reference.pipeline_version_id = 'test-pipeline-version-id';

    tree = shallow(<PipelineDetails {...generateProps(true)} />);
    await getV1RunSpy;
    await getV2RunSpy;
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();

    expect(tree.state('templateString')).toBe('test template');
  });

  it('use pipeline_version_id in recurring run to get pipeline template string (v2)', async () => {
    jest.spyOn(features, 'isFeatureEnabled').mockReturnValue(true);
    testV2RecurringRun.pipeline_version_reference.pipeline_version_id = 'test-pipeline-version-id';

    tree = shallow(<PipelineDetails {...generateProps(false, true)} />);
    await getV1RecurringRunSpy;
    await getV2RecurringRunSpy;
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();

    expect(tree.state('templateString')).toBe('test template');
  });

  it('shows load error banner when failing to parse the workflow source in embedded pipeline spec', async () => {
    testV1Run.run!.pipeline_spec = {
      pipeline_id: 'run-pipeline-id',
      workflow_manifest: 'not valid JSON',
    };

    render(<PipelineDetails {...generateProps(true)} />);
    await waitFor(() => {
      expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          additionalInfo: 'Unexpected token o in JSON at position 1',
          message: `Failed to parse pipeline spec from run with ID: ${
            testV1Run.run!.id
          }. Click Details for more information.`,
          mode: 'error',
        }),
      );
    });
  });

  it('shows load error banner when failing to get run details, when loading from run spec', async () => {
    TestUtils.makeErrorResponseOnce(getV1RunSpy, 'woops');
    tree = shallow(<PipelineDetails {...generateProps(true)} />);
    await getPipelineSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops',
        message: 'Cannot retrieve run details. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('shows load error banner when failing to get experiment details, when loading from run spec', async () => {
    testV2Run.experiment_id = 'test-experiment-id';
    TestUtils.makeErrorResponse(getExperimentSpy, 'woops');
    tree = shallow(<PipelineDetails {...generateProps(true)} />);
    await getPipelineSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops',
        message: 'Cannot retrieve run details. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('uses an empty string and does not show error when getTemplate response is empty', async () => {
    getPipelineVersionTemplateSpy.mockImplementationOnce(() => Promise.resolve({}));

    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineSpy;
    await TestUtils.flushPromises();

    // No errors
    expect(updateBannerSpy).toHaveBeenCalledTimes(1); // Once to clear banner
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({}));

    expect(tree.state('templateString')).toBe('');
  });

  it('shows load error banner when failing to get pipeline', async () => {
    TestUtils.makeErrorResponseOnce(getPipelineSpy, 'woops');
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops',
        message: 'Cannot retrieve pipeline details. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('shows load error banner when failing to get pipeline template', async () => {
    TestUtils.makeErrorResponseOnce(getPipelineVersionTemplateSpy, 'woops');
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops',
        message: 'Cannot retrieve pipeline template. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('shows no graph error banner when failing to parse graph', async () => {
    getPipelineVersionTemplateSpy.mockResolvedValue({
      template: `    
      apiVersion: argoproj.io/v1alpha1
      kind: Workflow
      metadata:
        generateName: entry-point-test-
      `,
    });
    TestUtils.makeErrorResponse(createGraphSpy, 'bad graph');
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'bad graph',
        message: 'Error: failed to generate Pipeline graph. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('clears the error banner when refreshing the page', async () => {
    TestUtils.makeErrorResponseOnce(getPipelineVersionTemplateSpy, 'woops');
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops',
        message: 'Cannot retrieve pipeline template. Click Details for more information.',
        mode: 'error',
      }),
    );

    (tree.instance() as PipelineDetails).refresh();

    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('has a new experiment button if it has a pipeline reference', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const newExperimentBtn = instance.getInitialToolbarState().actions[ButtonKeys.NEW_EXPERIMENT];
    expect(newExperimentBtn).toBeDefined();
  });

  it("has 'create run' toolbar button if viewing an embedded pipeline", async () => {
    tree = shallow(<PipelineDetails {...generateProps(true)} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    /* create run and create pipeline version, so 2 */
    expect(Object.keys(instance.getInitialToolbarState().actions)).toHaveLength(2);
    const newRunBtn = instance.getInitialToolbarState().actions[
      (ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION, ButtonKeys.NEW_PIPELINE_VERSION)
    ];
    expect(newRunBtn).toBeDefined();
  });

  it('clicking new run button when viewing embedded pipeline navigates to the new run page with run ID', async () => {
    tree = shallow(<PipelineDetails {...generateProps(true)} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const newRunBtn = instance.getInitialToolbarState().actions[
      ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION
    ];
    newRunBtn!.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_RUN + `?${QUERY_PARAMS.fromRunId}=${testV1Run.run!.id}`,
    );
  });

  it("has 'create run' toolbar button if not viewing an embedded pipeline", async () => {
    tree = shallow(<PipelineDetails {...generateProps(false)} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    /* create run, create pipeline version, create experiment and delete run, so 4 */
    expect(Object.keys(instance.getInitialToolbarState().actions)).toHaveLength(4);
    const newRunBtn = instance.getInitialToolbarState().actions[
      ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION
    ];
    expect(newRunBtn).toBeDefined();
  });

  it('clicking new run button navigates to the new run page', async () => {
    tree = shallow(<PipelineDetails {...generateProps(false)} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const newRunFromPipelineVersionBtn = instance.getInitialToolbarState().actions[
      ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION
    ];
    newRunFromPipelineVersionBtn.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_RUN +
        `?${QUERY_PARAMS.pipelineId}=${testPipeline.id}&${
          QUERY_PARAMS.pipelineVersionId
        }=${testPipeline.default_version!.id!}`,
    );
  });

  it('clicking new run button when viewing half-loaded page navigates to the new run page with pipeline ID and version ID', async () => {
    tree = shallow(<PipelineDetails {...generateProps(false)} />);
    // Intentionally don't wait until all network requests finish.
    const instance = tree.instance() as PipelineDetails;
    const newRunFromPipelineVersionBtn = instance.getInitialToolbarState().actions[
      ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION
    ];
    newRunFromPipelineVersionBtn.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_RUN +
        `?${QUERY_PARAMS.pipelineId}=${testPipeline.id}&${
          QUERY_PARAMS.pipelineVersionId
        }=${testPipeline.default_version!.id!}`,
    );
  });

  it('clicking new experiment button navigates to new experiment page', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const newExperimentBtn = instance.getInitialToolbarState().actions[ButtonKeys.NEW_EXPERIMENT];
    await newExperimentBtn.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_EXPERIMENT + `?${QUERY_PARAMS.pipelineId}=${testPipeline.id}`,
    );
  });

  it('clicking new experiment button when viewing half-loaded page navigates to the new experiment page with the pipeline ID', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    // Intentionally don't wait until all network requests finish.
    const instance = tree.instance() as PipelineDetails;
    const newExperimentBtn = instance.getInitialToolbarState().actions[ButtonKeys.NEW_EXPERIMENT];
    await newExperimentBtn.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_EXPERIMENT + `?${QUERY_PARAMS.pipelineId}=${testPipeline.id}`,
    );
  });

  it('has a delete button and it is enabled for pipeline version deletion', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const deleteBtn = instance.getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    expect(deleteBtn).toBeDefined();
    expect(deleteBtn.disabled).toBeFalsy();
  });

  it('has a delete button, and it is disabled because no version is selected', async () => {
    let pageProps = generateProps();
    pageProps.match.params = {
      [RouteParams.pipelineId]: testPipeline.id,
    };
    tree = shallow(<PipelineDetails {...pageProps} />);

    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const deleteBtn = instance.getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    expect(deleteBtn).toBeDefined();
    expect(deleteBtn.disabled).toBeTruthy();
  });

  it('shows delete confirmation dialog when delete button is clicked', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    const deleteBtn = (tree.instance() as PipelineDetails).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        title: 'Delete this pipeline version?',
      }),
    );
  });

  it('does not call delete API for selected pipeline when delete dialog is canceled', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    const deleteBtn = (tree.instance() as PipelineDetails).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const cancelBtn = call.buttons.find((b: any) => b.text === 'Cancel');
    await cancelBtn.onClick();
    expect(deletePipelineVersionSpy).not.toHaveBeenCalled();
  });

  it('calls delete API when delete dialog is confirmed', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    const deleteBtn = (tree.instance() as PipelineDetails).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    expect(deletePipelineVersionSpy).toHaveBeenCalledTimes(1);
    expect(deletePipelineVersionSpy).toHaveBeenLastCalledWith(testPipeline.default_version!.id!);
  });

  it('calls delete API when delete dialog is confirmed and page is half-loaded', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    // Intentionally don't wait until all network requests finish.
    const deleteBtn = (tree.instance() as PipelineDetails).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    expect(deletePipelineVersionSpy).toHaveBeenCalledTimes(1);
    expect(deletePipelineVersionSpy).toHaveBeenLastCalledWith(testPipeline.default_version!.id);
  });

  it('shows error dialog if deletion fails', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    TestUtils.makeErrorResponseOnce(deletePipelineVersionSpy, 'woops');
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    const deleteBtn = (tree.instance() as PipelineDetails).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    expect(updateDialogSpy).toHaveBeenCalledTimes(2); // Delete dialog + error dialog
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content: 'Failed to delete pipeline version: test-pipeline-version-id with error: "woops"',
        title: 'Failed to delete pipeline version',
      }),
    );
  });

  it('shows success snackbar if deletion succeeds', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    const deleteBtn = (tree.instance() as PipelineDetails).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    expect(updateSnackbarSpy).toHaveBeenCalledTimes(1);
    expect(updateSnackbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        message: 'Delete succeeded for this pipeline version',
        open: true,
      }),
    );
  });
});
