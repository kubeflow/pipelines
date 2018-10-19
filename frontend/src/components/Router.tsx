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
import Banner, { BannerProps } from '../components/Banner';
import Button from '@material-ui/core/Button';
import Compare from '../pages/Compare';
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogContent from '@material-ui/core/DialogContent';
import DialogTitle from '@material-ui/core/DialogTitle';
import JobDetails from '../pages/JobDetails';
import JobsAndRuns, { JobsAndRunsTab } from '../pages/JobsAndRuns';
import NewJob from '../pages/NewJob';
import PipelineDetails from '../pages/PipelineDetails';
import PipelineList from '../pages/PipelineList';
import RunDetails from '../pages/RunDetails';
import SideNav from './SideNav';
import Snackbar, { SnackbarProps } from '@material-ui/core/Snackbar';
import Toolbar, { ToolbarProps } from './Toolbar';
import { Route, Switch, Redirect, HashRouter } from 'react-router-dom';
import { classes, stylesheet } from 'typestyle';
import { commonCss } from '../Css';

const css = stylesheet({
  dialog: {
    minWidth: 250,
  },
});

export enum RouteParams {
  jobId = 'jid',
  pipelineId = 'pid',
  runId = 'rid',
}

// tslint:disable-next-line:variable-name
export const RoutePage = {
  COMPARE: `/compare`,
  JOBS: '/jobs',
  JOB_DETAILS: `/jobs/details/:${RouteParams.jobId}`,
  NEW_JOB: '/jobs/new',
  PIPELINES: '/pipelines',
  PIPELINE_DETAILS: `/pipelines/details/:${RouteParams.pipelineId}`,
  RUNS: '/runs',
  RUN_DETAILS: `/runs/details/:${RouteParams.runId}`,
};

export interface DialogProps {
  buttons?: Array<{ onClick?: () => any, text: string }>;
  // TODO: This can be generalized to React.ReactNode
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

class Router extends React.Component<{}, RouteComponentState> {

  constructor(props: any) {
    super(props);

    this.state = {
      bannerProps: {},
      dialogProps: { open: false },
      snackbarProps: { autoHideDuration: 5000, open: false },
      toolbarProps: { breadcrumbs: [{ displayName: '', href: '' }], actions: [] },
    };
  }

  public render() {
    const childProps = {
      toolbarProps: this.state.toolbarProps,
      updateBanner: this._updateBanner.bind(this),
      updateDialog: this._updateDialog.bind(this),
      updateSnackbar: this._updateSnackbar.bind(this),
      updateToolbar: this._setToolbarActions.bind(this),
    };

    return (
      <HashRouter>
        <div className={commonCss.page}>
          <div className={commonCss.flexGrow}>
            <Route render={({ ...props }) => (<SideNav page={props.location.pathname} {...props} />)} />
            <div className={classes(commonCss.page)}>
              <Toolbar {...this.state.toolbarProps} />
              {this.state.bannerProps.message
                && <Banner
                  message={this.state.bannerProps.message}
                  mode={this.state.bannerProps.mode}
                  additionalInfo={this.state.bannerProps.additionalInfo}
                  refresh={this.state.bannerProps.refresh} />}
              <Switch>
                {/* TODO: clean this up using a for-loop + array or something */}
                <Route exact={true} path={'/'} render={({ ...props }) => (
                  <Redirect to={RoutePage.PIPELINES} {...props} />
                )} />
                <Route exact={true} path={RoutePage.JOBS} render={({ ...props }) => (
                  <JobsAndRuns {...props} {...childProps} view={JobsAndRunsTab.JOBS} />
                )} />
                <Route exact={true} path={RoutePage.JOB_DETAILS} render={({ ...props }) => (
                  <JobDetails {...props} {...childProps} />
                )} />
                <Route exact={true} path={RoutePage.NEW_JOB} render={({ ...props }) => (
                  <NewJob {...props} {...childProps} />
                )} />
                <Route exact={true} path={RoutePage.PIPELINES} render={({ ...props }) => (
                  <PipelineList {...props} {...childProps} />
                )} />
                <Route exact={true} path={RoutePage.PIPELINE_DETAILS} render={({ ...props }) => (
                  <PipelineDetails {...props} {...childProps} />
                )} />
                <Route exact={true} path={RoutePage.RUNS} render={({ ...props }) => (
                  <JobsAndRuns {...props} {...childProps} view={JobsAndRunsTab.RUNS} />
                )} />
                <Route exact={true} path={RoutePage.RUN_DETAILS} render={({ ...props }) => (
                  <RunDetails {...props} {...childProps} />
                )} />
                <Route exact={true} path={RoutePage.COMPARE} render={({ ...props }) => (
                  <Compare {...props} {...childProps} />
                )} />
              </Switch>

              <Snackbar
                autoHideDuration={this.state.snackbarProps.autoHideDuration}
                message={this.state.snackbarProps.message}
                open={this.state.snackbarProps.open}
                onClose={this._handleSnackbarClose.bind(this)}
              />
            </div>
          </div>

          <Dialog open={this.state.dialogProps.open !== false} classes={{ paper: css.dialog }}
            className='dialog' onClose={() => this._handleDialogClosed()}>
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
                {this.state.dialogProps.buttons.map((b, i) =>
                  <Button key={i} onClick={() => this._handleDialogClosed(b.onClick)}
                    className='dialogButton' color='secondary'>
                    {b.text}
                  </Button>)}
              </DialogActions>
            )}
          </Dialog>
        </div>
      </HashRouter>
    );
  }

  private _updateDialog(dialogProps: DialogProps): void {
    // Assuming components will want to open the dialog by defaut.
    if (dialogProps.open === undefined) {
      dialogProps.open = true;
    }
    this.setState({ dialogProps });
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

  private _setToolbarActions(toolbarProps: ToolbarProps): void {
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

  private _handleSnackbarClose(): void {
    this.setState({ snackbarProps: { open: false, message: '' } });
  }
}

// TODO: loading/error experience until backend is reachable

// tslint:disable-next-line:no-default-export
export default Router;
