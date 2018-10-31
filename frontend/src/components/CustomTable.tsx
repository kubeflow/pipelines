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
import ChevronLeft from '@material-ui/icons/ChevronLeft';
import ChevronRight from '@material-ui/icons/ChevronRight';
import IconButton from '@material-ui/core/IconButton';
import MenuItem from '@material-ui/core/MenuItem';
import Radio from '@material-ui/core/Radio';
import TableSortLabel from '@material-ui/core/TableSortLabel';
import TextField from '@material-ui/core/TextField';
import Tooltip from '@material-ui/core/Tooltip';
import WarningIcon from '@material-ui/icons/WarningRounded';
import { BaseListRequest } from '../lib/Apis';
import Checkbox, { CheckboxProps } from '@material-ui/core/Checkbox';
import { TextFieldProps } from '@material-ui/core/TextField';
import { classes, stylesheet } from 'typestyle';
import { fontsize, dimension, commonCss, color, padding } from '../Css';
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
    fontSize: fontsize.base,
    marginRight: 6,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  columnName: {
    fontSize: fontsize.medium,
    fontWeight: 'bold',
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
    border: '1px solid ' + color.divider,
    borderRadius: 7,
    boxShadow: '0px 2px 7px #aaa',
    margin: '4px 2px',
  },
  expandedRow: {
    borderBottom: '1px solid transparent !important',
    boxSizing: 'border-box',
    height: '34px !important',
  },
  footer: {
    borderBottom: '1px solid ' + color.divider,
    textAlign: 'right',
  },
  header: {
    borderBottom: 'solid 1px ' + color.divider,
    color: color.strong,
    display: 'flex',
    flex: '0 0 50px',
    lineHeight: '50px', // must declare px
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
    marginRight: 10,
  },
});

interface CustomTableProps {
  columns: Column[];
  disablePaging?: boolean;
  disableSelection?: boolean;
  disableSorting?: boolean;
  getExpandComponent?: (index: number) => React.ReactNode;
  emptyMessage?: string;
  orderAscending: boolean;
  pageSize: number;
  reload: (request: BaseListRequest) => Promise<string>;
  rows: Row[];
  selectedIds?: string[];
  sortBy: string;
  toggleExpansion?: (rowId: number) => void;
  updateSelection?: (selectedIds: string[]) => void;
  useRadioButtons?: boolean;
}

interface CustomTableState {
  currentPage: number;
  maxPageIndex: number;
  tokenList: string[];
}

export default class CustomTable extends React.Component<CustomTableProps, CustomTableState> {
  constructor(props: any) {
    super(props);

    this.state = {
      currentPage: 0,
      maxPageIndex: Number.MAX_SAFE_INTEGER,
      tokenList: [''],
    };
  }

  public handleSelectAllClick(event: React.MouseEvent) {
    if (this.props.disableSelection === true) {
      return;
    }
    const selectedIds =
      (event.target as CheckboxProps).checked ? this.props.rows.map((v) => v.id) : [];
    if (this.props.updateSelection) {
      this.props.updateSelection(selectedIds);
    }
  }

  public handleClick(e: React.MouseEvent, id: string) {
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

  public isSelected(id: string) {
    return this.props.selectedIds && this.props.selectedIds.indexOf(id) !== -1;
  }

  public componentDidMount() {
    this._pageChanged(0);
  }

  public render() {
    const { disableSorting, orderAscending, sortBy, pageSize } = this.props;
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
          {this.props.columns.map((col, i) => {
            const isColumnSortable = !!this.props.columns[i].sortKey;
            const isCurrentSortColumn = sortBy === this.props.columns[i].sortKey;
            return (
              <div key={i} style={{ width: widths[i] + '%' }}
                className={css.columnName}>
                {disableSorting === true && <div>{col.label}</div>}
                {!disableSorting && (
                  <Tooltip title={isColumnSortable ? 'Sort' : 'Cannot sort by this column'}
                    enterDelay={300}>
                    <TableSortLabel active={isCurrentSortColumn} className={commonCss.ellipsis}
                      direction={isColumnSortable ? (orderAscending ? 'asc' : 'desc') : undefined}
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
                <div className={padding(20, 'lr')}>
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

  private async _requestSort(sortBy?: string) {
    if (sortBy) {
      const orderAscending = this.props.sortBy === sortBy ? !this.props.orderAscending : true;
      this._resetToFirstPage(await this.props.reload({ pageToken: '', orderAscending, sortBy }));
    }
  }

  private async _pageChanged(offset: number) {
    let newCurrentPage = this.state.currentPage + offset;
    let maxPageIndex = this.state.maxPageIndex;
    newCurrentPage = Math.max(0, newCurrentPage);
    newCurrentPage = Math.min(this.state.maxPageIndex, newCurrentPage);

    const newPageToken = await this.props.reload({
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

    // TODO: saw this warning:
    // Warning: Can't call setState (or forceUpdate) on an unmounted component.
    // This is a no-op, but it indicates a memory leak in your application.
    // To fix, cancel all subscriptions and asynchronous tasks in the componentWillUnmount method.
    this.setState({ currentPage: newCurrentPage, maxPageIndex });
  }

  private async _requestRowsPerPage(event: React.ChangeEvent) {
    const pageSize = (event.target as TextFieldProps).value as number;

    this._resetToFirstPage(await this.props.reload({ pageSize, pageToken: '' }));
  }

  private _resetToFirstPage(newPageToken?: string) {
    let maxPageIndex = Number.MAX_SAFE_INTEGER;
    const newTokenList = [''];

    if (newPageToken) {
      newTokenList.push(newPageToken);
    } else {
      maxPageIndex = 0;
    }

    // Reset state, since this invalidates the token list and page counter calculations
    this.setState({
      currentPage: 0,
      maxPageIndex,
      tokenList: newTokenList,
    });
  }

  private _expandButtonToggled(e: React.MouseEvent, rowIndex: number) {
    e.stopPropagation();
    if (this.props.toggleExpansion) {
      this.props.toggleExpansion(rowIndex);
    }
  }
}
