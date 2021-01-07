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

import * as React from 'react';
import { NewRun } from './NewRun';
import TestUtils, { defaultToolbarProps } from '../TestUtils';
import { shallow, ShallowWrapper, ReactWrapper, mount, render } from 'enzyme';
import { PageProps } from './Page';
import { Apis } from '../lib/Apis';
import { RoutePage, RouteParams, QUERY_PARAMS } from '../components/Router';
import { ApiExperiment, ApiListExperimentsResponse } from '../apis/experiment';
import { ApiPipeline, ApiPipelineVersion } from '../apis/pipeline';
import { ApiResourceType, ApiRunDetail, ApiParameter, ApiRelationship } from '../apis/run';
import { MemoryRouter } from 'react-router';
import { logger } from '../lib/Utils';
import { ApiFilter, PredicateOp } from '../apis/filter';
import { ExperimentStorageState } from '../apis/experiment';
import { ApiJob } from 'src/apis/job';
import { TFunction } from 'i18next';

jest.mock('react-i18next', () => ({
  // this mock makes sure any components using the translate hook can use it without a warning being shown
  withTranslation: () => (component: React.ComponentClass) => {
    component.defaultProps = { ...component.defaultProps, t: (key: string) => key };
    return component;
  },
}));

class TestNewRun extends NewRun {
  public _experimentSelectorClosed = super._experimentSelectorClosed;
  public _pipelineSelectorClosed = super._pipelineSelectorClosed;
  public _pipelineVersionSelectorClosed = super._pipelineVersionSelectorClosed;
  public _updateRecurringRunState = super._updateRecurringRunState;
  public _handleParamChange = super._handleParamChange;
}

function fillRequiredFields(instance: TestNewRun) {
  instance.handleChange('runName')({
    target: { value: 'test run name' },
  });
}

describe('NewRun', () => {
  let tree: ReactWrapper | ShallowWrapper;
  const consoleErrorSpy = jest.spyOn(console, 'error');
  const startJobSpy = jest.spyOn(Apis.jobServiceApi, 'createJob');
  const startRunSpy = jest.spyOn(Apis.runServiceApi, 'createRun');
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
  const listExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'listExperiment');
  const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
  const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipelineVersion');
  const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
  const getJobSpy = jest.spyOn(Apis.jobServiceApi, 'getJob');
  const loggerErrorSpy = jest.spyOn(logger, 'error');
  const historyPushSpy = jest.fn();
  const historyReplaceSpy = jest.fn();
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();

  let MOCK_EXPERIMENT = newMockExperiment();
  let MOCK_PIPELINE = newMockPipeline();
  let MOCK_PIPELINE_VERSION = newMockPipelineVersion();
  let MOCK_RUN_DETAIL = newMockRunDetail();
  let MOCK_RUN_WITH_EMBEDDED_PIPELINE = newMockRunWithEmbeddedPipeline();
  let t: TFunction = (key: string) => key;

  function muteErrors() {
    updateBannerSpy.mockImplementation(() => null);
    loggerErrorSpy.mockImplementation(() => null);
  }

  function newMockExperiment(): ApiExperiment {
    return {
      description: 'mock experiment description',
      id: 'some-mock-experiment-id',
      name: 'some mock experiment name',
    };
  }

  function newMockPipeline(): ApiPipeline {
    return {
      id: 'original-run-pipeline-id',
      name: 'original mock pipeline name',
      parameters: [],
      default_version: {
        id: 'original-run-pipeline-version-id',
        name: 'original mock pipeline version name',
      },
    };
  }

  function newMockPipelineWithParameters(): ApiPipeline {
    return {
      id: 'unoriginal-run-pipeline-id',
      name: 'unoriginal mock pipeline name',
      parameters: [
        {
          name: 'set value',
          value: 'abc',
        },
        {
          name: 'empty value',
          value: '',
        },
      ],
      default_version: {
        id: 'original-run-pipeline-version-id',
        name: 'original mock pipeline version name',
      },
    };
  }

  function newMockPipelineVersion(): ApiPipelineVersion {
    return {
      id: 'original-run-pipeline-version-id',
      name: 'original mock pipeline version name',
    };
  }

  function newMockRunDetail(): ApiRunDetail {
    return {
      pipeline_runtime: {
        workflow_manifest: '{}',
      },
      run: {
        id: 'some-mock-run-id',
        name: 'some mock run name',
        service_account: 'pipeline-runner',
        pipeline_spec: {
          pipeline_id: 'original-run-pipeline-id',
          workflow_manifest: '{}',
        },
      },
    };
  }

  function newMockJob(): ApiJob {
    return {
      id: 'job-id1',
      name: 'some mock job name',
      service_account: 'pipeline-runner',
      pipeline_spec: {
        pipeline_id: 'original-run-pipeline-id',
        workflow_manifest: '{}',
      },
      trigger: {
        periodic_schedule: {
          interval_second: '60',
        },
      },
    };
  }

  function newMockRunWithEmbeddedPipeline(): ApiRunDetail {
    const runDetail = newMockRunDetail();
    delete runDetail.run!.pipeline_spec!.pipeline_id;
    runDetail.run!.pipeline_spec!.workflow_manifest =
      '{"metadata": {"name": "embedded"}, "parameters": []}';
    return runDetail;
  }
  function generateProps(search?: string): any {
    return {
      history: { push: historyPushSpy, replace: historyReplaceSpy } as any,
      location: {
        pathname: RoutePage.NEW_RUN,
        // TODO: this should be removed once experiments are no longer required to reach this page.
        search: `` as any,
        t,
      } as any,
      match: '' as any,
      toolbarProps: defaultToolbarProps(),
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
      t,
    };
  }

  beforeEach(() => {
    jest.resetAllMocks();

    // TODO: decide this
    // consoleErrorSpy.mockImplementation(() => null);
    startRunSpy.mockImplementation(() => ({ id: 'new-run-id' }));
    getExperimentSpy.mockImplementation(() => MOCK_EXPERIMENT);
    listExperimentSpy.mockImplementation(() => {
      const response: ApiListExperimentsResponse = {
        experiments: [MOCK_EXPERIMENT],
        total_size: 1,
      };
      return response;
    });
    getPipelineSpy.mockImplementation(() => MOCK_PIPELINE);
    getPipelineVersionSpy.mockImplementation(() => MOCK_PIPELINE_VERSION);
    getRunSpy.mockImplementation(() => MOCK_RUN_DETAIL);
    updateBannerSpy.mockImplementation((opts: any) => {
      if (opts.mode) {
        // it's error or warning
        throw new Error('There was an error loading the page: ' + JSON.stringify(opts));
      }
    });

    MOCK_EXPERIMENT = newMockExperiment();
    MOCK_PIPELINE = newMockPipeline();
    MOCK_RUN_DETAIL = newMockRunDetail();
    MOCK_RUN_WITH_EMBEDDED_PIPELINE = newMockRunWithEmbeddedPipeline();
  });

  afterEach(async () => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    await tree.unmount();
  });

  it('renders the new run page', async () => {
    tree = shallow(<TestNewRun t={(key: any) => key} {...generateProps()} />);
    await TestUtils.flushPromises();

    expect(tree).toMatchSnapshot();
  });

  it('does not include any action buttons in the toolbar', async () => {
    const props = generateProps();
    // Clear the experiment ID from the query params, as it used at some point to update the
    // breadcrumb, and we cover that in a later test.
    props.location.search = '';

    tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
    await TestUtils.flushPromises();

    expect(updateToolbarSpy).toHaveBeenLastCalledWith({
      actions: {},
      breadcrumbs: [{ displayName: 'common:experiments', href: RoutePage.EXPERIMENTS }],
      pageTitle: 'common:start aRun',
    });
  });

  it('clears the banner when refresh is called', async () => {
    tree = shallow(<TestNewRun t={(key: any) => key} {...(generateProps() as any)} />);
    expect(updateBannerSpy).toHaveBeenCalledTimes(1);
    (tree.instance() as TestNewRun).refresh();
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({ t });
  });

  it('clears the banner when load is called', async () => {
    tree = shallow(<TestNewRun t={(key: any) => key} {...(generateProps() as any)} />);
    expect(updateBannerSpy).toHaveBeenCalledTimes(1);
    (tree.instance() as TestNewRun).load();
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({ t });
  });

  it('allows updating the run name', async () => {
    tree = shallow(<TestNewRun t={(key: any) => key} {...(generateProps() as any)} />);
    await TestUtils.flushPromises();

    (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'run name' } });

    expect(tree.state()).toHaveProperty('runName', 'run name');
  });

  it('reports validation error when missing the run name', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}&${
      QUERY_PARAMS.pipelineVersionId
    }=${MOCK_PIPELINE.default_version!.id}`;

    tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
    await TestUtils.flushPromises();

    (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: null } });

    expect(tree.state()).toHaveProperty('errorMessage', 'runNameRequired');
  });

  it('allows updating the run description', async () => {
    tree = shallow(<TestNewRun t={(key: any) => key} {...(generateProps() as any)} />);
    await TestUtils.flushPromises();
    (tree.instance() as TestNewRun).handleChange('description')({
      target: { value: 'run description' },
    });

    expect(tree.state()).toHaveProperty('description', 'run description');
  });

  it('changes title and form if the new run will recur, based on the radio buttons', async () => {
    // Default props do not include isRecurring in query params
    tree = shallow(<TestNewRun t={(key: any) => key} {...(generateProps() as any)} />);
    await TestUtils.flushPromises();

    (tree.instance() as TestNewRun)._updateRecurringRunState(true);
    await TestUtils.flushPromises();

    expect(tree).toMatchSnapshot();
  });

  it('changes title and form to default state if the new run is a one-off, based on the radio buttons', async () => {
    // Modify props to set page to recurring run form
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.isRecurring}=1`;
    tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
    await TestUtils.flushPromises();

    (tree.instance() as TestNewRun)._updateRecurringRunState(false);
    await TestUtils.flushPromises();

    expect(tree).toMatchSnapshot();
  });

  it('exits to the AllRuns page if there is no associated experiment', async () => {
    const props = generateProps();
    // Clear query params which might otherwise include an experiment ID.
    props.location.search = '';

    tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
    await TestUtils.flushPromises();
    tree.find('#exitNewRunPageBtn').simulate('click');

    expect(historyPushSpy).toHaveBeenCalledWith(RoutePage.RUNS);
  });

  it('fetches the associated experiment if one is present in the query params', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

    tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
    await TestUtils.flushPromises();

    expect(getExperimentSpy).toHaveBeenLastCalledWith(MOCK_EXPERIMENT.id);
  });

  it("updates the run's state with the associated experiment if one is present in the query params", async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

    tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
    await TestUtils.flushPromises();

    expect(tree.state()).toHaveProperty('experiment', MOCK_EXPERIMENT);
    expect(tree.state()).toHaveProperty('experimentName', MOCK_EXPERIMENT.name);
    expect(tree).toMatchSnapshot();
  });

  it('updates the breadcrumb with the associated experiment if one is present in the query params', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

    tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
    await TestUtils.flushPromises();

    expect(updateToolbarSpy).toHaveBeenLastCalledWith({
      actions: {},
      breadcrumbs: [
        { displayName: 'common:experiments', href: RoutePage.EXPERIMENTS },
        {
          displayName: MOCK_EXPERIMENT.name,
          href: RoutePage.EXPERIMENT_DETAILS.replace(
            ':' + RouteParams.experimentId,
            MOCK_EXPERIMENT.id!,
          ),
        },
      ],
      pageTitle: 'common:start aRun',
    });
  });

  it("exits to the associated experiment's details page if one is present in the query params", async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

    tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
    await TestUtils.flushPromises();
    tree.find('#exitNewRunPageBtn').simulate('click');

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, MOCK_EXPERIMENT.id!),
    );
  });

  it("changes the exit button's text if query params indicate this is the first run of an experiment", async () => {
    const props = generateProps();
    props.location.search =
      `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}` +
      `&${QUERY_PARAMS.firstRunInExperiment}=1`;

    tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
    await TestUtils.flushPromises();

    expect(tree).toMatchSnapshot();
  });

  it('shows a page error if getExperiment fails', async () => {
    muteErrors();

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

    TestUtils.makeErrorResponseOnce(getExperimentSpy, 'test error message');

    tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'test error message',
        message: `errorRetrieveAssocExperiment: some-mock-experiment-id. common:clickDetails`,
        mode: 'error',
      }),
    );
  });

  it('fetches the associated pipeline if one is present in the query params', async () => {
    const randomSpy = jest.spyOn(Math, 'random');
    randomSpy.mockImplementation(() => 0.5);

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}&${
      QUERY_PARAMS.pipelineVersionId
    }=${MOCK_PIPELINE.default_version!.id}`;

    tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
    await TestUtils.flushPromises();

    expect(tree.state()).toHaveProperty('pipeline', MOCK_PIPELINE);
    expect(tree.state()).toHaveProperty('pipelineName', MOCK_PIPELINE.name);
    expect(tree.state()).toHaveProperty('pipelineVersion', MOCK_PIPELINE_VERSION);
    expect((tree.state() as any).runName).toMatch(/Run of original mock pipeline version name/);
    expect(tree).toMatchSnapshot();

    randomSpy.mockRestore();
  });

  it('shows a page error if getPipeline fails', async () => {
    muteErrors();

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

    TestUtils.makeErrorResponseOnce(getPipelineSpy, 'test error message');

    tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'test error message',
        message: `errorRetrievePipeline: original-run-pipeline-id. common:clickDetails`,
        mode: 'error',
      }),
    );
  });

  it('shows a page error if getPipelineVersion fails', async () => {
    muteErrors();

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}&${QUERY_PARAMS.pipelineVersionId}=${MOCK_PIPELINE_VERSION.id}`;

    TestUtils.makeErrorResponseOnce(getPipelineVersionSpy, 'test error message');

    tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'test error message',
        message: `errorRetrievePipelineVersion: original-run-pipeline-version-id. common:clickDetails`,
        mode: 'error',
      }),
    );
  });

  it('renders a warning message if there are pipeline parameters with empty values', async () => {
    tree = TestUtils.mountWithRouter(
      <TestNewRun t={(key: any) => key} {...(generateProps() as any)} />,
    );
    await TestUtils.flushPromises();

    const pipeline = newMockPipelineWithParameters();
    tree.setState({ parameters: pipeline.parameters });

    // Ensure that at least one of the provided parameters has a missing value.
    expect((pipeline.parameters || []).some(parameter => !parameter.value)).toBe(true);
    expect(tree.find('#missing-parameters-message').exists()).toBe(true);
  });

  it('does not render a warning message if there are no pipeline parameters with empty values', async () => {
    tree = TestUtils.mountWithRouter(
      <TestNewRun t={(key: any) => key} {...(generateProps() as any)} />,
    );
    await TestUtils.flushPromises();

    const pipeline = newMockPipelineWithParameters();
    (pipeline.parameters || []).forEach(parameter => {
      parameter.value = 'I am not set';
    });
    tree.setState({ parameters: pipeline.parameters });

    // Ensure all provided parameters have valid values.
    expect((pipeline.parameters || []).every(parameter => !!parameter.value)).toBe(true);
    expect(tree.find('#missing-parameters-message').exists()).toBe(false);
  });

  describe('choosing a pipeline', () => {
    it("opens up the pipeline selector modal when users clicks 'Choose'", async () => {
      tree = TestUtils.mountWithRouter(
        <TestNewRun t={(key: any) => key} {...(generateProps() as any)} />,
      );
      await TestUtils.flushPromises();

      tree
        .find('#choosePipelineBtn')
        .at(0)
        .simulate('click');
      await TestUtils.flushPromises();
      expect(tree.state('pipelineSelectorOpen')).toBe(true);
    });

    it('closes the pipeline selector modal', async () => {
      tree = TestUtils.mountWithRouter(
        <TestNewRun t={(key: any) => key} {...(generateProps() as any)} />,
      );
      await TestUtils.flushPromises();

      tree
        .find('#choosePipelineBtn')
        .at(0)
        .simulate('click');
      expect(tree.state('pipelineSelectorOpen')).toBe(true);

      tree
        .find('#cancelPipelineSelectionBtn')
        .at(0)
        .simulate('click');
      expect(tree.state('pipelineSelectorOpen')).toBe(false);
    });

    it('sets the pipeline from the selector modal when confirmed', async () => {
      tree = TestUtils.mountWithRouter(
        <TestNewRun t={(key: any) => key} {...(generateProps() as any)} />,
      );
      await TestUtils.flushPromises();

      const oldPipeline = newMockPipeline();
      oldPipeline.id = 'old-pipeline-id';
      oldPipeline.name = 'old-pipeline-name';
      const newPipeline = newMockPipeline();
      newPipeline.id = 'new-pipeline-id';
      newPipeline.name = 'new-pipeline-name';
      getPipelineSpy.mockImplementation(() => newPipeline);
      tree.setState({ pipeline: oldPipeline, pipelineName: oldPipeline.name });

      tree
        .find('#choosePipelineBtn')
        .at(0)
        .simulate('click');
      expect(tree.state('pipelineSelectorOpen')).toBe(true);

      // Simulate selecting pipeline
      tree.setState({ unconfirmedSelectedPipeline: newPipeline });

      // Confirm pipeline selector
      tree
        .find('#usePipelineBtn')
        .at(0)
        .simulate('click');
      await TestUtils.flushPromises();
      expect(tree.state('pipelineSelectorOpen')).toBe(false);

      expect(tree.state('pipeline')).toEqual(newPipeline);
      expect(tree.state('pipelineName')).toEqual(newPipeline.name);
      expect(tree.state('pipelineSelectorOpen')).toBe(false);
      await TestUtils.flushPromises();
    });

    it('does not set the pipeline from the selector modal when cancelled', async () => {
      tree = TestUtils.mountWithRouter(
        <TestNewRun t={(key: any) => key} {...(generateProps() as any)} />,
      );
      await TestUtils.flushPromises();

      const oldPipeline = newMockPipeline();
      oldPipeline.id = 'old-pipeline-id';
      oldPipeline.name = 'old-pipeline-name';
      const newPipeline = newMockPipeline();
      newPipeline.id = 'new-pipeline-id';
      newPipeline.name = 'new-pipeline-name';
      getPipelineSpy.mockImplementation(() => newPipeline);
      tree.setState({ pipeline: oldPipeline, pipelineName: oldPipeline.name });

      tree
        .find('#choosePipelineBtn')
        .at(0)
        .simulate('click');
      expect(tree.state('pipelineSelectorOpen')).toBe(true);

      // Simulate selecting pipeline
      tree.setState({ unconfirmedSelectedPipeline: newPipeline });

      // Cancel pipeline selector
      tree
        .find('#cancelPipelineSelectionBtn')
        .at(0)
        .simulate('click');
      expect(tree.state('pipelineSelectorOpen')).toBe(false);

      expect(tree.state('pipeline')).toEqual(oldPipeline);
      expect(tree.state('pipelineName')).toEqual(oldPipeline.name);
      expect(tree.state('pipelineSelectorOpen')).toBe(false);
      await TestUtils.flushPromises();
    });
  });

  describe('choosing an experiment', () => {
    it("opens up the experiment selector modal when users clicks 'Choose'", async () => {
      tree = TestUtils.mountWithRouter(
        <TestNewRun t={(key: any) => key} {...(generateProps() as any)} />,
      );
      await TestUtils.flushPromises();

      tree
        .find('#chooseExperimentBtn')
        .at(0)
        .simulate('click');
      await TestUtils.flushPromises();
      expect(tree.state('experimentSelectorOpen')).toBe(true);
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
                string_value: ExperimentStorageState.ARCHIVED.toString(),
              },
            ],
          } as ApiFilter),
        ),
        undefined,
        undefined,
      );
    });

    it('lists available experiments by namespace if available', async () => {
      tree = TestUtils.mountWithRouter(
        <TestNewRun t={(key: any) => key} {...(generateProps() as any)} namespace='test-ns' />,
      );
      await TestUtils.flushPromises();

      tree
        .find('#chooseExperimentBtn')
        .at(0)
        .simulate('click');
      await TestUtils.flushPromises();
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
                string_value: ExperimentStorageState.ARCHIVED.toString(),
              },
            ],
          } as ApiFilter),
        ),
        'NAMESPACE',
        'test-ns',
      );
    });

    it('closes the experiment selector modal', async () => {
      tree = TestUtils.mountWithRouter(
        <TestNewRun t={(key: any) => key} {...(generateProps() as any)} />,
      );
      await TestUtils.flushPromises();

      tree
        .find('#chooseExperimentBtn')
        .at(0)
        .simulate('click');
      expect(tree.state('experimentSelectorOpen')).toBe(true);

      tree
        .find('#cancelExperimentSelectionBtn')
        .at(0)
        .simulate('click');
      expect(tree.state('experimentSelectorOpen')).toBe(false);
    });

    it('sets the experiment from the selector modal when confirmed', async () => {
      tree = TestUtils.mountWithRouter(
        <TestNewRun t={(key: any) => key} {...(generateProps() as any)} />,
      );
      await TestUtils.flushPromises();

      const oldExperiment = newMockExperiment();
      oldExperiment.id = 'old-experiment-id';
      oldExperiment.name = 'old-experiment-name';
      const newExperiment = newMockExperiment();
      newExperiment.id = 'new-experiment-id';
      newExperiment.name = 'new-experiment-name';
      getExperimentSpy.mockImplementation(() => newExperiment);
      tree.setState({ experiment: oldExperiment, experimentName: oldExperiment.name });

      tree
        .find('#chooseExperimentBtn')
        .at(0)
        .simulate('click');
      expect(tree.state('experimentSelectorOpen')).toBe(true);

      // Simulate selecting experiment
      tree.setState({ unconfirmedSelectedExperiment: newExperiment });

      // Confirm experiment selector
      tree
        .find('#useExperimentBtn')
        .at(0)
        .simulate('click');
      await TestUtils.flushPromises();
      expect(tree.state('experimentSelectorOpen')).toBe(false);

      expect(tree.state('experiment')).toEqual(newExperiment);
      expect(tree.state('experimentName')).toEqual(newExperiment.name);
      expect(tree.state('experimentSelectorOpen')).toBe(false);
      await TestUtils.flushPromises();
    });

    it('does not set the experiment from the selector modal when cancelled', async () => {
      tree = TestUtils.mountWithRouter(
        <TestNewRun t={(key: any) => key} {...(generateProps() as any)} />,
      );
      await TestUtils.flushPromises();

      const oldExperiment = newMockExperiment();
      oldExperiment.id = 'old-experiment-id';
      oldExperiment.name = 'old-experiment-name';
      const newExperiment = newMockExperiment();
      newExperiment.id = 'new-experiment-id';
      newExperiment.name = 'new-experiment-name';
      getExperimentSpy.mockImplementation(() => newExperiment);
      tree.setState({ experiment: oldExperiment, experimentName: oldExperiment.name });

      tree
        .find('#chooseExperimentBtn')
        .at(0)
        .simulate('click');
      expect(tree.state('experimentSelectorOpen')).toBe(true);

      // Simulate selecting experiment
      tree.setState({ unconfirmedSelectedExperiment: newExperiment });

      // Cancel experiment selector
      tree
        .find('#cancelExperimentSelectionBtn')
        .at(0)
        .simulate('click');
      expect(tree.state('experimentSelectorOpen')).toBe(false);

      expect(tree.state('experiment')).toEqual(oldExperiment);
      expect(tree.state('experimentName')).toEqual(oldExperiment.name);
      expect(tree.state('experimentSelectorOpen')).toBe(false);
      await TestUtils.flushPromises();
    });
  });

  // TODO: Add test for when dialog is dismissed. Due to the particulars of how the Dialog element
  // works, this will not be possible until it's wrapped in some manner, like UploadPipelineDialog
  // in PipelineList

  describe('cloning from a run', () => {
    it('fetches the original run if an ID is present in the query params', async () => {
      const run = newMockRunDetail().run!;
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${run.id}`;

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(getRunSpy).toHaveBeenCalledTimes(1);
      expect(getRunSpy).toHaveBeenLastCalledWith(run.id);
    });

    it("automatically generates the new run name based on the original run's name", async () => {
      const runDetail = newMockRunDetail();
      runDetail.run!.name = '-original run-';
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(tree.state('runName')).toBe('common:cloneOf -original run-');
    });

    it('automatically generates the new clone name if the original run was a clone', async () => {
      const runDetail = newMockRunDetail();
      runDetail.run!.name = 'Clone of some run';
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(tree.state('runName')).toBe('common:cloneMatcher some run');
    });

    it('uses service account in the original run', async () => {
      const defaultRunDetail = newMockRunDetail();
      const runDetail = {
        ...defaultRunDetail,
        run: {
          ...defaultRunDetail.run,
          service_account: 'sa1',
        },
      };
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;
      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(tree.state('serviceAccount')).toBe('sa1');
    });

    it('uses the query param experiment ID over the one in the original run if an ID is present in both', async () => {
      const experiment = newMockExperiment();
      const runDetail = newMockRunDetail();
      runDetail.run!.resource_references = [
        {
          key: { id: `${experiment.id}-different`, type: ApiResourceType.EXPERIMENT },
        },
      ];
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}` +
        `&${QUERY_PARAMS.experimentId}=${experiment.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(getRunSpy).toHaveBeenCalledTimes(1);
      expect(getRunSpy).toHaveBeenLastCalledWith(runDetail.run!.id);
      expect(getExperimentSpy).toHaveBeenCalledTimes(1);
      expect(getExperimentSpy).toHaveBeenLastCalledWith(experiment.id);
    });

    it('uses the experiment ID in the original run if no experiment ID is present in query params', async () => {
      const originalRunExperimentId = 'original-run-experiment-id';
      const runDetail = newMockRunDetail();
      runDetail.run!.resource_references = [
        {
          key: { id: originalRunExperimentId, type: ApiResourceType.EXPERIMENT },
        },
      ];
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(getRunSpy).toHaveBeenCalledTimes(1);
      expect(getRunSpy).toHaveBeenLastCalledWith(runDetail.run!.id);
      expect(getExperimentSpy).toHaveBeenCalledTimes(1);
      expect(getExperimentSpy).toHaveBeenLastCalledWith(originalRunExperimentId);
    });

    it('retrieves the pipeline from the original run, even if there is a pipeline ID in the query params', async () => {
      // The error is caused by incomplete mock data.
      muteErrors();

      const runDetail = newMockRunDetail();
      runDetail.run!.pipeline_spec = { pipeline_id: 'original-run-pipeline-id' };
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}` +
        `&${QUERY_PARAMS.pipelineId}=some-other-pipeline-id`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
      expect(getPipelineSpy).toHaveBeenLastCalledWith(runDetail.run!.pipeline_spec!.pipeline_id);
    });

    it('shows a page error if getPipeline fails to find the pipeline from the original run', async () => {
      muteErrors();

      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${MOCK_RUN_DETAIL.run!.id}`;

      TestUtils.makeErrorResponseOnce(getPipelineSpy, 'test error message');

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          additionalInfo: 'test error message',
          message: 'errorFindPipeline some-mock-run-id. common:clickDetails',
          mode: 'error',
        }),
      );
    });

    it('shows an error if getPipeline fails to find the pipeline from the original run', async () => {
      muteErrors();

      const runDetail = newMockRunDetail();
      runDetail.run!.pipeline_spec!.pipeline_id = undefined;
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          message: 'errorReadPipelineDef common:clickDetails',
          mode: 'error',
        }),
      );
    });

    it('does not call getPipeline if original run has pipeline spec instead of id', async () => {
      // Error expected because of incompelte mock data.
      muteErrors();

      const runDetail = newMockRunDetail();
      delete runDetail.run!.pipeline_spec!.pipeline_id;
      runDetail.run!.pipeline_spec!.workflow_manifest = 'test workflow yaml';
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(getPipelineSpy).not.toHaveBeenCalled();
    });

    it('shows a page error if parsing embedded pipeline yaml fails', async () => {
      muteErrors();

      const runDetail = newMockRunDetail();
      delete runDetail.run!.pipeline_spec!.pipeline_id;
      runDetail.run!.pipeline_spec!.workflow_manifest = '!definitely not yaml';
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          message: 'errorReadPipelineDef common:clickDetails',
          mode: 'error',
        }),
      );
    });

    it('loads and selects embedded pipeline from run', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${
        MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.id
      }`;

      getRunSpy.mockImplementation(() => MOCK_RUN_WITH_EMBEDDED_PIPELINE);

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenCalledTimes(1);
      expect(tree.state('workflowFromRun')).toEqual({
        metadata: { name: 'embedded' },
        parameters: [],
      });
      expect(tree.state('useWorkflowFromRun')).toBe(true);
    });

    it("shows a page error if the original run's workflow_manifest is undefined", async () => {
      muteErrors();

      const runDetail = newMockRunDetail();
      runDetail.run!.pipeline_spec!.workflow_manifest = undefined;
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          message: 'errorRun some-mock-run-id noWorkflowManifest ',
          mode: 'error',
        }),
      );
    });

    it("shows a page error if the original run's workflow_manifest is invalid JSON", async () => {
      muteErrors();

      const runDetail = newMockRunWithEmbeddedPipeline();
      runDetail.run!.pipeline_spec!.workflow_manifest = 'not json';
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          message: 'errorReadPipelineDef common:clickDetails',
          mode: 'error',
        }),
      );
    });

    it("gets the pipeline parameter values of the original run's pipeline", async () => {
      const runDetail = newMockRunDetail();
      const originalRunPipelineParams: ApiParameter[] = [
        { name: 'thisTestParam', value: 'thisTestVal' },
      ];
      runDetail.pipeline_runtime!.workflow_manifest = JSON.stringify({
        spec: {
          arguments: {
            parameters: originalRunPipelineParams,
          },
        },
      });
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(tree.state('parameters')).toEqual(originalRunPipelineParams);
    });

    it('shows a page error if getRun fails', async () => {
      muteErrors();

      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${MOCK_RUN_DETAIL.run!.id}`;

      TestUtils.makeErrorResponseOnce(getRunSpy, 'test error message');

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          additionalInfo: 'test error message',
          message: `errorRetrieveOrigRun: some-mock-run-id. common:clickDetails`,
          // again err is undefined
          mode: 'error',
        }),
      );
    });
  });

  // TODO: test other attributes and scenarios
  describe('cloning from a recurring run', () => {
    it('clones trigger schedule', async () => {
      const jobDetail = newMockJob();
      const startTime = new Date(1234);
      jobDetail.name = 'job1';
      jobDetail.trigger = {
        periodic_schedule: {
          interval_second: '360',
          start_time: startTime.toISOString() as any,
        },
      };
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRecurringRun}=${jobDetail.id}`;

      getJobSpy.mockImplementation(() => jobDetail);

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(tree.state('runName')).toBe('common:cloneOf job1');
      expect(tree.state('trigger')).toEqual({
        periodic_schedule: {
          interval_second: '360',
          start_time: '1970-01-01T00:00:01.234Z',
        },
      });
    });
  });

  describe('arriving from pipeline details page', () => {
    let mockEmbeddedPipelineProps: PageProps;
    beforeEach(() => {
      mockEmbeddedPipelineProps = generateProps();
      mockEmbeddedPipelineProps.location.search = `?${QUERY_PARAMS.fromRunId}=${
        MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.id
      }`;
      getRunSpy.mockImplementationOnce(() => MOCK_RUN_WITH_EMBEDDED_PIPELINE);
    });

    it('indicates that a pipeline is preselected and provides a means of selecting a different pipeline', async () => {
      tree = shallow(<TestNewRun t={(key: any) => key} {...(mockEmbeddedPipelineProps as any)} />);
      await TestUtils.flushPromises();

      expect(tree.state('useWorkflowFromRun')).toBe(true);
      expect(tree.state('usePipelineFromRunLabel')).toBe('usePipelinePrevPage');
      expect(tree).toMatchSnapshot();
    });

    it('retrieves the run with the embedded pipeline', async () => {
      tree = shallow(<TestNewRun t={(key: any) => key} {...(mockEmbeddedPipelineProps as any)} />);
      await TestUtils.flushPromises();

      expect(getRunSpy).toHaveBeenLastCalledWith(MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.id);
    });

    it('parses the embedded workflow and stores it in state', async () => {
      MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.pipeline_spec!.workflow_manifest = JSON.stringify(
        MOCK_PIPELINE,
      );

      tree = shallow(<TestNewRun t={(key: any) => key} {...(mockEmbeddedPipelineProps as any)} />);
      await TestUtils.flushPromises();

      expect(tree.state('workflowFromRun')).toEqual(MOCK_PIPELINE);
      expect(tree.state('parameters')).toEqual(MOCK_PIPELINE.parameters);
      expect(tree.state('useWorkflowFromRun')).toBe(true);
    });

    it('displays a page error if it fails to parse the embedded pipeline', async () => {
      muteErrors();

      MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.pipeline_spec!.workflow_manifest = 'not JSON';

      tree = shallow(<TestNewRun t={(key: any) => key} {...(mockEmbeddedPipelineProps as any)} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          additionalInfo: 'Unexpected token o in JSON at position 1',
          message: 'errorParsePipeline: not JSON. common:clickDetails',
          // err is undefined need to mock maybe
          mode: 'error',
        }),
      );
    });

    it('displays a page error if referenced run has no embedded pipeline', async () => {
      muteErrors();

      // Remove workflow_manifest entirely
      delete MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.pipeline_spec!.workflow_manifest;

      tree = mount(<TestNewRun t={(key: any) => key} {...(mockEmbeddedPipelineProps as any)} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          message: `errorRunProvided: some-mock-run-id noEmbeddedPipeline. `,
          mode: 'error',
        }),
      );
    });

    it('displays a page error if it fails to retrieve the run containing the embedded pipeline', async () => {
      muteErrors();

      getRunSpy.mockReset();
      TestUtils.makeErrorResponseOnce(getRunSpy, 'test - error!');

      tree = shallow(<TestNewRun t={(key: any) => key} {...(mockEmbeddedPipelineProps as any)} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          additionalInfo: 'test - error!',
          message: `errorRetrieveSpecRun: some-mock-run-id. common:clickDetails`,
          // again 'err' is undefined
          mode: 'error',
        }),
      );
    });
  });

  describe('starting a new run', () => {
    it("disables 'Start' new run button by default", async () => {
      tree = shallow(<TestNewRun t={(key: any) => key} {...(generateProps() as any)} />);
      await TestUtils.flushPromises();

      expect(tree.find('#startNewRunBtn').props()).toHaveProperty('disabled', true);
    });

    it("enables the 'Start' new run button if pipeline ID and pipeline version ID in query params and run name entered", async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}&${QUERY_PARAMS.pipelineVersionId}=${MOCK_PIPELINE_VERSION.id}`;

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'run name' } });
      await TestUtils.flushPromises();

      expect(tree.find('#startNewRunBtn').props()).toHaveProperty('disabled', false);
    });

    it("re-disables the 'Start' new run button if pipeline ID and pipeline version ID in query params and run name entered then cleared", async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}&${QUERY_PARAMS.pipelineVersionId}=${MOCK_PIPELINE_VERSION.id}`;

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'run name' } });
      await TestUtils.flushPromises();
      expect(tree.find('#startNewRunBtn').props()).toHaveProperty('disabled', false);

      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: '' } });
      expect(tree.find('#startNewRunBtn').props()).toHaveProperty('disabled', true);
    });

    it("sends a request to Start a run when 'Start' is clicked", async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}` +
        `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}` +
        `&${QUERY_PARAMS.pipelineVersionId}=${MOCK_PIPELINE_VERSION.id}`;

      tree = mount(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      (tree.instance() as TestNewRun).handleChange('runName')({
        target: { value: 'test run name' },
      });
      (tree.instance() as TestNewRun).handleChange('description')({
        target: { value: 'test run description' },
      });
      (tree.instance() as TestNewRun).handleChange('serviceAccount')({
        target: { value: 'service-account-name' },
      });
      await TestUtils.flushPromises();

      tree
        .find('#startNewRunBtn')
        .hostNodes()
        .simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(startRunSpy).toHaveBeenCalledTimes(1);
      expect(startRunSpy).toHaveBeenLastCalledWith({
        description: 'test run description',
        name: 'test run name',
        pipeline_spec: {
          parameters: MOCK_PIPELINE.parameters,
        },
        service_account: 'service-account-name',
        resource_references: [
          {
            key: {
              id: MOCK_EXPERIMENT.id,
              type: ApiResourceType.EXPERIMENT,
            },
            relationship: ApiRelationship.OWNER,
          },
          {
            key: {
              id: MOCK_PIPELINE_VERSION.id,
              type: ApiResourceType.PIPELINEVERSION,
            },
            relationship: ApiRelationship.CREATOR,
          },
        ],
      });
    });

    it('sends a request to Start a run with the json editor open', async () => {
      const props = generateProps();
      const pipeline = newMockPipelineWithParameters();
      pipeline.parameters = [{ name: 'testName', value: 'testValue' }];
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}` +
        `&${QUERY_PARAMS.pipelineId}=${pipeline.id}`;
      tree = TestUtils.mountWithRouter(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      tree.setState({ parameters: pipeline.parameters });
      (tree.instance() as TestNewRun).handleChange('runName')({
        target: { value: 'test run name' },
      });
      (tree.instance() as TestNewRun).handleChange('description')({
        target: { value: 'test run description' },
      });

      tree
        .find('input#newRunPipelineParam0')
        .simulate('change', { target: { value: '{"test2": "value2"}' } });

      tree.find('TextField#newRunPipelineParam0 Button').simulate('click');

      tree.find('BusyButton#startNewRunBtn').simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(startRunSpy).toHaveBeenCalledTimes(1);
      expect(startRunSpy).toHaveBeenLastCalledWith({
        description: 'test run description',
        name: 'test run name',
        pipeline_spec: {
          parameters: [{ name: 'testName', value: '{\n  "test2": "value2"\n}' }],
        },
        service_account: '',
        resource_references: [
          {
            key: {
              id: MOCK_EXPERIMENT.id,
              type: ApiResourceType.EXPERIMENT,
            },
            relationship: ApiRelationship.OWNER,
          },
          {
            key: {
              id: 'original-run-pipeline-version-id',
              type: ApiResourceType.PIPELINEVERSION,
            },
            relationship: ApiRelationship.CREATOR,
          },
        ],
      });
    });

    it('updates the parameters in state on handleParamChange', async () => {
      const props = generateProps();
      const pipeline = newMockPipeline();
      const pipelineVersion = newMockPipelineVersion();
      pipelineVersion.parameters = [
        { name: 'param-1', value: '' },
        { name: 'param-2', value: 'prefilled value' },
      ];
      props.location.search = `?${QUERY_PARAMS.pipelineId}=${pipeline.id}&${QUERY_PARAMS.pipelineVersionId}=${pipelineVersion.id}`;

      getPipelineSpy.mockImplementation(() => pipeline);
      getPipelineVersionSpy.mockImplementation(() => pipelineVersion);

      tree = mount(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();
      (tree.instance() as TestNewRun).handleChange('runName')({
        target: { value: 'test run name' },
      });
      // Fill in the first pipeline parameter
      (tree.instance() as TestNewRun)._handleParamChange(0, 'test param value');

      tree
        .find('#startNewRunBtn')
        .hostNodes()
        .simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(startRunSpy).toHaveBeenCalledTimes(1);
      expect(startRunSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          pipeline_spec: {
            parameters: [
              { name: 'param-1', value: 'test param value' },
              { name: 'param-2', value: 'prefilled value' },
            ],
          },
        }),
      );
    });

    it('copies pipeline from run in the start API call when cloning a run with embedded pipeline', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${
        MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.id
      }`;

      getRunSpy.mockImplementation(() => MOCK_RUN_WITH_EMBEDDED_PIPELINE);

      tree = mount(
        // Router is needed as context for Links to work.
        <MemoryRouter>
          <TestNewRun t={(key: any) => key} {...props} />
        </MemoryRouter>,
      );
      await TestUtils.flushPromises();

      tree
        .find('#startNewRunBtn')
        .hostNodes()
        .simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(startRunSpy).toHaveBeenCalledTimes(1);
      expect(startRunSpy).toHaveBeenLastCalledWith({
        description: '',
        name: 'common:cloneOf ' + MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.name,
        pipeline_spec: {
          parameters: [],
          pipeline_id: undefined,
          workflow_manifest: '{"metadata":{"name":"embedded"},"parameters":[]}',
        },
        service_account: 'pipeline-runner',
        resource_references: [],
      });
      // TODO: verify route change happens
    });

    it('updates the pipeline params as user selects different pipelines', async () => {
      tree = shallow(<TestNewRun t={(key: any) => key} {...generateProps()} />);
      await TestUtils.flushPromises();

      // No parameters should be showing
      expect(tree).toMatchSnapshot();

      // Select a pipeline version with parameters
      const pipeline = newMockPipeline();
      const pipelineVersionWithParams = newMockPipelineVersion();
      pipelineVersionWithParams.id = 'pipeline-version-with-params';
      pipelineVersionWithParams.parameters = [
        { name: 'param-1', value: 'prefilled value 1' },
        { name: 'param-2', value: 'prefilled value 2' },
      ];
      getPipelineSpy.mockImplementationOnce(() => pipeline);
      getPipelineVersionSpy.mockImplementationOnce(() => pipelineVersionWithParams);
      tree.setState({ unconfirmedSelectedPipeline: pipeline });
      tree.setState({ unconfirmedSelectedPipelineVersion: pipelineVersionWithParams });
      const instance = tree.instance() as TestNewRun;
      instance._pipelineSelectorClosed(true);
      instance._pipelineVersionSelectorClosed(true);
      await TestUtils.flushPromises();
      expect(tree).toMatchSnapshot();

      // Select a new pipeline with no parameters
      const noParamsPipeline = newMockPipeline();
      noParamsPipeline.id = 'no-params-pipeline';
      noParamsPipeline.parameters = [];
      const noParamsPipelineVersion = newMockPipelineVersion();
      noParamsPipelineVersion.id = 'no-params-pipeline-version';
      noParamsPipelineVersion.parameters = [];
      getPipelineSpy.mockImplementationOnce(() => noParamsPipeline);
      getPipelineVersionSpy.mockImplementationOnce(() => noParamsPipelineVersion);
      tree.setState({ unconfirmedSelectedPipeline: noParamsPipeline });
      tree.setState({ unconfirmedSelectedPipelineVersion: noParamsPipelineVersion });
      instance._pipelineSelectorClosed(true);
      instance._pipelineVersionSelectorClosed(true);
      await TestUtils.flushPromises();
      expect(tree).toMatchSnapshot();
    });

    it('trims whitespace from the pipeline params', async () => {
      tree = mount(
        <MemoryRouter>
          <TestNewRun t={(key: any) => key} {...generateProps()} />
        </MemoryRouter>,
      );
      await TestUtils.flushPromises();
      const instance = tree.find(TestNewRun).instance() as TestNewRun;
      fillRequiredFields(instance);

      // Select a pipeline with parameters
      const pipeline = newMockPipeline();
      const pipelineVersionWithParams = newMockPipeline();
      pipelineVersionWithParams.id = 'pipeline-version-with-params';
      pipelineVersionWithParams.parameters = [
        { name: 'param-1', value: '  whitespace on either side  ' },
        { name: 'param-2', value: 'value 2' },
      ];
      getPipelineSpy.mockImplementationOnce(() => pipeline);
      getPipelineVersionSpy.mockImplementationOnce(() => pipelineVersionWithParams);
      instance.setState({ unconfirmedSelectedPipeline: pipeline });
      instance.setState({ unconfirmedSelectedPipelineVersion: pipelineVersionWithParams });
      instance._pipelineSelectorClosed(true);
      instance._pipelineVersionSelectorClosed(true);
      tree
        .find('#startNewRunBtn')
        .hostNodes()
        .simulate('click');
      await TestUtils.flushPromises();

      expect(startRunSpy).toHaveBeenCalledTimes(1);
      expect(startRunSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          pipeline_spec: {
            parameters: [
              { name: 'param-1', value: 'whitespace on either side' },
              { name: 'param-2', value: 'value 2' },
            ],
          },
        }),
      );
    });

    it("sets the page to a busy state upon clicking 'Start'", async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}` +
        `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}` +
        `&${QUERY_PARAMS.pipelineVersionId}=${MOCK_PIPELINE_VERSION.id}`;

      tree = mount(<TestNewRun t={(key: any) => key} {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({
        target: { value: 'test run name' },
      });
      await TestUtils.flushPromises();

      tree
        .find('#startNewRunBtn')
        .hostNodes()
        .simulate('click');

      expect(tree.state('isBeingStarted')).toBe(true);
    });

    it('navigates to the ExperimentDetails page upon successful start if there was an experiment', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}` +
        `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}` +
        `&${QUERY_PARAMS.pipelineVersionId}=${MOCK_PIPELINE_VERSION.id}`;

      tree = mount(<TestNewRun t={(key: any) => key} {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({
        target: { value: 'test run name' },
      });
      await TestUtils.flushPromises();

      tree
        .find('#startNewRunBtn')
        .hostNodes()
        .simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(historyPushSpy).toHaveBeenCalledWith(
        RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, MOCK_EXPERIMENT.id!),
      );
    });

    it('navigates to the AllRuns page upon successful start if there was not an experiment', async () => {
      const props = generateProps();
      // No experiment in query params
      props.location.search = `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}&${QUERY_PARAMS.pipelineVersionId}=${MOCK_PIPELINE_VERSION.id}`;

      tree = mount(<TestNewRun t={(key: any) => key} {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({
        target: { value: 'test run name' },
      });
      await TestUtils.flushPromises();

      tree
        .find('#startNewRunBtn')
        .hostNodes()
        .simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(historyPushSpy).toHaveBeenCalledWith(RoutePage.RUNS);
    });

    it('shows an error dialog if Starting the new run fails', async () => {
      muteErrors();

      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}` +
        `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}` +
        `&${QUERY_PARAMS.pipelineVersionId}=${MOCK_PIPELINE_VERSION.id}`;

      TestUtils.makeErrorResponseOnce(startRunSpy, 'test error message');

      tree = mount(<TestNewRun t={(key: any) => key} {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({
        target: { value: 'test run name' },
      });
      await TestUtils.flushPromises();

      tree
        .find('#startNewRunBtn')
        .hostNodes()
        .simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(updateDialogSpy).toHaveBeenCalledTimes(1);
      expect(updateDialogSpy.mock.calls[0][0]).toMatchObject({
        content: 'test error message',
        title: 'runCreationFailed',
      });
    });

    it("shows an error dialog if 'Start' is clicked and the new run somehow has no pipeline", async () => {
      muteErrors();

      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({
        target: { value: 'test run name' },
      });
      await TestUtils.flushPromises();

      tree.find('#startNewRunBtn').simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(updateDialogSpy).toHaveBeenCalledTimes(1);
      expect(updateDialogSpy.mock.calls[0][0]).toMatchObject({
        content: 'cannotStartRun',
        title: 'runCreationFailed',
      });
    });

    it('unsets the page to a busy state if starting run fails', async () => {
      muteErrors();

      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}` +
        `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}` +
        `&${QUERY_PARAMS.pipelineVersionId}=${MOCK_PIPELINE_VERSION.id}`;

      TestUtils.makeErrorResponseOnce(startRunSpy, 'test error message');

      tree = mount(<TestNewRun t={(key: any) => key} {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({
        target: { value: 'test run name' },
      });
      await TestUtils.flushPromises();

      tree
        .find('#startNewRunBtn')
        .hostNodes()
        .simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(tree.state('isBeingStarted')).toBe(false);
    });

    it('shows snackbar confirmation after experiment is started', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}` +
        `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}` +
        `&${QUERY_PARAMS.pipelineVersionId}=${MOCK_PIPELINE_VERSION.id}`;

      tree = mount(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      (tree.instance() as TestNewRun).handleChange('runName')({
        target: { value: 'test run name' },
      });
      await TestUtils.flushPromises();

      tree
        .find('#startNewRunBtn')
        .hostNodes()
        .simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
        message: 'startNewRunSuccess: test run name',
        open: true,
      });
    });
  });

  describe('starting a new recurring run', () => {
    it('changes the title if the new run will recur, based on query param', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.isRecurring}=1`;
      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(updateToolbarSpy).toHaveBeenLastCalledWith({
        actions: {},
        breadcrumbs: [{ displayName: 'common:experiments', href: RoutePage.EXPERIMENTS }],
        pageTitle: 'common:start aRecurringRun',
      });
    });

    it('includes additional trigger input fields if run will be recurring', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.isRecurring}=1`;
      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      await TestUtils.flushPromises();

      expect(tree).toMatchSnapshot();
    });

    it("sends a request to start a new recurring run with default periodic schedule when 'Start' is clicked", async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.isRecurring}=1` +
        `&${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}` +
        `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}` +
        `&${QUERY_PARAMS.pipelineVersionId}=${MOCK_PIPELINE_VERSION.id}`;

      tree = TestUtils.mountWithRouter(<TestNewRun t={(key: any) => key} {...props} />);
      const instance = tree.instance() as TestNewRun;
      await TestUtils.flushPromises();

      instance.handleChange('runName')({ target: { value: 'test run name' } });
      instance.handleChange('description')({ target: { value: 'test run description' } });
      instance.handleChange('serviceAccount')({ target: { value: 'service-account-name' } });
      await TestUtils.flushPromises();

      tree
        .find('#startNewRunBtn')
        .at(0)
        .simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(startRunSpy).toHaveBeenCalledTimes(0);
      expect(startJobSpy).toHaveBeenCalledTimes(1);
      expect(startJobSpy).toHaveBeenLastCalledWith({
        description: 'test run description',
        enabled: true,
        max_concurrency: '10',
        name: 'test run name',
        no_catchup: false,
        pipeline_spec: {
          parameters: MOCK_PIPELINE.parameters,
        },
        service_account: 'service-account-name',
        resource_references: [
          {
            key: {
              id: MOCK_EXPERIMENT.id,
              type: ApiResourceType.EXPERIMENT,
            },
            relationship: ApiRelationship.OWNER,
          },
          {
            key: {
              id: MOCK_PIPELINE_VERSION.id,
              type: ApiResourceType.PIPELINEVERSION,
            },
            relationship: ApiRelationship.CREATOR,
          },
        ],
        // Default trigger
        trigger: {
          periodic_schedule: {
            end_time: undefined,
            interval_second: '60',
            start_time: undefined,
          },
        },
      });
    });

    it('displays an error message if periodic schedule end date/time is earlier than start date/time', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.isRecurring}=1` +
        `&${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}` +
        `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}` +
        `&${QUERY_PARAMS.pipelineVersionId}=${MOCK_PIPELINE_VERSION.id}`;

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({
        target: { value: 'test run name' },
      });
      tree.setState({
        trigger: {
          periodic_schedule: {
            end_time: new Date(2018, 4, 1),
            start_time: new Date(2018, 5, 1),
          },
        },
      });
      await TestUtils.flushPromises();

      expect(tree.state('errorMessage')).toBe('endDateBeforeStartDate');
    });

    it('displays an error message if cron schedule end date/time is earlier than start date/time', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.isRecurring}=1` +
        `&${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}` +
        `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}` +
        `&${QUERY_PARAMS.pipelineVersionId}=${MOCK_PIPELINE_VERSION.id}`;

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({
        target: { value: 'test run name' },
      });
      tree.setState({
        trigger: {
          cron_schedule: {
            end_time: new Date(2018, 4, 1),
            start_time: new Date(2018, 5, 1),
          },
        },
      });
      await TestUtils.flushPromises();

      expect(tree.state('errorMessage')).toBe('endDateBeforeStartDate');
    });

    it('displays an error message if max concurrent runs is negative', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.isRecurring}=1` +
        `&${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}` +
        `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}` +
        `&${QUERY_PARAMS.pipelineVersionId}=${MOCK_PIPELINE_VERSION.id}`;

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({
        target: { value: 'test run name' },
      });
      tree.setState({
        maxConcurrentRuns: '-1',
        trigger: {
          periodic_schedule: {
            interval_second: '60',
          },
        },
      });
      await TestUtils.flushPromises();

      expect(tree.state('errorMessage')).toBe('maxConcurrentRunsPositive');
    });

    it('displays an error message if max concurrent runs is not a number', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.isRecurring}=1` +
        `&${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}` +
        `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}` +
        `&${QUERY_PARAMS.pipelineVersionId}=${MOCK_PIPELINE_VERSION.id}`;

      tree = shallow(<TestNewRun t={(key: any) => key} {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({
        target: { value: 'test run name' },
      });
      tree.setState({
        maxConcurrentRuns: 'not a number',
        trigger: {
          periodic_schedule: {
            interval_second: '60',
          },
        },
      });
      await TestUtils.flushPromises();

      expect(tree.state('errorMessage')).toBe('maxConcurrentRunsPositive');
    });
  });
});
