import CustomTable, { Column, CustomRendererProps, Row, ExpandState } from './CustomTable';
import * as React from 'react';
import { Link, RouteComponentProps } from 'react-router-dom';
import {
  ApiListExperimentsResponse,
  ApiExperiment,
  ExperimentStorageState,
} from '../apis/experiment';
import { errorToMessage } from '../lib/Utils';
import { RoutePage, RouteParams } from './Router';
import { commonCss } from '../Css';
import { Apis, ExperimentSortKeys, ListRequest } from '../lib/Apis';
import { RunStorageState } from 'src/apis/run';
import RunList from '../pages/RunList';
import { PredicateOp, ApiFilter } from '../apis/filter';
import produce from 'immer';
import Tooltip from '@material-ui/core/Tooltip';
import { TFunction } from 'i18next';
import { withTranslation } from 'react-i18next';

export interface ExperimentListProps extends RouteComponentProps {
  t: TFunction;
  namespace?: string;
  storageState?: ExperimentStorageState;
  onError: (message: string, error: Error) => void;
}

interface DisplayExperiment extends ApiExperiment {
  error?: string;
  expandState?: ExpandState;
}

interface ExperimentListState {
  displayExperiments: DisplayExperiment[];
}

export class ExperimentList extends React.PureComponent<ExperimentListProps, ExperimentListState> {
  private _tableRef = React.createRef<CustomTable>();
  constructor(props: any) {
    super(props);

    this.state = {
      displayExperiments: [],
    };
  }

  public render(): JSX.Element {
    const { t } = this.props;
    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer,
        flex: 1,
        label: t('experiments:experimentName'),
        sortKey: ExperimentSortKeys.NAME,
      },
      {
        flex: 2,
        label: t('experiments:description'),
      },
    ];

    const rows: Row[] = this.state.displayExperiments.map(exp => {
      return {
        error: exp.error,
        expandState: exp.expandState,
        id: exp.id!,
        otherFields: [exp.name!, exp.description!],
      };
    });

    return (
      <div>
        <CustomTable
          columns={columns}
          rows={rows}
          ref={this._tableRef}
          disableSelection={true}
          initialSortColumn={ExperimentSortKeys.CREATED_AT}
          reload={this._loadExperiments.bind(this)}
          toggleExpansion={this._toggleRowExpand.bind(this)}
          getExpandComponent={this._getExpandedExperimentComponent.bind(this)}
          filterLabel={t('experiments:filterExperiments')}
          emptyMessage={t('experiments:noExperimentsFound')}
          t={t}
        />
      </div>
    );
  }

  public async refresh(): Promise<void> {
    if (this._tableRef.current) {
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

  protected async _loadExperiments(request: ListRequest): Promise<string> {
    let nextPageToken = '';
    let displayExperiments: DisplayExperiment[];

    if (this.props.storageState) {
      try {
        // Augment the request filter with the storage state predicate
        const filter = JSON.parse(
          decodeURIComponent(request.filter || '{"predicates": []}'),
        ) as ApiFilter;
        filter.predicates = (filter.predicates || []).concat([
          {
            key: 'storage_state',
            // Use EQUALS ARCHIVED or NOT EQUALS ARCHIVED to account for cases where the field
            // is missing, in which case it should be counted as available.
            op:
              this.props.storageState === ExperimentStorageState.ARCHIVED
                ? PredicateOp.EQUALS
                : PredicateOp.NOTEQUALS,
            string_value: ExperimentStorageState.ARCHIVED.toString(),
          },
        ]);
        request.filter = encodeURIComponent(JSON.stringify(filter));
      } catch (err) {
        const error = new Error(await errorToMessage(err));
        this.props.onError('Error: failed to parse request filter: ', error);
        return '';
      }
    }

    try {
      let response: ApiListExperimentsResponse;
      response = await Apis.experimentServiceApi.listExperiment(
        request.pageToken,
        request.pageSize,
        request.sortBy,
        request.filter,
        this.props.namespace ? 'NAMESPACE' : undefined,
        this.props.namespace || undefined,
      );
      nextPageToken = response.next_page_token || '';
      displayExperiments = response.experiments || [];
      displayExperiments.forEach(exp => (exp.expandState = ExpandState.COLLAPSED));
      this.setState({ displayExperiments });
    } catch (err) {
      const error = new Error(await errorToMessage(err));
      this.props.onError('Error: failed to list experiments: ', error);
      return '';
    }

    return nextPageToken;
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
        {...this.props}
        disablePaging={false}
        noFilterBox={true}
        storageState={
          this.props.storageState === ExperimentStorageState.ARCHIVED
            ? RunStorageState.ARCHIVED
            : RunStorageState.AVAILABLE
        }
        disableSorting={true}
        disableSelection={true}
      />
    );
  }
}
withTranslation(['experiments', 'common'])(ExperimentList);
export default ExperimentList;
