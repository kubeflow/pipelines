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
import RunList from './RunList';
import { Page } from './Page';
import { RoutePage } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { URLParser, QUERY_PARAMS } from '../lib/URLParser';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';

interface AllRunsListState {
  selectedIds: string[];
}

class AllRunsList extends Page<{}, AllRunsListState> {

  private _runlistRef = React.createRef<RunList>();

  constructor(props: any) {
    super(props);

    this.state = {
      selectedIds: [],
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: [
        {
          action: this._compareRuns.bind(this),
          disabled: true,
          disabledTitle: 'Select multiple runs to compare',
          id: 'compareBtn',
          title: 'Compare runs',
          tooltip: 'Compare up to 10 selected runs',
        },
        {
          action: this._cloneRun.bind(this),
          disabled: true,
          disabledTitle: 'Select a run to clone',
          id: 'cloneBtn',
          title: 'Clone run',
          tooltip: 'Create a copy from this run\s initial state',
        },
        {
          action: this.refresh.bind(this),
          id: 'refreshBtn',
          title: 'Refresh',
          tooltip: 'Refresh',
        },
      ],
      breadcrumbs: [{ displayName: 'All runs', href: '' }],
    };
  }

  public render(): JSX.Element {
    return <div className={classes(commonCss.page, padding(20, 'lr'))}>
      <RunList onError={this.showPageError.bind(this)} selectedIds={this.state.selectedIds}
        onSelectionChange={this._selectionChanged.bind(this)}
        {...this.props} ref={this._runlistRef} />
    </div>;
  }

  public async refresh(): Promise<void> {
    // Tell run list to refresh
    if (this._runlistRef.current) {
      this.clearBanner();
      await this._runlistRef.current.refresh();
    }
  }

  private _compareRuns(): void {
    const indices = this.state.selectedIds;
    if (indices.length > 1 && indices.length <= 10) {
      const runIds = this.state.selectedIds.join(',');
      const searchString = new URLParser(this.props).build({
        [QUERY_PARAMS.runlist]: runIds,
      });
      this.props.history.push(RoutePage.COMPARE + searchString);
    }
  }

  private _selectionChanged(selectedIds: string[]): void {
    const toolbarActions = [...this.props.toolbarProps.actions];
    // Compare runs button
    toolbarActions[0].disabled = selectedIds.length <= 1 || selectedIds.length > 10;
    // Clone run button
    toolbarActions[1].disabled = selectedIds.length !== 1;
    this.props.updateToolbar({ breadcrumbs: this.props.toolbarProps.breadcrumbs, actions: toolbarActions });
    this.setState({ selectedIds });
  }

  private _cloneRun(): void {
    if (this.state.selectedIds.length === 1) {
      const runId = this.state.selectedIds[0];
      const searchString = new URLParser(this.props).build({
        [QUERY_PARAMS.cloneFromRun]: runId || ''
      });
      this.props.history.push(RoutePage.NEW_RUN + searchString);
    }
  }
}

export default AllRunsList;
