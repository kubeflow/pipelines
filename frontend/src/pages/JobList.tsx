/*
 * Copyright 2021 Arrikto Inc.
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
import CustomTable, { Column, Row, CustomRendererProps } from '../components/CustomTable';
import RunUtils, { ExperimentInfo } from '../../src/lib/RunUtils';
import { Apis, JobSortKeys, ListRequest } from '../lib/Apis';
import { Link, RouteComponentProps } from 'react-router-dom';
import { RoutePage, RouteParams } from '../components/Router';
import { commonCss, color } from '../Css';
import { formatDateString, errorToMessage } from '../lib/Utils';
import Tooltip from '@material-ui/core/Tooltip';
import { ApiJob, ApiTrigger } from '../apis/job';

interface DisplayJob {
  experiment?: ExperimentInfo;
  job: ApiJob;
  error?: string;
}

// Both masks cannot be provided together.
type MaskProps = Exclude<
  { experimentIdMask?: string; namespaceMask?: string },
  { experimentIdMask: string; namespaceMask: string }
>;

export type JobListProps = MaskProps &
  RouteComponentProps & {
    disablePaging?: boolean;
    disableSelection?: boolean;
    disableSorting?: boolean;
    hideExperimentColumn?: boolean;
    noFilterBox?: boolean;
    onError: (message: string, error: Error) => void;
    onSelectionChange?: (selectedRunIds: string[]) => void;
    jobIdListMask?: string[];
    selectedIds?: string[];
  };

interface JobListState {
  jobs: DisplayJob[];
}

class JobList extends React.PureComponent<JobListProps, JobListState> {
  private _tableRef = React.createRef<CustomTable>();

  constructor(props: any) {
    super(props);

    this.state = {
      jobs: [],
    };
  }

  public render(): JSX.Element {
    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer,
        flex: 1.5,
        label: 'Job Name',
        sortKey: JobSortKeys.NAME,
      },
      { customRenderer: this._statusCustomRenderer, label: 'Status', flex: 0.5 },
      { customRenderer: this._triggerCustomRenderer, label: 'Trigger', flex: 1 },
      { label: 'Start time', flex: 1, sortKey: JobSortKeys.CREATED_AT },
    ];

    if (!this.props.hideExperimentColumn) {
      columns.splice(3, 0, {
        customRenderer: this._experimentCustomRenderer,
        flex: 1,
        label: 'Experiment',
      });
    }

    const rows: Row[] = this.state.jobs.map(j => {
      const row = {
        error: j.error,
        id: j.job.id!,
        otherFields: [
          j.job!.name,
          j.job.status,
          j.job.trigger,
          formatDateString(j.job.created_at),
        ] as any,
      };
      if (!this.props.hideExperimentColumn) {
        row.otherFields.splice(3, 0, j.experiment);
      }

      return row;
    });

    return (
      <div>
        <CustomTable
          columns={columns}
          rows={rows}
          selectedIds={this.props.selectedIds}
          initialSortColumn={JobSortKeys.CREATED_AT}
          ref={this._tableRef}
          filterLabel='Filter jobs'
          updateSelection={this.props.onSelectionChange}
          reload={this._loadJobs.bind(this)}
          disablePaging={this.props.disablePaging}
          disableSorting={this.props.disableSorting}
          disableSelection={this.props.disableSelection}
          noFilterBox={this.props.noFilterBox}
          emptyMessage={
            `No available jobs found` +
            `${
            this.props.experimentIdMask
              ? ' for this experiment'
              : this.props.namespaceMask
                ? ' for this namespace'
                : ''
            }.`
          }
        />
      </div>
    );
  }

  public async refresh(): Promise<void> {
    if (this._tableRef.current) {
      await this._tableRef.current.reload();
    }
  }

  private _nameCustomRenderer: React.FC<CustomRendererProps<string>> = (
    props: CustomRendererProps<string>,
  ) => {
    return (
      <Tooltip title={props.value || ''} enterDelay={300} placement='top-start'>
        <Link
          className={commonCss.link}
          onClick={e => e.stopPropagation()}
          to={RoutePage.JOB_DETAILS.replace(':' + RouteParams.runId, props.id)}
        >
          {props.value}
        </Link>
      </Tooltip>
    );
  };

  private _experimentCustomRenderer: React.FC<CustomRendererProps<ExperimentInfo>> = (
    props: CustomRendererProps<ExperimentInfo>,
  ) => {
    // If the getExperiment call failed or a run has no experiment, we display a placeholder.
    if (!props?.value?.id) {
      return <div>-</div>;
    }
    return (
      <Link
        className={commonCss.link}
        onClick={e => e.stopPropagation()}
        to={RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, props.value.id)}
      >
        {props.value.displayName}
      </Link>
    );
  };

  public _triggerCustomRenderer: React.FC<CustomRendererProps<ApiTrigger>> = (
    props: CustomRendererProps<ApiTrigger>,
  ) => {
    if (props.value?.cron_schedule) {
      return <div>Cron: {props.value.cron_schedule.cron}</div>;
    }
    if (props.value?.periodic_schedule?.interval_second) {
      const interval_in_seconds = parseInt(props.value.periodic_schedule.interval_second, 10);
      if (interval_in_seconds % 86400 === 0) {
        // days
        return <div>Every {interval_in_seconds / 86400} days</div>;
      }
      if (interval_in_seconds % 3600 === 0) {
        // hours
        return <div>Every {interval_in_seconds / 3600} hours</div>;
      }
      if (interval_in_seconds % 60 === 0) {
        // minutes
        return <div>Every {interval_in_seconds / 60} minutes</div>;
      }
      return <div>Every {interval_in_seconds} seconds</div>;
    }
    return <div>-</div>;
  };

  public _statusCustomRenderer: React.FC<CustomRendererProps<string>> = (
    props: CustomRendererProps<string>,
  ) => {
    if (!props.value) {
      return <div>-</div>;
    }
    const textColor =
      props.value === 'Enabled'
        ? color.success
        : props.value === 'Disabled'
          ? color.inactive
          : color.errorText;
    return <div style={{ color: textColor }}>{props.value}</div>;
  };

  protected async _loadJobs(request: ListRequest): Promise<string> {
    let displayJobs: DisplayJob[] = [];
    let nextPageToken = '';

    if (Array.isArray(this.props.jobIdListMask)) {
      displayJobs = this.props.jobIdListMask.map(id => ({ job: { id } }));
      // listJobs doesn't currently support batching by IDs, so in this case we
      // retrieve each job individually.
      await this._getAndSetJobs(displayJobs);
    } else {
      try {
        let resourceReference: {
          keyType?: 'EXPERIMENT' | 'NAMESPACE';
          keyId?: string;
        } = {};
        if (this.props.experimentIdMask) {
          resourceReference = {
            keyType: 'EXPERIMENT',
            keyId: this.props.experimentIdMask,
          };
        } else if (this.props.namespaceMask) {
          resourceReference = {
            keyType: 'NAMESPACE',
            keyId: this.props.namespaceMask,
          };
        }
        const response = await Apis.jobServiceApi.listJobs(
          request.pageToken,
          request.pageSize,
          request.sortBy,
          resourceReference.keyType,
          resourceReference.keyId,
          request.filter,
        );

        displayJobs = (response.jobs || []).map(j => ({ job: j }));
        nextPageToken = response.next_page_token || '';
      } catch (err) {
        const error = new Error(await errorToMessage(err));
        this.props.onError('Error: failed to fetch jobs.', error);
        // No point in continuing if we couldn't retrieve any jobs.
        return '';
      }
    }

    await this._setColumns(displayJobs);

    this.setState({
      jobs: displayJobs,
    });
    return nextPageToken;
  }

  private async _setColumns(displayJobs: DisplayJob[]): Promise<DisplayJob[]> {
    return Promise.all(
      displayJobs.map(async displayJob => {
        if (!this.props.hideExperimentColumn) {
          await this._getAndSetExperimentNames(displayJob);
        }
        return displayJob;
      }),
    );
  }

  /**
   * For each job ID, fetch its corresponding job, and set it in DisplayJobs
   */
  private _getAndSetJobs(displayJobs: DisplayJob[]): Promise<DisplayJob[]> {
    return Promise.all(
      displayJobs.map(async displayJob => {
        let getJobResponse: ApiJob;
        try {
          getJobResponse = await Apis.jobServiceApi.getJob(displayJob.job!.id!);
          displayJob.job = getJobResponse!;
        } catch (err) {
          displayJob.error = await errorToMessage(err);
        }
        return displayJob;
      }),
    );
  }

  /**
   * For the given DisplayJob, get its ApiJob and retrieve that ApiJob's Experiment ID if it has
   * one, then use that Experiment ID to fetch its associated Experiment and attach that
   * Experiment's name to the DisplayJob. If the ApiJob has no Experiment ID, then the corresponding
   * DisplayJob will show '-'.
   */
  private async _getAndSetExperimentNames(displayJob: DisplayJob): Promise<void> {
    const experimentId = RunUtils.getFirstExperimentReferenceId(displayJob.job);
    if (experimentId) {
      let experimentName = RunUtils.getFirstExperimentReferenceName(displayJob.job);
      if (!experimentName) {
        try {
          const experiment = await Apis.experimentServiceApi.getExperiment(experimentId);
          experimentName = experiment.name || '';
        } catch (err) {
          displayJob.error = 'Failed to get associated experiment: ' + (await errorToMessage(err));
          return;
        }
      }
      displayJob.experiment = {
        displayName: experimentName,
        id: experimentId,
      };
    }
  }
}

export default JobList;
