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
import { ArtifactLink } from 'src/components/ArtifactLink';
import CustomTable, { Column, CustomRendererProps, Row } from 'src/components/CustomTable';
import { RoutePageFactory } from 'src/components/Router';
import { ToolbarProps } from 'src/components/Toolbar';
import { commonCss, padding } from 'src/Css';
import { Apis, ListRequest } from 'src/lib/Apis';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { errorToMessage, formatDateString } from 'src/lib/Utils';
import { Page, PageProps } from 'src/pages/Page';
import { classes } from 'typestyle';

interface ArtifactListProps {
  // Retained while old bookmarked switcher routes converge on the native list.
  isGroupView?: boolean;
  namespace?: string;
}

interface ArtifactListState {
  rows: Row[];
}

const COLUMNS: Column[] = [
  { customRenderer: artifactNameRenderer, flex: 2, label: 'Name', sortKey: 'name' },
  { flex: 2, label: 'ID', sortKey: 'artifact_id' },
  { flex: 1, label: 'Type', sortKey: 'type' },
  { customRenderer: artifactUriRenderer, flex: 3, label: 'URI', sortKey: 'uri' },
  { flex: 1, label: 'Namespace', sortKey: 'namespace' },
  { flex: 1, label: 'Created at', sortKey: 'created_at' },
];

export class ArtifactList extends Page<ArtifactListProps, ArtifactListState> {
  private tableRef = React.createRef<CustomTable>();

  public state: ArtifactListState = { rows: [] };

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: {},
      breadcrumbs: [],
      pageTitle: 'Artifacts',
    };
  }

  public render(): React.JSX.Element {
    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <CustomTable
          ref={this.tableRef}
          columns={COLUMNS}
          rows={this.state.rows}
          disableSelection={true}
          reload={this.reload}
          initialSortColumn='created_at'
          initialSortOrder='desc'
          emptyMessage='No artifacts found.'
        />
      </div>
    );
  }

  public async refresh(): Promise<void> {
    await this.tableRef.current?.reload();
  }

  private reload = async (request: ListRequest): Promise<string> => {
    try {
      const response = await Apis.artifactServiceApiV2.artifacts(
        this.props.namespace,
        request.pageToken,
        request.pageSize,
        request.sortBy,
        request.filter,
      );
      this.setStateSafe({
        rows: (response.artifacts || []).map((artifact) => ({
          id: artifact.artifact_id || '',
          otherFields: [
            artifact.name || '[unnamed]',
            artifact.artifact_id || '-',
            artifact.type || '-',
            artifact.uri || '',
            artifact.namespace || '-',
            formatDateString(artifact.created_at),
          ],
        })),
      });
      this.clearBanner();
      return response.next_page_token || '';
    } catch (error) {
      const message = await errorToMessage(error);
      this.showPageError(message || 'Error: failed to list artifacts.', error);
      this.setStateSafe({ rows: [] });
      return '';
    }
  };
}

function artifactNameRenderer({ id, value }: CustomRendererProps<string>) {
  return (
    <Link
      onClick={(event) => event.stopPropagation()}
      className={commonCss.link}
      to={RoutePageFactory.artifactDetails(id)}
    >
      {value}
    </Link>
  );
}

function artifactUriRenderer({ value }: CustomRendererProps<string>) {
  return <ArtifactLink artifactUri={value || ''} />;
}

const EnhancedArtifactList = (props: PageProps) => {
  const namespace = React.useContext(NamespaceContext);
  return <ArtifactList key={namespace} {...props} namespace={namespace} />;
};

export default EnhancedArtifactList;
