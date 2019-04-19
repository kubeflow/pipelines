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
import Buttons from '../lib/Buttons';
import RunList from './RunList';
import { Page } from './Page';
import { RunStorageState } from '../apis/run';
import { ToolbarProps } from '../components/Toolbar';
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
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    return {
      actions: [
        buttons.newRun(),
        buttons.newExperiment(),
        buttons.compareRuns(() => this.state.selectedIds),
        buttons.cloneRun(() => this.state.selectedIds, false),
        buttons.archive(
          () => this.state.selectedIds,
          false,
          selectedIds => this._selectionChanged(selectedIds),
        ),
        buttons.refresh(this.refresh.bind(this)),
      ],
      breadcrumbs: [],
      pageTitle: 'Experiments',
    };
  }

  public render(): JSX.Element {
    return <div className={classes(commonCss.page, padding(20, 'lr'))}>
      <RunList onError={this.showPageError.bind(this)} selectedIds={this.state.selectedIds}
        onSelectionChange={this._selectionChanged.bind(this)} ref={this._runlistRef}
        storageState={RunStorageState.AVAILABLE} {...this.props} />
    </div>;
  }

  public async refresh(): Promise<void> {
    // Tell run list to refresh
    if (this._runlistRef.current) {
      this.clearBanner();
      await this._runlistRef.current.refresh();
    }
  }

  private _selectionChanged(selectedIds: string[]): void {
    const toolbarActions = [...this.props.toolbarProps.actions];
    // TODO: keeping track of indices in the toolbarActions array is not ideal. This should be
    // refactored so that individual buttons can be referenced with something other than indices.
    // Compare runs button
    toolbarActions[2].disabled = selectedIds.length <= 1 || selectedIds.length > 10;
    // Clone run button
    toolbarActions[3].disabled = selectedIds.length !== 1;
    // Archive run button
    toolbarActions[4].disabled = !selectedIds.length;
    this.props.updateToolbar({ breadcrumbs: this.props.toolbarProps.breadcrumbs, actions: toolbarActions });
    this.setState({ selectedIds });
  }
}

export default AllRunsList;
