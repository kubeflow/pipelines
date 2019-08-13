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
import Buttons, { ButtonKeys } from '../lib/Buttons';
import CustomTable, { Column, Row, ExpandState, CustomRendererProps } from '../components/CustomTable';
import RunList from './RunList';
import produce from 'immer';
import { ApiFilter, PredicateOp } from '../apis/filter';
import { ApiListExperimentsResponse, ApiExperiment } from '../apis/experiment';
import { ApiResourceType, ApiRun, RunStorageState } from '../apis/run';
import { Apis, ExperimentSortKeys, ListRequest, RunSortKeys } from '../lib/Apis';
import { Link } from 'react-router-dom';
import { NodePhase } from '../lib/StatusUtils';
import { Page } from './Page';
import { RoutePage, RouteParams } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { logger } from '../lib/Utils';
import { statusToIcon } from './Status';

interface DisplayExperiment extends ApiExperiment {
  last5Runs?: ApiRun[];
  error?: string;
  expandState?: ExpandState;
}

interface ExperimentListState {
  displayExperiments: DisplayExperiment[];
  selectedIds: string[];
  selectedTab: number;
}

class ExperimentList extends Page<{}, ExperimentListState> {
  private _tableRef = React.createRef<CustomTable>();

  constructor(props: any) {
    super(props);

    this.state = {
      displayExperiments: [],
      selectedIds: [],
      selectedTab: 0,
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    return {
      actions: buttons
        .newRun()
        .newExperiment()
        .compareRuns(() => this.state.selectedIds)
        .cloneRun(() => this.state.selectedIds, false)
        .archive(
          () => this.state.selectedIds,
          false,
          ids => this._selectionChanged(ids),
        )
        .refresh(this.refresh.bind(this))
        .getToolbarActionMap(),
      breadcrumbs: [],
      pageTitle: 'Experiments',
    };
  }

  public render(): JSX.Element {
    const columns: Column[] = [{
      customRenderer: this._nameCustomRenderer,
      flex: 1,
      label: 'Experiment name',
      sortKey: ExperimentSortKeys.NAME,
    }, {
      flex: 2,
      label: 'Description',
    }, {
      customRenderer: this._last5RunsCustomRenderer,
      flex: 1,
      label: 'Last 5 runs',
    }];

    const rows: Row[] = this.state.displayExperiments.map((exp) => {
      return {
        error: exp.error,
        expandState: exp.expandState,
        id: exp.id!,
        otherFields: [
          exp.name!,
          exp.description!,
          exp.expandState === ExpandState.EXPANDED ? [] : exp.last5Runs,
        ]
      };
    });

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <CustomTable columns={columns} rows={rows} ref={this._tableRef}
          disableSelection={true} initialSortColumn={ExperimentSortKeys.CREATED_AT}
          reload={this._reload.bind(this)} toggleExpansion={this._toggleRowExpand.bind(this)}
          getExpandComponent={this._getExpandedExperimentComponent.bind(this)}
          filterLabel='Filter experiments'
          emptyMessage='No experiments found. Click "Create experiment" to start.' />
      </div>
    );
  }

  public async refresh(): Promise<void> {
    if (this._tableRef.current) {
      this.clearBanner();
      await this._tableRef.current.reload();
    }
  }

  public _nameCustomRenderer: React.FC<CustomRendererProps<string>> = (props: CustomRendererProps<string>) => {
    return <Link className={commonCss.link} onClick={(e) => e.stopPropagation()}
      to={RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, props.id)}>{props.value}</Link>;
  }

  public _last5RunsCustomRenderer: React.FC<CustomRendererProps<ApiRun[]>> = (props: CustomRendererProps<ApiRun[]>) => {
    return <div className={commonCss.flex}>
      {(props.value || []).map((run, i) => (
        <span key={i} style={{ margin: '0 1px' }}>
          {statusToIcon(run.status as NodePhase || NodePhase.UNKNOWN, run.created_at)}
        </span>
      ))}
    </div>;
  }

  private async _reload(request: ListRequest): Promise<string> {
    // Fetch the list of experiments
    let response: ApiListExperimentsResponse;
    let displayExperiments: DisplayExperiment[];
    try {
      response = await Apis.experimentServiceApi.listExperiment(
        request.pageToken, request.pageSize, request.sortBy, request.filter);
      displayExperiments = response.experiments || [];
      displayExperiments.forEach((exp) => exp.expandState = ExpandState.COLLAPSED);
    } catch (err) {
      await this.showPageError('Error: failed to retrieve list of experiments.', err);
      // No point in continuing if we couldn't retrieve any experiments.
      return '';
    }

    // Fetch and set last 5 runs' statuses for each experiment
    await Promise.all(displayExperiments.map(async experiment => {
      // TODO: should we aggregate errors here? What if they fail for different reasons?
      try {
        const listRunsResponse = await Apis.runServiceApi.listRuns(
          undefined /* pageToken */,
          5 /* pageSize */,
          RunSortKeys.CREATED_AT + ' desc',
          ApiResourceType.EXPERIMENT.toString(),
          experiment.id,
          encodeURIComponent(JSON.stringify({
            predicates: [{
              key: 'storage_state',
              op: PredicateOp.NOTEQUALS,
              string_value: RunStorageState.ARCHIVED.toString(),
            }]
          } as ApiFilter)),
        );
        experiment.last5Runs = listRunsResponse.runs || [];
      } catch (err) {
        experiment.error = 'Failed to load the last 5 runs of this experiment';
        logger.error(
          `Error: failed to retrieve run statuses for experiment: ${experiment.name}.`,
          err);
      }
    }));

    this.setState({ displayExperiments });
    return response.next_page_token || '';
  }

  private _selectionChanged(selectedIds: string[]): void {
    const actions = this.props.toolbarProps.actions;
    actions[ButtonKeys.COMPARE].disabled = selectedIds.length <= 1 || selectedIds.length > 10;
    actions[ButtonKeys.CLONE_RUN].disabled = selectedIds.length !== 1;
    actions[ButtonKeys.ARCHIVE].disabled = !selectedIds.length;
    this.props.updateToolbar({ actions });
    this.setState({ selectedIds });
  }

  private _toggleRowExpand(rowIndex: number): void {
    const displayExperiments = produce(this.state.displayExperiments, draft => {
      draft[rowIndex].expandState =
        draft[rowIndex].expandState === ExpandState.COLLAPSED ?
          ExpandState.EXPANDED :
          ExpandState.COLLAPSED;
    });

    this.setState({ displayExperiments });
  }

  private _getExpandedExperimentComponent(experimentIndex: number): JSX.Element {
    const experiment = this.state.displayExperiments[experimentIndex];
    return <RunList hideExperimentColumn={true} experimentIdMask={experiment.id} onError={() => null} {...this.props}
      disablePaging={true} selectedIds={this.state.selectedIds} noFilterBox={true}
      storageState={RunStorageState.AVAILABLE}
      onSelectionChange={this._selectionChanged.bind(this)} disableSorting={true} />;
  }

}

export default ExperimentList;
