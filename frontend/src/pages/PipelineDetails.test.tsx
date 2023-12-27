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

import { render, screen, waitFor } from '@testing-library/react';
import { graphlib } from 'dagre';
import { ReactWrapper, shallow, ShallowWrapper } from 'enzyme';
import * as React from 'react';
import * as JsYaml from 'js-yaml';
import { ApiPipeline, ApiPipelineVersion } from 'src/apis/pipeline';
import { V2beta1Pipeline, V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import { ApiRunDetail } from 'src/apis/run';
import { QUERY_PARAMS, RoutePage, RouteParams } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import { ButtonKeys } from 'src/lib/Buttons';
import * as StaticGraphParser from 'src/lib/StaticGraphParser';
import TestUtils from 'src/TestUtils';
import * as features from 'src/features';
import { PageProps } from './Page';
import PipelineDetails from './PipelineDetails';
import { ApiJob } from 'src/apis/job';
import { V2beta1Run } from 'src/apisv2beta1/run';
import { V2beta1RecurringRun } from 'src/apisv2beta1/recurringrun';
import { V2beta1Experiment } from 'src/apisv2beta1/experiment';

describe('PipelineDetails', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const getV1PipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
  const getV1PipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipelineVersion');
  const getV1TemplateSpy = jest.spyOn(Apis.pipelineServiceApi, 'getTemplate');
  const getV1PipelineVersionTemplateSpy = jest.spyOn(
    Apis.pipelineServiceApi,
    'getPipelineVersionTemplate',
  );
  const listV1PipelineVersionsSpy = jest.spyOn(Apis.pipelineServiceApi, 'listPipelineVersions');
  const getV1RunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
  const getV1RecurringRunSpy = jest.spyOn(Apis.jobServiceApi, 'getJob');
  const getV2PipelineSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
  const getV2PipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
  const listV2PipelineVersionsSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'listPipelineVersions');
  const getV2RunSpy = jest.spyOn(Apis.runServiceApiV2, 'getRun');
  const getV2RecurringRunSpy = jest.spyOn(Apis.recurringRunServiceApi, 'getRecurringRun');
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApiV2, 'getExperiment');
  const deletePipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'deletePipelineVersion');
  const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');
  const PIPELINE_VERSION_ID = 'test-pipeline-version-id';

  let tree: ShallowWrapper | ReactWrapper;
  let testV1Pipeline: ApiPipeline = {};
  let testV1PipelineVersion: ApiPipelineVersion = {};
  let testV1Run: ApiRunDetail = {};
  let testV1RecurringRun: ApiJob = {};
  let testV2Pipeline: V2beta1Pipeline = {};
  let originalTestV2PipelineVersion: V2beta1PipelineVersion = {};
  let newTestV2PipelineVersion: V2beta1PipelineVersion = {};
  let testV2Run: V2beta1Run = {};
  let testV2RecurringRun: V2beta1RecurringRun = {};

  function generateProps(
    versionId?: string,
    fromRunSpec = false,
    fromRecurringRunSpec = false,
  ): PageProps {
    let params = {};
    // If no fromXXX parameter is provided, it means KFP UI expects to
    // show Pipeline detail with pipeline version ID
    if (!fromRunSpec && !fromRecurringRunSpec) {
      params = {
        [RouteParams.pipelineId]: testV2Pipeline.pipeline_id,
        [RouteParams.pipelineVersionId]: versionId || '',
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

    testV1Pipeline = {
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

    testV1PipelineVersion = {
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

    testV2Pipeline = {
      created_at: new Date(2018, 8, 5, 4, 3, 2),
      description: 'test pipeline description',
      pipeline_id: 'test-pipeline-id',
      display_name: 'test pipeline',
    };

    originalTestV2PipelineVersion = {
      display_name: 'test-pipeline-version',
      pipeline_id: 'test-pipeline-id',
      pipeline_version_id: 'test-pipeline-version-id',
      pipeline_spec: JsYaml.safeLoad(
        'spec:\n  arguments:\n    parameters:\n      - name: output\n',
      ),
    };

    newTestV2PipelineVersion = {
      display_name: 'new-test-pipeline-version',
      pipeline_id: 'test-pipeline-id',
      pipeline_version_id: 'new-test-pipeline-version-id',
      pipeline_spec: JsYaml.safeLoad(
        'spec:\n  arguments:\n    parameters:\n      - name: output\n',
      ),
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

    getV1PipelineSpy.mockImplementation(() => Promise.resolve(testV1Pipeline));
    getV1PipelineVersionSpy.mockImplementation(() => Promise.resolve(testV1PipelineVersion));
    getV1TemplateSpy.mockImplementation(() => Promise.resolve({ template: 'test template' }));
    getV1PipelineVersionTemplateSpy.mockImplementation(() =>
      Promise.resolve({ template: 'test version template' }),
    );
    listV1PipelineVersionsSpy.mockImplementation(() =>
      Promise.resolve({ versions: [testV1PipelineVersion] }),
    );
    getV1RunSpy.mockImplementation(() => Promise.resolve(testV1Run));
    getV1RecurringRunSpy.mockImplementation(() => Promise.resolve(testV1RecurringRun));

    getV2PipelineSpy.mockImplementation(() => Promise.resolve(testV2Pipeline));
    getV2PipelineVersionSpy.mockImplementation(() =>
      Promise.resolve(originalTestV2PipelineVersion),
    );
    listV2PipelineVersionsSpy.mockImplementation(() =>
      Promise.resolve({ pipeline_versions: [originalTestV2PipelineVersion] }),
    );
    getV2RunSpy.mockImplementation(() => Promise.resolve(testV2Run));
    getV2RecurringRunSpy.mockImplementation(() => Promise.resolve(testV2RecurringRun));

    getExperimentSpy.mockImplementation(() =>
      Promise.resolve({
        experiment_id: 'test-experiment-id',
        display_name: 'test experiment',
      } as V2beta1Experiment),
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
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        breadcrumbs: [{ displayName: 'Pipelines', href: RoutePage.PIPELINES }],
        pageTitle:
          testV2Pipeline.display_name + ' (' + originalTestV2PipelineVersion.display_name + ')',
      }),
    );
  });

  it(
    'shows all runs breadcrumbs, and "Pipeline details" as page title when the pipeline ' +
      'comes from a run spec that does not have an experiment',
    async () => {
      tree = shallow(<PipelineDetails {...generateProps(undefined, true)} />);
      await getV1RunSpy;
      await getV2RunSpy;
      await createGraphSpy;
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
      tree = shallow(<PipelineDetails {...generateProps(undefined, false, true)} />);
      await getV1RecurringRunSpy;
      await getV2RecurringRunSpy;
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
      tree = shallow(<PipelineDetails {...generateProps(undefined, true)} />);
      await getV1RunSpy;
      await getV2RunSpy;
      await getExperimentSpy;
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
      tree = shallow(<PipelineDetails {...generateProps(undefined, false, true)} />);
      await getV1RecurringRunSpy;
      await getV2RecurringRunSpy;
      await getExperimentSpy;
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

  it(
    'parses the workflow source in embedded pipeline spec as JSON ' +
      'and then converts it to YAML (v1)',
    async () => {
      testV1Run.run!.pipeline_spec = {
        pipeline_id: 'run-pipeline-id',
        workflow_manifest: '{"spec": {"arguments": {"parameters": [{"name": "output"}]}}}',
      };

      tree = shallow(<PipelineDetails {...generateProps(undefined, true)} />);
      await getV1RunSpy;
      await getV2RunSpy;
      await TestUtils.flushPromises();

      expect(tree.state('templateString')).toBe(
        'spec:\n  arguments:\n    parameters:\n      - name: output\n',
      );
    },
  );

  it(
    'directly use pipeline_manifest dumped from ' + 'pipeline_spec in run as template string (v2)',
    async () => {
      jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
        if (featureKey === features.FeatureKey.V2_ALPHA) {
          return true;
        }
        return false;
      });
      testV2Run.pipeline_spec = { spec: { arguments: { parameters: [{ name: 'output' }] } } };

      tree = shallow(<PipelineDetails {...generateProps(undefined, true)} />);
      await getV1RunSpy;
      await getV2RunSpy;
      await TestUtils.flushPromises();

      expect(tree.state('templateString')).toBe(
        'spec:\n  arguments:\n    parameters:\n      - name: output\n',
      );
    },
  );

  it(
    'directly use pipeline_manifest dumped from pipeline_spec ' +
      'in recurring run as template string (v2)',
    async () => {
      jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
        if (featureKey === features.FeatureKey.V2_ALPHA) {
          return true;
        }
        return false;
      });
      testV2RecurringRun.pipeline_spec = {
        spec: { arguments: { parameters: [{ name: 'output' }] } },
      };

      tree = shallow(<PipelineDetails {...generateProps(undefined, false, true)} />);
      await getV1RecurringRunSpy;
      await getV2RecurringRunSpy;
      await TestUtils.flushPromises();

      expect(tree.state('templateString')).toBe(
        'spec:\n  arguments:\n    parameters:\n      - name: output\n',
      );
    },
  );

  it('use pipeline_version_id in run to get pipeline template string (v2)', async () => {
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      if (featureKey === features.FeatureKey.V2_ALPHA) {
        return true;
      }
      return false;
    });
    testV2Run.pipeline_version_reference.pipeline_id = 'test-pipeline-id';
    testV2Run.pipeline_version_reference.pipeline_version_id = 'test-pipeline-version-id';

    tree = shallow(<PipelineDetails {...generateProps(undefined, true)} />);
    await getV1RunSpy;
    await getV2RunSpy;
    await getV2PipelineVersionSpy;
    await TestUtils.flushPromises();

    expect(tree.state('templateString')).toBe(
      'spec:\n  arguments:\n    parameters:\n      - name: output\n',
    );
  });

  it('calls listPipelineVersions() if no pipeline version id', async () => {
    listV2PipelineVersionsSpy.mockImplementation(() =>
      Promise.resolve({
        pipeline_versions: [newTestV2PipelineVersion, originalTestV2PipelineVersion],
      }),
    );
    render(<PipelineDetails {...generateProps()} />);

    await waitFor(() => {
      expect(listV2PipelineVersionsSpy).toHaveBeenCalled();
    });

    expect(updateToolbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        breadcrumbs: [{ displayName: 'Pipelines', href: RoutePage.PIPELINES }],
        pageTitle: testV2Pipeline.display_name + ' (' + newTestV2PipelineVersion.display_name + ')',
      }),
    );
  });

  it('renders "No graph to show" if it is empty pipeline', async () => {
    TestUtils.makeErrorResponse(getV2PipelineVersionSpy, 'No pipeline version is found');
    render(<PipelineDetails {...generateProps(PIPELINE_VERSION_ID)} />);

    await waitFor(() => {
      expect(getV2PipelineVersionSpy).toHaveBeenCalled();
    });

    screen.getByText('No graph to show');
  });

  it('use pipeline_version_id in recurring run to get pipeline template string (v2)', async () => {
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      if (featureKey === features.FeatureKey.V2_ALPHA) {
        return true;
      }
      return false;
    });
    testV2RecurringRun.pipeline_version_reference.pipeline_id = 'test-pipeline-id';
    testV2RecurringRun.pipeline_version_reference.pipeline_version_id = 'test-pipeline-version-id';

    tree = shallow(<PipelineDetails {...generateProps(undefined, false, true)} />);
    await getV1RecurringRunSpy;
    await getV2RecurringRunSpy;
    await getV2PipelineVersionSpy;
    await TestUtils.flushPromises();

    expect(tree.state('templateString')).toBe(
      'spec:\n  arguments:\n    parameters:\n      - name: output\n',
    );
  });

  it(
    'shows load error banner when failing to parse the workflow source ' +
      'in embedded pipeline spec',
    async () => {
      testV1Run.run!.pipeline_spec = {
        pipeline_id: 'run-pipeline-id',
        workflow_manifest: 'not valid JSON',
      };
      render(<PipelineDetails {...generateProps(undefined, true)} />);

      await waitFor(() => {
        expect(getV1RunSpy).toHaveBeenCalled();
      });

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
    },
  );

  it('shows load error banner when failing to get run details, when loading from run spec', async () => {
    TestUtils.makeErrorResponseOnce(getV1RunSpy, 'woops');
    tree = shallow(<PipelineDetails {...generateProps(undefined, true)} />);
    await getV1PipelineSpy;
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

  it(
    'shows load error banner when failing to get experiment details, ' +
      'when loading from run spec',
    async () => {
      testV2Run.experiment_id = 'test-experiment-id';
      TestUtils.makeErrorResponse(getExperimentSpy, 'woops');
      tree = shallow(<PipelineDetails {...generateProps(undefined, true)} />);
      await getV1PipelineSpy;
      await TestUtils.flushPromises();
      expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          additionalInfo: 'woops',
          message: 'Cannot retrieve run details. Click Details for more information.',
          mode: 'error',
        }),
      );
    },
  );

  it('shows load error banner when failing to get pipeline', async () => {
    TestUtils.makeErrorResponseOnce(getV1PipelineSpy, 'woops');
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getV1PipelineSpy;
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

  it('shows load error banner when failing to get pipeline version', async () => {
    TestUtils.makeErrorResponse(getV2PipelineVersionSpy, 'No pipeline version is found');
    render(<PipelineDetails {...generateProps(PIPELINE_VERSION_ID)} />);

    await waitFor(() => {
      expect(getV2PipelineVersionSpy).toHaveBeenCalled();
      // get version error will use empty string as template string, which won't call createGraph()
      expect(createGraphSpy).toHaveBeenCalledTimes(0);
    });

    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'No pipeline version is found',
        message: 'Cannot retrieve pipeline version. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it(
    'uses an empty string and does not show error ' +
      'when pipeline_spec in the response of getPipelineVersion() is undefined' +
      'and v1 getPipelineVersionTemplate() returns empty string',
    async () => {
      jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
        if (featureKey === features.FeatureKey.V2_ALPHA) {
          return true;
        }
        return false;
      });
      getV2PipelineVersionSpy.mockResolvedValue({
        display_name: 'test-pipeline-version',
        pipeline_id: 'test-pipeline-id',
        pipeline_version_id: 'test-pipeline-version-id',
        pipeline_spec: undefined, // empty pipeline_spec
      });
      getV1PipelineVersionTemplateSpy.mockResolvedValue({ template: '' });
      render(<PipelineDetails {...generateProps(PIPELINE_VERSION_ID)} />);

      await waitFor(() => {
        expect(getV2PipelineVersionSpy).toHaveBeenCalled();
        expect(getV1PipelineVersionTemplateSpy).toHaveBeenCalled();
        // empty template string from empty pipeline_spec and it won't call createGraph()
        expect(createGraphSpy).toHaveBeenCalledTimes(0);
      });

      // No errors
      expect(updateBannerSpy).toHaveBeenCalledTimes(1); // Once to clear banner
      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({}));
    },
  );

  it(
    'uses an empty string and does not show error ' +
      'when pipeline_spec in the response of getPipelineVersion() is undefined' +
      'and v1 getTemplate() returns empty string',
    async () => {
      jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
        if (featureKey === features.FeatureKey.V2_ALPHA) {
          return true;
        }
        return false;
      });
      getV2PipelineVersionSpy.mockResolvedValue({
        display_name: 'test-pipeline-version',
        pipeline_id: 'test-pipeline-id',
        pipeline_version_id: undefined,
        pipeline_spec: undefined, // empty pipeline_spec
      });
      getV1TemplateSpy.mockResolvedValue({ template: '' });
      render(<PipelineDetails {...generateProps(PIPELINE_VERSION_ID)} />);

      await waitFor(() => {
        expect(getV2PipelineVersionSpy).toHaveBeenCalled();
        expect(getV1TemplateSpy).toHaveBeenCalled(); // because no pipeline version id
        // empty template string from empty pipeline_spec and it won't call createGraph()
        expect(createGraphSpy).toHaveBeenCalledTimes(0);
      });

      // No errors
      expect(updateBannerSpy).toHaveBeenCalledTimes(1); // Once to clear banner
      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({}));
    },
  );

  it(
    'shows no graph error banner ' +
      'when pipeline_spec in the response of getPipelineVersion() is invalid format',
    async () => {
      jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
        if (featureKey === features.FeatureKey.V2_ALPHA) {
          return true;
        }
        return false;
      });
      getV2PipelineVersionSpy.mockResolvedValue({
        display_name: 'test-pipeline-version',
        pipeline_id: 'test-pipeline-id',
        pipeline_version_id: 'test-pipeline-version-id',
        pipeline_spec: {}, // invalid pipeline_spec
      });
      render(<PipelineDetails {...generateProps(PIPELINE_VERSION_ID)} />);

      await waitFor(() => {
        expect(getV2PipelineVersionSpy).toHaveBeenCalled();
      });

      expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          additionalInfo: 'Important infomation is missing. Pipeline Spec is invalid.',
          message: 'Error: failed to generate Pipeline graph. Click Details for more information.',
          mode: 'error',
        }),
      );
    },
  );

  it('shows no graph error banner when failing to parse graph', async () => {
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      if (featureKey === features.FeatureKey.V2_ALPHA) {
        return true;
      }
      return false;
    });
    getV2PipelineVersionSpy.mockResolvedValue({
      display_name: 'test-pipeline-version',
      pipeline_id: 'test-pipeline-id',
      pipeline_version_id: 'test-pipeline-version-id',
      pipeline_spec: {
        apiVersion: 'argoproj.io/v1alpha1',
        kind: 'Workflow',
      },
    });
    TestUtils.makeErrorResponse(createGraphSpy, 'bad graph');
    render(<PipelineDetails {...generateProps(PIPELINE_VERSION_ID)} />);

    await waitFor(() => {
      expect(getV2PipelineVersionSpy).toHaveBeenCalled();
      expect(createGraphSpy).toHaveBeenCalled();
    });

    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'bad graph',
        message: 'Error: failed to generate Pipeline graph. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('has a new experiment button if it has a pipeline reference', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const newExperimentBtn = instance.getInitialToolbarState().actions[ButtonKeys.NEW_EXPERIMENT];
    expect(newExperimentBtn).toBeDefined();
  });

  it("has 'clone run' toolbar button if viewing an embedded pipeline", async () => {
    tree = shallow(<PipelineDetails {...generateProps(undefined, true)} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    /* create run and create pipeline version, so 2 */
    expect(Object.keys(instance.getInitialToolbarState().actions)).toHaveLength(1);
    const cloneRunBtn = instance.getInitialToolbarState().actions[ButtonKeys.CLONE_RUN];
    expect(cloneRunBtn).toBeDefined();
  });

  it("has 'clone recurring run' toolbar button if viewing an embedded pipeline from recurring run", async () => {
    tree = shallow(<PipelineDetails {...generateProps(undefined, false, true)} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    /* create run and create pipeline version, so 2 */
    expect(Object.keys(instance.getInitialToolbarState().actions)).toHaveLength(1);
    const cloneRecurringRunBtn = instance.getInitialToolbarState().actions[
      ButtonKeys.CLONE_RECURRING_RUN
    ];
    expect(cloneRecurringRunBtn).toBeDefined();
  });

  it(
    'clicking clone run button when viewing embedded pipeline navigates to ' +
      'the new run page (clone a run) with run ID',
    async () => {
      tree = shallow(<PipelineDetails {...generateProps(undefined, true)} />);
      await TestUtils.flushPromises();
      const instance = tree.instance() as PipelineDetails;
      const cloneRunBtn = instance.getInitialToolbarState().actions[ButtonKeys.CLONE_RUN];
      cloneRunBtn!.action();
      expect(historyPushSpy).toHaveBeenCalledTimes(1);
      expect(historyPushSpy).toHaveBeenLastCalledWith(
        RoutePage.NEW_RUN + `?${QUERY_PARAMS.cloneFromRun}=${testV1Run.run!.id}`,
      );
    },
  );

  it(
    'clicking clone recurring run button when viewing embedded pipeline from recurring run' +
      'navigates to the new run page (clone a recurring run) with recurring run ID',
    async () => {
      tree = shallow(<PipelineDetails {...generateProps(undefined, false, true)} />);
      await TestUtils.flushPromises();
      const instance = tree.instance() as PipelineDetails;
      const cloneRecurringRunBtn = instance.getInitialToolbarState().actions[
        ButtonKeys.CLONE_RECURRING_RUN
      ];
      cloneRecurringRunBtn!.action();
      expect(historyPushSpy).toHaveBeenCalledTimes(1);
      expect(historyPushSpy).toHaveBeenLastCalledWith(
        RoutePage.NEW_RUN +
          `?${QUERY_PARAMS.cloneFromRecurringRun}=${testV1RecurringRun.id}&recurring=1`,
      );
    },
  );

  it("has 'create run' toolbar button if not viewing an embedded pipeline", async () => {
    tree = shallow(<PipelineDetails {...generateProps(undefined, false)} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    /* create run, create pipeline version, create experiment and delete run, so 4 */
    expect(Object.keys(instance.getInitialToolbarState().actions)).toHaveLength(4);
    const newRunBtn = instance.getInitialToolbarState().actions[
      ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION
    ];
    expect(newRunBtn).toBeDefined();
  });

  it('uses selected version ID to create run if URL does not contain version ID', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const newRunFromPipelineVersionBtn = instance.getInitialToolbarState().actions[
      ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION
    ];
    newRunFromPipelineVersionBtn.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_RUN +
        `?${QUERY_PARAMS.pipelineId}=${testV2Pipeline.pipeline_id}&${QUERY_PARAMS.pipelineVersionId}=${originalTestV2PipelineVersion.pipeline_version_id}`,
    );
  });

  it('clicking new run button navigates to the new run page', async () => {
    tree = shallow(<PipelineDetails {...generateProps(PIPELINE_VERSION_ID, false)} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const newRunFromPipelineVersionBtn = instance.getInitialToolbarState().actions[
      ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION
    ];
    newRunFromPipelineVersionBtn.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_RUN +
        `?${QUERY_PARAMS.pipelineId}=${testV2Pipeline.pipeline_id}&${QUERY_PARAMS.pipelineVersionId}=${PIPELINE_VERSION_ID}`,
    );
  });

  it(
    'clicking new run button when viewing half-loaded page navigates to ' +
      'the new run page with pipeline ID and version ID',
    async () => {
      tree = shallow(<PipelineDetails {...generateProps(PIPELINE_VERSION_ID, false)} />);
      // Intentionally don't wait until all network requests finish.
      const instance = tree.instance() as PipelineDetails;
      const newRunFromPipelineVersionBtn = instance.getInitialToolbarState().actions[
        ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION
      ];
      newRunFromPipelineVersionBtn.action();
      expect(historyPushSpy).toHaveBeenCalledTimes(1);
      expect(historyPushSpy).toHaveBeenLastCalledWith(
        RoutePage.NEW_RUN +
          `?${QUERY_PARAMS.pipelineId}=${testV2Pipeline.pipeline_id}&${QUERY_PARAMS.pipelineVersionId}=${PIPELINE_VERSION_ID}`,
      );
    },
  );

  it('clicking new experiment button navigates to new experiment page', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const newExperimentBtn = instance.getInitialToolbarState().actions[ButtonKeys.NEW_EXPERIMENT];
    await newExperimentBtn.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_EXPERIMENT + `?${QUERY_PARAMS.pipelineId}=${testV1Pipeline.id}`,
    );
  });

  it(
    'clicking new experiment button when viewing half-loaded page navigates to ' +
      'the new experiment page with the pipeline ID',
    async () => {
      tree = shallow(<PipelineDetails {...generateProps()} />);
      // Intentionally don't wait until all network requests finish.
      const instance = tree.instance() as PipelineDetails;
      const newExperimentBtn = instance.getInitialToolbarState().actions[ButtonKeys.NEW_EXPERIMENT];
      await newExperimentBtn.action();
      expect(historyPushSpy).toHaveBeenCalledTimes(1);
      expect(historyPushSpy).toHaveBeenLastCalledWith(
        RoutePage.NEW_EXPERIMENT + `?${QUERY_PARAMS.pipelineId}=${testV1Pipeline.id}`,
      );
    },
  );

  it('has a delete button and it is enabled for pipeline version deletion', async () => {
    tree = shallow(<PipelineDetails {...generateProps(PIPELINE_VERSION_ID)} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const deleteBtn = instance.getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    expect(deleteBtn).toBeDefined();
    expect(deleteBtn.disabled).toBeFalsy();
  });

  it('has a delete button, and it is disabled because no version is selected', async () => {
    let pageProps = generateProps();
    tree = shallow(<PipelineDetails {...pageProps} />);

    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const deleteBtn = instance.getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    expect(deleteBtn).toBeDefined();
    expect(deleteBtn.disabled).toBeTruthy();
  });

  it('shows delete confirmation dialog when delete button is clicked', async () => {
    tree = shallow(<PipelineDetails {...generateProps(PIPELINE_VERSION_ID)} />);
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
    tree = shallow(<PipelineDetails {...generateProps(PIPELINE_VERSION_ID)} />);
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
    tree = shallow(<PipelineDetails {...generateProps(PIPELINE_VERSION_ID)} />);
    await TestUtils.flushPromises();
    const deleteBtn = (tree.instance() as PipelineDetails).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    expect(deletePipelineVersionSpy).toHaveBeenCalledTimes(1);
    expect(deletePipelineVersionSpy).toHaveBeenLastCalledWith(
      testV1Pipeline.id,
      testV1Pipeline.default_version!.id!,
    );
  });

  it('calls delete API when delete dialog is confirmed and page is half-loaded', async () => {
    tree = shallow(<PipelineDetails {...generateProps(PIPELINE_VERSION_ID)} />);
    // Intentionally don't wait until all network requests finish.
    const deleteBtn = (tree.instance() as PipelineDetails).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    expect(deletePipelineVersionSpy).toHaveBeenCalledTimes(1);
    expect(deletePipelineVersionSpy).toHaveBeenLastCalledWith(
      testV1Pipeline.id,
      testV1Pipeline.default_version!.id!,
    );
  });

  it('shows error dialog if deletion fails', async () => {
    tree = shallow(<PipelineDetails {...generateProps(PIPELINE_VERSION_ID)} />);
    TestUtils.makeErrorResponseOnce(deletePipelineVersionSpy, 'woops');
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
    tree = shallow(<PipelineDetails {...generateProps(PIPELINE_VERSION_ID)} />);
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
