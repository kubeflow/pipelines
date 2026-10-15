/*
 * Copyright 2018 The Kubeflow Authors
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
import { ListRequest } from '../lib/Apis';
import { LocalStorage } from '../lib/LocalStorage';
import { debounce } from 'lodash';
import type { V2beta1Predicate } from 'src/apisv2beta1/filter';
import { encodeNameFilter } from 'src/lib/ApiFilter';

export enum ExpandState {
  COLLAPSED,
  EXPANDED,
  NONE,
}

export interface Column {
  flex?: number;
  label: string;
  sortKey?: string;
  customRenderer?: React.FC<CustomRendererProps<any | undefined>>;
}

export interface CustomRendererProps<T> {
  value?: T;
  id: string;
}

export interface Row {
  expandState?: ExpandState;
  error?: string;
  id: string;
  otherFields: any[];
}

export interface CustomTableRenderModel {
  columns: Column[];
  rows: Row[];
  selectedIds: string[];
  filter: string;
  filterLabel: string;
  filterActions?: React.ReactNode;
  onFilterChange: (value: string) => void;
  sortBy: string;
  sortOrder: 'asc' | 'desc';
  onSort: (key: string) => void;
  onSelect: (id: string) => void;
  onSelectAll: (checked: boolean) => void;
  isBusy: boolean;
  emptyMessage?: string;
  errorMessage?: string;
  pageSize: number;
  onPageSizeChange: (size: number) => void;
  canPrevious: boolean;
  canNext: boolean;
  onPrevious: () => void;
  onNext: () => void;
  disablePaging?: boolean;
  disableSelection?: boolean;
  disableSorting?: boolean;
  noFilterBox?: boolean;
  useRadioButtons?: boolean;
  disableAdditionalSelection?: boolean;
  getExpandedContent?: (index: number) => React.ReactNode;
  onToggleExpansion?: (index: number) => void;
}

interface CustomTableProps {
  columns: Column[];
  renderTable: (model: CustomTableRenderModel) => React.ReactNode;
  disablePaging?: boolean;
  disableSelection?: boolean;
  disableSorting?: boolean;
  emptyMessage?: string;
  errorMessage?: string;
  filterLabel?: string;
  /** Changes reload the first page while preserving the current name filter and sort. */
  filterPredicates?: V2beta1Predicate[];
  filterActions?: React.ReactNode;
  getExpandComponent?: (index: number) => React.ReactNode;
  initialSortColumn?: string;
  initialSortOrder?: 'asc' | 'desc';
  initialFilterString?: string;
  setFilterString?: (filterString: string) => void;
  noFilterBox?: boolean;
  reload: (request: ListRequest) => Promise<string>;
  rows: Row[];
  selectedIds?: string[];
  toggleExpansion?: (rowId: number) => void;
  updateSelection?: (selectedIds: string[]) => void;
  useRadioButtons?: boolean;
  disableAdditionalSelection?: boolean;
}

interface CustomTableState {
  currentPage: number;
  filterString: string;
  filterStringEncoded: string;
  isBusy: boolean;
  maxPageIndex: number;
  sortOrder: 'asc' | 'desc';
  pageSize: number;
  sortBy: string;
  tokenList: string[];
}

export default class CustomTable extends React.Component<CustomTableProps, CustomTableState> {
  private _isMounted = true;

  /** Suppresses stale paging and busy updates when requests overlap or the table remounts. */
  private _activeReloadGeneration = 0;

  private _debouncedFilterRequest = debounce(
    (filterString: string) => this._requestFilter(filterString),
    300,
  );

  constructor(props: CustomTableProps) {
    super(props);

    this.state = {
      currentPage: 0,
      filterString: this.props.initialFilterString || '',
      filterStringEncoded: encodeNameFilter(
        props.initialFilterString || '',
        props.filterPredicates,
      ),
      isBusy: false,
      maxPageIndex: Number.MAX_SAFE_INTEGER,
      pageSize: LocalStorage.getTablePageSize(this._getPageId()),
      sortBy:
        props.initialSortColumn || (props.columns.length ? props.columns[0].sortKey || '' : ''),
      sortOrder: props.initialSortOrder || 'desc',
      tokenList: [''],
    };
  }

  private selectAll(checked: boolean): void {
    if (this.props.disableSelection) return;
    const selectedIds = checked ? this.props.rows.map((v) => v.id) : [];
    if (this.props.updateSelection) {
      this.props.updateSelection(selectedIds);
    }
  }

  private selectRow(id: string): void {
    if (this.props.disableSelection === true) {
      return;
    }

    let newSelected: string[];
    if (this.props.useRadioButtons) {
      newSelected = [id];
    } else {
      const selectedIds = this.props.selectedIds || [];
      const selectedIndex = selectedIds.indexOf(id);
      newSelected =
        selectedIndex === -1
          ? selectedIds.concat(id)
          : selectedIds.slice(0, selectedIndex).concat(selectedIds.slice(selectedIndex + 1));
    }

    if (this.props.updateSelection) {
      this.props.updateSelection(newSelected);
    }
  }

  public componentDidMount(): void {
    this._isMounted = true;
    this._pageChanged(0);
  }

  public componentDidUpdate(previousProps: CustomTableProps): void {
    // Predicates synchronize the table with its external API scope. Compare their
    // encoded values because callers may construct equivalent arrays on each render.
    if (
      encodeNameFilter('', previousProps.filterPredicates) !==
      encodeNameFilter('', this.props.filterPredicates)
    ) {
      this._debouncedFilterRequest.cancel();
      void this._requestFilter(this.state.filterString);
    }
  }

  public componentWillUnmount(): void {
    this._isMounted = false;
    this._activeReloadGeneration++;
    this._debouncedFilterRequest.cancel();
  }

  public render(): React.JSX.Element {
    const { filterString, pageSize, sortBy, sortOrder } = this.state;
    return (
      <>
        {this.props.renderTable({
          columns: this.props.columns,
          rows: this.props.rows,
          selectedIds: this.props.selectedIds || [],
          filter: filterString,
          filterActions: this.props.filterActions,
          filterLabel: this.props.filterLabel || 'Filter',
          onFilterChange: this.changeFilter,
          sortBy,
          sortOrder,
          onSort: (key) => this._requestSort(key),
          onSelect: (id) => this.selectRow(id),
          onSelectAll: (checked) => this.selectAll(checked),
          isBusy: this.state.isBusy,
          emptyMessage: this.props.emptyMessage,
          errorMessage: this.props.errorMessage,
          pageSize,
          onPageSizeChange: (size) => this.changePageSize(size),
          canPrevious: this.state.currentPage > 0,
          canNext: this.state.currentPage < this.state.maxPageIndex,
          onPrevious: () => this._pageChanged(-1),
          onNext: () => this._pageChanged(1),
          disablePaging: this.props.disablePaging,
          disableSelection: this.props.disableSelection,
          disableSorting: this.props.disableSorting,
          noFilterBox: this.props.noFilterBox,
          useRadioButtons: this.props.useRadioButtons,
          disableAdditionalSelection: this.props.disableAdditionalSelection,
          getExpandedContent: this.props.getExpandComponent,
          onToggleExpansion: this.props.toggleExpansion,
        })}
      </>
    );
  }

  public reload(loadRequest?: ListRequest): Promise<string> {
    return this._reload(loadRequest, loadRequest ? 'unchanged' : 'refresh');
  }

  private async _reload(
    loadRequest: ListRequest | undefined,
    paging: 'unchanged' | 'refresh' | 'reset' | number,
  ): Promise<string> {
    // Override the current state with incoming request
    const request: ListRequest = Object.assign(
      {
        filter: this.state.filterStringEncoded,
        orderAscending: this.state.sortOrder === 'asc',
        pageSize: this.state.pageSize,
        pageToken: this.state.tokenList[this.state.currentPage],
        sortBy: this.state.sortBy,
      },
      loadRequest,
    );

    const reloadGeneration = ++this._activeReloadGeneration;
    const refreshedPage = this.state.currentPage;
    let result: string;
    try {
      this.setStateSafe({
        filterStringEncoded: request.filter,
        isBusy: true,
        pageSize: request.pageSize,
        sortBy: request.sortBy,
        sortOrder: request.orderAscending ? 'asc' : 'desc',
      });

      if (request.sortBy && !request.orderAscending) {
        request.sortBy += ' desc';
      }

      result = await this.props.reload(request);
      if (this._isMounted && reloadGeneration === this._activeReloadGeneration) {
        if (paging === 'reset') {
          this._resetToFirstPage(result);
        } else if (paging === 'refresh') {
          const tokenList = this.state.tokenList.slice(0, refreshedPage + 1);
          if (result) tokenList.push(result);
          this.setStateSafe({
            tokenList,
            maxPageIndex: result ? Number.MAX_SAFE_INTEGER : refreshedPage,
          });
        } else if (typeof paging === 'number') {
          const tokenList = [...this.state.tokenList];
          if (result && paging + 1 === tokenList.length) tokenList.push(result);
          this.setStateSafe({
            currentPage: paging,
            maxPageIndex: result ? this.state.maxPageIndex : paging,
            tokenList,
          });
        }
      }
    } finally {
      if (this._isMounted && reloadGeneration === this._activeReloadGeneration) {
        this.setStateSafe({ isBusy: false });
      }
    }
    return result;
  }

  private changeFilter = (value: string) => {
    if (this.props.setFilterString) {
      this.props.setFilterString(value || '');
    }
    // Set state here so that the UI will be updated even if the actual filter request is debounced
    this.setStateSafe(
      { filterString: value } as any,
      async () => await this._debouncedFilterRequest(value as string),
    );
  };

  // Exposed for testing
  protected async _requestFilter(filterString?: string): Promise<void> {
    const filterStringEncoded = encodeNameFilter(filterString || '', this.props.filterPredicates);
    this.setStateSafe({ filterStringEncoded });
    await this._reload({ filter: filterStringEncoded, pageToken: '' }, 'reset');
  }

  private setStateSafe(newState: Partial<CustomTableState>, cb?: () => void): void {
    if (this._isMounted) {
      this.setState(newState as any, cb);
    }
  }

  private _requestSort(sortBy?: string): void {
    if (sortBy) {
      // Set the sort column to the provided column if it's different, and
      // invert the sort order it if it's the same column
      const sortOrder =
        this.state.sortBy === sortBy ? (this.state.sortOrder === 'asc' ? 'desc' : 'asc') : 'asc';
      this.setStateSafe({ sortOrder, sortBy }, async () => {
        await this._reload({ pageToken: '', orderAscending: sortOrder === 'asc', sortBy }, 'reset');
      });
    }
  }

  private async _pageChanged(offset: number): Promise<void> {
    let newCurrentPage = this.state.currentPage + offset;
    newCurrentPage = Math.max(0, newCurrentPage);
    newCurrentPage = Math.min(this.state.maxPageIndex, newCurrentPage);
    await this._reload({ pageToken: this.state.tokenList[newCurrentPage] }, newCurrentPage);
  }

  private async changePageSize(pageSize: number): Promise<void> {
    LocalStorage.saveTablePageSize(pageSize, this._getPageId());
    await this._reload({ pageSize, pageToken: '' }, 'reset');
  }

  private _getPageId(): string | undefined {
    const pathSegments = window.location.hash
      .split('?')[0]
      .replace(/^#\//, '')
      .split('/')
      .filter(Boolean);
    if (pathSegments.length === 0) {
      return undefined;
    }
    const [root, second] = pathSegments;
    if (root === 'shared' && second === 'pipelines') {
      return 'shared/pipelines';
    }
    if (root === 'archive' && second) {
      return `${root}/${second}`;
    }
    if (second && ['details', 'new', 'lineage'].includes(second)) {
      return `${root}/${second}`;
    }
    return root;
  }

  private _resetToFirstPage(newPageToken?: string): void {
    let maxPageIndex = Number.MAX_SAFE_INTEGER;
    const newTokenList = [''];

    if (newPageToken) {
      newTokenList.push(newPageToken);
    } else {
      maxPageIndex = 0;
    }

    // Reset state, since this invalidates the token list and page counter calculations
    this.setStateSafe({
      currentPage: 0,
      maxPageIndex,
      tokenList: newTokenList,
    });
  }
}
