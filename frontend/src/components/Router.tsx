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

import Button from '@material-ui/core/Button';
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogContent from '@material-ui/core/DialogContent';
import DialogTitle from '@material-ui/core/DialogTitle';
import Snackbar, { SnackbarProps } from '@material-ui/core/Snackbar';
import * as React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';
import Compare from 'src/pages/Compare';
import FrontendFeatures from 'src/pages/FrontendFeatures';
import RunDetailsRouter from 'src/pages/RunDetailsRouter';
import { classes, stylesheet } from 'typestyle';
import Banner, { BannerProps } from '../components/Banner';
import { commonCss } from '../Css';
import { Deployments, KFP_FLAGS } from '../lib/Flags';
import Page404 from '../pages/404';
import AllExperimentsAndArchive, {
  AllExperimentsAndArchiveTab,
} from '../pages/AllExperimentsAndArchive';
import AllRecurringRunsList from '../pages/AllRecurringRunsList';
import AllRunsAndArchive, { AllRunsAndArchiveTab } from '../pages/AllRunsAndArchive';
import ArtifactDetails from '../pages/ArtifactDetails';
import ArtifactList from '../pages/ArtifactList';
import ExecutionDetails from '../pages/ExecutionDetails';
import ExecutionList from '../pages/ExecutionList';
import ExperimentDetails from '../pages/ExperimentDetails';
import { GettingStarted } from '../pages/GettingStarted';
import NewExperiment from '../pages/NewExperiment';
import NewPipelineVersion from '../pages/NewPipelineVersion';
import NewRunSwitcher from '../pages/NewRunSwitcher';
import PipelineDetails from '../pages/PipelineDetails';
import PrivateAndSharedPipelines, { PrivateAndSharedTab } from '../pages/PrivateAndSharedPipelines';
import RecurringRunDetailsRouter from 'src/pages/RecurringRunDetailsRouter';
import SideNav from './SideNav';
import Toolbar, { ToolbarProps } from './Toolbar';
import { BuildInfoContext } from 'src/lib/BuildInfo';

export type RouteConfig = {
  path: string;
  Component: React.ComponentType<any>;
  view?: any;
  notExact?: boolean;
};

const css = stylesheet({
  dialog: {
    minWidth: 250,
  },
});

export enum QUERY_PARAMS {
  cloneFromRun = 'cloneFromRun',
  cloneFromRecurringRun = 'cloneFromRecurringRun',
  experimentId = 'experimentId',
  isRecurring = 'recurring',
  firstRunInExperiment = 'firstRunInExperiment',
  pipelineId = 'pipelineId',
  pipelineVersionId = 'pipelineVersionId',
  fromRunId = 'fromRun',
  fromRecurringRunId = 'fromRecurringRun',
  runlist = 'runlist',
  view = 'view',
}

export enum RouteParams {
  experimentId = 'eid',
  pipelineId = 'pid',
  pipelineVersionId = 'vid',
  runId = 'rid',
  recurringRunId = 'rrid',
  // TODO: create one of these for artifact and execution?
  ID = 'id',
  executionId = 'executionid',
}

// tslint:disable-next-line:variable-name
export const RoutePrefix = {
  ARTIFACT: '/artifact',
  EXECUTION: '/execution',
  RECURRING_RUN: '/recurringrun',
};

// tslint:disable-next-line:variable-name
export const RoutePage = {
  ARCHIVED_RUNS: '/archive/runs',
  ARCHIVED_EXPERIMENTS: '/archive/experiments',
  ARTIFACTS: '/artifacts',
  ARTIFACT_DETAILS: `/artifacts/:${RouteParams.ID}`,
  COMPARE: `/compare`,
  EXECUTIONS: '/executions',
  EXECUTION_DETAILS: `/executions/:${RouteParams.ID}`,
  EXPERIMENTS: '/experiments',
  EXPERIMENT_DETAILS: `/experiments/details/:${RouteParams.experimentId}`,
  NEW_EXPERIMENT: '/experiments/new',
  NEW_PIPELINE_VERSION: '/pipeline_versions/new',
  NEW_RUN: '/runs/new',
  PIPELINES: '/pipelines',
  PIPELINES_SHARED: '/shared/pipelines',
  PIPELINE_DETAILS: `/pipelines/details/:${RouteParams.pipelineId}/version/:${RouteParams.pipelineVersionId}?`,
  PIPELINE_DETAILS_NO_VERSION: `/pipelines/details/:${RouteParams.pipelineId}?`, // pipelineId is optional
  RUNS: '/runs',
  RUN_DETAILS: `/runs/details/:${RouteParams.runId}`,
  RUN_DETAILS_WITH_EXECUTION: `/runs/details/:${RouteParams.runId}/execution/:${RouteParams.executionId}`,
  RECURRING_RUNS: '/recurringruns',
  RECURRING_RUN_DETAILS: `/recurringrun/details/:${RouteParams.recurringRunId}`,
  START: '/start',
  FRONTEND_FEATURES: '/frontend_features',
};

export const RoutePageFactory = {
  artifactDetails: (artifactId: number) => {
    return RoutePage.ARTIFACT_DETAILS.replace(`:${RouteParams.ID}`, '' + artifactId);
  },
  executionDetails: (executionId: number) => {
    return RoutePage.EXECUTION_DETAILS.replace(`:${RouteParams.ID}`, '' + executionId);
  },
  pipelineDetails: (id: string) => {
    return RoutePage.PIPELINE_DETAILS_NO_VERSION.replace(`:${RouteParams.pipelineId}`, id);
  },
};

export const ExternalLinks = {
  DOCUMENTATION: 'https://www.kubeflow.org/docs/pipelines/',
  GITHUB: 'https://github.com/kubeflow/pipelines',
  GITHUB_ISSUE: 'https://github.com/kubeflow/pipelines/issues/new/choose',
};

export interface DialogProps {
  buttons?: Array<{ onClick?: () => any; text: string }>;
  // TODO: This should be generalized to any react component.
  content?: string;
  onClose?: () => any;
  open?: boolean;
  title?: string;
}

interface RouteComponentState {
  bannerProps: BannerProps;
  dialogProps: DialogProps;
  snackbarProps: SnackbarProps;
  toolbarProps: ToolbarProps;
}

export interface RouterProps {
  configs?: RouteConfig[]; // only used in tests
}

const DEFAULT_ROUTE =
  KFP_FLAGS.DEPLOYMENT === Deployments.MARKETPLACE ? RoutePage.START : RoutePage.PIPELINES;

// This component is made as a wrapper to separate toolbar state for different pages.
const Router: React.FC<RouterProps> = ({ configs }) => {
  const buildInfo = React.useContext(BuildInfoContext);

  let routes: RouteConfig[] = configs || [
    { path: RoutePage.START, Component: GettingStarted },
    {
      Component: AllRunsAndArchive,
      path: RoutePage.ARCHIVED_RUNS,
      view: AllRunsAndArchiveTab.ARCHIVE,
    },
    {
      Component: AllExperimentsAndArchive,
      path: RoutePage.ARCHIVED_EXPERIMENTS,
      view: AllExperimentsAndArchiveTab.ARCHIVE,
    },
    { path: RoutePage.ARTIFACTS, Component: ArtifactList },
    { path: RoutePage.ARTIFACT_DETAILS, Component: ArtifactDetails, notExact: true },
    { path: RoutePage.EXECUTIONS, Component: ExecutionList },
    { path: RoutePage.EXECUTION_DETAILS, Component: ExecutionDetails },
    {
      Component: AllExperimentsAndArchive,
      path: RoutePage.EXPERIMENTS,
      view: AllExperimentsAndArchiveTab.EXPERIMENTS,
    },
    { path: RoutePage.EXPERIMENT_DETAILS, Component: ExperimentDetails },
    { path: RoutePage.NEW_EXPERIMENT, Component: NewExperiment },
    { path: RoutePage.NEW_PIPELINE_VERSION, Component: NewPipelineVersion },
    { path: RoutePage.NEW_RUN, Component: NewRunSwitcher },
    {
      path: RoutePage.PIPELINES,
      Component: PrivateAndSharedPipelines,
      view: PrivateAndSharedTab.PRIVATE,
    },
    {
      path: RoutePage.PIPELINES_SHARED,
      Component: PrivateAndSharedPipelines,
      view: PrivateAndSharedTab.SHARED,
    },
    { path: RoutePage.PIPELINE_DETAILS, Component: PipelineDetails },
    { path: RoutePage.PIPELINE_DETAILS_NO_VERSION, Component: PipelineDetails },
    { path: RoutePage.RUNS, Component: AllRunsAndArchive, view: AllRunsAndArchiveTab.RUNS },
    { path: RoutePage.RECURRING_RUNS, Component: AllRecurringRunsList },
    { path: RoutePage.RECURRING_RUN_DETAILS, Component: RecurringRunDetailsRouter },
    { path: RoutePage.RUN_DETAILS, Component: RunDetailsRouter },
    { path: RoutePage.RUN_DETAILS_WITH_EXECUTION, Component: RunDetailsRouter },
    { path: RoutePage.COMPARE, Component: Compare },
    { path: RoutePage.FRONTEND_FEATURES, Component: FrontendFeatures },
  ];

  if (!buildInfo?.apiServerMultiUser) {
    routes = routes.filter(r => r.path !== RoutePage.PIPELINES_SHARED);
  }

  return (
    // There will be only one instance of SideNav, throughout UI usage.
    <SideNavLayout>
      <Switch>
        <Route
          exact={true}
          path={'/'}
          render={({ ...props }) => <Redirect to={DEFAULT_ROUTE} {...props} />}
        />

        {/* Normal routes */}
        {routes.map((route, i) => {
          const { path } = { ...route };
          return (
            // Setting a key here, so that two different routes are considered two instances from
            // react. Therefore, they don't share toolbar state. This avoids many bugs like dangling
            // network response handlers.
            <Route
              key={i}
              exact={!route.notExact}
              path={path}
              render={props => <RoutedPage key={props.location.key} route={route} />}
            />
          );
        })}

        {/* 404 */}
        {
          <Route>
            <RoutedPage />
          </Route>
        }
      </Switch>
    </SideNavLayout>
  );
};

class RoutedPage extends React.Component<{ route?: RouteConfig }, RouteComponentState> {
  private childProps = {
    toolbarProps: {
      breadcrumbs: [{ displayName: '', href: '' }],
      actions: {},
      pageTitle: '',
    } as ToolbarProps,
    updateBanner: this._updateBanner.bind(this),
    updateDialog: this._updateDialog.bind(this),
    updateSnackbar: this._updateSnackbar.bind(this),
    updateToolbar: this._updateToolbar.bind(this),
  };

  constructor(props: any) {
    super(props);

    this.state = {
      bannerProps: {},
      dialogProps: { open: false },
      snackbarProps: { autoHideDuration: 5000, open: false },
      toolbarProps: { breadcrumbs: [{ displayName: '', href: '' }], actions: [], ...props },
    };
  }

  public render(): JSX.Element {
    this.childProps.toolbarProps = this.state.toolbarProps;
    const route = this.props.route;

    return (
      <div className={classes(commonCss.page)}>
        <Route render={({ ...props }) => <Toolbar {...this.state.toolbarProps} {...props} />} />
        {this.state.bannerProps.message && (
          <Banner
            message={this.state.bannerProps.message}
            mode={this.state.bannerProps.mode}
            additionalInfo={this.state.bannerProps.additionalInfo}
            refresh={this.state.bannerProps.refresh}
            showTroubleshootingGuideLink={true}
          />
        )}
        <Switch>
          {route &&
            (() => {
              const { path, Component, ...otherProps } = { ...route };
              return (
                <Route
                  exact={!route.notExact}
                  path={path}
                  render={({ ...props }) => (
                    <Component {...props} {...this.childProps} {...otherProps} />
                  )}
                />
              );
            })()}

          {/* 404 */}
          {!!route && (
            <Route render={({ ...props }) => <Page404 {...props} {...this.childProps} />} />
          )}
        </Switch>

        <Snackbar
          autoHideDuration={this.state.snackbarProps.autoHideDuration}
          message={this.state.snackbarProps.message}
          open={this.state.snackbarProps.open}
          onClose={this._handleSnackbarClose.bind(this)}
        />

        <Dialog
          open={this.state.dialogProps.open !== false}
          classes={{ paper: css.dialog }}
          className='dialog'
          onClose={() => this._handleDialogClosed()}
        >
          {this.state.dialogProps.title && (
            <DialogTitle> {this.state.dialogProps.title}</DialogTitle>
          )}
          {this.state.dialogProps.content && (
            <DialogContent className={commonCss.prewrap}>
              {this.state.dialogProps.content}
            </DialogContent>
          )}
          {this.state.dialogProps.buttons && (
            <DialogActions>
              {this.state.dialogProps.buttons.map((b, i) => (
                <Button
                  key={i}
                  onClick={() => this._handleDialogClosed(b.onClick)}
                  className='dialogButton'
                  color='secondary'
                >
                  {b.text}
                </Button>
              ))}
            </DialogActions>
          )}
        </Dialog>
      </div>
    );
  }

  private _updateDialog(dialogProps: DialogProps): void {
    // Assuming components will want to open the dialog by defaut.
    if (dialogProps.open === undefined) {
      dialogProps.open = true;
    }
    this.setState({ dialogProps });
  }

  private _updateToolbar(newToolbarProps: Partial<ToolbarProps>): void {
    const toolbarProps = Object.assign(this.state.toolbarProps, newToolbarProps);
    this.setState({ toolbarProps });
  }

  private _updateBanner(bannerProps: BannerProps): void {
    this.setState({ bannerProps });
  }

  private _updateSnackbar(snackbarProps: SnackbarProps): void {
    snackbarProps.autoHideDuration =
      snackbarProps.autoHideDuration || this.state.snackbarProps.autoHideDuration;
    this.setState({ snackbarProps });
  }

  private _handleDialogClosed(onClick?: () => void): void {
    this.setState({ dialogProps: { open: false } });
    if (onClick) {
      onClick();
    }
    if (this.state.dialogProps.onClose) {
      this.state.dialogProps.onClose();
    }
  }
  private _handleSnackbarClose(): void {
    this.setState({ snackbarProps: { open: false, message: '' } });
  }
}

// TODO: loading/error experience until backend is reachable

export default Router;

const SideNavLayout: React.FC<{}> = ({ children }) => (
  <div className={classes(commonCss.page)}>
    <div className={classes(commonCss.flexGrow)}>
      <Route render={({ ...props }) => <SideNav page={props.location.pathname} {...props} />} />
      {children}
    </div>
  </div>
);
