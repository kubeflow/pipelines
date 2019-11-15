/*
 * Copyright 2019 Google LLC
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
import CustomTable, {
  Column,
  Row,
  ExpandState,
  CustomRendererProps,
} from '../components/CustomTable';
import { Page } from './Page';
import { ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import {
  getResourceProperty,
  rowCompareFn,
  rowFilterFn,
  groupRows,
  getExpandedRow,
  serviceErrorToString,
} from '../lib/Utils';
import { RoutePageFactory } from '../components/Router';
import { Link } from 'react-router-dom';
import { Artifact, ArtifactType } from '../generated/src/apis/metadata/metadata_store_pb';
import { ArtifactProperties, ArtifactCustomProperties, ListRequest, Apis } from '../lib/Apis';
import {
  GetArtifactTypesRequest,
  GetArtifactsRequest,
} from '../generated/src/apis/metadata/metadata_store_service_pb';
import { getArtifactCreationTime } from '../lib/MetadataUtils';
import { ArtifactLink } from '../components/ArtifactLink';

interface ArtifactListState {
  artifacts: Artifact[];
  rows: Row[];
  expandedRows: Map<number, Row[]>;
  columns: Column[];
}

class ArtifactList extends Page<{}, ArtifactListState> {
  private tableRef = React.createRef<CustomTable>();

  constructor(props: any) {
    super(props);
    this.state = {
      artifacts: [],
      columns: [
        {
          customRenderer: this.nameCustomRenderer,
          flex: 2,
          label: 'Pipeline/Workspace',
          sortKey: 'pipelineName',
        },
        {
          customRenderer: this.nameCustomRenderer,
          flex: 1,
          label: 'Name',
          sortKey: 'name',
        },
        { label: 'ID', flex: 1, sortKey: 'id' },
        { label: 'Type', flex: 2, sortKey: 'type' },
        { label: 'URI', flex: 2, sortKey: 'uri', customRenderer: this.uriCustomRenderer },
        { label: 'Created at', flex: 1, sortKey: 'created_at' },
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
          disablePaging={true}
          disableSelection={true}
          reload={this.reload}
          initialSortColumn='pipelineName'
          initialSortOrder='asc'
          getExpandComponent={this.getExpandedArtifactsRow}
          toggleExpansion={this.toggleRowExpand}
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
    const { response: res, error: err } = await Apis.getMetadataServicePromiseClient().getArtifacts(
      new GetArtifactsRequest(),
    );

    if (err) {
      // Code === 5 means no record found in backend. This is a temporary workaround.
      // TODO: remove err.code !== 5 check when backend is fixed.
      if (err.code !== 5) {
        this.showPageError(serviceErrorToString(err));
      }
      return '';
    }

    const artifacts = (res && res.getArtifactsList()) || [];
    await this.getRowsFromArtifacts(request, artifacts);
    return '';
  }

  private nameCustomRenderer: React.FC<CustomRendererProps<string>> = (
    props: CustomRendererProps<string>,
  ) => {
    const [artifactType, artifactId] = props.id.split(':');
    return (
      <Link
        onClick={e => e.stopPropagation()}
        className={commonCss.link}
        to={RoutePageFactory.artifactDetails(artifactType, Number(artifactId))}
      >
        {props.value}
      </Link>
    );
  };

  private uriCustomRenderer: React.FC<CustomRendererProps<string>> = ({ value }) => (
    <ArtifactLink artifactUri={value} />
  );

  /**
   * Temporary solution to apply sorting, filtering, and pagination to the
   * local list of artifacts until server-side handling is available
   * TODO: Replace once https://github.com/kubeflow/metadata/issues/73 is done.
   * @param request
   */
  private async getRowsFromArtifacts(request: ListRequest, artifacts: Artifact[]): Promise<void> {
    const artifactTypesMap = new Map<number, ArtifactType>();
    // TODO: Consider making an Api method for returning and caching types
    const {
      response: res,
      error: err,
    } = await Apis.getMetadataServicePromiseClient().getArtifactTypes(
      new GetArtifactTypesRequest(),
    );
    if (err) {
      this.showPageError(serviceErrorToString(err));
      return;
    }

    ((res && res.getArtifactTypesList()) || []).forEach(artifactType => {
      artifactTypesMap.set(artifactType.getId()!, artifactType);
    });

    try {
      // TODO: When backend supports sending creation time back when we list
      // artifacts, let's use it directly.
      const artifactsWithCreationTimes = await Promise.all(
        artifacts.map(async artifact => {
          const artifactId = artifact.getId();
          if (!artifactId) {
            return { artifact };
          }

          return {
            artifact,
            creationTime: await getArtifactCreationTime(artifactId),
          };
        }),
      );

      const collapsedAndExpandedRows = groupRows(
        artifactsWithCreationTimes
          .map(({ artifact, creationTime }) => {
            const typeId = artifact.getTypeId();
            const type =
              typeId && artifactTypesMap && artifactTypesMap.get(typeId)
                ? artifactTypesMap.get(typeId)!.getName()
                : typeId;
            return {
              id: `${type}:${artifact.getId()}`, // Join with colon so we can build the link
              otherFields: [
                getResourceProperty(artifact, ArtifactProperties.PIPELINE_NAME) ||
                  getResourceProperty(artifact, ArtifactCustomProperties.WORKSPACE, true),
                getResourceProperty(artifact, ArtifactProperties.NAME),
                artifact.getId(),
                type,
                artifact.getUri(),
                creationTime || '',
              ],
            } as Row;
          })
          .filter(rowFilterFn(request))
          .sort(rowCompareFn(request, this.state.columns)),
      );

      this.setState({
        artifacts,
        expandedRows: collapsedAndExpandedRows.expandedRows,
        rows: collapsedAndExpandedRows.collapsedRows,
      });
    } catch (err) {
      if (err.message) {
        this.showPageError(err.message, err);
      } else {
        this.showPageError('Unknown error', err);
      }
    }
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
