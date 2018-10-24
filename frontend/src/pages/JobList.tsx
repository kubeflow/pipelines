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

import { Apis, JobSortKeys, BaseListRequest, ListJobsRequest } from '../lib/Apis';
import * as React from 'react';
import AddIcon from '@material-ui/icons/Add';
import CloneIcon from '@material-ui/icons/FileCopy';
import CustomTable, { Column, Row, ExpandState } from '../components/CustomTable';
import DeleteIcon from '@material-ui/icons/Delete';
import RefreshIcon from '@material-ui/icons/Refresh';
import RunList from './RunList';
import { ApiRun } from '../apis/run';
import { BannerProps } from '../components/Banner';
import { DialogProps, RoutePage, RouteParams } from '../components/Router';
import { Link } from 'react-router-dom';
import { RouteComponentProps } from 'react-router';
import { SnackbarProps } from '@material-ui/core/Snackbar';
import { ToolbarActionConfig, ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { logger, getLastInStatusList } from '../lib/Utils';
import { statusToIcon, NodePhase } from './Status';
import { URLParser, QUERY_PARAMS } from '../lib/URLParser';
import { ApiListJobsResponse, ApiJob } from '../apis/job';
import { triggerDisplayString } from '../lib/TriggerUtils';

interface DisplayJob extends ApiJob {
  last5Runs?: ApiRun[];
  pipelineName?: string;
  error?: string;
  expandState?: ExpandState;
}

interface JobListProps extends RouteComponentProps {
  toolbarProps: ToolbarProps;
  updateBanner: (bannerProps: BannerProps) => void;
  updateDialog: (dialogProps: DialogProps) => void;
  updateSnackbar: (snackbarProps: SnackbarProps) => void;
  updateToolbar: (toolbarProps: ToolbarProps) => void;
}

interface JobListState {
  displayJobs: DisplayJob[];
  orderAscending: boolean;
  pageSize: number;
  pageToken: string;
  selectedJobIds: string[];
  selectedTab: number;
  sortBy: string;
}

class JobList extends React.Component<JobListProps, JobListState> {

  private _toolbarActions: ToolbarActionConfig[] = [
    {
      action: this._newJobClicked.bind(this),
      disabled: false,
      icon: AddIcon,
      id: 'newJobBtn',
      title: 'New job',
      tooltip: 'New job',
    },
    {
      action: this._loadJobs.bind(this),
      disabled: false,
      icon: RefreshIcon,
      id: 'refreshBtn',
      title: 'Refresh',
      tooltip: 'Refresh',
    },
    {
      action: this._cloneJob.bind(this),
      disabled: true,
      disabledTitle: 'Select a job to clone',
      icon: CloneIcon,
      id: 'cloneBtn',
      title: 'Clone',
      tooltip: 'Clone',
    },
    {
      action: () => this.props.updateDialog({
        buttons: [
          { onClick: () => this._deleteDialogClosed(true), text: 'Delete' },
          { onClick: () => this._deleteDialogClosed(false), text: 'Cancel' },
        ],
        onClose: () => this._deleteDialogClosed(false),
        title: `Delete ${this.state.selectedJobIds.length} job${this.state.selectedJobIds.length === 1 ? '' : 's'}?`,
      }),
      disabled: true,
      disabledTitle: 'Select at least one job to delete',
      icon: DeleteIcon,
      id: 'deleteBtn',
      title: 'Delete',
      tooltip: 'Delete',
    },
  ];

  constructor(props: any) {
    super(props);

    this.state = {
      displayJobs: [],
      orderAscending: true,
      pageSize: 10,
      pageToken: '',
      selectedJobIds: [],
      selectedTab: 0,
      sortBy: JobSortKeys.CREATED_AT,
    };
  }

  public componentWillMount() {
    this.props.updateToolbar({ actions: this._toolbarActions, breadcrumbs: [{ displayName: 'Jobs', href: RoutePage.JOBS }] });
  }

  public componentWillUnmount() {
    this.props.updateBanner({});
  }

  public render() {
    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer.bind(this),
        flex: 2,
        label: 'Job name',
        sortKey: JobSortKeys.NAME,
      },
      {
        customRenderer: this._last5RunsCustomRenderer.bind(this),
        flex: 1,
        label: 'Last 5 runs',
      },
      {
        flex: 2,
        label: 'Pipeline',
      },
      { label: 'Created at', sortKey: JobSortKeys.CREATED_AT, flex: 1 },
      { label: 'Schedule', flex: 1 },
      { label: 'Enabled', flex: 1 },
    ];

    const rows: Row[] = this.state.displayJobs.map(j => {
      return {
        error: j.error,
        expandState: j.expandState,
        id: j.id!,
        otherFields: [
          j.name!,
          j.expandState === ExpandState.EXPANDED ? [] : j.last5Runs,
          j.pipelineName,
          j.created_at!.toLocaleString(),
          triggerDisplayString(j.trigger),
          j.enabled,
        ]
      } as Row;
    });

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <CustomTable columns={columns} rows={rows} orderAscending={this.state.orderAscending}
          updateSelection={this._selectionChanged.bind(this)} sortBy={this.state.sortBy}
          reload={this._loadJobs.bind(this)} selectedIds={this.state.selectedJobIds}
          toggleExpansion={this._toggleRowExpand.bind(this)}
          pageSize={this.state.pageSize} getExpandComponent={this._getExpandedJobComponent.bind(this)}
          emptyMessage='No jobs found. Click "New job" to start.' />
      </div>
    );
  }

  private async _loadJobs(loadRequest?: BaseListRequest): Promise<string> {
    // Override the current state with incoming request
    const request: ListJobsRequest = Object.assign({
      orderAscending: this.state.orderAscending,
      pageSize: this.state.pageSize,
      pageToken: this.state.pageToken,
      sortBy: this.state.sortBy,
    }, loadRequest);

    // Fetch the list of jobs
    let response: ApiListJobsResponse;
    let displayJobs: DisplayJob[];
    try {
      response =
        await Apis.jobServiceApi.listJobs(
          request.pageToken,
          request.pageSize,
          request.sortBy ? request.sortBy + (request.orderAscending ? ' asc' : ' desc') : ''
        );
      displayJobs = response.jobs || [];
      displayJobs.forEach(j => j.expandState = ExpandState.COLLAPSED);
    } catch (err) {
      this._handlePageError('Error: failed to retrieve list of jobs.', err);
      // No point in continuing if we couldn't retrieve any jobs.
      return '';
    }

    // Fetch and set last 5 runs' statuses for each job
    await Promise.all(displayJobs.map(async job => {
      // TODO: should we aggregate errors here? What if they fail for different reasons?
      try {
        const listRunsResponse = await Apis.jobServiceApi.listJobRuns(job.id!, '', 5, '');
        job.last5Runs = listRunsResponse.runs || [];
      } catch (err) {
        job.error = 'Failed to load the last 5 runs of this job';
        logger.error(`Error: failed to retrieve run statuses for job: ${job.name}.`, err);
      }
    }));

    // Fetch and set pipeline name for each job
    await Promise.all(displayJobs.map(async job => {
      // TODO: should we aggregate errors here? What if they fail for different reasons?
      try {
        const pipeline = await Apis.pipelineServiceApi.getPipeline(job.pipeline_id!);
        job.pipelineName = pipeline.name;
      } catch (err) {
        // TODO: job name ideally needs padding-top 1px when there's an error like this.
        job.error = 'Failed to load pipeline for this job';
        logger.error(`Error: failed to retrieve pipeline for job: ${job.name}.`, err);
      }
    }));

    // TODO: saw this warning:
    // Warning: Can't call setState (or forceUpdate) on an unmounted component.
    // This is a no-op, but it indicates a memory leak in your application.
    // To fix, cancel all subscriptions and asynchronous tasks in the componentWillUnmount method.
    this.setState({
      displayJobs,
      orderAscending: request.orderAscending!,
      pageSize: request.pageSize!,
      pageToken: request.pageToken!,
      sortBy: request.sortBy!,
    });

    return response.next_page_token || '';
  }

  private _handlePageError(message: string, error: Error): void {
    this.props.updateBanner({
      additionalInfo: error.message,
      message: message + (error.message ? ' Click Details for more information.' : ''),
      mode: 'error',
      refresh: this._loadJobs.bind(this),
    });
  }

  private _cloneJob() {
    if (this.state.selectedJobIds.length === 1) {
      const job = this.state.displayJobs.find(j => j.id === this.state.selectedJobIds[0]);
      if (!job) {
        logger.error('Could not get a job with the id:', this.state.selectedJobIds[0]);
        return;
      }
      const jobId = job.id;
      const searchString = new URLParser(this.props).build({
        [QUERY_PARAMS.cloneFromJob]: jobId || ''
      });
      this.props.history.push(RoutePage.NEW_JOB + searchString);
    }
  }

  private async _deleteDialogClosed(deleteConfirmed: boolean): Promise<void> {
    if (deleteConfirmed) {
      const unsuccessfulDeleteIds: string[] = [];
      const errorMessages: string[] = [];
      // TODO: Show spinner during wait.
      await Promise.all(this.state.selectedJobIds.map(async (id) => {
        try {
          await Apis.jobServiceApi.deleteJob(id);
        } catch (err) {
          unsuccessfulDeleteIds.push(id);
          const job = this.state.displayJobs.find((p) => p.id === id);
          errorMessages.push(
            `Deleting job${job ? ':' + job.name : ''} failed with error: "${err}"`);
        }
      }));

      const successfulDeletes = this.state.selectedJobIds.length - unsuccessfulDeleteIds.length;
      if (successfulDeletes > 0) {
        this.props.updateSnackbar({
          message: `Successfully deleted ${successfulDeletes} job${successfulDeletes === 1 ? '' : 's'}!`,
          open: true,
        });
        this._loadJobs();
      }

      if (unsuccessfulDeleteIds.length > 0) {
        this.props.updateDialog({
          buttons: [{ text: 'Dismiss' }],
          content: errorMessages.join('\n\n'),
          title: `Failed to delete ${unsuccessfulDeleteIds.length} job${unsuccessfulDeleteIds.length === 1 ? '' : 's'}`,
        });
      }

      this._selectionChanged(unsuccessfulDeleteIds);
    }
  }

  private _nameCustomRenderer(value: string, id: string) {
    return <Link className={commonCss.link} onClick={(e) => e.stopPropagation()}
      to={RoutePage.JOB_DETAILS.replace(':' + RouteParams.jobId, id)}>{value}</Link>;
  }

  private _last5RunsCustomRenderer(runs: ApiRun[]) {
    return <div className={commonCss.flex}>
      {runs.map((run, i) => (
        <span key={i} style={{ margin: '0 1px' }}>
          {statusToIcon(getLastInStatusList(run.status || '') || NodePhase.ERROR)}
        </span>
      ))}
    </div>;
  }

  private _selectionChanged(selectedJobIds: string[]) {
    const toolbarActions = [...this.props.toolbarProps.actions];
    toolbarActions[2].disabled = selectedJobIds.length !== 1;
    toolbarActions[3].disabled = !selectedJobIds.length;
    this.props.updateToolbar({ breadcrumbs: this.props.toolbarProps.breadcrumbs, actions: toolbarActions });
    this.setState({ selectedJobIds });
  }

  private _newJobClicked() {
    this.props.history.push(RoutePage.NEW_JOB);
  }

  private _toggleRowExpand(rowIndex: number) {
    const displayJobs = this.state.displayJobs;
    displayJobs[rowIndex].expandState =
      displayJobs[rowIndex].expandState === ExpandState.COLLAPSED ?
        ExpandState.EXPANDED :
        ExpandState.COLLAPSED;

    this.setState({ displayJobs });
  }

  private _getExpandedJobComponent(jobIndex: number) {
    const job = this.state.displayJobs[jobIndex];
    const runIds = (job.last5Runs || []).map(r => r.id!);
    return <RunList runIdListMask={runIds} onError={() => null} {...this.props}
      disablePaging={true} selectedIds={this.state.selectedJobIds}
      onSelectionChange={this._selectionChanged.bind(this)} disableSorting={true} />;
  }
}

export default JobList;
