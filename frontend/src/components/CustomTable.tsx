/*
 * Copyright 2018 Google LLC
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
import ArrowRight from '@material-ui/icons/ArrowRight';
import Checkbox, { CheckboxProps } from '@material-ui/core/Checkbox';
import ChevronLeft from '@material-ui/icons/ChevronLeft';
import ChevronRight from '@material-ui/icons/ChevronRight';
import IconButton from '@material-ui/core/IconButton';
import MenuItem from '@material-ui/core/MenuItem';
import Radio from '@material-ui/core/Radio';
import Separator from '../atoms/Separator';
import TableSortLabel from '@material-ui/core/TableSortLabel';
import TextField from '@material-ui/core/TextField';
import Tooltip from '@material-ui/core/Tooltip';
import WarningIcon from '@material-ui/icons/WarningRounded';
import { ListRequest } from '../lib/Apis';
import { TextFieldProps } from '@material-ui/core/TextField';
import { classes, stylesheet } from 'typestyle';
import { fonts, fontsize, dimension, commonCss, color, padding } from '../Css';
import { logger } from '../lib/Utils';

export enum ExpandState {
  COLLAPSED,
  EXPANDED,
  NONE,
}

export interface Column {
  flex?: number;
  label: string;
  sortKey?: string;
  customRenderer?: (value: any, id: string) => React.StatelessComponent;
}

export interface Row {
  expandState?: ExpandState;
  error?: string;
  id: string;
  otherFields: any[];
}

const rowHeight = 40;

export const css = stylesheet({
  cell: {
    $nest: {
      '&:not(:nth-child(2))': {
        color: color.inactive,
      },
    },
    alignSelf: 'center',
    borderBottom: 'initial',
    color: color.foreground,
    fontFamily: fonts.secondary,
    fontSize: fontsize.base,
    letterSpacing: 0.25,
    marginRight: 20,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  columnName: {
    color: '#1F1F1F',
    fontSize: fontsize.small,
    fontWeight: 'bold',
    letterSpacing: 0.25,
    marginRight: 20,
  },
  emptyMessage: {
    padding: 20,
    textAlign: 'center',
  },
  expandButton: {
    marginRight: 10,
    padding: 3,
    transition: 'transform 0.3s',
  },
  expandButtonExpanded: {
    transform: 'rotate(90deg)',
  },
  expandedContainer: {
    borderRadius: 10,
    boxShadow: '0 1px 2px 0 rgba(60,64,67,0.30), 0 1px 3px 1px rgba(60,64,67,0.15)',
    margin: '16px 2px',
  },
  expandedRow: {
    borderBottom: '1px solid transparent !important',
    boxSizing: 'border-box',
    height: '40px !important',
  },
  footer: {
    borderBottom: '1px solid ' + color.divider,
    fontFamily: fonts.secondary,
    height: 40,
    textAlign: 'right',
  },
  header: {
    borderBottom: 'solid 1px ' + color.divider,
    color: color.strong,
    display: 'flex',
    flex: '0 0 40px',
    lineHeight: '40px', // must declare px
    marginTop: 20,
  },
  icon: {
    color: color.alert,
    height: 18,
    paddingRight: 4,
    verticalAlign: 'sub',
    width: 18,
  },
  row: {
    $nest: {
      '&:hover': {
        backgroundColor: '#f3f3f3',
      },
    },
    borderBottom: '1px solid #ddd',
    display: 'flex',
    flexShrink: 0,
    height: rowHeight,
    outline: 'none',
  },
  rowsPerPage: {
    color: color.strong,
    height: dimension.xsmall,
    minWidth: dimension.base,
  },
  selected: {
    backgroundColor: color.activeBg,
  },
  selectionToggle: {
    marginRight: 12,
  },
});

interface CustomTableProps {
  columns: Column[];
  disablePaging?: boolean;
  disableSelection?: boolean;
  disableSorting?: boolean;
  emptyMessage?: string;
  getExpandComponent?: (index: number) => React.ReactNode;
  initialSortColumn?: string;
  initialSortOrder?: 'asc' | 'desc';
  reload: (request: ListRequest) => Promise<string>;
  rows: Row[];
  selectedIds?: string[];
  toggleExpansion?: (rowId: number) => void;
  updateSelection?: (selectedIds: string[]) => void;
  useRadioButtons?: boolean;
}

interface CustomTableState {
  currentPage: number;
  maxPageIndex: number;
  sortOrder: 'asc' | 'desc';
  pageSize: number;
  sortBy: string;
  tokenList: string[];
}

export default class CustomTable extends React.Component<CustomTableProps, CustomTableState> {
  private _isMounted = true;

  constructor(props: CustomTableProps) {
    super(props);

    this.state = {
      currentPage: 0,
      maxPageIndex: Number.MAX_SAFE_INTEGER,
      pageSize: 10,
      sortBy: props.initialSortColumn ||
        (props.columns.length ? props.columns[0].sortKey || '' : ''),
      sortOrder: props.initialSortOrder || 'desc',
      tokenList: [''],
    };
  }

  public handleSelectAllClick(event: React.MouseEvent): void {
    if (this.props.disableSelection === true) {
      return;
    }
    const selectedIds =
      (event.target as CheckboxProps).checked ? this.props.rows.map((v) => v.id) : [];
    if (this.props.updateSelection) {
      this.props.updateSelection(selectedIds);
    }
  }

  public handleClick(e: React.MouseEvent, id: string): void {
    if (this.props.disableSelection === true) {
      return;
    }

    let newSelected = [];
    if (this.props.useRadioButtons) {
      newSelected = [id];
    } else {
      const selectedIds = this.props.selectedIds || [];
      const selectedIndex = selectedIds.indexOf(id);
      newSelected = selectedIndex === -1 ?
        selectedIds.concat(id) :
        selectedIds.slice(0, selectedIndex).concat(selectedIds.slice(selectedIndex + 1));
    }

    if (this.props.updateSelection) {
      this.props.updateSelection(newSelected);
    }

    e.stopPropagation();
  }

  public isSelected(id: string): boolean {
    return !!this.props.selectedIds && this.props.selectedIds.indexOf(id) !== -1;
  }

  public componentDidMount(): void {
    this._pageChanged(0);
  }

  public componentWillUnmount(): void {
    this._isMounted = false;
  }

  public render(): JSX.Element {
    const { pageSize, sortBy, sortOrder } = this.state;
    const numSelected = (this.props.selectedIds || []).length;
    const totalFlex = this.props.columns.reduce((total, c) => total += (c.flex || 1), 0);
    const widths = this.props.columns.map(c => (c.flex || 1) / totalFlex * 100);

    return (
      <div className={commonCss.pageOverflowHidden}>

        {/* Header */}
        <div className={classes(
          css.header, (this.props.disableSelection || this.props.useRadioButtons) && padding(20, 'l'))}>
          {(this.props.disableSelection !== true && this.props.useRadioButtons !== true) && (
            <div className={classes(css.columnName, css.cell, css.selectionToggle)}>
              <Checkbox indeterminate={!!numSelected && numSelected < this.props.rows.length}
                color='primary' checked={!!numSelected && numSelected === this.props.rows.length}
                onChange={this.handleSelectAllClick.bind(this)} />
            </div>
          )}
          {/* Shift cells to account for expand button */}
          {!!this.props.getExpandComponent && (
            <Separator orientation='horizontal' units={40} />
          )}
          {this.props.columns.map((col, i) => {
            const isColumnSortable = !!this.props.columns[i].sortKey;
            const isCurrentSortColumn = sortBy === this.props.columns[i].sortKey;
            return (
              <div key={i} style={{ width: widths[i] + '%' }}
                className={css.columnName}>
                {this.props.disableSorting === true && <div>{col.label}</div>}
                {!this.props.disableSorting && (
                  <Tooltip title={isColumnSortable ? 'Sort' : 'Cannot sort by this column'}
                    enterDelay={300}>
                    <TableSortLabel active={isCurrentSortColumn} className={commonCss.ellipsis}
                      direction={isColumnSortable ? sortOrder : undefined}
                      onClick={() => this._requestSort(this.props.columns[i].sortKey)}>
                      {col.label}
                    </TableSortLabel>
                  </Tooltip>
                )}
              </div>
            );
          })}
        </div>

        {/* Body */}
        <div className={commonCss.scrollContainer}>
          {/* Empty experience */}
          {this.props.rows.length === 0 && !!this.props.emptyMessage && (
            <div className={css.emptyMessage}>{this.props.emptyMessage}</div>
          )}
          {this.props.rows.map((row, i) => {
            if (row.otherFields.length !== this.props.columns.length) {
              logger.error('Rows must have the same number of cells defined in columns');
              return null;
            }
            return (<div className={classes(row.expandState === ExpandState.EXPANDED && css.expandedContainer)} key={i}>
              <div role='checkbox' tabIndex={-1} className={
                classes(
                  'tableRow',
                  css.row,
                  this.props.disableSelection === true && padding(20, 'l'),
                  this.isSelected(row.id) && css.selected,
                  row.expandState === ExpandState.EXPANDED && css.expandedRow
                )}
                onClick={e => this.handleClick(e, row.id)}>
                {(this.props.disableSelection !== true || !!this.props.getExpandComponent) && (
                  <div className={classes(css.cell, css.selectionToggle)}>
                    {/* If using checkboxes */}
                    {(this.props.disableSelection !== true && this.props.useRadioButtons !== true) && (
                      <Checkbox color='primary' checked={this.isSelected(row.id)} />)}
                    {/* If using radio buttons */}
                    {(this.props.disableSelection !== true && this.props.useRadioButtons) && (
                      <Radio color='primary' checked={this.isSelected(row.id)} />)}
                    {!!this.props.getExpandComponent && (
                      <IconButton className={classes(css.expandButton,
                        row.expandState === ExpandState.EXPANDED && css.expandButtonExpanded)}
                        onClick={(e) => this._expandButtonToggled(e, i)}>
                        <ArrowRight />
                      </IconButton>
                    )}
                  </div>
                )}
                {row.otherFields.map((cell, c) => (
                  <div key={c} style={{ width: widths[c] + '%' }} className={css.cell}>
                    {c === 0 && row.error && (
                      <Tooltip title={row.error}><WarningIcon className={css.icon} /></Tooltip>
                    )}
                    {this.props.columns[c].customRenderer ?
                      this.props.columns[c].customRenderer!(cell, row.id) : cell}
                  </div>
                ))}
              </div>
              {row.expandState === ExpandState.EXPANDED && this.props.getExpandComponent && (
                <div className={padding(20, 'lrb')}>
                  {this.props.getExpandComponent(i)}
                </div>
              )}
            </div>);
          })}
        </div>

        {/* Footer */}
        {!this.props.disablePaging && (
          <div className={css.footer}>
            <span className={padding(10, 'r')}>Rows per page:</span>
            <TextField select={true} variant='standard' className={css.rowsPerPage}
              InputProps={{ disableUnderline: true }} onChange={this._requestRowsPerPage.bind(this)}
              value={pageSize}>
              {[10, 20, 50, 100].map((size, i) => (
                <MenuItem key={i} value={size}>{size}</MenuItem>
              ))}
            </TextField>

            <IconButton onClick={() => this._pageChanged(-1)} disabled={!this.state.currentPage}>
              <ChevronLeft />
            </IconButton>
            <IconButton onClick={() => this._pageChanged(1)}
              disabled={this.state.currentPage >= this.state.maxPageIndex}>
              <ChevronRight />
            </IconButton>
          </div>
        )}
      </div>
    );
  }

  public reload(loadRequest?: ListRequest): Promise<string> {
    // Override the current state with incoming request
    const request: ListRequest = Object.assign({
      orderAscending: this.state.sortOrder === 'asc',
      pageSize: this.state.pageSize,
      pageToken: this.state.tokenList[this.state.currentPage],
      sortBy: this.state.sortBy,
    }, loadRequest);

    this.setStateSafe({
      pageSize: request.pageSize!,
      sortBy: request.sortBy!,
      sortOrder: request.orderAscending ? 'asc' : 'desc',
    });

    if (request.sortBy && !request.orderAscending) {
      request.sortBy += ' desc';
    }

    return this.props.reload(request);
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
      const sortOrder = this.state.sortBy === sortBy ?
        (this.state.sortOrder === 'asc' ? 'desc' : 'asc') : 'asc';
      this.setStateSafe({ sortOrder, sortBy }, async () => {
        this._resetToFirstPage(
          await this.reload({ pageToken: '', orderAscending: sortOrder === 'asc', sortBy }));
      });
    }
  }

  private async _pageChanged(offset: number): Promise<void> {
    let newCurrentPage = this.state.currentPage + offset;
    let maxPageIndex = this.state.maxPageIndex;
    newCurrentPage = Math.max(0, newCurrentPage);
    newCurrentPage = Math.min(this.state.maxPageIndex, newCurrentPage);

    const newPageToken = await this.reload({
      pageToken: this.state.tokenList[newCurrentPage],
    });

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

  private async _requestRowsPerPage(event: React.ChangeEvent): Promise<void> {
    const pageSize = (event.target as TextFieldProps).value as number;

    this._resetToFirstPage(await this.reload({ pageSize, pageToken: '' }));
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

  private _expandButtonToggled(e: React.MouseEvent, rowIndex: number): void {
    e.stopPropagation();
    if (this.props.toggleExpansion) {
      this.props.toggleExpansion(rowIndex);
    }
  }
}
