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

import { SnackbarProps } from '@mui/material/Snackbar';
import * as React from 'react';
import { Navigate, Route, Routes, useLocation, useNavigate, matchPath } from 'react-router';
import { NavigationProps } from 'src/lib/Navigation';
import Page404 from 'src/pages/404';
import Compare from 'src/pages/Compare';
import FrontendFeatures from 'src/pages/FrontendFeatures';
import RunDetailsRouter from 'src/pages/RunDetailsRouter';
import { classes, stylesheet } from 'typestyle';
import Banner, { BannerProps } from 'src/components/Banner';
import { commonCss } from 'src/Css';
import { Deployments, KFP_FLAGS } from 'src/lib/Flags';
import AllExperimentsAndArchive, {
  AllExperimentsAndArchiveTab,
} from 'src/pages/AllExperimentsAndArchive';
import AllRecurringRunsPage from 'src/pages/AllRecurringRunsList';
import AllRunsAndArchive, { AllRunsAndArchiveTab } from 'src/pages/AllRunsAndArchive';
import ArtifactDetails from 'src/pages/ArtifactDetails';
import EnhancedArtifactList from 'src/pages/ArtifactList';
import ExperimentDetailsPage from 'src/pages/ExperimentDetails';
import { GettingStarted } from 'src/pages/GettingStarted';
import NewExperimentPage from 'src/pages/NewExperiment';
import NewPipelineVersionPage from 'src/pages/NewPipelineVersion';
import NewRunSwitcher from 'src/pages/NewRunSwitcher';
import PipelineDetails from 'src/pages/PipelineDetails';
import PrivateAndSharedPipelines, {
  PrivateAndSharedTab,
} from 'src/pages/PrivateAndSharedPipelines';
import RecurringRunDetailsRouter from 'src/pages/RecurringRunDetailsRouter';
import {
  BookOpen,
  FlaskConical,
  Info,
  Package,
  PlayCircle,
  Repeat,
  Workflow,
  X,
} from 'lucide-react';
import { ApplicationShell } from './modernization/ApplicationShell';
import type { AppShellNavItem } from './modernization/AppShell';
import { ModernPageChrome } from './modernization/ModernPageChrome';
import { Button as ModernButton } from './ui/button';
import Toolbar, { ToolbarProps } from './Toolbar';
import { BuildInfoContext } from 'src/lib/BuildInfo';

import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Snackbar,
} from '@mui/material';

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
  returnTo = 'returnTo',
  fromRunId = 'fromRun',
  fromRecurringRunId = 'fromRecurringRun',
  runlist = 'runlist',
  taskId = 'task',
  view = 'view',
  executionRedirect = 'executionRedirect',
}

export enum RouteParams {
  experimentId = 'eid',
  pipelineId = 'pid',
  pipelineVersionId = 'vid',
  runId = 'rid',
  recurringRunId = 'rrid',
  // Legacy execution routes below are retained only to redirect old bookmarks.
  ID = 'id',
  executionId = 'executionid',
}

// tslint:disable-next-line:variable-name
export const RoutePrefix = {
  ARTIFACT: '/artifact',
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
  artifactDetails: (artifactId: string | number) => {
    return RoutePage.ARTIFACT_DETAILS.replace(`:${RouteParams.ID}`, '' + artifactId);
  },
  runDetails: (runId: string) => {
    return RoutePage.RUN_DETAILS.replace(`:${RouteParams.runId}`, runId);
  },
  runDetailsTask: (runId: string, taskId: string) => {
    const search = new URLSearchParams({ [QUERY_PARAMS.taskId]: taskId });
    return `${RoutePageFactory.runDetails(runId)}?${search.toString()}`;
  },
  pipelineDetails: (id: string) => {
    return RoutePage.PIPELINE_DETAILS_NO_VERSION.replace(`:${RouteParams.pipelineId}`, id);
  },
};

export function getSafeReturnPath(path: string | null): string | undefined {
  if (!path || !path.startsWith('/') || path.startsWith('//') || path.includes('://')) {
    return undefined;
  }
  return path;
}

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

const RemovedExecutionRoute = ({ params }: NavigationProps) => {
  return (
    <Navigate
      replace
      to={`${RoutePage.RUNS}?${QUERY_PARAMS.executionRedirect}=${params[RouteParams.ID] ? 'detail' : 'list'}`}
    />
  );
};

const LegacyRunExecutionRoute = ({ location, params }: NavigationProps) => {
  const query = new URLSearchParams(location.search);
  query.set(QUERY_PARAMS.executionRedirect, 'detail');
  return (
    <Navigate
      replace
      to={{
        pathname: RoutePageFactory.runDetails(encodeURIComponent(params[RouteParams.runId]!)),
        search: `?${query}`,
        hash: location.hash,
      }}
    />
  );
};

// Keep navigation guidance separate from page-owned loading and error banners.
const ExecutionRedirectNotice = ({ modern = false }: { modern?: boolean }) => {
  const location = useLocation();
  const navigate = useNavigate();
  const query = new URLSearchParams(location.search);
  const reason = query.get(QUERY_PARAMS.executionRedirect);
  if (reason !== 'list' && reason !== 'detail') return null;
  const closeNotice = () => {
    query.delete(QUERY_PARAMS.executionRedirect);
    navigate(
      { ...location, search: query.size ? `?${query}` : '' },
      { replace: true, state: location.state },
    );
  };
  const message = (
    <>
      Execution pages have moved to Runs and task details. Open a run and select a task to view its
      inputs, outputs, status, and logs.
      {reason === 'detail' &&
        ' This legacy execution link cannot select the corresponding task automatically.'}
    </>
  );
  if (modern) {
    return (
      <div role='alert' className='kfp-navigation-notice'>
        <Info size={18} aria-hidden='true' />
        <p>{message}</p>
        <ModernButton variant='ghost' size='icon' aria-label='Close' onClick={closeNotice}>
          <X size={16} aria-hidden='true' />
        </ModernButton>
      </div>
    );
  }
  return (
    <Alert severity='info' onClose={closeNotice}>
      {message}
    </Alert>
  );
};

// This component is made as a wrapper to separate toolbar state for different pages.
const Router: React.FC<RouterProps> = ({ configs }) => {
  const buildInfo = React.useContext(BuildInfoContext);
  const defaultRoute =
    KFP_FLAGS.DEPLOYMENT === Deployments.MARKETPLACE ? RoutePage.START : RoutePage.PIPELINES;

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
    { path: RoutePage.ARTIFACTS, Component: EnhancedArtifactList },
    { path: RoutePage.ARTIFACT_DETAILS, Component: ArtifactDetails, notExact: true },
    { path: RoutePage.EXECUTIONS, Component: RemovedExecutionRoute },
    { path: RoutePage.EXECUTION_DETAILS, Component: RemovedExecutionRoute },
    {
      Component: AllExperimentsAndArchive,
      path: RoutePage.EXPERIMENTS,
      view: AllExperimentsAndArchiveTab.EXPERIMENTS,
    },
    { path: RoutePage.EXPERIMENT_DETAILS, Component: ExperimentDetailsPage },
    { path: RoutePage.NEW_EXPERIMENT, Component: NewExperimentPage },
    { path: RoutePage.NEW_PIPELINE_VERSION, Component: NewPipelineVersionPage },
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
    { path: RoutePage.RECURRING_RUNS, Component: AllRecurringRunsPage },
    { path: RoutePage.RECURRING_RUN_DETAILS, Component: RecurringRunDetailsRouter },
    { path: RoutePage.RUN_DETAILS, Component: RunDetailsRouter },
    { path: RoutePage.RUN_DETAILS_WITH_EXECUTION, Component: LegacyRunExecutionRoute },
    { path: RoutePage.COMPARE, Component: Compare },
    { path: RoutePage.FRONTEND_FEATURES, Component: FrontendFeatures },
  ];

  if (!buildInfo?.apiServerMultiUser) {
    routes = routes.filter((r) => r.path !== RoutePage.PIPELINES_SHARED);
  }

  return (
    // Keep the shell mounted across route changes.
    <ApplicationLayout>
      <Routes>
        <Route path='/' element={<Navigate replace to={defaultRoute} />} />
        {routes.map((route) => (
          <Route
            key={route.path}
            path={route.notExact ? `${route.path}/*` : route.path}
            element={<RoutePageElement route={route} />}
          />
        ))}
        <Route path='*' element={<RoutePageElement />} />
      </Routes>
    </ApplicationLayout>
  );
};

// Run Details owns task selection in the query. Other pages initialize forms from it.
function RoutePageElement({ route }: { route?: RouteConfig }) {
  const location = useLocation();
  const navigate = useNavigate();
  // Match the encoded pathname before decoding once: useParams turns a literal
  // "%2F" (encoded as "%252F" in the URL) into a slash in React Router 8.
  const match = route && matchPath({ path: route.path, end: !route.notExact }, location.pathname);
  const params = Object.fromEntries(
    Object.entries(match?.params || {}).map(([name, value]) => {
      try {
        return [name, value === undefined ? undefined : decodeURIComponent(value)];
      } catch {
        return [name, value];
      }
    }),
  );
  const routeIdentity =
    route?.path === RoutePage.RUN_DETAILS
      ? location.pathname
      : location.key || `${location.pathname}${location.search}${location.hash}`;
  return (
    <RoutedPage
      key={`${route?.path}:${routeIdentity}`}
      route={route}
      location={location}
      navigate={navigate}
      params={params}
    />
  );
}

class RoutedPage extends React.Component<
  NavigationProps & { route?: RouteConfig },
  RouteComponentState
> {
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

  public render(): React.JSX.Element {
    this.childProps.toolbarProps = this.state.toolbarProps;
    const { route, location, navigate, params } = this.props;
    const Component = route?.Component ?? Page404;
    const navigation = { location, navigate, params };
    const navigationNotice =
      route?.path === RoutePage.RUNS || route?.path === RoutePage.RUN_DETAILS ? (
        <ExecutionRedirectNotice modern={route?.path === RoutePage.RUNS} />
      ) : undefined;
    const page = <Component {...navigation} {...this.childProps} view={route?.view} />;

    if (route?.path === RoutePage.RUNS || route?.path === RoutePage.ARCHIVED_RUNS) {
      return (
        <ModernPageChrome
          toolbarProps={{ ...this.state.toolbarProps, navigate }}
          bannerProps={this.state.bannerProps}
          dialogProps={this.state.dialogProps}
          snackbarProps={this.state.snackbarProps}
          onDialogClose={this._handleDialogClosed}
          onSnackbarClose={this._handleSnackbarClose}
          navigationNotice={navigationNotice}
          showThemeControl={KFP_FLAGS.HIDE_SIDENAV}
        >
          {page}
        </ModernPageChrome>
      );
    }

    return (
      <div className={classes(commonCss.page, 'kfp-legacy-page')}>
        <Toolbar {...this.state.toolbarProps} navigate={navigate} />
        {navigationNotice}
        {this.state.bannerProps.message && (
          <Banner
            message={this.state.bannerProps.message}
            mode={this.state.bannerProps.mode}
            additionalInfo={this.state.bannerProps.additionalInfo}
            refresh={this.state.bannerProps.refresh}
            showTroubleshootingGuideLink={true}
          />
        )}
        {page}

        <Snackbar
          autoHideDuration={this.state.snackbarProps.autoHideDuration}
          message={this.state.snackbarProps.message}
          open={this.state.snackbarProps.open}
          onClose={this._handleSnackbarClose}
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

  private _handleDialogClosed = (onClick?: () => void): void => {
    this.setState({ dialogProps: { open: false } });
    if (onClick) {
      onClick();
    }
    if (this.state.dialogProps.onClose) {
      this.state.dialogProps.onClose();
    }
  };
  private _handleSnackbarClose = (): void => {
    this.setState({ snackbarProps: { open: false, message: '' } });
  };
}

// TODO: loading/error experience until backend is reachable

export default Router;

const ApplicationLayout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { pathname } = useLocation();
  const items: AppShellNavItem[] = [
    {
      id: 'pipelines',
      elementId: 'pipelinesBtn',
      label: 'Pipelines',
      href: RoutePage.PIPELINES,
      icon: Workflow,
      active:
        pathname.startsWith(RoutePage.PIPELINES) || pathname.startsWith(RoutePage.PIPELINES_SHARED),
    },
    {
      id: 'experiments',
      label: 'Experiments',
      href: RoutePage.EXPERIMENTS,
      icon: FlaskConical,
      active:
        pathname.startsWith(RoutePage.EXPERIMENTS) || pathname === RoutePage.ARCHIVED_EXPERIMENTS,
    },
    {
      id: 'runs',
      elementId: 'runsBtn',
      label: 'Runs',
      href: RoutePage.RUNS,
      icon: PlayCircle,
      active:
        pathname.startsWith(RoutePage.RUNS) ||
        pathname.startsWith(RoutePage.COMPARE) ||
        pathname === RoutePage.ARCHIVED_RUNS,
    },
    {
      id: 'recurring-runs',
      label: 'Recurring runs',
      href: RoutePage.RECURRING_RUNS,
      icon: Repeat,
      active:
        pathname.startsWith(RoutePage.RECURRING_RUNS) ||
        pathname.startsWith(RoutePrefix.RECURRING_RUN),
    },
    {
      id: 'artifacts',
      label: 'Artifacts',
      href: RoutePage.ARTIFACTS,
      icon: Package,
      active: pathname.startsWith(RoutePrefix.ARTIFACT),
    },
  ];
  if (KFP_FLAGS.DEPLOYMENT === Deployments.MARKETPLACE) {
    items.unshift({
      id: 'getting-started',
      label: 'Getting Started',
      href: RoutePage.START,
      icon: BookOpen,
      active: pathname.startsWith(RoutePage.START),
    });
  }
  return (
    <ApplicationShell items={items} currentPath={pathname}>
      {children}
    </ApplicationShell>
  );
};
