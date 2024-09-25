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

import * as React from 'react';
import Buttons, { ButtonKeys } from 'src/lib/Buttons';
import CustomTable, {
  Column,
  Row,
  ExpandState,
  CustomRendererProps,
} from 'src/components/CustomTable';
import RunList from './RunList';
import produce from 'immer';
import {
  V2beta1ListExperimentsResponse,
  V2beta1Experiment,
  V2beta1ExperimentStorageState,
} from 'src/apisv2beta1/experiment';
import { V2beta1Filter, V2beta1PredicateOperation } from 'src/apisv2beta1/filter';
import { V2beta1Run, V2beta1RunStorageState } from 'src/apisv2beta1/run';
import { Apis, ExperimentSortKeys, ListRequest, RunSortKeys } from 'src/lib/Apis';
import { Link } from 'react-router-dom';
import { Page, PageProps } from './Page';
import { RoutePage, RouteParams } from 'src/components/Router';
import { ToolbarProps } from 'src/components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from 'src/Css';
import { logger } from 'src/lib/Utils';
import { statusToIcon } from './StatusV2';
import Tooltip from '@material-ui/core/Tooltip';
import { NamespaceContext } from 'src/lib/KubeflowClient';

interface DisplayExperiment extends V2beta1Experiment {
  last5Runs?: V2beta1Run[];
  error?: string;
  expandState?: ExpandState;
}

interface ExperimentListState {
  displayExperiments: DisplayExperiment[];
  selectedIds: string[];
  selectedTab: number;
}

export class ExperimentList extends Page<{ namespace?: string }, ExperimentListState> {
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
        .newExperiment()
        .compareRuns(() => this.state.selectedIds)
        .cloneRun(() => this.state.selectedIds, false)
        .archive(
          'run',
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
    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer,
        flex: 1,
        label: 'Experiment name',
        sortKey: ExperimentSortKeys.NAME,
      },
      {
        flex: 2,
        label: 'Description',
      },
      {
        customRenderer: this._last5RunsCustomRenderer,
        flex: 1,
        label: 'Last 5 runs',
        sortKey: ExperimentSortKeys.LAST_RUN_CREATED_AT,
      },
    ];

    const rows: Row[] = this.state.displayExperiments.map(exp => {
      return {
        error: exp.error,
        expandState: exp.expandState,
        id: exp.experiment_id!,
        otherFields: [
          exp.display_name!,
          exp.description!,
          exp.expandState === ExpandState.EXPANDED ? [] : exp.last5Runs,
        ],
      };
    });

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <CustomTable
          columns={columns}
          rows={rows}
          ref={this._tableRef}
          disableSelection={true}
          initialSortColumn={ExperimentSortKeys.CREATED_AT}
          reload={this._reload.bind(this)}
          toggleExpansion={this._toggleRowExpand.bind(this)}
          getExpandComponent={this._getExpandedExperimentComponent.bind(this)}
          filterLabel='Filter experiments'
          emptyMessage='No experiments found. Click "Create experiment" to start.'
        />
      </div>
    );
  }

  public async refresh(): Promise<void> {
    if (this._tableRef.current) {
      this.clearBanner();
      await this._tableRef.current.reload();
    }
  }

  public _nameCustomRenderer: React.FC<CustomRendererProps<string>> = (
    props: CustomRendererProps<string>,
  ) => {
    return (
      <Tooltip title={props.value} enterDelay={300} placement='top-start'>
        <Link
          className={commonCss.link}
          onClick={e => e.stopPropagation()}
          to={RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, props.id)}
        >
          {props.value}
        </Link>
      </Tooltip>
    );
  };

  public _last5RunsCustomRenderer: React.FC<CustomRendererProps<V2beta1Run[]>> = (
    props: CustomRendererProps<V2beta1Run[]>,
  ) => {
    return (
      <div className={commonCss.flex}>
        {(props.value || []).map((run, i) => (
          <span key={i} style={{ margin: '0 1px' }}>
            {statusToIcon(run.state, run.created_at)}
          </span>
        ))}
      </div>
    );
  };

  private async _reload(request: ListRequest): Promise<string> {
    // Fetch the list of experiments
    let response: V2beta1ListExperimentsResponse;
    let displayExperiments: DisplayExperiment[];
    try {
      // This ExperimentList page is used as the "All experiments" tab
      // inside ExperimentAndRuns. Here we only list unarchived experiments.
      // Archived experiments are listed in "Archive" page.
      const filter = JSON.parse(
        decodeURIComponent(request.filter || '{"predicates": []}'),
      ) as V2beta1Filter;
      filter.predicates = (filter.predicates || []).concat([
        {
          key: 'storage_state',
          operation: V2beta1PredicateOperation.NOTEQUALS,
          string_value: V2beta1ExperimentStorageState.ARCHIVED.toString(),
        },
      ]);
      request.filter = encodeURIComponent(JSON.stringify(filter));
      response = await Apis.experimentServiceApiV2.listExperiments(
        request.pageToken,
        request.pageSize,
        request.sortBy,
        request.filter,
        this.props.namespace || undefined,
      );
      displayExperiments = response.experiments || [];
      displayExperiments.forEach(exp => (exp.expandState = ExpandState.COLLAPSED));
    } catch (err) {
      await this.showPageError('Error: failed to retrieve list of experiments.', err);
      // No point in continuing if we couldn't retrieve any experiments.
      return '';
    }

    // Fetch and set last 5 runs' statuses for each experiment
    await Promise.all(
      displayExperiments.map(async experiment => {
        // TODO: should we aggregate errors here? What if they fail for different reasons?
        try {
          const listRunsResponse = await Apis.runServiceApiV2.listRuns(
            undefined,
            experiment.experiment_id,
            undefined /* pageToken */,
            5 /* pageSize */,
            RunSortKeys.CREATED_AT + ' desc',
            encodeURIComponent(
              JSON.stringify({
                predicates: [
                  {
                    key: 'storage_state',
                    operation: V2beta1PredicateOperation.NOTEQUALS,
                    string_value: V2beta1RunStorageState.ARCHIVED.toString(),
                  },
                ],
              } as V2beta1Filter),
            ),
          );
          experiment.last5Runs = listRunsResponse.runs || [];
        } catch (err) {
          experiment.error = 'Failed to load the last 5 runs of this experiment';
          logger.error(
            `Error: failed to retrieve run statuses for experiment: ${experiment.display_name}.`,
            err,
          );
        }
      }),
    );

    this.setStateSafe({ displayExperiments });
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
        draft[rowIndex].expandState === ExpandState.COLLAPSED
          ? ExpandState.EXPANDED
          : ExpandState.COLLAPSED;
    });

    this.setState({ displayExperiments });
  }

  private _getExpandedExperimentComponent(experimentIndex: number): JSX.Element {
    const experiment = this.state.displayExperiments[experimentIndex];
    return (
      <RunList
        hideExperimentColumn={true}
        experimentIdMask={experiment.experiment_id}
        onError={() => null}
        {...this.props}
        disablePaging={false}
        selectedIds={this.state.selectedIds}
        noFilterBox={true}
        storageState={V2beta1RunStorageState.AVAILABLE}
        onSelectionChange={this._selectionChanged.bind(this)}
        disableSorting={true}
      />
    );
  }
}

const EnhancedExperimentList: React.FC<PageProps> = props => {
  const namespace = React.useContext(NamespaceContext);
  return <ExperimentList key={namespace} {...props} namespace={namespace} />;
};

export default EnhancedExperimentList;
