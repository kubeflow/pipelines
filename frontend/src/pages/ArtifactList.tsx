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
import CustomTable, { Column, Row, ExpandState, CustomRendererProps } from '../components/CustomTable';
import { Page } from './Page';
import { ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { getResourceProperty, rowCompareFn, rowFilterFn, groupRows, getExpandedRow, serviceErrorToString } from '../lib/Utils';
import { RoutePage, RouteParams } from '../components/Router';
import { Link } from 'react-router-dom';
import { Artifact, ArtifactType } from '../generated/src/apis/metadata/metadata_store_pb';
import { ArtifactProperties, ArtifactCustomProperties, ListRequest, Apis } from '../lib/Apis';
import { GetArtifactTypesRequest, GetArtifactsRequest } from '../generated/src/apis/metadata/metadata_store_service_pb';

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
          sortKey: 'pipelineName'
        },
        {
          customRenderer: this.nameCustomRenderer,
          flex: 1,
          label: 'Name',
          sortKey: 'name',
        },
        { label: 'ID', flex: 1, sortKey: 'id' },
        { label: 'Type', flex: 2, sortKey: 'type' },
        { label: 'URI', flex: 2, sortKey: 'uri', },
        // TODO: Get timestamp from the event that created this artifact.
        // {label: 'Created at', flex: 1, sortKey: 'created_at'},
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
        <CustomTable ref={this.tableRef}
          columns={columns}
          rows={rows}
          disablePaging={true}
          disableSelection={true}
          reload={this.reload}
          initialSortColumn='pipelineName'
          initialSortOrder='asc'
          getExpandComponent={this.getExpandedArtifactsRow}
          toggleExpansion={this.toggleRowExpand}
          emptyMessage='No artifacts found.' />
      </div>
    );
  }

  public async refresh(): Promise<void> {
    if (this.tableRef.current) {
      await this.tableRef.current.reload();
    }
  }

  private async reload(request: ListRequest): Promise<string> {
    Apis.getMetadataServiceClient().getArtifacts(new GetArtifactsRequest(), (err, res) => {
      if (err) {
        this.showPageError(serviceErrorToString(err));
        return;
      }

      const artifacts = (res && res.getArtifactsList()) || [];
      this.getRowsFromArtifacts(request, artifacts);
      this.clearBanner();
    });
    return '';
  }


  private nameCustomRenderer: React.FC<CustomRendererProps<string>> =
    (props: CustomRendererProps<string>) => {
      const [artifactType, artifactId] = props.id.split(':');
      const link = RoutePage.ARTIFACT_DETAILS
        .replace(`:${RouteParams.ARTIFACT_TYPE}+`, artifactType)
        .replace(`:${RouteParams.ID}`, artifactId);
      return (
        <Link onClick={(e) => e.stopPropagation()}
          className={commonCss.link}
          to={link}>
          {props.value}
        </Link>
      );
  }

  /**
   * Temporary solution to apply sorting, filtering, and pagination to the
   * local list of artifacts until server-side handling is available
   * TODO: Replace once https://github.com/kubeflow/metadata/issues/73 is done.
   * @param request
   */
  private getRowsFromArtifacts(request: ListRequest, artifacts: Artifact[]): void {
    const artifactTypesMap = new Map<number, ArtifactType>();
    // TODO: Consider making an Api method for returning and caching types
    Apis.getMetadataServiceClient().getArtifactTypes(new GetArtifactTypesRequest(), (err, res) => {
      if (err) {
        this.showPageError(serviceErrorToString(err));
        return;
      }

      (res && res.getArtifactTypesList() || []).forEach((artifactType) => {
        artifactTypesMap.set(artifactType.getId()!, artifactType);
      });

      const collapsedAndExpandedRows =
        groupRows(artifacts
          .map((artifact) => { // Flattens
            const typeId = artifact.getTypeId();
            const type = (typeId && artifactTypesMap && artifactTypesMap.get(typeId))
              ? artifactTypesMap.get(typeId)!.getName()
              : typeId;
            return {
              id: `${type}:${artifact.getId()}`, // Join with colon so we can build the link
              otherFields: [
                getResourceProperty(artifact, ArtifactProperties.PIPELINE_NAME)
                || getResourceProperty(artifact, ArtifactCustomProperties.WORKSPACE, true),
                getResourceProperty(artifact, ArtifactProperties.NAME),
                artifact.getId(),
                type,
                artifact.getUri(),
                // TODO: Get timestamp from the event that created this artifact.
                // formatDateString(
                //   getArtifactProperty(a, ArtifactProperties.CREATE_TIME) || ''),
              ],
            } as Row;
          })
          .filter(rowFilterFn(request))
          .sort(rowCompareFn(request, this.state.columns))
        );

      this.setState({
        artifacts,
        expandedRows: collapsedAndExpandedRows.expandedRows,
        rows: collapsedAndExpandedRows.collapsedRows,
      });
    });
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
    rows[index].expandState = rows[index].expandState === ExpandState.EXPANDED ?
      ExpandState.COLLAPSED : ExpandState.EXPANDED;
    this.setState({ rows });
  }

  private getExpandedArtifactsRow(index: number): React.ReactNode {
    return getExpandedRow(this.state.expandedRows, this.state.columns)(index);
  }
}

export default ArtifactList;
