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
import { Link } from 'react-router';
import { ArtifactLink } from 'src/components/ArtifactLink';
import CustomTable, { Column, CustomRendererProps, Row } from 'src/components/CustomTable';
import { RoutePageFactory } from 'src/components/Router';
import { ToolbarProps } from 'src/lib/PageChromeTypes';
import { Apis, ListRequest } from 'src/lib/Apis';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { errorToMessage, formatDateString } from 'src/lib/Utils';
import { getArtifactTypeName } from 'src/lib/v2/RuntimeArtifactUtils';
import { PageTokenTracker } from 'src/lib/v2/PaginationUtils';
import { Page, PageProps } from 'src/pages/Page';
import { ResourceTable } from 'src/components/modernization/ResourceTable';
import { Button } from 'src/components/ui/button';
import type { V2beta1Filter } from 'src/apisv2beta1/filter';
import './Artifacts.css';

interface ArtifactListProps {
  namespace?: string;
}

interface ArtifactListState {
  rows: Row[];
  selectedType: number;
}

interface ArtifactUriCell {
  namespace?: string;
  uri: string;
}

// Numeric values are the ArtifactType wire values in backend/api/v2beta1/artifact.proto.
// The list service filters its integer Type column; totals are not fetched for these choices.
const TYPE_FILTERS = [
  { label: 'All', values: [] },
  { label: 'Model', values: [2] },
  { label: 'Dataset', values: [3] },
  { label: 'Metrics', values: [6, 7, 8] },
  { label: 'HTML', values: [4] },
];

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

  private nameFilter = '';
  private sortColumn = 'created_at';
  private sortOrder: 'asc' | 'desc' = 'desc';
  public state: ArtifactListState = { rows: [], selectedType: 0 };

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: {},
      breadcrumbs: [],
      pageTitle: 'Artifacts',
    };
  }

  public render(): React.JSX.Element {
    return (
      <div className='kfp-artifact-list'>
        <div className='kfp-artifact-type-filters' role='group' aria-label='Artifact types'>
          {TYPE_FILTERS.map((type, index) => (
            <Button
              key={type.label}
              variant='ghost'
              size='sm'
              aria-pressed={this.state.selectedType === index}
              onClick={() => {
                if (this.state.selectedType === index) return;
                this.activeReloadGeneration++;
                this.setState({ selectedType: index, rows: [] });
              }}
            >
              {type.label}
            </Button>
          ))}
        </div>
        <CustomTable
          key={this.state.selectedType}
          initialFilterString={this.nameFilter}
          setFilterString={(value) => {
            this.nameFilter = value;
          }}
          filterLabel='Filter artifacts by name'
          renderTable={(table) => (
            <ResourceTable
              table={{
                ...table,
                onSort: (key) => {
                  this.sortColumn = key;
                  this.sortOrder =
                    table.sortBy === key && table.sortOrder === 'asc' ? 'desc' : 'asc';
                  table.onSort(key);
                },
              }}
              onOpenRow={(id) =>
                this.props.navigate(RoutePageFactory.artifactDetails(encodeURIComponent(id)))
              }
              label='Artifacts'
              singular='artifact'
              plural='artifacts'
              minWidth={920}
            />
          )}
          ref={this.tableRef}
          columns={COLUMNS}
          rows={this.state.rows}
          disableSelection={true}
          reload={this.reload}
          initialSortColumn={this.sortColumn}
          initialSortOrder={this.sortOrder}
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
    const types = TYPE_FILTERS[this.state.selectedType].values;
    const filter: V2beta1Filter = request.filter
      ? JSON.parse(decodeURIComponent(request.filter))
      : { predicates: [] };
    if (types.length)
      filter.predicates = [
        ...(filter.predicates || []),
        { key: 'type', operation: 'IN', int_values: { values: types } },
      ];
    try {
      const response = await Apis.artifactServiceApiV2.artifacts(
        this.props.namespace,
        request.pageToken,
        request.pageSize,
        request.sortBy,
        types.length ? encodeURIComponent(JSON.stringify(filter)) : request.filter,
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
      type: this.state.selectedType,
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
      className='kfp-artifact-link'
      style={{ whiteSpace: 'normal', overflowWrap: 'anywhere' }}
      title={value}
      to={RoutePageFactory.artifactDetails(encodeURIComponent(id))}
    >
      {value}
    </Link>
  );
}

function artifactIdRenderer({ id, value = '' }: CustomRendererProps<string>) {
  const shortId = value.length > 16 ? `${value.slice(0, 8)}…${value.slice(-4)}` : value;
  return (
    <span title={value}>
      <Link
        aria-label={`Artifact ID ${value}`}
        className='kfp-artifact-link'
        onClick={(event) => event.stopPropagation()}
        to={RoutePageFactory.artifactDetails(encodeURIComponent(id))}
      >
        {shortId}
      </Link>
    </span>
  );
}

function wrappedTextRenderer({ value }: CustomRendererProps<string>) {
  return <span style={{ whiteSpace: 'normal', overflowWrap: 'anywhere' }}>{value}</span>;
}

function artifactTypeRenderer({ value = '' }: CustomRendererProps<string>) {
  const kind = value.includes('Dataset')
    ? 'dataset'
    : value.includes('Model')
      ? 'model'
      : value.includes('Metric')
        ? 'metric'
        : 'other';
  return (
    <span title={value} className='kfp-artifact-type' data-kind={kind}>
      {value}
    </span>
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
  return (
    <span className='kfp-artifact-uri' title={value?.uri}>
      <ArtifactLink artifactUri={value?.uri || ''} namespace={value?.namespace} />
    </span>
  );
}

const EnhancedArtifactList = (props: PageProps) => {
  const namespace = React.useContext(NamespaceContext);
  return <ArtifactList key={namespace} {...props} namespace={namespace} />;
};

export default EnhancedArtifactList;
