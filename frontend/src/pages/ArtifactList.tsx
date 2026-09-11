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
import { Tooltip } from '@mui/material';
import { Link } from 'react-router-dom';
import { ArtifactLink } from 'src/components/ArtifactLink';
import CustomTable, { Column, CustomRendererProps, Row } from 'src/components/CustomTable';
import { RoutePageFactory } from 'src/components/Router';
import { ToolbarProps } from 'src/components/Toolbar';
import { commonCss, padding } from 'src/Css';
import { Apis, ListRequest } from 'src/lib/Apis';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { errorToMessage, formatDateString } from 'src/lib/Utils';
import { getArtifactTypeName } from 'src/lib/v2/RuntimeArtifactUtils';
import { PageTokenTracker } from 'src/lib/v2/PaginationUtils';
import { Page, PageProps } from 'src/pages/Page';
import { classes } from 'typestyle';

interface ArtifactListProps {
  namespace?: string;
}

interface ArtifactListState {
  rows: Row[];
}

interface ArtifactUriCell {
  namespace?: string;
  uri: string;
}

const COLUMNS: Column[] = [
  { customRenderer: artifactNameRenderer, flex: 2, label: 'Name', sortKey: 'name' },
  { customRenderer: artifactIdRenderer, flex: 1.5, label: 'ID', sortKey: 'artifact_id' },
  { customRenderer: artifactTypeRenderer, flex: 2, label: 'Type', sortKey: 'type' },
  { customRenderer: artifactUriRenderer, flex: 1.8, label: 'URI', sortKey: 'uri' },
  { customRenderer: wrappedTextRenderer, flex: 1.2, label: 'Namespace', sortKey: 'namespace' },
  { customRenderer: artifactDateRenderer, flex: 1.5, label: 'Created at', sortKey: 'created_at' },
];

export class ArtifactList extends Page<ArtifactListProps, ArtifactListState> {
  private tableRef = React.createRef<CustomTable>();
  private activeReloadGeneration = 0;
  private lastSuccessfulRequestKey?: string;
  private pageTokenTracker = new PageTokenTracker();

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
    const reloadGeneration = ++this.activeReloadGeneration;
    const requestKey = this.getRequestKey(request);
    try {
      const response = await Apis.artifactServiceApiV2.artifacts(
        this.props.namespace,
        request.pageToken,
        request.pageSize,
        request.sortBy,
        request.filter,
      );
      const nextPageToken = response.next_page_token || '';
      if (reloadGeneration === this.activeReloadGeneration) {
        let artifactsWithoutId = 0;
        const rows = (response.artifacts || []).flatMap<Row>((artifact) => {
          const artifactId = artifact.artifact_id;
          if (!artifactId) {
            artifactsWithoutId++;
            return [];
          }
          return [
            {
              id: artifactId,
              otherFields: [
                artifact.name || '[unnamed]',
                artifactId,
                getArtifactTypeName(artifact),
                { namespace: artifact.namespace || this.props.namespace, uri: artifact.uri || '' },
                artifact.namespace || '-',
                artifact.created_at,
              ],
            },
          ];
        });
        const repeatedPageToken = this.pageTokenTracker.isRepeated(
          this.getPaginationContextKey(request),
          request.pageToken,
          nextPageToken,
        );
        this.lastSuccessfulRequestKey = requestKey;
        this.setStateSafe({ rows });
        if (repeatedPageToken) {
          this.showPageError(
            `Artifact service returned a repeated page token: ${nextPageToken}`,
            new Error(`Repeated artifact page token: ${nextPageToken}`),
          );
          return '';
        }
        if (artifactsWithoutId) {
          const message = `${artifactsWithoutId} artifact${artifactsWithoutId === 1 ? '' : 's'} could not be displayed because the Artifact service returned no ID. Refresh the page; if the problem persists, contact your administrator.`;
          this.showPageError(message, new Error(message));
        } else {
          this.clearBanner();
        }
      }
      return nextPageToken;
    } catch (error) {
      const message = await errorToMessage(error);
      if (reloadGeneration === this.activeReloadGeneration) {
        this.showPageError(message || 'Error: failed to list artifacts.', error);
        if (this.lastSuccessfulRequestKey !== requestKey) {
          this.setStateSafe({ rows: [] });
        }
      }
      return '';
    }
  };

  private getRequestKey(request: ListRequest): string {
    return JSON.stringify({
      ...this.getPaginationContext(request),
      pageToken: request.pageToken || '',
    });
  }

  private getPaginationContextKey(request: ListRequest): string {
    return JSON.stringify(this.getPaginationContext(request));
  }

  private getPaginationContext(request: ListRequest) {
    return {
      filter: request.filter || '',
      namespace: this.props.namespace || '',
      pageSize: request.pageSize || 0,
      sortBy: request.sortBy || '',
    };
  }
}

function artifactNameRenderer({ id, value }: CustomRendererProps<string>) {
  return (
    <Link
      onClick={(event) => event.stopPropagation()}
      className={commonCss.link}
      style={{ whiteSpace: 'normal', overflowWrap: 'anywhere' }}
      title={value}
      to={RoutePageFactory.artifactDetails(id)}
    >
      {value}
    </Link>
  );
}

function artifactIdRenderer({ id, value = '' }: CustomRendererProps<string>) {
  const shortId = value.length > 16 ? `${value.slice(0, 8)}…${value.slice(-4)}` : value;
  return (
    <Tooltip title={value}>
      <Link
        aria-label={`Artifact ID ${value}`}
        className={commonCss.link}
        onClick={(event) => event.stopPropagation()}
        to={RoutePageFactory.artifactDetails(id)}
      >
        {shortId}
      </Link>
    </Tooltip>
  );
}

function wrappedTextRenderer({ value }: CustomRendererProps<string>) {
  return <span style={{ whiteSpace: 'normal', overflowWrap: 'anywhere' }}>{value}</span>;
}

function artifactTypeRenderer({ value }: CustomRendererProps<string>) {
  return (
    <Tooltip title={value}>
      <span
        style={{
          display: 'block',
          whiteSpace: 'nowrap',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
        }}
      >
        {value}
      </span>
    </Tooltip>
  );
}

function artifactDateRenderer({ value }: CustomRendererProps<Date | string | undefined>) {
  const date = typeof value === 'string' ? new Date(value) : value;
  if (!date || Number.isNaN(date.getTime())) {
    return <span>{formatDateString(value)}</span>;
  }
  return (
    <time dateTime={date.toISOString()} title={formatDateString(date)}>
      <span style={{ display: 'block', whiteSpace: 'nowrap' }}>{date.toLocaleDateString()}</span>
      <span style={{ display: 'block', whiteSpace: 'nowrap' }}>{date.toLocaleTimeString()}</span>
    </time>
  );
}

function artifactUriRenderer({ value }: CustomRendererProps<ArtifactUriCell>) {
  return <ArtifactLink artifactUri={value?.uri || ''} namespace={value?.namespace} />;
}

const EnhancedArtifactList = (props: PageProps) => {
  const namespace = React.useContext(NamespaceContext);
  return <ArtifactList key={namespace} {...props} namespace={namespace} />;
};

export default EnhancedArtifactList;
