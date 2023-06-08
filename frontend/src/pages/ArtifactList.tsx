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

import * as React from 'react';
import { Link } from 'react-router-dom';
import { ListRequest } from 'src/lib/Apis';
import {
  Api,
  ArtifactCustomProperties,
  ArtifactProperties,
  getArtifactCreationTime,
  getArtifactTypes,
  getResourcePropertyViaFallBack,
} from 'src/mlmd/library';
import { Artifact, ArtifactType, GetArtifactsRequest } from 'src/third_party/mlmd';
import { ListOperationOptions } from 'src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_pb';
import { classes } from 'typestyle';
import { ArtifactLink } from 'src/components/ArtifactLink';
import CustomTable, {
  Column,
  CustomRendererProps,
  ExpandState,
  Row,
} from 'src/components/CustomTable';
import { RoutePageFactory } from 'src/components/Router';
import { ToolbarProps } from 'src/components/Toolbar';
import { commonCss, padding } from 'src/Css';
import {
  CollapsedAndExpandedRows,
  getExpandedRow,
  getStringEnumKey,
  groupRows,
  rowFilterFn,
  serviceErrorToString,
} from 'src/lib/Utils';
import { Page } from 'src/pages/Page';

interface ArtifactListProps {
  isGroupView: boolean;
}

interface ArtifactListState {
  artifacts: Artifact[];
  rows: Row[];
  expandedRows: Map<number, Row[]>;
  columns: Column[];
}

const ARTIFACT_PROPERTY_REPOS = [ArtifactProperties, ArtifactCustomProperties];
const PIPELINE_WORKSPACE_FIELDS = ['RUN_ID', 'PIPELINE_NAME', 'WORKSPACE'];
const NAME_FIELDS = [
  getStringEnumKey(ArtifactCustomProperties, ArtifactCustomProperties.NAME),
  getStringEnumKey(ArtifactCustomProperties, ArtifactCustomProperties.DISPLAY_NAME),
];

export class ArtifactList extends Page<ArtifactListProps, ArtifactListState> {
  private tableRef = React.createRef<CustomTable>();
  private api = Api.getInstance();
  private artifactTypesMap: Map<number, ArtifactType>;

  constructor(props: any) {
    super(props);
    this.state = {
      artifacts: [],
      columns: [
        {
          customRenderer: this.nameCustomRenderer,
          flex: 2,
          label: 'Pipeline/Workspace',
        },
        {
          customRenderer: this.nameCustomRenderer,
          flex: 1,
          label: 'Name',
        },
        { label: 'ID', flex: 1 },
        { label: 'Type', flex: 2 },
        { label: 'URI', flex: 2, customRenderer: this.uriCustomRenderer },
        { label: 'Created at', flex: 1 },
      ],
      expandedRows: new Map(),
      rows: [],
    };
    this.reload = this.reload.bind(this);
    this.toggleRowExpand = this.toggleRowExpand.bind(this);
    this.getExpandedArtifactsRow = this.getExpandedArtifactsRow.bind(this);
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: {},
      breadcrumbs: [],
      pageTitle: 'Artifacts',
    };
  }

  public render(): JSX.Element {
    const { rows, columns } = this.state;
    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <CustomTable
          ref={this.tableRef}
          columns={columns}
          rows={rows}
          disablePaging={this.props.isGroupView}
          disableSelection={true}
          reload={this.reload}
          initialSortColumn='pipelineName'
          initialSortOrder='asc'
          getExpandComponent={this.props.isGroupView ? this.getExpandedArtifactsRow : undefined}
          toggleExpansion={this.props.isGroupView ? this.toggleRowExpand : undefined}
          emptyMessage='No artifacts found.'
        />
      </div>
    );
  }

  public async refresh(): Promise<void> {
    if (this.tableRef.current) {
      await this.tableRef.current.reload();
    }
  }

  private async reload(request: ListRequest): Promise<string> {
    const listOperationOpts = new ListOperationOptions();
    if (request.pageSize) {
      listOperationOpts.setMaxResultSize(request.pageSize);
    }
    if (request.pageToken) {
      listOperationOpts.setNextPageToken(request.pageToken);
    }
    // TODO(jlyaoyuli): Add filter functionality for "entire" artifact list.

    // TODO: Consider making an Api method for returning and caching types
    if (!this.artifactTypesMap || !this.artifactTypesMap.size) {
      this.artifactTypesMap = await getArtifactTypes(
        this.api.metadataStoreService,
        this.showPageError.bind(this),
      );
    }
    const artifacts = this.props.isGroupView
      ? await this.getArtifacts()
      : await this.getArtifacts(listOperationOpts);
    this.clearBanner();
    let flattenedRows = await this.getFlattenedRowsFromArtifacts(request, artifacts);
    let groupedRows = await this.getGroupedRowsFromArtifacts(request, artifacts);
    // TODO(jlyaoyuli): Consider to support grouped rows with pagination.
    this.setState({
      artifacts,
      expandedRows: this.props.isGroupView ? groupedRows?.expandedRows : new Map(),
      rows: this.props.isGroupView ? groupedRows?.collapsedRows : flattenedRows,
    });

    return listOperationOpts.getNextPageToken();
  }

  private nameCustomRenderer: React.FC<CustomRendererProps<string>> = (
    props: CustomRendererProps<string>,
  ) => {
    return (
      <Link
        onClick={e => e.stopPropagation()}
        className={commonCss.link}
        to={RoutePageFactory.artifactDetails(Number(props.id))}
      >
        {props.value}
      </Link>
    );
  };

  private uriCustomRenderer: React.FC<CustomRendererProps<string>> = ({ value }) => (
    <ArtifactLink artifactUri={value} />
  );

  private async getArtifacts(listOperationOpts?: ListOperationOptions): Promise<Artifact[]> {
    try {
      const response = await this.api.metadataStoreService.getArtifacts(
        new GetArtifactsRequest().setOptions(listOperationOpts),
      );
      listOperationOpts?.setNextPageToken(response.getNextPageToken());
      return response.getArtifactsList();
    } catch (err) {
      // Code === 5 means no record found in backend. This is a temporary workaround.
      // TODO: remove err.code !== 5 check when backend is fixed.
      if (err.code !== 5) {
        this.showPageError(serviceErrorToString(err));
      }
    }
    return [];
  }

  private async getFlattenedRowsFromArtifacts(
    request: ListRequest,
    artifacts: Artifact[],
  ): Promise<Row[]> {
    try {
      const artifactsWithCreationTimes = await Promise.all(
        artifacts.map(async artifact => {
          const artifactId = artifact.getId();
          if (!artifactId) {
            return { artifact };
          }

          return {
            artifact,
            creationTime: await getArtifactCreationTime(artifactId, this.api.metadataStoreService),
          };
        }),
      );

      const flattenedRows: Row[] = artifactsWithCreationTimes
        .map(({ artifact, creationTime }) => {
          const typeId = artifact.getTypeId();
          const artifactType = this.artifactTypesMap!.get(typeId);
          const type = artifactType ? artifactType.getName() : artifact.getTypeId();
          return {
            id: `${artifact.getId()}`,
            otherFields: [
              getResourcePropertyViaFallBack(
                artifact,
                ARTIFACT_PROPERTY_REPOS,
                PIPELINE_WORKSPACE_FIELDS,
              ) || '[unknown]',
              getResourcePropertyViaFallBack(artifact, ARTIFACT_PROPERTY_REPOS, NAME_FIELDS) ||
                '[unknown]',
              artifact.getId(),
              type,
              artifact.getUri(),
              creationTime || '',
            ],
          } as Row;
        })
        .filter(rowFilterFn(request));
      // TODO(jlyaoyuli): Add sort functionality for entire artifact list.

      return flattenedRows;
    } catch (err) {
      if (err.message) {
        this.showPageError(err.message, err);
      } else {
        this.showPageError('Unknown error', err);
      }
    }
    return [];
  }

  /**
   * Temporary solution to apply sorting, filtering, and pagination to the
   * local list of artifacts until server-side handling is available
   * TODO: Replace once https://github.com/kubeflow/metadata/issues/73 is done.
   * @param request
   * @param artifacts
   */
  private async getGroupedRowsFromArtifacts(
    request: ListRequest,
    artifacts: Artifact[],
  ): Promise<CollapsedAndExpandedRows> {
    const flattenedRows = await this.getFlattenedRowsFromArtifacts(request, artifacts);
    return groupRows(flattenedRows);
  }

  /**
   * Toggles the expansion state of a row
   * @param index
   */
  private toggleRowExpand(index: number): void {
    const { rows } = this.state;
    if (!rows[index]) {
      return;
    }
    rows[index].expandState =
      rows[index].expandState === ExpandState.EXPANDED
        ? ExpandState.COLLAPSED
        : ExpandState.EXPANDED;
    this.setState({ rows });
  }

  private getExpandedArtifactsRow(index: number): React.ReactNode {
    return getExpandedRow(this.state.expandedRows, this.state.columns)(index);
  }
}

export default ArtifactList;
