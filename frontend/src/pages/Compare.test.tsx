import * as React from 'react';
import { createMemoryHistory } from 'history';
import EnhancedCompare, { TEST_ONLY, TaggedViewerConfig } from './CompareV1';
import TestUtils from '../TestUtils';
import { ReactWrapper, ShallowWrapper, shallow } from 'enzyme';
import { Apis } from '../lib/Apis';
import { PageProps } from './Page';
import { RoutePage, QUERY_PARAMS, RouteParams } from '../components/Router';
import { ApiRunDetail } from '../apis/run';
import { PlotType } from '../components/viewers/Viewer';
import { OutputArtifactLoader } from '../lib/OutputArtifactLoader';
import { ButtonKeys } from '../lib/Buttons';
import { render } from '@testing-library/react';
import { Router } from 'react-router-dom';
import Compare, { CompareProps } from './Compare';

describe('Switch between v1 and v2 Compare runs pages', () => {
  let tree: ReactWrapper | ShallowWrapper;

  const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => null);
  const consoleLogSpy = jest.spyOn(console, 'log').mockImplementation(() => null);

  const updateToolbarSpy = jest.fn();
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
  const outputArtifactLoaderSpy = jest.spyOn(OutputArtifactLoader, 'load');

  const RUN_ID = '1';
  const RUN_ID_2 = '2';

  function generateProps(): CompareProps {
    const pageProps: PageProps = {
      history: { push: historyPushSpy } as any,
      location: {
        search: `?${QUERY_PARAMS.runlist}=test-run-id=${RUN_ID},${RUN_ID_2}`
        // '?runlist=7ec912bc-2bf1-45e7-9906-464c8256ebf9,6ed2fde9-989c-4fc2-b5ea-ed9bacef08a5',
      } as any,
      match: '' as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
    return Object.assign(pageProps, {
      collapseSections: {},
      fullscreenViewerConfig: null,
      metricsCompareProps: { rows: [], xLabels: [], yLabels: [] },
      paramsCompareProps: { rows: [], xLabels: [], yLabels: [] },
      runs: [],
      selectedIds: [],
      viewersMap: new Map(),
      workflowObjects: [],
    });
  }

  const MOCK_RUN_1_ID = 'mock-run-1-id';
  const MOCK_RUN_2_ID = 'mock-run-2-id';
  const MOCK_RUN_3_ID = 'mock-run-3-id';

  let runs: ApiRunDetail[] = [];

  function newMockRun(id?: string): ApiRunDetail {
    return {
      pipeline_runtime: {
        workflow_manifest: '{}',
      },
      run: {
        id: id || 'test-run-id',
        name: 'test run ' + id,
      },
    };
  }

  /**
   * After calling this function, the global 'tree' will be a Compare instance with a table viewer
   * and a tensorboard viewer.
   */
  async function setUpViewersAndShallowMount(): Promise<void> {
    // Simulate returning a tensorboard and table viewer
    outputArtifactLoaderSpy.mockImplementation(() => [
      { type: PlotType.TENSORBOARD, url: 'gs://path' },
      { data: [[]], labels: ['col1, col2'], type: PlotType.TABLE },
    ]);

    const workflow = {
      status: {
        nodes: {
          node1: {
            outputs: {
              artifacts: [
                {
                  name: 'mlpipeline-ui-metadata',
                  s3: { s3Bucket: { bucket: 'test bucket' }, key: 'test key' },
                },
              ],
            },
          },
        },
      },
    };
    const run1 = newMockRun('run-with-workflow-1');
    run1.pipeline_runtime!.workflow_manifest = JSON.stringify(workflow);
    const run2 = newMockRun('run-with-workflow-2');
    run2.pipeline_runtime!.workflow_manifest = JSON.stringify(workflow);
    runs.push(run1, run2);

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.runlist}=run-with-workflow-1,run-with-workflow-2`;

    tree = shallow(<TestCompare {...props} />);
    await TestUtils.flushPromises();
  }

  beforeEach(async () => {
    // Reset mocks
    consoleErrorSpy.mockReset();
    consoleLogSpy.mockReset();
    updateBannerSpy.mockReset();
    updateDialogSpy.mockReset();
    updateSnackbarSpy.mockReset();
    updateToolbarSpy.mockReset();
    historyPushSpy.mockReset();
    outputArtifactLoaderSpy.mockReset();

    getRunSpy.mockClear();

    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];

    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));
  });

  afterEach(async () => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    if (tree && tree.exists()) {
      await tree.unmount();
    }
  });

  const v1PipelineSpecTemplate = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: entry-point-test-
spec:
  arguments:
    parameters: []
  entrypoint: entry-point-test
  templates:
  - dag:
      tasks:
      - name: recurse-1
        template: recurse-1
      - name: leaf-1
        template: leaf-1
    name: start
  - dag:
      tasks:
      - name: start
        template: start
      - name: recurse-2
        template: recurse-2
    name: recurse-1
  - dag:
      tasks:
      - name: start
        template: start
      - name: leaf-2
        template: leaf-2
      - name: recurse-3
        template: recurse-3
    name: recurse-2
  - dag:
      tasks:
      - name: start
        template: start
      - name: recurse-1
        template: recurse-1
      - name: recurse-2
        template: recurse-2
    name: recurse-3
  - dag:
      tasks:
      - name: start
        template: start
    name: entry-point-test
  - container:
    name: leaf-1
  - container:
    name: leaf-2
    `;

  it('Show error if not valid v1 template and disabled v2 feature', async () => {
    // v2 feature is turn off.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      return false;
    });
    Apis.pipelineServiceApi.getPipelineVersionTemplate = jest
      .fn()
      .mockResolvedValue({ template: 'bad graph' });
    const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');
    TestUtils.makeErrorResponse(createGraphSpy, 'bad graph');

    render(<Compare {...generateProps()} />);
    await TestUtils.flushPromises();

    screen.getByTestId('pipeline-detail-v1');
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo:
          'Unable to convert string response from server to Argo workflow template: https://argoproj.github.io/argo-workflows/workflow-templates/',
        message: 'Error: failed to generate Pipeline graph. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('Show error if v1 template cannot generate graph and disabled v2 feature', async () => {
    // v2 feature is turn off.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      return false;
    });
    Apis.pipelineServiceApi.getPipelineVersionTemplate = jest.fn().mockResolvedValue({
      template: `    
      apiVersion: argoproj.io/v1alpha1
      kind: Workflow
      metadata:
        generateName: entry-point-test-
      `,
    });
    const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');
    TestUtils.makeErrorResponse(createGraphSpy, 'bad graph');

    render(<PipelineDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    screen.getByTestId('pipeline-detail-v1');
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'bad graph',
        message: 'Error: failed to generate Pipeline graph. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('Show error if not valid v2 template and enabled v2 feature', async () => {
    // v2 feature is turn on.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      if (featureKey === features.FeatureKey.V2_ALPHA) {
        return true;
      }
      return false;
    });
    const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');
    TestUtils.makeErrorResponse(createGraphSpy, 'bad graph');

    render(<PipelineDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    screen.getByTestId('pipeline-detail-v1');
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'Important infomation is missing. Pipeline Spec is invalid.',
        message: 'Error: failed to generate Pipeline graph. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('Show v1 page if valid v1 template and enabled v2 feature flag', async () => {
    // v2 feature is turn on.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      if (featureKey === features.FeatureKey.V2_ALPHA) {
        return true;
      }
      return false;
    });

    const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');
    createGraphSpy.mockImplementation(() => new graphlib.Graph());
    Apis.pipelineServiceApi.getTemplate = jest
      .fn()
      .mockResolvedValue({ template: v1PipelineSpecTemplate });
    Apis.pipelineServiceApi.getPipelineVersionTemplate = jest
      .fn()
      .mockResolvedValue({ template: v1PipelineSpecTemplate });

    render(<PipelineDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    screen.getByTestId('pipeline-detail-v1');
    expect(updateBannerSpy).toHaveBeenCalledTimes(1);
  });

  it('Show v1 page if valid v1 template and disabled v2 feature flag', async () => {
    // v2 feature is turn off.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      return false;
    });
    Apis.pipelineServiceApi.getPipelineVersionTemplate = jest
      .fn()
      .mockResolvedValue({ template: v1PipelineSpecTemplate });
    const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');
    createGraphSpy.mockImplementation(() => new graphlib.Graph());

    render(<PipelineDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    screen.getByTestId('pipeline-detail-v1');
    expect(updateBannerSpy).toHaveBeenCalledTimes(1);
  });

  it('Show v2 page if valid v2 template and enabled v2 feature', async () => {
    // v2 feature is turn on.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      if (featureKey === features.FeatureKey.V2_ALPHA) {
        return true;
      }
      return false;
    });
    const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');
    TestUtils.makeErrorResponse(createGraphSpy, 'bad graph');
    Apis.pipelineServiceApi.getPipelineVersionTemplate = jest
      .fn()
      .mockResolvedValue({ template: v2YamlTemplateString });

    render(<PipelineDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    screen.getByTestId('pipeline-detail-v2');
    expect(updateBannerSpy).toHaveBeenCalledTimes(1);
  });
});
