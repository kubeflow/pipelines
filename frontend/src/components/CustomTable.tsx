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
import { V2beta1Filter, V2beta1PredicateOperation } from 'src/apisv2beta1/filter';

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

  /** Suppresses stale `isBusy` updates when reload() overlaps (e.g. React StrictMode remounts). */
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
      filterStringEncoded: this.props.initialFilterString
        ? this._createAndEncodeFilterV2(this.props.initialFilterString)
        : '',
      isBusy: false,
      maxPageIndex: Number.MAX_SAFE_INTEGER,
      pageSize: this.readPageSize(),
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

  public async reload(loadRequest?: ListRequest): Promise<string> {
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
      // A direct refresh keeps this page but must replace its next-page token.
      // Filter, sort and page navigation requests reconcile paging in their callers.
      if (!loadRequest && this._isMounted && reloadGeneration === this._activeReloadGeneration) {
        const tokenList = this.state.tokenList.slice(0, refreshedPage + 1);
        if (result) tokenList.push(result);
        this.setStateSafe({
          tokenList,
          maxPageIndex: result ? Number.MAX_SAFE_INTEGER : refreshedPage,
        });
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
    const filterStringEncoded = filterString ? this._createAndEncodeFilterV2(filterString) : '';
    this.setStateSafe({ filterStringEncoded });
    const generation = this._activeReloadGeneration + 1;
    const nextToken = await this.reload({ filter: filterStringEncoded, pageToken: '' });
    if (this._isMounted && generation === this._activeReloadGeneration) {
      this._resetToFirstPage(nextToken);
    }
  }

  private _createAndEncodeFilterV2(filterString: string): string {
    const filter: V2beta1Filter = {
      predicates: [
        {
          // TODO: remove this hardcoding once more sophisticated filtering is supported
          key: 'name',
          operation: V2beta1PredicateOperation.IS_SUBSTRING,
          string_value: filterString,
        },
      ],
    };
    return encodeURIComponent(JSON.stringify(filter));
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
        const generation = this._activeReloadGeneration + 1;
        const nextToken = await this.reload({
          pageToken: '',
          orderAscending: sortOrder === 'asc',
          sortBy,
        });
        if (this._isMounted && generation === this._activeReloadGeneration) {
          this._resetToFirstPage(nextToken);
        }
      });
    }
  }

  private async _pageChanged(offset: number): Promise<void> {
    let newCurrentPage = this.state.currentPage + offset;
    let maxPageIndex = this.state.maxPageIndex;
    newCurrentPage = Math.max(0, newCurrentPage);
    newCurrentPage = Math.min(this.state.maxPageIndex, newCurrentPage);

    const reloadGeneration = this._activeReloadGeneration + 1;
    const newPageToken = await this.reload({
      pageToken: this.state.tokenList[newCurrentPage],
    });
    if (!this._isMounted || reloadGeneration !== this._activeReloadGeneration) {
      return;
    }

    if (newPageToken) {
      // If we're using the greatest yet known page, then the pageToken will be new.
      if (newCurrentPage + 1 === this.state.tokenList.length) {
        this.state.tokenList.push(newPageToken);
      }
    } else {
      maxPageIndex = newCurrentPage;
    }

    this.setStateSafe({ currentPage: newCurrentPage, maxPageIndex });
  }

  private async changePageSize(pageSize: number): Promise<void> {
    try {
      LocalStorage.saveTablePageSize(pageSize, this._getPageId());
    } catch {
      // Keep paging usable if browser policy or quota prevents saving the preference.
    }
    const generation = this._activeReloadGeneration + 1;
    const nextToken = await this.reload({ pageSize, pageToken: '' });
    if (this._isMounted && generation === this._activeReloadGeneration) {
      this._resetToFirstPage(nextToken);
    }
  }

  private readPageSize(): number {
    try {
      return LocalStorage.getTablePageSize(this._getPageId());
    } catch {
      return 10;
    }
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
