/*
 * Copyright 2019 The Kubeflow Authors
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

import { CircularProgress } from '@mui/material';
import { QueryClient, useQuery, useQueryClient } from '@tanstack/react-query';
import * as React from 'react';
import { Link, Route, Switch } from 'react-router-dom';
import {
  ArtifactArtifactType,
  V2beta1Artifact,
  V2beta1ArtifactTask,
  V2beta1IOType,
} from 'src/apisv2beta1/artifact';
import { V2beta1Filter, V2beta1PredicateOperation } from 'src/apisv2beta1/filter';
import MD2Tabs from 'src/atoms/MD2Tabs';
import ArtifactPreview from 'src/components/ArtifactPreview';
import Banner from 'src/components/Banner';
import CustomTable, { Column, CustomRendererProps, Row } from 'src/components/CustomTable';
import DetailsTable from 'src/components/DetailsTable';
import { RoutePage, RoutePageFactory, RouteParams } from 'src/components/Router';
import { ToolbarProps } from 'src/components/Toolbar';
import { RuntimeMetricsVisualizations } from 'src/components/viewers/RuntimeMetricsVisualizations';
import { commonCss, padding } from 'src/Css';
import { queryKeys } from 'src/hooks/queryKeys';
import { Apis, ListRequest } from 'src/lib/Apis';
import { KeyValue } from 'src/lib/StaticGraphParser';
import { errorToMessage, formatDateString, logger } from 'src/lib/Utils';
import {
  getArtifactTypeName,
  isVisualizableArtifact,
  LEGACY_UI_METADATA_ARTIFACT_KEY,
} from 'src/lib/v2/RuntimeArtifactUtils';
import { Page, PageProps } from 'src/pages/Page';
import { classes } from 'typestyle';

export enum ArtifactDetailsTab {
  OVERVIEW = 0,
  RELATED_TASKS = 1,
}

const RELATED_TASKS_PATH = 'lineage';
const TAB_NAMES = ['Overview', 'Related tasks'];
const OUTPUT_RELATIONSHIP_TYPES = new Set<V2beta1IOType>([
  V2beta1IOType.OUTPUT,
  V2beta1IOType.ITERATOR_OUTPUT,
  V2beta1IOType.ONE_OF_OUTPUT,
  V2beta1IOType.TASK_FINAL_STATUS_OUTPUT,
]);
const RELATED_TASK_COLUMNS: Column[] = [
  { flex: 2, label: 'Relationship', sortKey: 'id' },
  { customRenderer: RelatedTaskLink, flex: 3, label: 'Task' },
];

interface ArtifactDetailsState {
  artifact?: V2beta1Artifact;
  hasError?: boolean;
}

class ArtifactDetails extends Page<{}, ArtifactDetailsState> {
  private relationshipsTableRef = React.createRef<CustomTable>();

  public state: ArtifactDetailsState = {};

  private get id(): string {
    return this.props.match.params[RouteParams.ID];
  }

  public async componentDidMount(): Promise<void> {
    this._isMounted = true;
    await this.load();
  }

  public render(): React.JSX.Element {
    const { artifact, hasError } = this.state;
    if (!artifact && !hasError) {
      return (
        <div className={commonCss.page}>
          <CircularProgress className={commonCss.absoluteCenter} />
        </div>
      );
    }
    if (!artifact) {
      return <div className={commonCss.page} />;
    }

    return (
      <div className={commonCss.page}>
        <Switch>
          <Route path={this.props.match.path} exact={true}>
            <ArtifactOverview artifact={artifact} onSwitch={this.switchTab} />
          </Route>
          <Route path={`${this.props.match.path}/${RELATED_TASKS_PATH}`} exact={true}>
            <ArtifactRelationshipsLoader
              artifactId={this.id}
              onSwitch={this.switchTab}
              tableRef={this.relationshipsTableRef}
            />
          </Route>
        </Switch>
      </div>
    );
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: {},
      breadcrumbs: [{ displayName: 'Artifacts', href: RoutePage.ARTIFACTS }],
      pageTitle: `Artifact ${this.id}`,
    };
  }

  public async refresh(): Promise<void> {
    await Promise.all([this.load(), this.relationshipsTableRef.current?.reload()]);
  }

  private load = async (): Promise<void> => {
    this.setStateSafe({ hasError: false });
    try {
      const artifact = await Apis.artifactServiceApiV2.artifact_1(this.id);
      if (!this._isMounted) {
        return;
      }
      this.props.updateToolbar({ pageTitle: artifact.name || `Artifact ${this.id}` });
      this.setStateSafe({ artifact, hasError: false });
      this.clearBanner();
    } catch (error) {
      const message = await errorToMessage(error);
      this.setStateSafe({ hasError: true });
      this.showPageError(message || `Error: failed to load artifact ${this.id}.`, error);
    }
  };

  private switchTab = (selectedTab: number) => {
    switch (selectedTab) {
      case ArtifactDetailsTab.RELATED_TASKS:
        this.props.history.push(`${this.props.match.url}/${RELATED_TASKS_PATH}`);
        return;
      case ArtifactDetailsTab.OVERVIEW:
        this.props.history.push(this.props.match.url.replace(`/${RELATED_TASKS_PATH}`, ''));
        return;
      default:
        logger.error(`Unknown selected tab ${selectedTab}.`);
    }
  };
}

function ArtifactOverview({
  artifact,
  onSwitch,
}: {
  artifact: V2beta1Artifact;
  onSwitch: (selectedTab: number) => void;
}) {
  const directlyVisualizable = isVisualizableArtifact(artifact);
  const shouldLookUpLegacyKey =
    !directlyVisualizable &&
    !!artifact.artifact_id &&
    (!artifact.type ||
      artifact.type === ArtifactArtifactType.TYPE_UNSPECIFIED ||
      artifact.type === ArtifactArtifactType.Artifact);
  const {
    data: legacyArtifactKey,
    error: legacyKeyError,
    isError: legacyKeyIsError,
  } = useQuery<string | undefined, Error>({
    queryKey: queryKeys.artifactVisualizationKey(artifact.artifact_id || ''),
    queryFn: () => findLegacyUiMetadataArtifactKey(artifact.artifact_id!),
    enabled: shouldLookUpLegacyKey,
    retry: false,
    staleTime: Infinity,
  });
  const details: Array<KeyValue<string>> = [
    ['Artifact ID', artifact.artifact_id || '-'],
    ['Name', artifact.name || '-'],
    ['Type', getArtifactTypeName(artifact)],
    ['Description', artifact.description || '-'],
    ['Namespace', artifact.namespace || '-'],
    ['Created at', formatDateString(artifact.created_at)],
  ];
  if (artifact.number_value !== undefined) {
    details.push(['Value', String(artifact.number_value)]);
  }
  if (artifact.metadata && Object.keys(artifact.metadata).length) {
    details.push(['Metadata', JSON.stringify(artifact.metadata)]);
  }

  return (
    <>
      <ArtifactTabs selectedTab={ArtifactDetailsTab.OVERVIEW} onSwitch={onSwitch} />
      <div className={classes(padding(20, 'lr'))}>
        <DetailsTable title='Artifact details' fields={details} />
        {artifact.uri && (
          <DetailsTable
            title='Artifact URI'
            fields={[[artifact.name || 'Artifact', { uri: artifact.uri }]]}
            valueComponent={ArtifactPreview}
            valueComponentProps={{ namespace: artifact.namespace }}
          />
        )}
        {legacyKeyIsError && (
          <Banner
            message='Unable to determine whether this artifact contains legacy UI visualizations. Refresh the page to try again.'
            additionalInfo={legacyKeyError.message}
            mode='error'
          />
        )}
        {(directlyVisualizable || legacyArtifactKey) && (
          <RuntimeMetricsVisualizations
            artifacts={[artifact]}
            artifactKey={legacyArtifactKey}
            namespace={artifact.namespace}
          />
        )}
      </div>
    </>
  );
}

async function findLegacyUiMetadataArtifactKey(artifactId: string): Promise<string | undefined> {
  const filter: V2beta1Filter = {
    predicates: [
      {
        key: 'key',
        operation: V2beta1PredicateOperation.EQUALS,
        string_value: LEGACY_UI_METADATA_ARTIFACT_KEY,
      },
    ],
  };
  const response = await Apis.artifactServiceApiV2.artifactTasks(
    undefined,
    undefined,
    [artifactId],
    undefined,
    undefined,
    1,
    'id asc',
    encodeURIComponent(JSON.stringify(filter)),
  );
  return response.artifact_tasks?.[0]?.key === LEGACY_UI_METADATA_ARTIFACT_KEY
    ? LEGACY_UI_METADATA_ARTIFACT_KEY
    : undefined;
}

interface ArtifactRelationshipsLoaderProps {
  artifactId: string;
  onSwitch: (selectedTab: number) => void;
  tableRef: React.RefObject<CustomTable | null>;
}

function ArtifactRelationshipsLoader(props: ArtifactRelationshipsLoaderProps) {
  const queryClient = useQueryClient();
  return <ArtifactRelationshipsTable {...props} queryClient={queryClient} />;
}

interface ArtifactRelationshipsTableProps extends ArtifactRelationshipsLoaderProps {
  queryClient: QueryClient;
}

interface ArtifactRelationshipsLoaderState {
  error?: string;
  rows: Row[];
}

interface PageTokenChain {
  nextTokens: Set<string>;
  successors: Map<string, string>;
}

class ArtifactRelationshipsTable extends React.PureComponent<
  ArtifactRelationshipsTableProps,
  ArtifactRelationshipsLoaderState
> {
  public state: ArtifactRelationshipsLoaderState = { rows: [] };

  private activeReloadGeneration = 0;
  private pageTokenChains = new Map<number, PageTokenChain>();

  public componentWillUnmount(): void {
    this.activeReloadGeneration++;
  }

  public render(): React.JSX.Element {
    return (
      <>
        <ArtifactTabs
          selectedTab={ArtifactDetailsTab.RELATED_TASKS}
          onSwitch={this.props.onSwitch}
        />
        <div className={classes(padding(20, 'lr'))}>
          <div className={commonCss.header2}>Producing and consuming tasks</div>
          {this.state.error && (
            <Banner
              message='Unable to load related tasks. Refresh the page to try again.'
              additionalInfo={this.state.error}
              mode='error'
            />
          )}
          <CustomTable
            ref={this.props.tableRef}
            columns={RELATED_TASK_COLUMNS}
            rows={this.state.rows}
            disableSelection={true}
            disableSorting={true}
            emptyMessage={this.state.error ? undefined : 'No related tasks found.'}
            initialSortColumn='id'
            initialSortOrder='asc'
            noFilterBox={true}
            reload={this.reload}
          />
        </div>
      </>
    );
  }

  private reload = async (request: ListRequest): Promise<string> => {
    const reloadGeneration = ++this.activeReloadGeneration;
    try {
      const response = await this.props.queryClient.fetchQuery({
        queryKey: queryKeys.artifactTasksPage(
          this.props.artifactId,
          request.pageToken,
          request.pageSize,
        ),
        queryFn: () =>
          Apis.artifactServiceApiV2.artifactTasks(
            undefined,
            undefined,
            [this.props.artifactId],
            undefined,
            request.pageToken,
            request.pageSize,
            'id asc',
          ),
      });
      const nextPageToken = response.next_page_token || '';
      if (reloadGeneration !== this.activeReloadGeneration) {
        return nextPageToken;
      }
      const repeatedPageToken = this.isRepeatedPageToken(request, nextPageToken);
      this.setState({
        error: repeatedPageToken
          ? `Artifact service returned a repeated page token: ${nextPageToken}`
          : undefined,
        rows: (response.artifact_tasks || []).map((artifactTask, index) => ({
          id: artifactTask.id || `${request.pageToken || 'first-page'}-${index}`,
          otherFields: [relationshipLabel(artifactTask, index), artifactTask],
        })),
      });
      return repeatedPageToken ? '' : nextPageToken;
    } catch (error) {
      const message = await errorToMessage(error);
      if (reloadGeneration === this.activeReloadGeneration) {
        this.setState({
          error: message || 'Artifact service failed to list related tasks.',
          rows: [],
        });
      }
      return '';
    }
  };

  private isRepeatedPageToken(request: ListRequest, nextPageToken: string): boolean {
    if (!nextPageToken) {
      return false;
    }
    const pageSize = request.pageSize || 0;
    const requestPageToken = request.pageToken || '';
    let chain = this.pageTokenChains.get(pageSize);
    if (!chain) {
      chain = { nextTokens: new Set(), successors: new Map() };
      this.pageTokenChains.set(pageSize, chain);
    }
    if (chain.successors.get(requestPageToken) === nextPageToken) {
      return false;
    }
    const repeated = requestPageToken === nextPageToken || chain.nextTokens.has(nextPageToken);
    chain.successors.set(requestPageToken, nextPageToken);
    chain.nextTokens.add(nextPageToken);
    return repeated;
  }
}

function RelatedTaskLink({ value }: CustomRendererProps<V2beta1ArtifactTask>) {
  const artifactTask = value;
  if (!artifactTask?.run_id) {
    return <>{artifactTask?.task_id || '-'}</>;
  }
  return (
    <Link className={commonCss.link} to={RoutePageFactory.runDetails(artifactTask.run_id)}>
      Run {artifactTask.run_id}
      {artifactTask.task_id ? ` · Task ${artifactTask.task_id}` : ''}
    </Link>
  );
}

function ArtifactTabs({
  selectedTab,
  onSwitch,
}: {
  selectedTab: ArtifactDetailsTab;
  onSwitch: (selectedTab: number) => void;
}) {
  return (
    <div className={classes(padding(20, 't'))}>
      <MD2Tabs tabs={TAB_NAMES} selectedTab={selectedTab} onSwitch={onSwitch} />
    </div>
  );
}

function relationshipLabel(artifactTask: V2beta1ArtifactTask, index: number): string {
  const direction =
    artifactTask.type && OUTPUT_RELATIONSHIP_TYPES.has(artifactTask.type)
      ? 'Produced as'
      : 'Consumed as';
  return `${direction} ${artifactTask.key || artifactTask.producer?.task_name || index + 1}`;
}

const EnhancedArtifactDetails = (props: PageProps) => (
  <ArtifactDetails {...props} key={props.match.params[RouteParams.ID]} />
);

export default EnhancedArtifactDetails;
