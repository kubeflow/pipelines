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

import { mount, ReactWrapper, shallow, ShallowWrapper } from 'enzyme';
import * as React from 'react';
import { MemoryRouter, RouterProps } from 'react-router';
import { Apis } from '../lib/Apis';
import { LocalStorage } from '../lib/LocalStorage';
import { RoutePage } from './Router';
import { css, SideNav } from './SideNav';
import { GkeMetadata } from '../lib/GkeMetadata';
import { createMemoryHistory } from 'history';
import TestUtils from '../TestUtils';

const wideWidth = 1000;
const narrowWidth = 200;
const isCollapsed = (tree: ShallowWrapper<any>) =>
  tree.find('WithStyles(IconButton)').hasClass(css.collapsedChevron);
const routerProps: RouterProps = { history: {} as any };
const defaultProps = {
  ...routerProps,
  gkeMetadata: {},
  buildInfo: {
    apiServerCommitHash: '0a7b9e38f2b9bcdef4bbf3234d971e1635b50cd5',
    apiServerTagName: '1.0.0',
    apiServerReady: true,
    buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
    frontendCommitHash: '302e93ce99099173f387c7e0635476fe1b69ea98',
    frontendTagName: '1.0.0-rc1',
  },
};

describe('SideNav', () => {
  let tree: ReactWrapper | ShallowWrapper;

  const consoleErrorSpy = jest.spyOn(console, 'error');
  const checkHubSpy = jest.spyOn(Apis, 'isJupyterHubAvailable');
  const localStorageHasKeySpy = jest.spyOn(LocalStorage, 'hasKey');
  const localStorageIsCollapsedSpy = jest.spyOn(LocalStorage, 'isNavbarCollapsed');

  beforeEach(() => {
    jest.clearAllMocks();

    consoleErrorSpy.mockImplementation(() => null);
    checkHubSpy.mockImplementation(() => ({ ok: true }));
    localStorageHasKeySpy.mockImplementation(() => false);
    localStorageIsCollapsedSpy.mockImplementation(() => false);
  });

  afterEach(async () => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    await tree.unmount();
    jest.resetAllMocks();
    (window as any).innerWidth = wideWidth;
  });

  it('renders expanded state', () => {
    localStorageHasKeySpy.mockImplementationOnce(() => false);
    (window as any).innerWidth = wideWidth;
    tree = shallow(<SideNav page={RoutePage.PIPELINES} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders collapsed state', () => {
    localStorageHasKeySpy.mockImplementationOnce(() => false);
    (window as any).innerWidth = narrowWidth;
    tree = shallow(<SideNav page={RoutePage.PIPELINES} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders Pipelines as active page', () => {
    tree = shallow(<SideNav page={RoutePage.PIPELINES} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders Pipelines as active when on PipelineDetails page', () => {
    tree = shallow(<SideNav page={RoutePage.PIPELINE_DETAILS} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page', () => {
    tree = shallow(<SideNav page={RoutePage.EXPERIMENTS} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active when on ExperimentDetails page', () => {
    tree = shallow(<SideNav page={RoutePage.EXPERIMENT_DETAILS} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on NewExperiment page', () => {
    tree = shallow(<SideNav page={RoutePage.NEW_EXPERIMENT} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on Compare page', () => {
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on AllRuns page', () => {
    tree = shallow(<SideNav page={RoutePage.RUNS} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on RunDetails page', () => {
    tree = shallow(<SideNav page={RoutePage.RUN_DETAILS} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on RecurringRunDetails page', () => {
    tree = shallow(<SideNav page={RoutePage.RECURRING_RUN_DETAILS} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on NewRun page', () => {
    tree = shallow(<SideNav page={RoutePage.NEW_RUN} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders recurring runs as active page', () => {
    tree = shallow(<SideNav page={RoutePage.RECURRING_RUNS} {...defaultProps} />);
    expect(tree).toMatchInlineSnapshot(`
      <div
        className="root flexColumn noShrink"
        id="sideNav"
      >
        <div
          style={
            Object {
              "flexGrow": 1,
            }
          }
        >
          <div
            className="indicator indicatorHidden"
          />
          <WithStyles(Tooltip)
            disableFocusListener={true}
            disableHoverListener={true}
            disableTouchListener={true}
            enterDelay={300}
            placement="right-start"
            title="Pipeline List"
          >
            <Link
              className="unstyled"
              id="pipelinesBtn"
              replace={false}
              to="/pipelines"
            >
              <WithStyles(Button)
                className="button"
              >
                <div
                  className="flex flex-row flex-shrink-0"
                >
                  <div
                    className="alignItems"
                  >
                    <PipelinesIcon
                      color="#9aa0a6"
                    />
                  </div>
                  <span
                    className="label"
                  >
                    Pipelines
                  </span>
                </div>
              </WithStyles(Button)>
            </Link>
          </WithStyles(Tooltip)>
          <div
            className="indicator indicatorHidden"
          />
          <WithStyles(Tooltip)
            disableFocusListener={true}
            disableHoverListener={true}
            disableTouchListener={true}
            enterDelay={300}
            placement="right-start"
            title="Experiment List"
          >
            <Link
              className="unstyled"
              id="experimentsBtn"
              replace={false}
              to="/experiments"
            >
              <WithStyles(Button)
                className="button"
              >
                <div
                  className="flex flex-row flex-shrink-0"
                >
                  <div
                    className="alignItems"
                  >
                    <ExperimentsIcon
                      color="#9aa0a6"
                    />
                  </div>
                  <span
                    className="label"
                  >
                    Experiments
                  </span>
                </div>
              </WithStyles(Button)>
            </Link>
          </WithStyles(Tooltip)>
          <div
            className="indicator indicatorHidden"
          />
          <WithStyles(Tooltip)
            disableFocusListener={true}
            disableHoverListener={true}
            disableTouchListener={true}
            enterDelay={300}
            placement="right-start"
            title="Runs List"
          >
            <Link
              className="unstyled"
              id="runsBtn"
              replace={false}
              to="/runs"
            >
              <WithStyles(Button)
                className="button"
              >
                <div
                  className="flex flex-row flex-shrink-0"
                >
                  <pure(DirectionsRunIcon) />
                  <span
                    className="label"
                  >
                    Runs
                  </span>
                </div>
              </WithStyles(Button)>
            </Link>
          </WithStyles(Tooltip)>
          <div
            className="indicator"
          />
          <WithStyles(Tooltip)
            disableFocusListener={true}
            disableHoverListener={true}
            disableTouchListener={true}
            enterDelay={300}
            placement="right-start"
            title="Recurring Runs List"
          >
            <Link
              className="unstyled"
              id="recurringRunsBtn"
              replace={false}
              to="/recurringruns"
            >
              <WithStyles(Button)
                className="button active"
              >
                <div
                  className="flex flex-row flex-shrink-0"
                >
                  <pure(AlarmIcon) />
                  <span
                    className="label"
                  >
                    Recurring Runs
                  </span>
                </div>
              </WithStyles(Button)>
            </Link>
          </WithStyles(Tooltip)>
          <div
            className="indicator indicatorHidden"
          />
          <WithStyles(Tooltip)
            disableFocusListener={true}
            disableHoverListener={true}
            disableTouchListener={true}
            enterDelay={300}
            placement="right-start"
            title="Artifacts List"
          >
            <Link
              className="unstyled"
              id="artifactsBtn"
              replace={false}
              to="/artifacts"
            >
              <WithStyles(Button)
                className="button"
              >
                <div
                  className="flex flex-row flex-shrink-0"
                >
                  <pure(BubbleChartIcon) />
                  <span
                    className="label"
                  >
                    Artifacts
                  </span>
                </div>
              </WithStyles(Button)>
            </Link>
          </WithStyles(Tooltip)>
          <div
            className="indicator indicatorHidden"
          />
          <WithStyles(Tooltip)
            disableFocusListener={true}
            disableHoverListener={true}
            disableTouchListener={true}
            enterDelay={300}
            placement="right-start"
            title="Executions List"
          >
            <Link
              className="unstyled"
              id="executionsBtn"
              replace={false}
              to="/executions"
            >
              <WithStyles(Button)
                className="button"
              >
                <div
                  className="flex flex-row flex-shrink-0"
                >
                  <pure(PlayArrowIcon) />
                  <span
                    className="label"
                  >
                    Executions
                  </span>
                </div>
              </WithStyles(Button)>
            </Link>
          </WithStyles(Tooltip)>
          <hr
            className="separator"
          />
          <ExternalUri
            collapsed={false}
            icon={[Function]}
            title="Documentation"
            to="https://www.kubeflow.org/docs/pipelines/"
          />
          <ExternalUri
            collapsed={false}
            icon={[Function]}
            title="Github Repo"
            to="https://github.com/kubeflow/pipelines"
          />
          <hr
            className="separator"
          />
          <WithStyles(IconButton)
            className="chevron"
            onClick={[Function]}
          >
            <pure(ChevronLeftIcon) />
          </WithStyles(IconButton)>
        </div>
        <div
          className="infoVisible"
        >
          <WithStyles(Tooltip)
            enterDelay={300}
            placement="top-start"
            title="Build date: 10/23/2018, Commit hash: 0a7b9e3"
          >
            <div
              className="envMetadata"
            >
              <span>
                Version: 
              </span>
              <a
                className="link unstyled"
                href="https://www.github.com/kubeflow/pipelines/commit/0a7b9e38f2b9bcdef4bbf3234d971e1635b50cd5"
                rel="noopener"
                target="_blank"
              >
                1.0.0
              </a>
            </div>
          </WithStyles(Tooltip)>
          <WithStyles(Tooltip)
            enterDelay={300}
            placement="top-start"
            title="Report an Issue"
          >
            <div
              className="envMetadata"
            >
              <a
                className="link unstyled"
                href="https://github.com/kubeflow/pipelines/issues/new/choose"
                rel="noopener"
                target="_blank"
              >
                Report an Issue
              </a>
            </div>
          </WithStyles(Tooltip)>
        </div>
      </div>
    `);
  });

  it('renders jobs as active page when on JobDetails page', () => {
    tree = shallow(<SideNav page={RoutePage.RECURRING_RUN_DETAILS} {...defaultProps} />);
    expect(tree).toMatchInlineSnapshot(`
      <div
        className="root flexColumn noShrink"
        id="sideNav"
      >
        <div
          style={
            Object {
              "flexGrow": 1,
            }
          }
        >
          <div
            className="indicator indicatorHidden"
          />
          <WithStyles(Tooltip)
            disableFocusListener={true}
            disableHoverListener={true}
            disableTouchListener={true}
            enterDelay={300}
            placement="right-start"
            title="Pipeline List"
          >
            <Link
              className="unstyled"
              id="pipelinesBtn"
              replace={false}
              to="/pipelines"
            >
              <WithStyles(Button)
                className="button"
              >
                <div
                  className="flex flex-row flex-shrink-0"
                >
                  <div
                    className="alignItems"
                  >
                    <PipelinesIcon
                      color="#9aa0a6"
                    />
                  </div>
                  <span
                    className="label"
                  >
                    Pipelines
                  </span>
                </div>
              </WithStyles(Button)>
            </Link>
          </WithStyles(Tooltip)>
          <div
            className="indicator indicatorHidden"
          />
          <WithStyles(Tooltip)
            disableFocusListener={true}
            disableHoverListener={true}
            disableTouchListener={true}
            enterDelay={300}
            placement="right-start"
            title="Experiment List"
          >
            <Link
              className="unstyled"
              id="experimentsBtn"
              replace={false}
              to="/experiments"
            >
              <WithStyles(Button)
                className="button"
              >
                <div
                  className="flex flex-row flex-shrink-0"
                >
                  <div
                    className="alignItems"
                  >
                    <ExperimentsIcon
                      color="#9aa0a6"
                    />
                  </div>
                  <span
                    className="label"
                  >
                    Experiments
                  </span>
                </div>
              </WithStyles(Button)>
            </Link>
          </WithStyles(Tooltip)>
          <div
            className="indicator indicatorHidden"
          />
          <WithStyles(Tooltip)
            disableFocusListener={true}
            disableHoverListener={true}
            disableTouchListener={true}
            enterDelay={300}
            placement="right-start"
            title="Runs List"
          >
            <Link
              className="unstyled"
              id="runsBtn"
              replace={false}
              to="/runs"
            >
              <WithStyles(Button)
                className="button"
              >
                <div
                  className="flex flex-row flex-shrink-0"
                >
                  <pure(DirectionsRunIcon) />
                  <span
                    className="label"
                  >
                    Runs
                  </span>
                </div>
              </WithStyles(Button)>
            </Link>
          </WithStyles(Tooltip)>
          <div
            className="indicator"
          />
          <WithStyles(Tooltip)
            disableFocusListener={true}
            disableHoverListener={true}
            disableTouchListener={true}
            enterDelay={300}
            placement="right-start"
            title="Recurring Runs List"
          >
            <Link
              className="unstyled"
              id="recurringRunsBtn"
              replace={false}
              to="/recurringruns"
            >
              <WithStyles(Button)
                className="button active"
              >
                <div
                  className="flex flex-row flex-shrink-0"
                >
                  <pure(AlarmIcon) />
                  <span
                    className="label"
                  >
                    Recurring Runs
                  </span>
                </div>
              </WithStyles(Button)>
            </Link>
          </WithStyles(Tooltip)>
          <div
            className="indicator indicatorHidden"
          />
          <WithStyles(Tooltip)
            disableFocusListener={true}
            disableHoverListener={true}
            disableTouchListener={true}
            enterDelay={300}
            placement="right-start"
            title="Artifacts List"
          >
            <Link
              className="unstyled"
              id="artifactsBtn"
              replace={false}
              to="/artifacts"
            >
              <WithStyles(Button)
                className="button"
              >
                <div
                  className="flex flex-row flex-shrink-0"
                >
                  <pure(BubbleChartIcon) />
                  <span
                    className="label"
                  >
                    Artifacts
                  </span>
                </div>
              </WithStyles(Button)>
            </Link>
          </WithStyles(Tooltip)>
          <div
            className="indicator indicatorHidden"
          />
          <WithStyles(Tooltip)
            disableFocusListener={true}
            disableHoverListener={true}
            disableTouchListener={true}
            enterDelay={300}
            placement="right-start"
            title="Executions List"
          >
            <Link
              className="unstyled"
              id="executionsBtn"
              replace={false}
              to="/executions"
            >
              <WithStyles(Button)
                className="button"
              >
                <div
                  className="flex flex-row flex-shrink-0"
                >
                  <pure(PlayArrowIcon) />
                  <span
                    className="label"
                  >
                    Executions
                  </span>
                </div>
              </WithStyles(Button)>
            </Link>
          </WithStyles(Tooltip)>
          <hr
            className="separator"
          />
          <ExternalUri
            collapsed={false}
            icon={[Function]}
            title="Documentation"
            to="https://www.kubeflow.org/docs/pipelines/"
          />
          <ExternalUri
            collapsed={false}
            icon={[Function]}
            title="Github Repo"
            to="https://github.com/kubeflow/pipelines"
          />
          <hr
            className="separator"
          />
          <WithStyles(IconButton)
            className="chevron"
            onClick={[Function]}
          >
            <pure(ChevronLeftIcon) />
          </WithStyles(IconButton)>
        </div>
        <div
          className="infoVisible"
        >
          <WithStyles(Tooltip)
            enterDelay={300}
            placement="top-start"
            title="Build date: 10/23/2018, Commit hash: 0a7b9e3"
          >
            <div
              className="envMetadata"
            >
              <span>
                Version: 
              </span>
              <a
                className="link unstyled"
                href="https://www.github.com/kubeflow/pipelines/commit/0a7b9e38f2b9bcdef4bbf3234d971e1635b50cd5"
                rel="noopener"
                target="_blank"
              >
                1.0.0
              </a>
            </div>
          </WithStyles(Tooltip)>
          <WithStyles(Tooltip)
            enterDelay={300}
            placement="top-start"
            title="Report an Issue"
          >
            <div
              className="envMetadata"
            >
              <a
                className="link unstyled"
                href="https://github.com/kubeflow/pipelines/issues/new/choose"
                rel="noopener"
                target="_blank"
              >
                Report an Issue
              </a>
            </div>
          </WithStyles(Tooltip)>
        </div>
      </div>
    `);
  });

  it('show jupyterhub link if accessible', () => {
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...defaultProps} />);
    tree.setState({ jupyterHubAvailable: true });
    expect(tree).toMatchSnapshot();
  });

  it('collapses if collapse state is true localStorage', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => true);
    localStorageHasKeySpy.mockImplementationOnce(() => true);

    (window as any).innerWidth = wideWidth;
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...defaultProps} />);
    expect(isCollapsed(tree)).toBe(true);
  });

  it('expands if collapse state is false in localStorage', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => true);

    tree = shallow(<SideNav page={RoutePage.COMPARE} {...defaultProps} />);
    expect(isCollapsed(tree)).toBe(false);
  });

  it('collapses if no collapse state in localStorage, and window is too narrow', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);

    (window as any).innerWidth = narrowWidth;
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...defaultProps} />);
    expect(isCollapsed(tree)).toBe(true);
  });

  it('expands if no collapse state in localStorage, and window is wide', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);

    (window as any).innerWidth = wideWidth;
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...defaultProps} />);
    expect(isCollapsed(tree)).toBe(false);
  });

  it('collapses if no collapse state in localStorage, and window goes from wide to narrow', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);

    (window as any).innerWidth = wideWidth;
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...defaultProps} />);
    expect(isCollapsed(tree)).toBe(false);

    (window as any).innerWidth = narrowWidth;
    const resizeEvent = new Event('resize');
    window.dispatchEvent(resizeEvent);
    expect(isCollapsed(tree)).toBe(true);
  });

  it('expands if no collapse state in localStorage, and window goes from narrow to wide', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);

    (window as any).innerWidth = narrowWidth;
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...defaultProps} />);
    expect(isCollapsed(tree)).toBe(true);

    (window as any).innerWidth = wideWidth;
    const resizeEvent = new Event('resize');
    window.dispatchEvent(resizeEvent);
    expect(isCollapsed(tree)).toBe(false);
  });

  it('saves state in localStorage if chevron is clicked', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);
    const spy = jest.spyOn(LocalStorage, 'saveNavbarCollapsed');

    (window as any).innerWidth = narrowWidth;
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...defaultProps} />);
    expect(isCollapsed(tree)).toBe(true);

    tree.find('WithStyles(IconButton)').simulate('click');
    expect(spy).toHaveBeenCalledWith(false);
  });

  it('does not collapse if collapse state is saved in localStorage, and window resizes', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => true);

    (window as any).innerWidth = wideWidth;
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...defaultProps} />);
    expect(isCollapsed(tree)).toBe(false);

    (window as any).innerWidth = narrowWidth;
    const resizeEvent = new Event('resize');
    window.dispatchEvent(resizeEvent);
    expect(isCollapsed(tree)).toBe(false);
  });

  it('populates the display build information using the default props', async () => {
    tree = TestUtils.mountWithRouter(<SideNav page={RoutePage.PIPELINES} {...defaultProps} />);
    const instance = tree.find(SideNav).instance() as any;

    expect(tree).toMatchSnapshot();

    const buildInfo = defaultProps.buildInfo;
    expect(instance._getBuildInfo()).toEqual({
      tagName: buildInfo.apiServerTagName,
      commitHash: buildInfo.apiServerCommitHash.substring(0, 7),
      commitUrl:
        'https://www.github.com/kubeflow/pipelines/commit/' + buildInfo.apiServerCommitHash,
      date: new Date(buildInfo.buildDate).toLocaleDateString(),
    });
  });

  it('display the correct GKE metadata', async () => {
    const clusterName = 'some-cluster-name';
    const projectId = 'some-project-id';
    const gkeMetadata: GkeMetadata = { clusterName, projectId };

    const newProps = {
      ...defaultProps,
      gkeMetadata,
      buildInfo: {},
    };

    tree = TestUtils.mountWithRouter(<SideNav page={RoutePage.PIPELINES} {...newProps} />);

    expect(tree).toMatchSnapshot();
  });

  it('displays the frontend tag name if the api server hash is not returned', async () => {
    const newProps = {
      ...defaultProps,
      buildInfo: {
        apiServerReady: true,
        // No apiServerCommitHash or apiServerTagName
        buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
        frontendCommitHash: '302e93ce99099173f387c7e0635476fe1b69ea98',
        frontendTagName: '1.0.0',
      },
    };

    tree = TestUtils.mountWithRouter(<SideNav page={RoutePage.PIPELINES} {...newProps} />);
    const instance = tree.find(SideNav).instance() as any;

    expect(instance._getBuildInfo()).toEqual(
      expect.objectContaining({
        commitHash: newProps.buildInfo.frontendCommitHash.substring(0, 7),
        tagName: newProps.buildInfo.frontendTagName,
      }),
    );
  });

  it('uses the frontend commit hash for the link URL if the api server hash is not returned', async () => {
    const newProps = {
      ...defaultProps,
      buildInfo: {
        apiServerReady: true,
        // No apiServerCommitHash
        buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
        frontendCommitHash: '302e93ce99099173f387c7e0635476fe1b69ea98',
      },
    };

    tree = TestUtils.mountWithRouter(<SideNav page={RoutePage.PIPELINES} {...newProps} />);
    const instance = tree.find(SideNav).instance() as any;

    expect(instance._getBuildInfo()).toEqual(
      expect.objectContaining({
        commitUrl:
          'https://www.github.com/kubeflow/pipelines/commit/' +
          newProps.buildInfo.frontendCommitHash,
      }),
    );
  });

  it("displays 'unknown' if the frontend and api server tag names/commit hashes are not returned", async () => {
    const newProps = {
      ...defaultProps,
      buildInfo: {
        apiServerReady: true,
        // No apiServerCommitHash
        buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
        // No frontendCommitHash
      },
    };

    tree = TestUtils.mountWithRouter(<SideNav page={RoutePage.PIPELINES} {...newProps} />);
    const instance = tree.find(SideNav).instance() as any;

    expect(instance._getBuildInfo()).toEqual(
      expect.objectContaining({
        commitHash: 'unknown',
        tagName: 'unknown',
      }),
    );
  });

  it('links to the github repo root if the frontend and api server commit hashes are not returned', async () => {
    const newProps = {
      ...defaultProps,
      buildInfo: {
        apiServerReady: true,
        // No apiServerCommitHash
        buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
        // No frontendCommitHash
      },
    };

    tree = TestUtils.mountWithRouter(<SideNav page={RoutePage.PIPELINES} {...newProps} />);
    const instance = tree.find(SideNav).instance() as any;
    expect(instance._getBuildInfo()).toEqual(
      expect.objectContaining({
        commitUrl: 'https://www.github.com/kubeflow/pipelines',
      }),
    );
  });

  it("displays 'unknown' if the date is not returned", async () => {
    const newProps = {
      ...defaultProps,
      buildInfo: {
        apiServerCommitHash: '0a7b9e38f2b9bcdef4bbf3234d971e1635b50cd5',
        apiServerReady: true,
        // No buildDate
        frontendCommitHash: '302e93ce99099173f387c7e0635476fe1b69ea98',
      },
    };

    tree = TestUtils.mountWithRouter(<SideNav page={RoutePage.PIPELINES} {...newProps} />);
    const instance = tree.find(SideNav).instance() as any;

    expect(instance._getBuildInfo()).toEqual(
      expect.objectContaining({
        date: 'unknown',
      }),
    );
  });
});
