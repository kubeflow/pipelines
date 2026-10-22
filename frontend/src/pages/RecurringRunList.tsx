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

import { scheduleDescription } from '../lib/ScheduleDisplay';
import { NavigationProps } from 'src/lib/Navigation';
import * as React from 'react';
import { ResourceTable } from 'src/components/tables/ResourceTable';
import CustomTable, { Column, Row, CustomRendererProps } from 'src/components/CustomTable';
import { ExperimentInfo } from 'src/lib/ExperimentInfo';
import { Apis, JobSortKeys, ListRequest } from 'src/lib/Apis';
import { Link } from 'react-router';
import { RoutePageFactory } from 'src/components/Router';
import { formatDateString, errorToMessage } from 'src/lib/Utils';
import { V2RecurringRun, V2RecurringRunStatus, V2Trigger } from 'src/apisv2/recurringrun';
import { V2ListExperimentsResponse } from 'src/apisv2/experiment';
import { RecurringRunSwitch } from 'src/components/runs/RecurringRunSwitch';
import 'src/components/runs/ExperimentWorkflows.css';

/** Extracts start_time and end_time from the trigger's cron or periodic schedule. */
export function getScheduleTimes(trigger?: V2Trigger): { startTime?: Date; endTime?: Date } {
  const schedule = trigger?.cron_schedule || trigger?.periodic_schedule;
  return {
    startTime: schedule?.start_time,
    endTime: schedule?.end_time,
  };
}

/** Derives schedule status (Active, Scheduled, Expired, Disabled) from run state and trigger times. */
export function getScheduleStatus(recurringRun: V2RecurringRun): string {
  if (recurringRun.status === V2RecurringRunStatus.DISABLED) {
    return 'Disabled';
  }
  if (recurringRun.status !== V2RecurringRunStatus.ENABLED) {
    return '-';
  }

  const { startTime, endTime } = getScheduleTimes(recurringRun.trigger);
  const now = new Date();
  if (startTime && now < startTime) {
    return 'Scheduled';
  }
  if (endTime && now > endTime) {
    return 'Expired';
  }
  return 'Active';
}

import { DateListFilter } from 'src/components/tables/ColumnFilter';

interface DisplayRecurringRun {
  experiment?: ExperimentInfo;
  recurringRun: V2RecurringRun;
  error?: string;
}

// Both masks cannot be provided together.
type MaskProps = Exclude<
  { experimentIdMask?: string; namespaceMask?: string },
  { experimentIdMask: string; namespaceMask: string }
>;

export type RecurringRunListProps = MaskProps &
  NavigationProps & {
    disablePaging?: boolean;
    disableSelection?: boolean;
    disableSorting?: boolean;
    hideExperimentColumn?: boolean;
    noFilterBox?: boolean;
    onError: (message: string, error: Error) => void;
    onSelectionChange?: (selectedRunIds: string[]) => void;
    recurringRunIdListMask?: string[];
    selectedIds?: string[];
    refreshCount: number;
  };

interface RecurringRunListState {
  recurringRuns: DisplayRecurringRun[];
}

class RecurringRunList extends React.PureComponent<RecurringRunListProps, RecurringRunListState> {
  private _tableRef = React.createRef<CustomTable>();

  constructor(props: any) {
    super(props);

    this.state = {
      recurringRuns: [],
    };
  }

  componentDidUpdate(prevProps: { refreshCount: number }) {
    if (prevProps.refreshCount === this.props.refreshCount) {
      return;
    }
    this.refresh();
  }

  public render(): React.JSX.Element {
    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer,
        flex: 1.5,
        label: 'Recurring run name',
        sortKey: JobSortKeys.NAME,
      },
      { customRenderer: this._statusCustomRenderer, label: 'Status', flex: 0.5, width: 120 },
      { customRenderer: this._triggerCustomRenderer, label: 'Schedule', flex: 1.5, width: 260 },
      { label: 'Start time', flex: 1, width: 190 },
      { label: 'End time', flex: 1, width: 190 },
      {
        customRenderer: this._scheduleStatusCustomRenderer,
        label: 'Schedule status',
        flex: 0.7,
        width: 120,
      },
      { label: 'Created at', flex: 1, width: 190, sortKey: JobSortKeys.CREATED_AT },
    ];

    if (!this.props.hideExperimentColumn) {
      columns.splice(6, 0, {
        customRenderer: this._experimentCustomRenderer,
        flex: 1,
        label: 'Experiment',
        width: 160,
      });
    }

    const rows: Row[] = this.state.recurringRuns.map((j) => {
      const { startTime, endTime } = getScheduleTimes(j.recurringRun.trigger);
      const row = {
        error: j.error,
        id: j.recurringRun.recurring_run_id!,
        otherFields: [
          j.recurringRun!.display_name,
          j.recurringRun.status,
          j.recurringRun.trigger,
          formatDateString(startTime),
          formatDateString(endTime),
          getScheduleStatus(j.recurringRun),
          formatDateString(j.recurringRun.created_at),
        ] as any,
      };
      if (!this.props.hideExperimentColumn) {
        row.otherFields.splice(6, 0, j.experiment);
      }

      return row;
    });

    return (
      <div>
        <DateListFilter
          label='Created'
          field='created_at'
          disabled={!!this.props.recurringRunIdListMask || this.props.noFilterBox}
        >
          {(date) => (
            <CustomTable
              filterPredicates={date.predicates}
              renderTable={(table) => (
                <ResourceTable
                  table={{
                    ...table,
                    emptyMessage: date.active
                      ? 'No results match the selected filters.'
                      : table.emptyMessage,
                  }}
                  headerActions={{ created_at: date.control }}
                  activeFilters={date.chip}
                  label='Recurring runs'
                  minWidth={this.props.hideExperimentColumn ? 1250 : 1410}
                  singular='schedule'
                  plural='schedules'
                />
              )}
              columns={columns}
              rows={rows}
              selectedIds={this.props.selectedIds}
              initialSortColumn={JobSortKeys.CREATED_AT}
              ref={this._tableRef}
              filterLabel='Search recurring runs'
              updateSelection={this.props.onSelectionChange}
              reload={this._loadRecurringRuns.bind(this)}
              disablePaging={this.props.disablePaging}
              disableSorting={this.props.disableSorting}
              disableSelection={this.props.disableSelection}
              noFilterBox={this.props.noFilterBox}
              emptyMessage={
                `No available recurring runs found` +
                `${
                  this.props.experimentIdMask
                    ? ' for this experiment'
                    : this.props.namespaceMask
                      ? ' for this namespace'
                      : ''
                }.`
              }
            />
          )}
        </DateListFilter>
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
      <Link
        className='kfp-workflow-link'
        title={props.value || ''}
        onClick={(e) => e.stopPropagation()}
        to={RoutePageFactory.recurringRunDetails(props.id)}
      >
        {props.value}
      </Link>
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
        className='kfp-workflow-link'
        title={props.value.displayName}
        onClick={(e) => e.stopPropagation()}
        to={RoutePageFactory.experimentDetails(props.value.id)}
      >
        {props.value.displayName}
      </Link>
    );
  };

  public _triggerCustomRenderer: React.FC<CustomRendererProps<V2Trigger>> = (
    props: CustomRendererProps<V2Trigger>,
  ) => {
    return (
      <div
        title={
          props.value?.cron_schedule?.cron ? `Cron: ${props.value.cron_schedule.cron}` : undefined
        }
      >
        {scheduleDescription(props.value)}
      </div>
    );
  };

  public _statusCustomRenderer: React.FC<CustomRendererProps<V2RecurringRunStatus>> = (
    props: CustomRendererProps<V2RecurringRunStatus>,
  ) => {
    return (
      <RecurringRunSwitch
        id={props.id}
        name={
          this.state.recurringRuns.find((item) => item.recurringRun.recurring_run_id === props.id)
            ?.recurringRun.display_name || props.id
        }
        status={props.value}
        onUpdated={this.refresh.bind(this)}
      />
    );
  };

  public _scheduleStatusCustomRenderer: React.FC<CustomRendererProps<string>> = (
    props: CustomRendererProps<string>,
  ) => (
    <span className='kfp-schedule-status' data-status={props.value}>
      {props.value || '-'}
    </span>
  );

  protected async _loadRecurringRuns(request: ListRequest): Promise<string> {
    let displayRecurringRuns: DisplayRecurringRun[];
    let nextPageToken = '';

    if (Array.isArray(this.props.recurringRunIdListMask)) {
      displayRecurringRuns = this.props.recurringRunIdListMask.map((id) => ({
        recurringRun: { recurring_run_id: id },
      }));
      // listRecurringRuns doesn't currently support batching by IDs, so in this case we
      // retrieve each recurring run individually.
      await this._getAndSetRecurringRuns(displayRecurringRuns);
    } else {
      try {
        const response = await Apis.recurringRunServiceApi.listRecurringRuns(
          request.pageToken,
          request.pageSize,
          request.sortBy,
          this.props.namespaceMask,
          request.filter,
          this.props.experimentIdMask,
        );

        displayRecurringRuns = (response.recurringRuns || []).map((rr) => ({ recurringRun: rr }));
        nextPageToken = response.next_page_token || '';
      } catch (err) {
        const error = new Error(await errorToMessage(err));
        this.props.onError('Error: failed to fetch recurring runs.', error);
        // No point in continuing if we couldn't retrieve any recurring runs.
        return '';
      }
    }

    await this._setColumns(displayRecurringRuns);

    this.setState({
      recurringRuns: displayRecurringRuns,
    });
    return nextPageToken;
  }

  private async _setColumns(
    displayRecurringRuns: DisplayRecurringRun[],
  ): Promise<DisplayRecurringRun[]> {
    let experimentsResponse: V2ListExperimentsResponse;
    let experimentsGetError: string;
    try {
      if (!this.props.namespaceMask) {
        // Single-user mode.
        experimentsResponse = await Apis.experimentServiceApiV2.listExperiments();
      } else {
        // Multi-user mode.
        experimentsResponse = await Apis.experimentServiceApiV2.listExperiments(
          undefined,
          undefined,
          undefined,
          undefined,
          this.props.namespaceMask,
        );
      }
    } catch (error) {
      experimentsGetError = 'Failed to get associated experiment: ' + (await errorToMessage(error));
    }

    return Promise.all(
      displayRecurringRuns.map((displayRecurringRun) => {
        if (!this.props.hideExperimentColumn) {
          const experimentId = displayRecurringRun.recurringRun.experiment_id;

          if (experimentId) {
            const experiment = experimentsResponse?.experiments?.find(
              (e) => e.experiment_id === displayRecurringRun.recurringRun.experiment_id,
            );
            // If matching experiment id not found (typically because it has
            // been deleted), set display name to "-".
            const displayName = experiment?.display_name || '-';
            if (experimentsGetError) {
              displayRecurringRun.error = experimentsGetError;
            } else {
              displayRecurringRun.experiment = {
                displayName: displayName,
                id: experimentId,
              };
            }
          }
        }
        return displayRecurringRun;
      }),
    );
  }

  /**
   * For each recurring run ID, fetch its corresponding recurring run,
   * and set it in DisplayRecurringRuns
   */
  private _getAndSetRecurringRuns(
    displayRecurringRuns: DisplayRecurringRun[],
  ): Promise<DisplayRecurringRun[]> {
    return Promise.all(
      displayRecurringRuns.map(async (displayRecurringRun) => {
        let getRecurringRunResponse: V2RecurringRun;
        try {
          getRecurringRunResponse = await Apis.recurringRunServiceApi.getRecurringRun(
            displayRecurringRun.recurringRun!.recurring_run_id!,
          );
          displayRecurringRun.recurringRun = getRecurringRunResponse!;
        } catch (err) {
          displayRecurringRun.error = await errorToMessage(err);
        }
        return displayRecurringRun;
      }),
    );
  }
}

export default RecurringRunList;
