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

import * as Apis from '../lib/Apis';
import * as React from 'react';
import { BannerProps } from '../components/Banner';
import CloneIcon from '@material-ui/icons/FileCopy';
import CompareIcon from '@material-ui/icons/CompareArrows';
import DeleteIcon from '@material-ui/icons/Delete';
import DetailsTable from '../components/DetailsTable';
import DisableIcon from '@material-ui/icons/Stop';
import EnableIcon from '@material-ui/icons/PlayCircleFilled';
import MD2Tabs from '../atoms/MD2Tabs';
import RefreshIcon from '@material-ui/icons/Refresh';
import RunList from '../pages/RunList';
import Separator from '../atoms/Separator';
import { ToolbarActionConfig, ToolbarProps } from '../components/Toolbar';
import { RouteComponentProps } from 'react-router';
import { DialogProps, RoutePage, RouteParams } from '../components/Router';
import { apiJob } from '../../../frontend/src/api/job';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { formatDateString, enabledDisplayString, logger } from '../lib/Utils';
import { triggerDisplayString } from '../lib/TriggerUtils';
import { SnackbarProps } from '@material-ui/core/Snackbar';
import { URLParser, QUERY_PARAMS } from '../lib/URLParser';

interface JobDetailsProps extends RouteComponentProps {
  toolbarProps: ToolbarProps;
  updateBanner: (bannerProps: BannerProps) => void;
  updateDialog: (dialogProps: DialogProps) => void;
  updateSnackbar: (snackbarProps: SnackbarProps) => void;
  updateToolbar: (toolbarProps: ToolbarProps) => void;
}

interface JobDetailsState {
  job: apiJob | null;
  selectedRunIds: string[];
  selectedTab: number;
}

class JobDetails extends React.Component<JobDetailsProps, JobDetailsState> {

  private _runlistRef = React.createRef<RunList>();
  private _toolbarActions: ToolbarActionConfig[] = [
    {
      action: this._compareRuns.bind(this),
      disabled: true,
      icon: CompareIcon,
      id: 'compareBtn',
      title: 'Compare runs',
      tooltip: 'Compare up to 10 selected runs',
    },
    {
      action: this._cloneJob.bind(this),
      disabled: true,
      icon: CloneIcon,
      id: 'cloneBtn',
      title: 'Clone',
      tooltip: 'Clone this job',
    },
    {
      action: this._loadJob.bind(this),
      disabled: false,
      icon: RefreshIcon,
      id: 'refreshBtn',
      title: 'Refresh',
      tooltip: 'Refresh',
    },
    {
      action: () => this._setEnabledState(true),
      disabled: true,
      disabledTitle: 'Job already enabled',
      icon: EnableIcon,
      id: 'enableBtn',
      title: 'Enable',
      tooltip: 'Enable the job\'s trigger',
    },
    {
      action: () => this._setEnabledState(false),
      disabled: true,
      disabledTitle: 'Job already disabled',
      icon: DisableIcon,
      id: 'disableBtn',
      title: 'Disable',
      tooltip: 'Disable the job\'s trigger',
    },
    {
      action: () => this.props.updateDialog({
        buttons: [
          { onClick: () => this._deleteDialogClosed(true), text: 'Delete' },
          { onClick: () => this._deleteDialogClosed(false), text: 'Cancel' },
        ],
        onClose: () => this._deleteDialogClosed(false),
        title: 'Delete this job?',
      }),
      disabled: false,
      icon: DeleteIcon,
      id: 'deleteBtn',
      title: 'Delete',
      tooltip: 'Delete this job',
    },
  ];

  constructor(props: any) {
    super(props);

    this.state = {
      job: null,
      selectedRunIds: [],
      selectedTab: 0,
    };
  }

  public componentWillMount() {
    const { job } = this.state;
    // TODO: job status next to page name
    this.props.updateToolbar({
      actions: this._toolbarActions,
      breadcrumbs: [
        { displayName: 'Jobs', href: RoutePage.JOBS },
        { displayName: job && job.name ? job.name : this.props.match.params[RouteParams.jobId], href: '' }
      ],
    });
  }

  public componentDidMount() {
    this._loadJob();
  }

  public componentWillUnmount() {
    this.props.updateBanner({});
  }

  public render() {
    const { job } = this.state;
    const { selectedTab } = this.state;
    let jobDetails: string[][] = [];
    let inputParameters: string[][] = [];
    let triggerDetails: string[][] = [];
    if (job) {
      jobDetails = [
        ['Job pipeline ID', job.pipeline_id!],
        ['Description', job.description!],
        ['Created at', formatDateString(job.created_at)],
      ];
      inputParameters = (job.parameters || []).map(p => [p.name || '', p.value || '']);
      if (job.trigger) {
        triggerDetails = [
          ['Trigger', triggerDisplayString(job.trigger)],
          ['Enabled', enabledDisplayString(job.trigger, job.enabled!)],
        ];
      }
    }

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>

        {job && (
          <div className={commonCss.page}>
            <MD2Tabs selectedTab={selectedTab} onSwitch={(tab: number) => this.setState({ selectedTab: tab })}
              tabs={['Runs', 'Config']} />
            <div className={commonCss.page}>

              {selectedTab === 0 && <RunList handleError={this._handlePageError.bind(this)}
                jobIdMask={job.id} ref={this._runlistRef} selectedIds={this.state.selectedRunIds}
                updateSelection={this._selectionChanged.bind(this)} {...this.props} />}

              {selectedTab === 1 && (<div className={padding()}>
                <div className={commonCss.header}>Job details</div>
                <DetailsTable fields={jobDetails} />

                {!!inputParameters.length && <div>
                  <Separator orientation='vertical' />
                  <div className={commonCss.header}>Input parameters</div>
                  <DetailsTable fields={inputParameters} />
                </div>}

                {!!triggerDetails.length && <div>
                  <Separator orientation='vertical' />
                  <div className={commonCss.header}>Job trigger</div>
                  <DetailsTable fields={triggerDetails} />
                </div>}
              </div>
              )}
            </div>
          </div>
        )}
      </div>
    );
  }

  private async _loadJob() {
    const jobId = this.props.match.params[RouteParams.jobId];

    try {
      const job = await Apis.getJob(jobId);

      const toolbarActions = [...this.props.toolbarProps.actions];
      toolbarActions[3].disabled = job.enabled === true;
      toolbarActions[4].disabled = job.enabled === false;

      this._updateToolbar(toolbarActions);

      this.setState({ job }, () => this._runlistRef.current && this._runlistRef.current.refresh());
    } catch (err) {
      this._handlePageError(
        `Error: failed to retrieve job: ${jobId}. Click Details for more information.`, err);
      logger.error(`Error loading job: ${jobId}`, err);
    }
  }

  private _handlePageError(message: string, error: Error): void {
    this.props.updateBanner({
      additionalInfo: error.message,
      message,
      mode: 'error',
      refresh: this._loadJob.bind(this),
    });

    if (this._runlistRef.current) {
      this._runlistRef.current.refresh();
    }
  }

  private _selectionChanged(selectedRunIds: string[]) {
    const toolbarActions = [...this.props.toolbarProps.actions];
    toolbarActions[0].disabled = selectedRunIds.length <= 1 || selectedRunIds.length > 10;
    // TODO: is this cloning a run, or a job? should be run.
    toolbarActions[1].disabled = selectedRunIds.length !== 1;

    this._updateToolbar(toolbarActions);
    this.setState({ selectedRunIds });
  }

  private _compareRuns() {
    const indices = this.state.selectedRunIds;
    if (indices.length > 1 && indices.length <= 10) {
      const runIds = this.state.selectedRunIds.join(',');
      const searchString = new URLParser(this.props).build({
        [QUERY_PARAMS.runlist]: runIds,
      });
      this.props.history.push(RoutePage.COMPARE + searchString);
    }
  }

  private _cloneJob() {
    if (this.state.job && this.state.job.id) {
      const searchString = new URLParser(this.props).build({
        [QUERY_PARAMS.cloneFromJob]: this.state.job!.id || ''
      });
      this.props.history.push(RoutePage.NEW_JOB + searchString);
    }
  }

  private _showErrorDialog(title: string, content: string): void {
    this.props.updateDialog({
      buttons: [{ text: 'Dismiss' }],
      content,
      title,
    });
  }

  private async _setEnabledState(enabled: boolean) {
    if (this.state.job) {
      const toolbarActions = [...this.props.toolbarProps.actions];

      const buttonIndex = enabled ? 3 : 4;
      const id = this.state.job.id!;

      toolbarActions[buttonIndex].busy = true;
      this._updateToolbar(toolbarActions);
      try {
        await (enabled ? Apis.enableJob(id) : Apis.disableJob(id));
        this._loadJob();
      } catch (err) {
        this._showErrorDialog(`Failed to ${enabled ? 'enable' : 'disable'} job`, err.message);
      } finally {
        toolbarActions[buttonIndex].busy = false;
        this._updateToolbar(toolbarActions);
      }
    }
  }

  private _updateToolbar(actions: ToolbarActionConfig[]): void {
    this.props.updateToolbar({ breadcrumbs: this.props.toolbarProps.breadcrumbs, actions });
  }

  private async _deleteDialogClosed(deleteConfirmed: boolean): Promise<void> {
    if (deleteConfirmed) {
      // TODO: Show spinner during wait.
      try {
        await Apis.deleteJob(this.state.job!.id!);
        this.props.history.push(RoutePage.JOBS);
        this.props.updateSnackbar({
          message: `Successfully deleted job: ${this.state.job!.name}`,
          open: true,
        });
      } catch (err) {
        this._showErrorDialog('Failed to delete job', err.message);
        logger.error('Deleting job failed with error:', err);
      }
    }
  }
}

export default JobDetails;
