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

import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import * as React from 'react';
import * as JsYaml from 'js-yaml';
import { MemoryRouter } from 'react-router';
import { vi } from 'vitest';
import { V2Experiment } from 'src/apisv2/experiment';
import { V2Pipeline, V2PipelineVersion } from 'src/apisv2/pipeline';
import { V2RecurringRun } from 'src/apisv2/recurringrun';
import { V2Run } from 'src/apisv2/run';
import { QUERY_PARAMS, RoutePage, RouteParams } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import { ButtonKeys } from 'src/lib/Buttons';
import * as StaticFlow from 'src/lib/v2/StaticFlow';
import template from 'src/data/test/lightweight_python_functions_v2_pipeline_rev.yaml?raw';
import TestUtils, { mockResizeObserver } from 'src/TestUtils';
import { PageProps } from './Page';
import PipelineDetailsPage from './PipelineDetailsPage';

type PipelineDetailsPageState = PipelineDetailsPage['state'];

class PipelineDetailsWrapper {
  private _instance: PipelineDetailsPage;
  private _renderResult: ReturnType<typeof render>;

  public constructor(instance: PipelineDetailsPage, renderResult: ReturnType<typeof render>) {
    this._instance = instance;
    this._renderResult = renderResult;
  }

  public instance(): PipelineDetailsPage {
    return this._instance;
  }

  public state<K extends keyof PipelineDetailsPageState>(
    key?: K,
  ): PipelineDetailsPageState | PipelineDetailsPageState[K] {
    const state = this._instance.state;
    return key ? state[key] : state;
  }

  public setState(state: Partial<PipelineDetailsPageState>): void {
    act(() => {
      this._instance.setState(state);
    });
  }

  public unmount(): void {
    this._renderResult.unmount();
  }
}

function renderPipelineDetailsElement(element: React.ReactElement): PipelineDetailsWrapper {
  const detailsRef = React.createRef<PipelineDetailsPage>();
  const elementWithRef = React.cloneElement(element, { ref: detailsRef });
  const result = render(<MemoryRouter>{elementWithRef}</MemoryRouter>);
  if (!detailsRef.current) {
    throw new Error('PipelineDetails instance is not available');
  }
  return new PipelineDetailsWrapper(detailsRef.current, result);
}

function renderPipelineDetailsPage(element: React.ReactElement): ReturnType<typeof render> {
  return render(<MemoryRouter>{element}</MemoryRouter>);
}

describe('PipelineDetails', () => {
  const updateBannerSpy = vi.fn();
  const updateDialogSpy = vi.fn();
  const updateSnackbarSpy = vi.fn();
  const updateToolbarSpy = vi.fn();
  const navigateSpy = vi.fn();
  const getV2PipelineSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
  const getV2PipelineVersionSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
  const listV2PipelineVersionsSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'listPipelineVersions');
  const getV2RunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
  const getV2RecurringRunSpy = vi.spyOn(Apis.recurringRunServiceApi, 'getRecurringRun');
  const getExperimentSpy = vi.spyOn(Apis.experimentServiceApiV2, 'getExperiment');
  const deletePipelineVersionSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'deletePipelineVersion');
  const createGraphSpy = vi.spyOn(StaticFlow, 'convertFlowElements');
  const PIPELINE_VERSION_ID = 'test-pipeline-version-id';

  let tree: PipelineDetailsWrapper | undefined;
  let testV2Pipeline: V2Pipeline = {};
  let originalTestV2PipelineVersion: V2PipelineVersion = {};
  let newTestV2PipelineVersion: V2PipelineVersion = {};
  let testV2Run: V2Run = {};
  let testV2RecurringRun: V2RecurringRun = {};

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

    const location = { search } as any;
    const pageProps = TestUtils.generatePageProps(
      PipelineDetailsPage,
      location,
      params,
      navigateSpy,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
    );
    return pageProps;
  }

  beforeAll(() => vi.spyOn(console, 'error').mockImplementation());

  beforeEach(() => {
    vi.clearAllMocks();
    mockResizeObserver();

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
      pipeline_spec: JsYaml.load(template),
    };

    newTestV2PipelineVersion = {
      display_name: 'new-test-pipeline-version',
      pipeline_id: 'test-pipeline-id',
      pipeline_version_id: 'new-test-pipeline-version-id',
      pipeline_spec: JsYaml.load(template),
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

    getV2PipelineSpy.mockImplementation(() => Promise.resolve(testV2Pipeline));
    getV2PipelineVersionSpy.mockImplementation(() =>
      Promise.resolve(originalTestV2PipelineVersion),
    );
    listV2PipelineVersionsSpy.mockImplementation(() =>
      Promise.resolve({ pipeline_versions: [originalTestV2PipelineVersion] }),
    );
    deletePipelineVersionSpy.mockResolvedValue(undefined as any);
    getV2RunSpy.mockImplementation(() => Promise.resolve(testV2Run));
    getV2RecurringRunSpy.mockImplementation(() => Promise.resolve(testV2RecurringRun));

    getExperimentSpy.mockImplementation(() =>
      Promise.resolve({
        experiment_id: 'test-experiment-id',
        display_name: 'test experiment',
      } as V2Experiment),
    );
    createGraphSpy.mockReturnValue([]);
  });

  afterEach(() => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    if (tree) {
      tree.unmount();
      tree = undefined;
    }
    vi.clearAllMocks();
  });

  it('provides full version management at the versions URL without loading a graph', async () => {
    const props = generateProps();
    props.location = { ...props.location, search: '?tab=versions' };
    tree = renderPipelineDetailsElement(<PipelineDetailsPage {...props} />);
    const selection = await screen.findByRole('checkbox', {
      name: 'Select version test-pipeline-version',
    });
    expect(screen.getByRole('searchbox', { name: 'Search versions' })).toBeVisible();
    expect(screen.getByRole('combobox', { name: 'Rows per page' })).toBeVisible();
    expect(createGraphSpy).not.toHaveBeenCalled();
    expect(getV2PipelineVersionSpy).not.toHaveBeenCalled();
    fireEvent.click(selection);
    expect(tree.instance().getInitialToolbarState().actions[ButtonKeys.DELETE_RUN].disabled).toBe(
      false,
    );
    await act(async () => {
      await tree!.instance().refresh();
    });
    expect(tree.instance().getInitialToolbarState().actions[ButtonKeys.DELETE_RUN].disabled).toBe(
      true,
    );
  });

  it('deletes selected versions from management and stays on the versions page', async () => {
    listV2PipelineVersionsSpy.mockResolvedValue({
      pipeline_versions: [originalTestV2PipelineVersion, newTestV2PipelineVersion],
    });
    const props = generateProps();
    props.location = { ...props.location, search: '?tab=versions' };
    tree = renderPipelineDetailsElement(<PipelineDetailsPage {...props} />);
    fireEvent.click(
      await screen.findByRole('checkbox', { name: 'Select version test-pipeline-version' }),
    );
    fireEvent.click(
      screen.getByRole('checkbox', { name: 'Select version new-test-pipeline-version' }),
    );
    await act(async () => {
      await tree!.instance().getInitialToolbarState().actions[ButtonKeys.DELETE_RUN].action();
    });
    const dialog = updateDialogSpy.mock.calls[updateDialogSpy.mock.calls.length - 1][0];
    await act(async () => {
      await dialog.buttons.find((button: { text: string }) => button.text === 'Delete').onClick();
    });
    expect(deletePipelineVersionSpy).toHaveBeenCalledWith(
      'test-pipeline-id',
      'test-pipeline-version-id',
    );
    expect(deletePipelineVersionSpy).toHaveBeenCalledWith(
      'test-pipeline-id',
      'new-test-pipeline-version-id',
    );
    expect(navigateSpy).not.toHaveBeenCalled();
    expect(tree.instance().state.selectedVersionIds).toEqual([]);
  });

  it('puts the version selector beside the title and deletion in overflow', async () => {
    tree = renderPipelineDetailsElement(
      <PipelineDetailsPage {...generateProps(PIPELINE_VERSION_ID)} />,
    );
    await TestUtils.flushPromises();
    const toolbar = tree.instance().getInitialToolbarState();
    expect(toolbar.actions[ButtonKeys.DELETE_RUN].overflow).toBe(true);
    render(<>{toolbar.titleAccessory}</>);
    expect(screen.getByRole('combobox', { name: 'Version' })).toHaveValue(
      originalTestV2PipelineVersion.pipeline_version_id,
    );
  });

  it('shows pipeline name in page name, and breadcrumb to go back to pipelines', async () => {
    tree = renderPipelineDetailsElement(<PipelineDetailsPage {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        breadcrumbs: [{ displayName: 'Pipelines', href: RoutePage.PIPELINES }],
        pageTitle: testV2Pipeline.display_name,
      }),
    );
  });

  it(
    'shows all runs breadcrumbs, and "Pipeline details" as page title when the pipeline ' +
      'comes from a run spec that does not have an experiment',
    async () => {
      tree = renderPipelineDetailsElement(
        <PipelineDetailsPage {...generateProps(undefined, true)} />,
      );
      await getV2RunSpy;
      await createGraphSpy;
      await TestUtils.flushPromises();
      expect(updateToolbarSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          breadcrumbs: [
            { displayName: 'All runs', href: RoutePage.RUNS },
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
      'comes from a recurring run spec that does not have an experiment',
    async () => {
      tree = renderPipelineDetailsElement(
        <PipelineDetailsPage {...generateProps(undefined, false, true)} />,
      );
      await getV2RecurringRunSpy;
      await TestUtils.flushPromises();
      expect(updateToolbarSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          breadcrumbs: [
            { displayName: 'All recurring runs', href: RoutePage.RECURRING_RUNS },
            {
              displayName: testV2RecurringRun.display_name,
              href: RoutePage.RECURRING_RUN_DETAILS.replace(
                ':' + RouteParams.recurringRunId,
                testV2RecurringRun.recurring_run_id!,
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
      tree = renderPipelineDetailsElement(
        <PipelineDetailsPage {...generateProps(undefined, true)} />,
      );
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
      tree = renderPipelineDetailsElement(
        <PipelineDetailsPage {...generateProps(undefined, false, true)} />,
      );
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
              displayName: testV2RecurringRun.display_name,
              href: RoutePage.RECURRING_RUN_DETAILS.replace(
                ':' + RouteParams.recurringRunId,
                testV2RecurringRun.recurring_run_id!,
              ),
            },
          ],
          pageTitle: 'Pipeline details',
        }),
      );
    },
  );

  it(
    'directly use YAML dumped from ' + 'pipeline_spec in run as template string (v2)',
    async () => {
      testV2Run.pipeline_spec = JsYaml.load(template);

      tree = renderPipelineDetailsElement(
        <PipelineDetailsPage {...generateProps(undefined, true)} />,
      );
      await getV2RunSpy;
      await TestUtils.flushPromises();

      expect(tree.state('templateString')).toBe(JsYaml.dump(JsYaml.load(template)));
    },
  );

  it(
    'directly use YAML dumped from pipeline_spec ' + 'in recurring run as template string (v2)',
    async () => {
      testV2RecurringRun.pipeline_spec = JsYaml.load(template);

      tree = renderPipelineDetailsElement(
        <PipelineDetailsPage {...generateProps(undefined, false, true)} />,
      );
      await getV2RecurringRunSpy;
      await TestUtils.flushPromises();

      expect(tree.state('templateString')).toBe(JsYaml.dump(JsYaml.load(template)));
    },
  );

  it('use pipeline_version_id in run to get pipeline template string (v2)', async () => {
    testV2Run.pipeline_version_reference.pipeline_id = 'test-pipeline-id';
    testV2Run.pipeline_version_reference.pipeline_version_id = 'test-pipeline-version-id';

    tree = renderPipelineDetailsElement(
      <PipelineDetailsPage {...generateProps(undefined, true)} />,
    );
    await getV2RunSpy;
    await getV2PipelineVersionSpy;
    await TestUtils.flushPromises();

    expect(tree.state('templateString')).toBe(JsYaml.dump(JsYaml.load(template)));
  });

  it('calls listPipelineVersions() if no pipeline version id', async () => {
    listV2PipelineVersionsSpy.mockImplementation(() =>
      Promise.resolve({
        pipeline_versions: [newTestV2PipelineVersion, originalTestV2PipelineVersion],
      }),
    );
    renderPipelineDetailsPage(<PipelineDetailsPage {...generateProps()} />);

    await waitFor(() => expect(listV2PipelineVersionsSpy).toHaveBeenCalled());

    expect(updateToolbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        breadcrumbs: [{ displayName: 'Pipelines', href: RoutePage.PIPELINES }],
        pageTitle: testV2Pipeline.display_name,
      }),
    );
  });

  it('renders the pipeline details shell if it is empty', async () => {
    TestUtils.makeErrorResponse(getV2PipelineVersionSpy, 'No pipeline version is found');
    renderPipelineDetailsPage(<PipelineDetailsPage {...generateProps(PIPELINE_VERSION_ID)} />);

    await waitFor(() => expect(getV2PipelineVersionSpy).toHaveBeenCalled());

    expect(screen.getByTestId('pipeline-detail-v2')).toBeInTheDocument();
  });

  it('use pipeline_version_id in recurring run to get pipeline template string (v2)', async () => {
    testV2RecurringRun.pipeline_version_reference.pipeline_id = 'test-pipeline-id';
    testV2RecurringRun.pipeline_version_reference.pipeline_version_id = 'test-pipeline-version-id';

    tree = renderPipelineDetailsElement(
      <PipelineDetailsPage {...generateProps(undefined, false, true)} />,
    );
    await getV2RecurringRunSpy;
    await getV2PipelineVersionSpy;
    await TestUtils.flushPromises();

    expect(tree.state('templateString')).toBe(JsYaml.dump(JsYaml.load(template)));
  });

  it('shows load error banner when failing to get run details, when loading from run spec', async () => {
    TestUtils.makeErrorResponseOnce(getV2RunSpy, 'woops');
    tree = renderPipelineDetailsElement(
      <PipelineDetailsPage {...generateProps(undefined, true)} />,
    );
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledWith(
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
      tree = renderPipelineDetailsElement(
        <PipelineDetailsPage {...generateProps(undefined, true)} />,
      );
      await TestUtils.flushPromises();
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
    TestUtils.makeErrorResponseOnce(getV2PipelineSpy, 'woops');
    tree = renderPipelineDetailsElement(<PipelineDetailsPage {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops',
        message: 'Cannot retrieve pipeline details. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('shows load error banner when failing to get pipeline version', async () => {
    TestUtils.makeErrorResponse(getV2PipelineVersionSpy, 'No pipeline version is found');
    renderPipelineDetailsPage(<PipelineDetailsPage {...generateProps(PIPELINE_VERSION_ID)} />);

    await waitFor(() => expect(getV2PipelineVersionSpy).toHaveBeenCalled());
    expect(createGraphSpy).toHaveBeenCalledTimes(0);

    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'No pipeline version is found',
        message: 'Cannot retrieve pipeline version. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('warns when the selected pipeline version has no pipeline_spec', async () => {
    getV2PipelineVersionSpy.mockResolvedValue({
      display_name: 'test-pipeline-version',
      pipeline_id: 'test-pipeline-id',
      pipeline_version_id: 'test-pipeline-version-id',
      pipeline_spec: undefined, // empty pipeline_spec
    });
    renderPipelineDetailsPage(<PipelineDetailsPage {...generateProps(PIPELINE_VERSION_ID)} />);

    await waitFor(() => expect(getV2PipelineVersionSpy).toHaveBeenCalled());
    // empty template string from empty pipeline_spec and it won't call createGraph()
    expect(createGraphSpy).toHaveBeenCalledTimes(0);

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          mode: 'warning',
          message: expect.stringContaining('no pipeline spec'),
        }),
      ),
    );
  });

  it('warns when the returned version has neither an ID nor a pipeline spec', async () => {
    getV2PipelineVersionSpy.mockResolvedValue({
      display_name: 'test-pipeline-version',
      pipeline_id: 'test-pipeline-id',
      pipeline_version_id: undefined,
      pipeline_spec: undefined, // empty pipeline_spec
    });
    renderPipelineDetailsPage(<PipelineDetailsPage {...generateProps(PIPELINE_VERSION_ID)} />);

    await waitFor(() => expect(getV2PipelineVersionSpy).toHaveBeenCalled());
    // empty template string from empty pipeline_spec and it won't call createGraph()
    expect(createGraphSpy).toHaveBeenCalledTimes(0);

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          mode: 'warning',
          message: expect.stringContaining('no pipeline spec'),
        }),
      ),
    );
  });

  it(
    'shows no graph error banner ' +
      'when pipeline_spec in the response of getPipelineVersion() is invalid format',
    async () => {
      getV2PipelineVersionSpy.mockResolvedValue({
        display_name: 'test-pipeline-version',
        pipeline_id: 'test-pipeline-id',
        pipeline_version_id: 'test-pipeline-version-id',
        pipeline_spec: {}, // invalid pipeline_spec
      });
      renderPipelineDetailsPage(<PipelineDetailsPage {...generateProps(PIPELINE_VERSION_ID)} />);

      await waitFor(() => expect(getV2PipelineVersionSpy).toHaveBeenCalled());

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
    getV2PipelineVersionSpy.mockResolvedValue({
      display_name: 'test-pipeline-version',
      pipeline_id: 'test-pipeline-id',
      pipeline_version_id: 'test-pipeline-version-id',
      pipeline_spec: JsYaml.load(template),
    });
    createGraphSpy.mockImplementationOnce(() => {
      throw new Error('bad graph');
    });
    renderPipelineDetailsPage(<PipelineDetailsPage {...generateProps(PIPELINE_VERSION_ID)} />);

    await waitFor(() => expect(getV2PipelineVersionSpy).toHaveBeenCalled());
    await waitFor(() => expect(createGraphSpy).toHaveBeenCalled());

    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'bad graph',
        message: 'Error: failed to generate Pipeline graph. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('does not offer create experiment from pipeline details', async () => {
    tree = renderPipelineDetailsElement(<PipelineDetailsPage {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetailsPage;
    const newExperimentBtn = instance.getInitialToolbarState().actions[ButtonKeys.NEW_EXPERIMENT];
    expect(newExperimentBtn).toBeUndefined();
  });

  it("has 'clone run' toolbar button if viewing an embedded pipeline", async () => {
    tree = renderPipelineDetailsElement(
      <PipelineDetailsPage {...generateProps(undefined, true)} />,
    );
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetailsPage;
    /* create run and create pipeline version, so 2 */
    expect(Object.keys(instance.getInitialToolbarState().actions)).toHaveLength(1);
    const cloneRunBtn = instance.getInitialToolbarState().actions[ButtonKeys.CLONE_RUN];
    expect(cloneRunBtn).toBeDefined();
  });

  it("has 'clone recurring run' toolbar button if viewing an embedded pipeline from recurring run", async () => {
    tree = renderPipelineDetailsElement(
      <PipelineDetailsPage {...generateProps(undefined, false, true)} />,
    );
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetailsPage;
    /* create run and create pipeline version, so 2 */
    expect(Object.keys(instance.getInitialToolbarState().actions)).toHaveLength(1);
    const cloneRecurringRunBtn =
      instance.getInitialToolbarState().actions[ButtonKeys.CLONE_RECURRING_RUN];
    expect(cloneRecurringRunBtn).toBeDefined();
  });

  it(
    'clicking clone run button when viewing embedded pipeline navigates to ' +
      'the new run page (clone a run) with run ID',
    async () => {
      tree = renderPipelineDetailsElement(
        <PipelineDetailsPage {...generateProps(undefined, true)} />,
      );
      await TestUtils.flushPromises();
      const instance = tree.instance() as PipelineDetailsPage;
      const cloneRunBtn = instance.getInitialToolbarState().actions[ButtonKeys.CLONE_RUN];
      cloneRunBtn!.action();
      expect(navigateSpy).toHaveBeenCalledTimes(1);
      expect(navigateSpy).toHaveBeenLastCalledWith(
        RoutePage.NEW_RUN + `?${QUERY_PARAMS.cloneFromRun}=${testV2Run.run_id}`,
      );
    },
  );

  it(
    'clicking clone recurring run button when viewing embedded pipeline from recurring run' +
      'navigates to the new run page (clone a recurring run) with recurring run ID',
    async () => {
      tree = renderPipelineDetailsElement(
        <PipelineDetailsPage {...generateProps(undefined, false, true)} />,
      );
      await TestUtils.flushPromises();
      const instance = tree.instance() as PipelineDetailsPage;
      const cloneRecurringRunBtn =
        instance.getInitialToolbarState().actions[ButtonKeys.CLONE_RECURRING_RUN];
      cloneRecurringRunBtn!.action();
      expect(navigateSpy).toHaveBeenCalledTimes(1);
      expect(navigateSpy).toHaveBeenLastCalledWith(
        RoutePage.NEW_RUN +
          `?${QUERY_PARAMS.cloneFromRecurringRun}=${testV2RecurringRun.recurring_run_id}&recurring=1`,
      );
    },
  );

  it("has 'create run' toolbar button if not viewing an embedded pipeline", async () => {
    tree = renderPipelineDetailsElement(
      <PipelineDetailsPage {...generateProps(undefined, false)} />,
    );
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetailsPage;
    /* create run, create pipeline version, create experiment and delete run, so 4 */
    expect(Object.keys(instance.getInitialToolbarState().actions)).toHaveLength(3);
    const newRunBtn =
      instance.getInitialToolbarState().actions[ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION];
    expect(newRunBtn).toBeDefined();
  });

  it('uses selected version ID to create run if URL does not contain version ID', async () => {
    tree = renderPipelineDetailsElement(<PipelineDetailsPage {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetailsPage;
    const newRunFromPipelineVersionBtn =
      instance.getInitialToolbarState().actions[ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION];
    newRunFromPipelineVersionBtn.action();
    expect(navigateSpy).toHaveBeenCalledTimes(1);
    expect(navigateSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_RUN +
        `?${QUERY_PARAMS.pipelineId}=${testV2Pipeline.pipeline_id}&${QUERY_PARAMS.pipelineVersionId}=${originalTestV2PipelineVersion.pipeline_version_id}`,
    );
  });

  it('clicking new run button navigates to the new run page', async () => {
    tree = renderPipelineDetailsElement(
      <PipelineDetailsPage {...generateProps(PIPELINE_VERSION_ID, false)} />,
    );
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetailsPage;
    const newRunFromPipelineVersionBtn =
      instance.getInitialToolbarState().actions[ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION];
    newRunFromPipelineVersionBtn.action();
    expect(navigateSpy).toHaveBeenCalledTimes(1);
    expect(navigateSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_RUN +
        `?${QUERY_PARAMS.pipelineId}=${testV2Pipeline.pipeline_id}&${QUERY_PARAMS.pipelineVersionId}=${PIPELINE_VERSION_ID}`,
    );
  });

  it(
    'clicking new run button when viewing half-loaded page navigates to ' +
      'the new run page with pipeline ID and version ID',
    async () => {
      tree = renderPipelineDetailsElement(
        <PipelineDetailsPage {...generateProps(PIPELINE_VERSION_ID, false)} />,
      );
      // Intentionally don't wait until all network requests finish.
      const instance = tree.instance() as PipelineDetailsPage;
      const newRunFromPipelineVersionBtn =
        instance.getInitialToolbarState().actions[ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION];
      newRunFromPipelineVersionBtn.action();
      expect(navigateSpy).toHaveBeenCalledTimes(1);
      expect(navigateSpy).toHaveBeenLastCalledWith(
        RoutePage.NEW_RUN +
          `?${QUERY_PARAMS.pipelineId}=${testV2Pipeline.pipeline_id}&${QUERY_PARAMS.pipelineVersionId}=${PIPELINE_VERSION_ID}`,
      );
    },
  );

  it('has a delete button and it is enabled for pipeline version deletion', async () => {
    tree = renderPipelineDetailsElement(
      <PipelineDetailsPage {...generateProps(PIPELINE_VERSION_ID)} />,
    );
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetailsPage;
    const deleteBtn = instance.getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    expect(deleteBtn).toBeDefined();
    expect(deleteBtn.disabled).toBeFalsy();
  });

  it('has a delete button, and it is disabled because no version is selected', async () => {
    let pageProps = generateProps();
    tree = renderPipelineDetailsElement(<PipelineDetailsPage {...pageProps} />);

    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetailsPage;
    const deleteBtn = instance.getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    expect(deleteBtn).toBeDefined();
    expect(deleteBtn.disabled).toBeTruthy();
  });

  it('shows delete confirmation dialog when delete button is clicked', async () => {
    tree = renderPipelineDetailsElement(
      <PipelineDetailsPage {...generateProps(PIPELINE_VERSION_ID)} />,
    );
    const deleteBtn = (tree.instance() as PipelineDetailsPage).getInitialToolbarState().actions[
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
    tree = renderPipelineDetailsElement(
      <PipelineDetailsPage {...generateProps(PIPELINE_VERSION_ID)} />,
    );
    const deleteBtn = (tree.instance() as PipelineDetailsPage).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const cancelBtn = call.buttons.find((b: any) => b.text === 'Cancel');
    await cancelBtn.onClick();
    expect(deletePipelineVersionSpy).not.toHaveBeenCalled();
  });

  it('calls delete API when delete dialog is confirmed', async () => {
    tree = renderPipelineDetailsElement(
      <PipelineDetailsPage {...generateProps(PIPELINE_VERSION_ID)} />,
    );
    await TestUtils.flushPromises();
    const deleteBtn = (tree.instance() as PipelineDetailsPage).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    expect(deletePipelineVersionSpy).toHaveBeenCalledTimes(1);
    expect(deletePipelineVersionSpy).toHaveBeenLastCalledWith(
      testV2Pipeline.pipeline_id,
      originalTestV2PipelineVersion.pipeline_version_id!,
    );
  });

  it('calls delete API when delete dialog is confirmed and page is half-loaded', async () => {
    tree = renderPipelineDetailsElement(
      <PipelineDetailsPage {...generateProps(PIPELINE_VERSION_ID)} />,
    );
    // Intentionally don't wait until all network requests finish.
    const deleteBtn = (tree.instance() as PipelineDetailsPage).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    expect(deletePipelineVersionSpy).toHaveBeenCalledTimes(1);
    expect(deletePipelineVersionSpy).toHaveBeenLastCalledWith(
      testV2Pipeline.pipeline_id,
      originalTestV2PipelineVersion.pipeline_version_id!,
    );
  });

  it('shows error dialog if deletion fails', async () => {
    tree = renderPipelineDetailsElement(
      <PipelineDetailsPage {...generateProps(PIPELINE_VERSION_ID)} />,
    );
    TestUtils.makeErrorResponseOnce(deletePipelineVersionSpy, 'woops');
    await TestUtils.flushPromises();
    const deleteBtn = (tree.instance() as PipelineDetailsPage).getInitialToolbarState().actions[
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
    tree = renderPipelineDetailsElement(
      <PipelineDetailsPage {...generateProps(PIPELINE_VERSION_ID)} />,
    );
    await TestUtils.flushPromises();
    const deleteBtn = (tree.instance() as PipelineDetailsPage).getInitialToolbarState().actions[
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
