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
import CustomTable, {
  Column,
  Row,
  ExpandState,
  CustomRendererProps,
} from '../components/CustomTable';
import RunList from './RunList';
import produce from 'immer';
import { ApiFilter, PredicateOp } from '../apis/filter';
import {
  ApiListExperimentsResponse,
  ApiExperiment,
  ExperimentStorageState,
} from '../apis/experiment';
import { ApiRun, RunStorageState } from '../apis/run';
import { Apis, ExperimentSortKeys, ListRequest, RunSortKeys } from '../lib/Apis';
import { Link } from 'react-router-dom';
import { NodePhase } from '../lib/StatusUtils';
import { Page, PageProps } from './Page';
import { RoutePage, RouteParams } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { logger } from '../lib/Utils';
import { statusToIcon } from './Status';
import Tooltip from '@material-ui/core/Tooltip';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { TFunction } from 'i18next';
import { useTranslation } from 'react-i18next';

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

export class ExperimentList extends Page<
  { namespace?: string; t: TFunction },
  ExperimentListState
> {
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
    const { t } = this.props;
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    return {
      actions: buttons
        .newRun()
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
      pageTitle: t('common:experiments'),
      t,
    };
  }

  public render(): JSX.Element {
    const { t } = this.props;
    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer,
        flex: 1,
        label: t('experimentName'),
        sortKey: ExperimentSortKeys.NAME,
      },
      {
        flex: 2,
        label: t('common:description'),
      },
      {
        customRenderer: this._last5RunsCustomRenderer,
        flex: 1,
        label: t('last5Runs'),
      },
    ];

    const rows: Row[] = this.state.displayExperiments.map(exp => {
      return {
        error: exp.error,
        expandState: exp.expandState,
        id: exp.id!,
        otherFields: [
          exp.name!,
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
          filterLabel={t('filterExperiments')}
          emptyMessage={t('noExperimentsFound')}
          t={t}
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

  public _last5RunsCustomRenderer: React.FC<CustomRendererProps<ApiRun[]>> = (
    props: CustomRendererProps<ApiRun[]>,
  ) => {
    const { t } = this.props;
    return (
      <div className={commonCss.flex}>
        {(props.value || []).map((run, i) => (
          <span key={i} style={{ margin: '0 1px' }}>
            {statusToIcon(t, (run.status as NodePhase) || NodePhase.UNKNOWN, run.created_at)}
          </span>
        ))}
      </div>
    );
  };

  private async _reload(request: ListRequest): Promise<string> {
    // Fetch the list of experiments
    let response: ApiListExperimentsResponse;
    let displayExperiments: DisplayExperiment[];
    const { t } = this.props;
    try {
      // This ExperimentList page is used as the "All experiments" tab
      // inside ExperimentAndRuns. Here we only list unarchived experiments.
      // Archived experiments are listed in "Archive" page.
      const filter = JSON.parse(
        decodeURIComponent(request.filter || '{"predicates": []}'),
      ) as ApiFilter;
      filter.predicates = (filter.predicates || []).concat([
        {
          key: 'storage_state',
          op: PredicateOp.NOTEQUALS,
          string_value: ExperimentStorageState.ARCHIVED.toString(),
        },
      ]);
      request.filter = encodeURIComponent(JSON.stringify(filter));
      response = await Apis.experimentServiceApi.listExperiment(
        request.pageToken,
        request.pageSize,
        request.sortBy,
        request.filter,
        this.props.namespace ? 'NAMESPACE' : undefined,
        this.props.namespace || undefined,
      );
      displayExperiments = response.experiments || [];
      displayExperiments.forEach(exp => (exp.expandState = ExpandState.COLLAPSED));
    } catch (err) {
      await this.showPageError(t('experimentListError'), err);
      // No point in continuing if we couldn't retrieve any experiments.
      return '';
    }

    // Fetch and set last 5 runs' statuses for each experiment
    await Promise.all(
      displayExperiments.map(async experiment => {
        // TODO: should we aggregate errors here? What if they fail for different reasons?
        const { t } = this.props;
        try {
          const listRunsResponse = await Apis.runServiceApi.listRuns(
            undefined /* pageToken */,
            5 /* pageSize */,
            RunSortKeys.CREATED_AT + ' desc',
            'EXPERIMENT',
            experiment.id,
            encodeURIComponent(
              JSON.stringify({
                predicates: [
                  {
                    key: 'storage_state',
                    op: PredicateOp.NOTEQUALS,
                    string_value: RunStorageState.ARCHIVED.toString(),
                  },
                ],
              } as ApiFilter),
            ),
          );
          experiment.last5Runs = listRunsResponse.runs || [];
        } catch (err) {
          experiment.error = t('last5RunsFailed');
          logger.error(`${t('runStatusError')}: ${experiment.name}.`, err);
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
        experimentIdMask={experiment.id}
        onError={() => null}
        {...this.props}
        disablePaging={false}
        selectedIds={this.state.selectedIds}
        noFilterBox={true}
        storageState={RunStorageState.AVAILABLE}
        onSelectionChange={this._selectionChanged.bind(this)}
        disableSorting={true}
      />
    );
  }
}

const EnhancedExperimentList: React.FC<PageProps> = props => {
  const { t } = useTranslation(['experiments', 'common']);
  const namespace = React.useContext(NamespaceContext);
  return <ExperimentList key={namespace} {...props} namespace={namespace} t={t} />;
};

export default EnhancedExperimentList;
