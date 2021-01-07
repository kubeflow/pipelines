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

import {
  Api,
  Artifact,
  ArtifactProperties,
  ArtifactCustomProperties,
  ListRequest,
  ArtifactType,
  getArtifactCreationTime,
  getArtifactTypes,
  getResourcePropertyViaFallBack,
  GetArtifactsRequest,
} from '@kubeflow/frontend';
import * as React from 'react';
import { Link } from 'react-router-dom';
import { classes } from 'typestyle';
import CustomTable, {
  Column,
  Row,
  ExpandState,
  CustomRendererProps,
} from '../components/CustomTable';
import { Page } from './Page';
import { ToolbarProps } from '../components/Toolbar';
import { commonCss, padding } from '../Css';
import {
  rowCompareFn,
  rowFilterFn,
  groupRows,
  getExpandedRow,
  serviceErrorToString,
  CollapsedAndExpandedRows,
} from '../lib/Utils';
import { RoutePageFactory } from '../components/Router';
import { ArtifactLink } from '../components/ArtifactLink';
import { TFunction } from 'i18next';
import { withTranslation } from 'react-i18next';

interface ArtifactListState {
  artifacts: Artifact[];
  rows: Row[];
  expandedRows: Map<number, Row[]>;
  columns: Column[];
}

const ARTIFACT_PROPERTY_REPOS = [ArtifactProperties, ArtifactCustomProperties];
const PIPELINE_WORKSPACE_FIELDS = ['RUN_ID', 'PIPELINE_NAME', 'WORKSPACE'];
const NAME_FIELDS = ['NAME'];

class ArtifactList extends Page<{ t: TFunction }, ArtifactListState> {
  private tableRef = React.createRef<CustomTable>();
  private api = Api.getInstance();
  private artifactTypesMap: Map<number, ArtifactType>;

  constructor(props: any) {
    super(props);
    const { t } = this.props;
    this.state = {
      artifacts: [],
      columns: [
        {
          customRenderer: this.nameCustomRenderer,
          flex: 2,
          label: t('pipelineWorkspace'),
          sortKey: 'pipelineName',
        },
        {
          customRenderer: this.nameCustomRenderer,
          flex: 1,
          label: t('common:name'),
          sortKey: 'name',
        },
        { label: t('common:id'), flex: 1, sortKey: 'id' },
        { label: t('common:type'), flex: 2, sortKey: 'type' },
        { label: t('common:uri'), flex: 2, sortKey: 'uri', customRenderer: this.uriCustomRenderer },
        { label: t('common:createdAt'), flex: 1, sortKey: 'created_at' },
      ],
      expandedRows: new Map(),
      rows: [],
    };
    this.reload = this.reload.bind(this);
    this.toggleRowExpand = this.toggleRowExpand.bind(this);
    this.getExpandedArtifactsRow = this.getExpandedArtifactsRow.bind(this);
  }

  public getInitialToolbarState(): ToolbarProps {
    const { t } = this.props;
    return {
      actions: {},
      breadcrumbs: [],
      pageTitle: t('common:artifacts'),
      t,
    };
  }

  public render(): JSX.Element {
    const { rows, columns } = this.state;
    const { t } = this.props;
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
          emptyMessage={t('noArtifactsFound')}
          t={t}
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
    // TODO: Consider making an Api method for returning and caching types
    if (!this.artifactTypesMap || !this.artifactTypesMap.size) {
      this.artifactTypesMap = await getArtifactTypes(
        this.api.metadataStoreService,
        this.showPageError.bind(this),
      );
    }
    if (!this.state.artifacts.length) {
      const artifacts = await this.getArtifacts();
      this.clearBanner();
      const collapsedAndExpandedRows = await this.getRowsFromArtifacts(request, artifacts);
      if (collapsedAndExpandedRows) {
        this.setState({
          artifacts,
          expandedRows: collapsedAndExpandedRows.expandedRows,
          rows: collapsedAndExpandedRows.collapsedRows,
        });
      }
    }
    return '';
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

  private async getArtifacts(): Promise<Artifact[]> {
    try {
      const response = await this.api.metadataStoreService.getArtifacts(new GetArtifactsRequest());
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

  /**
   * Temporary solution to apply sorting, filtering, and pagination to the
   * local list of artifacts until server-side handling is available
   * TODO: Replace once https://github.com/kubeflow/metadata/issues/73 is done.
   * @param request
   * @param artifacts
   */
  private async getRowsFromArtifacts(
    request: ListRequest,
    artifacts: Artifact[],
  ): Promise<CollapsedAndExpandedRows | undefined> {
    const { t } = this.props;
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
            creationTime: await getArtifactCreationTime(artifactId, this.api.metadataStoreService),
          };
        }),
      );

      return groupRows(
        artifactsWithCreationTimes
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
                ),
                getResourcePropertyViaFallBack(artifact, ARTIFACT_PROPERTY_REPOS, NAME_FIELDS),
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
    } catch (err) {
      if (err.message) {
        this.showPageError(err.message, err);
      } else {
        this.showPageError(t('common:unknownError'), err);
      }
    }
    return;
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

export default withTranslation(['artifacts', 'common'])(ArtifactList);
