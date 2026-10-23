/*
 * Copyright 2019 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the 'License');
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an 'AS IS' BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { Button, CircularProgress } from '@mui/material';
import { QueryClient, useQuery, useQueryClient } from '@tanstack/react-query';
import * as React from 'react';
import { Link, Route, Routes } from 'react-router';
import { ArtifactArtifactType, V2Artifact, V2ArtifactTask, V2IOType } from 'src/apisv2/artifact';
import { V2Filter, V2PredicateOperation } from 'src/apisv2/filter';
import { PipelineTaskTaskType } from 'src/apisv2/run';
import MD2Tabs from 'src/atoms/MD2Tabs';
import ArtifactPreview from 'src/components/ArtifactPreview';
import NativeArtifactLineage from 'src/components/NativeArtifactLineage';
import Banner from 'src/components/Banner';
import CustomTable, { Column, CustomRendererProps, Row } from 'src/components/CustomTable';
import DetailsTable from 'src/components/DetailsTable';
import { RoutePage, RoutePageFactory, RouteParams } from 'src/components/Router';
import { ToolbarProps } from 'src/components/Toolbar';
import { RuntimeMetricsVisualizations } from 'src/components/viewers/RuntimeMetricsVisualizations';
import { commonCss, padding } from 'src/Css';
import { queryKeys } from 'src/hooks/queryKeys';
import { Apis, ListRequest } from 'src/lib/Apis';
import { KeyValue } from 'src/lib/DetailsTableTypes';
import { errorToMessage, formatDateString, logger } from 'src/lib/Utils';
import {
  isInputArtifactTaskType,
  isOutputArtifactTaskType,
  OUTPUT_ARTIFACT_TASK_TYPES,
} from 'src/lib/v2/ArtifactTaskUtils';
import {
  getArtifactTypeName,
  hasArtifactContentPreview,
  isLegacyUiMetadataArtifact,
  isVisualizableArtifact,
  LEGACY_UI_METADATA_ARTIFACT_KEYS,
} from 'src/lib/v2/RuntimeArtifactUtils';
import { PageTokenTracker } from 'src/lib/v2/PaginationUtils';
import { getTaskDisplayName, listAllRunTasks } from 'src/lib/v2/RunTaskUtils';
import { Page, PageProps } from 'src/pages/Page';
import { classes } from 'typestyle';
import MD2Tabs from '../atoms/MD2Tabs';
import { ResourceInfo, ResourceType } from '../components/ResourceInfo';
import { RoutePage, RoutePageFactory, RouteParams } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { commonCss, padding } from '../Css';
import {
  errorToMessage,
  isServiceError,
  logger,
  serviceErrorToString,
  titleCase,
} from '../lib/Utils';
import { Page, PageProps } from './Page';
import { ArtifactHelpers } from 'src/mlmd/MlmdUtils';

export enum ArtifactDetailsTab {
  OVERVIEW = 0,
  LINEAGE_EXPLORER = 1,
}

const LINEAGE_PATH = 'lineage';

const TABS = {
  [ArtifactDetailsTab.OVERVIEW]: { name: 'Overview' },
  [ArtifactDetailsTab.LINEAGE_EXPLORER]: { name: 'Lineage Explorer' },
};

const TAB_NAMES = [ArtifactDetailsTab.OVERVIEW, ArtifactDetailsTab.LINEAGE_EXPLORER].map(
  (tabConfig) => TABS[tabConfig].name,
);

interface ArtifactDetailsState {
  artifact?: Artifact;
  artifactType?: ArtifactType;
  hasError?: boolean;
}

class ArtifactDetails extends Page<{}, ArtifactDetailsState> {
  private get fullTypeName(): string {
    return this.state.artifactType?.getName() || '';
  }

  private get properTypeName(): string {
    const parts = this.fullTypeName.split('/');
    if (!parts.length) {
      return '';
    }
    return titleCase(parts[parts.length - 1]);
  }

  private get id(): number {
    return Number(this.props.match.params[RouteParams.ID]);
  }

  private static buildResourceDetailsPageRoute(
    resource: LineageResource,
    _: string, // typename is no longer used
  ): string {
    // HACK: this distinguishes artifact from execution, only artifacts have
    // the getUri() method.
    // TODO: switch to use typedResource
    if ('getUri' in resource && typeof resource.getUri === 'function') {
      return RoutePageFactory.artifactDetails(resource.getId());
    } else {
      return RoutePageFactory.executionDetails(resource.getId());
    }
  }

  public state: ArtifactDetailsState = {};

  private api = Api.getInstance();

  public async componentDidMount(): Promise<void> {
    this._isMounted = true;
    return this.load();
  }

  public render(): React.JSX.Element {
    if (!this.state.artifact && !this.state.hasError) {
      return (
        <div className={commonCss.page}>
          <CircularProgress className={commonCss.absoluteCenter} />
        </div>
      );
    }
    if (!this.state.artifact) {
      return <div className={commonCss.page} />;
    }
    return (
      <div className={classes(commonCss.page)}>
        <Switch>
          {/*
           ** This is react-router's nested route feature.
           ** reference: https://reacttraining.com/react-router/web/example/nesting
           */}
          <Route path={this.props.match.path} exact={true}>
            <>
              <div className={classes(padding(20, 't'))}>
                <MD2Tabs
                  tabs={TAB_NAMES}
                  selectedTab={ArtifactDetailsTab.OVERVIEW}
                  onSwitch={this.switchTab}
                />
              </div>
              <div className={classes(padding(20, 'lr'))}>
                <ResourceInfo
                  resourceType={ResourceType.ARTIFACT}
                  typeName={this.properTypeName}
                  resource={this.state.artifact}
                />
              </div>
            </>
          </Route>
          <Route path={`${this.props.match.path}/${LINEAGE_PATH}`} exact={true}>
            <>
              <div className={classes(padding(20, 't'))}>
                <MD2Tabs
                  tabs={TAB_NAMES}
                  selectedTab={ArtifactDetailsTab.LINEAGE_EXPLORER}
                  onSwitch={this.switchTab}
                />
              </div>
              <LineageView
                target={this.state.artifact}
                buildResourceDetailsPageRoute={ArtifactDetails.buildResourceDetailsPageRoute}
              />
            </>
          </Route>
        </Switch>
      </div>
    );
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: {},
      breadcrumbs: [{ displayName: 'Artifacts', href: RoutePage.ARTIFACTS }],
      pageTitle: `Artifact #${this.id}`,
    };
  }

  public async refresh(): Promise<void> {
    return this.load();
  }

  private load = async (): Promise<void> => {
    this.setStateSafe({ hasError: false });
    const request = new GetArtifactsByIDRequest();
    request.setArtifactIdsList([Number(this.id)]);

    try {
      const response = await this.api.metadataStoreService.getArtifactsByID(request);
      if (response.getArtifactsList().length === 0) {
        this.setStateSafe({ hasError: true });
        this.showPageError(`No artifact identified by id: ${this.id}`);
        return;
      }
      if (response.getArtifactsList().length > 1) {
        this.setStateSafe({ hasError: true });
        this.showPageError(`Found multiple artifacts with ID: ${this.id}`);
        return;
      }
      const artifact = response.getArtifactsList()[0];
      const typeRequest = new GetArtifactTypesByIDRequest();
      typeRequest.setTypeIdsList([artifact.getTypeId()]);
      const typeResponse = await this.api.metadataStoreService.getArtifactTypesByID(typeRequest);
      const artifactType = typeResponse.getArtifactTypesList()[0] || undefined;

      let title = ArtifactHelpers.getName(artifact);
      const version = getResourceProperty(artifact, ArtifactProperties.VERSION);
      if (version) {
        title += ` (version: ${version})`;
      }
      if (!this._isMounted) {
        return;
      }

      this.props.updateToolbar({
        pageTitle: title,
      });
      this.setStateSafe({ artifact, artifactType });
    } catch (err) {
      this.setStateSafe({ hasError: true });
      if (isServiceError(err)) {
        this.showPageError(serviceErrorToString(err));
      } else {
        const errorMessage = await errorToMessage(err);
        this.showPageError(
          errorMessage ? `Error: ${errorMessage}` : 'Error: failed to load artifact.',
        );
      }
    }
  };

  private switchTab = (selectedTab: number) => {
    switch (selectedTab) {
      case ArtifactDetailsTab.LINEAGE_EXPLORER:
        return this.props.history.push(`${this.props.match.url}/${LINEAGE_PATH}`);
      case ArtifactDetailsTab.OVERVIEW:
        return this.props.history.push(this.props.match.url);
      default:
        logger.error(`Unknown selected tab ${selectedTab}.`);
    }
  };
}

function ArtifactOverview({
  artifact,
  onSwitch,
  refreshGeneration,
}: {
  artifact: V2Artifact;
  onSwitch: (selectedTab: number) => void;
  refreshGeneration: number;
}) {
  const [mountedAt] = React.useState(() => Date.now());
  const [openedPreviewId, setOpenedPreviewId] = React.useState<string>();
  const hasContentPreview = hasArtifactContentPreview(artifact);
  const contentPreviewOpen = !!artifact.artifact_id && openedPreviewId === artifact.artifact_id;
  const directlyVisualizable = isVisualizableArtifact(artifact);
  const shouldLookUpLegacyKey =
    !directlyVisualizable &&
    !!artifact.artifact_id &&
    (!artifact.type ||
      artifact.type === ArtifactArtifactType.TYPE_UNSPECIFIED ||
      artifact.type === ArtifactArtifactType.Artifact);
  const artifactCreatedAt = artifact.created_at?.getTime();
  const artifactAge = artifactCreatedAt === undefined ? undefined : mountedAt - artifactCreatedAt;
  // The timestamp comes from the API server while mountedAt comes from the browser. A bounded skew
  // allowance preserves eventual-consistency recovery without polling every historical artifact.
  const shouldReconcileMissingKey =
    artifactAge !== undefined &&
    artifactAge >= -LEGACY_KEY_CLOCK_SKEW_TOLERANCE_MS &&
    artifactAge <= LEGACY_KEY_RECONCILIATION_WINDOW_MS + LEGACY_KEY_CLOCK_SKEW_TOLERANCE_MS;
  const queryClient = useQueryClient();
  const visualizationQueryKey = [
    ...queryKeys.artifactVisualizationKey(artifact.artifact_id || ''),
    refreshGeneration,
  ];
  const previousVisualizationQueryKey = [
    ...queryKeys.artifactVisualizationKey(artifact.artifact_id || ''),
    refreshGeneration - 1,
  ];
  const { data: legacyKeyResult } = useQuery<LegacyUiMetadataKeyResult, Error>({
    queryKey: visualizationQueryKey,
    queryFn: async () => {
      const result = await findLegacyUiMetadataArtifactKey(artifact.artifact_id!);
      const previousResult =
        queryClient.getQueryData<LegacyUiMetadataKeyResult>(visualizationQueryKey) ||
        (refreshGeneration > 0
          ? queryClient.getQueryData<LegacyUiMetadataKeyResult>(previousVisualizationQueryKey)
          : undefined);
      return !result.key && result.errors.length && previousResult?.key
        ? { ...result, key: previousResult.key }
        : result;
    },
    enabled: shouldLookUpLegacyKey,
    placeholderData: (previousResult) => previousResult,
    retry: false,
    staleTime: (query) =>
      query.state.data?.key && query.state.data.errors.length === 0 ? Infinity : 0,
    refetchInterval: (query) => {
      const result = query.state.data;
      const isComplete = !!result?.key && result.errors.length === 0;
      const attemptCount = query.state.dataUpdateCount + query.state.errorUpdateCount;
      const hasRelationshipErrors = !!result?.errors.length || query.state.errorUpdateCount > 0;
      const shouldReconcile = hasRelationshipErrors || (!result?.key && shouldReconcileMissingKey);
      return !isComplete && shouldReconcile && attemptCount < LEGACY_KEY_RECONCILIATION_ATTEMPTS
        ? LEGACY_KEY_RECONCILIATION_INTERVAL_MS
        : false;
    },
  });
  const legacyArtifactKey = legacyKeyResult?.key;
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
        {!!legacyKeyResult?.errors.length && (
          <Banner
            message='Some artifact relationships could not be checked. Available visualization information is still shown; refresh the page to try again.'
            additionalInfo={legacyKeyResult.errors.join('\n')}
            mode='warning'
          />
        )}
        {(directlyVisualizable || legacyArtifactKey) && (
          <>
            {hasContentPreview && (
              <Button
                size='small'
                aria-expanded={contentPreviewOpen}
                onClick={() =>
                  setOpenedPreviewId(contentPreviewOpen ? undefined : artifact.artifact_id)
                }
              >
                {contentPreviewOpen ? 'Hide artifact visualization' : 'Show artifact visualization'}
              </Button>
            )}
            <RuntimeMetricsVisualizations
              artifacts={[artifact]}
              artifactKey={legacyArtifactKey}
              namespace={artifact.namespace}
              active={!hasContentPreview || contentPreviewOpen}
              refreshGeneration={refreshGeneration}
            />
          </>
        )}
      </div>
    </>
  );
}

async function findLegacyUiMetadataArtifactKey(
  artifactId: string,
): Promise<LegacyUiMetadataKeyResult> {
  const filter: V2Filter = {
    predicates: [
      {
        key: 'key',
        operation: V2PredicateOperation.IN,
        string_values: { values: [...LEGACY_UI_METADATA_ARTIFACT_KEYS] },
      },
    ],
  };
  const results = await Promise.allSettled(
    // The API accepts only one IO type per request. Query each known producing type rather than
    // using an untyped first page, where a same-key input could hide the producing relationship.
    OUTPUT_ARTIFACT_TASK_TYPES.map((type) =>
      Apis.artifactServiceApiV2.artifactTasks(
        undefined,
        undefined,
        [artifactId],
        type,
        undefined,
        1,
        'id asc',
        encodeURIComponent(JSON.stringify(filter)),
      ),
    ),
  );
  const errors: string[] = [];
  const responses = [];
  for (const [index, result] of results.entries()) {
    if (result.status === 'fulfilled') {
      responses.push(result.value);
    } else {
      errors.push(`${OUTPUT_ARTIFACT_TASK_TYPES[index]}: ${await errorToMessage(result.reason)}`);
    }
  }
  const producingRelationship = responses
    .flatMap((response) => response.artifact_tasks ?? [])
    .find(
      (artifactTask) =>
        isLegacyUiMetadataArtifact({}, artifactTask.key) &&
        isOutputArtifactTaskType(artifactTask.type),
    );
  return {
    errors,
    key: producingRelationship?.key,
  };
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

type ArtifactTasksResponse = Awaited<ReturnType<typeof Apis.artifactServiceApiV2.artifactTasks>>;

class ArtifactRelationshipsTable extends React.PureComponent<
  ArtifactRelationshipsTableProps,
  ArtifactRelationshipsLoaderState
> {
  public state: ArtifactRelationshipsLoaderState = { rows: [] };

  private activeReloadGeneration = 0;
  private pageTokenTracker = new PageTokenTracker();

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
    const queryKey = queryKeys.artifactTasksPage(
      this.props.artifactId,
      request.pageToken,
      request.pageSize,
    );
    try {
      const response = await this.props.queryClient.fetchQuery({
        queryKey,
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
      const repeatedPageToken = this.pageTokenTracker.isRepeated(
        String(request.pageSize || 0),
        request.pageToken,
        nextPageToken,
      );
      this.setState({
        error: repeatedPageToken
          ? `Artifact service returned a repeated page token: ${nextPageToken}`
          : undefined,
        rows: buildArtifactTaskRows(response, request.pageToken),
      });
      return repeatedPageToken ? '' : nextPageToken;
    } catch (error) {
      const message = await errorToMessage(error);
      if (reloadGeneration === this.activeReloadGeneration) {
        const cachedResponse = this.props.queryClient.getQueryData<ArtifactTasksResponse>(queryKey);
        this.setState({
          error: message || 'Artifact service failed to list related tasks.',
          rows: cachedResponse ? buildArtifactTaskRows(cachedResponse, request.pageToken) : [],
        });
      }
      return '';
    }
  };
}

function buildArtifactTaskRows(response: ArtifactTasksResponse, pageToken?: string): Row[] {
  return (response.artifact_tasks || []).map((artifactTask, index) => ({
    id: artifactTask.id || `${pageToken || 'first-page'}-${index}`,
    otherFields: [relationshipLabel(artifactTask, index), artifactTask],
  }));
}

function relatedTaskRenderer(props: CustomRendererProps<V2ArtifactTask>) {
  return <RelatedTaskLink {...props} />;
}

function RelatedTaskLink({ value }: CustomRendererProps<V2ArtifactTask>) {
  const artifactTask = value;
  const producerName =
    artifactTask && isOutputArtifactTaskType(artifactTask.type)
      ? artifactTask.producer?.task_name
      : undefined;
  const runId = artifactTask?.run_id || '';
  const run = useQuery({
    queryKey: queryKeys.v2RunDetail(runId),
    queryFn: () => Apis.runServiceApiV2.getRun(runId),
    enabled: !!runId,
    staleTime: 60_000,
    retry: false,
  });
  // Rows in the same run share one paginated task query, including in-flight requests.
  const tasks = useQuery({
    queryKey: queryKeys.runTasks(runId),
    queryFn: () => listAllRunTasks(runId),
    enabled: !!runId && !!artifactTask?.task_id,
    staleTime: 60_000,
    retry: false,
  });
  if (!artifactTask?.run_id) {
    return <>{producerName || artifactTask?.task_id || '-'}</>;
  }
  const task = tasks.data?.find((candidate) => candidate.task_id === artifactTask.task_id);
  const runName = run.data?.display_name || `Run ${runId}`;
  const taskName = task
    ? getTaskDisplayName(task, `Task ${artifactTask.task_id}`)
    : producerName || `Task ${artifactTask.task_id}`;
  const taskLabel =
    task?.type === PipelineTaskTaskType.ROOT
      ? isOutputArtifactTaskType(artifactTask.type)
        ? 'Run output (root)'
        : 'Root task'
      : taskName;
  return (
    <Link
      className={commonCss.link}
      title={`Run ${runId}${artifactTask.task_id ? ` · Task ${artifactTask.task_id}` : ''}`}
      to={
        artifactTask.task_id
          ? RoutePageFactory.runDetailsTask(artifactTask.run_id, artifactTask.task_id)
          : RoutePageFactory.runDetails(artifactTask.run_id)
      }
    >
      {runName}
      {artifactTask.task_id ? ` · ${taskLabel}` : ''}
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

function relationshipLabel(artifactTask: V2ArtifactTask, index: number): string {
  const relationshipName = artifactTask.key || artifactTask.producer?.task_name || index + 1;
  if (isOutputArtifactTaskType(artifactTask.type)) {
    return `Produced as ${relationshipName}`;
  }
  if (isInputArtifactTaskType(artifactTask.type)) {
    return `Consumed as ${relationshipName}`;
  }
  const relationshipType =
    artifactTask.type && artifactTask.type !== V2IOType.UNSPECIFIED ? artifactTask.type : 'unknown';
  return `Related as ${relationshipType}: ${relationshipName}`;
}

const EnhancedArtifactDetails = (props: PageProps) => {
  return <ArtifactDetails {...props} key={props.match.params[RouteParams.ID]} />;
};

export default EnhancedArtifactDetails;
