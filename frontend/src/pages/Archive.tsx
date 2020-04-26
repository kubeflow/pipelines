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
import Buttons, { ButtonKeys } from '../lib/Buttons';
import RunList from './RunList';
import { Page, PageProps } from './Page';
import { RunStorageState } from '../apis/run';
import { ExperimentStorageState } from '../apis/experiment';
import { ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import ExperimentListComponent from 'src/components/ExperimentListComponent';
import FormControlLabel from '@material-ui/core/FormControlLabel';
import Radio from '@material-ui/core/Radio';

export enum ArchiveTab {
  RUNS = 0,
  EXPERIMENTS = 1,
}

export interface ArchiveProps extends PageProps {
  namespace?: string;
}

interface ArchiveState {
  selectedTab: ArchiveTab;
  selectedIds: string[];
}

export class Archive extends Page<ArchiveProps, ArchiveState> {
  private _runlistRef = React.createRef<RunList>();
  private _experimentlistRef = React.createRef<ExperimentListComponent>();

  constructor(props: any) {
    super(props);

    this.state = {
      selectedTab: ArchiveTab.RUNS,
      selectedIds: [],
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    return {
      actions: buttons
        .restore('run', () => this.state.selectedIds, false, this._selectionChanged.bind(this))
        .refresh(this.refresh.bind(this))
        .delete(
          () => this.state.selectedIds,
          'run',
          this._selectionChanged.bind(this),
          false /* useCurrentResource */,
        )
        .getToolbarActionMap(),
      breadcrumbs: [],
      pageTitle: 'Archive',
    };
  }

  public render(): JSX.Element {
    return (
      <div>
      <div className={classes(commonCss.flex, padding(10, 'b'))}>
        <FormControlLabel
          id='viewArchivedRuns'
          label='Runs'
          checked={this.state.selectedTab === ArchiveTab.RUNS}
          control={<Radio color='primary' />}
          onChange={() =>
            this.setState({
              selectedTab: ArchiveTab.RUNS,
            })
          }
        />
        <FormControlLabel
          id='viewArchivedExperiments'
          label='Experiments'
          checked={this.state.selectedTab === ArchiveTab.EXPERIMENTS}
          control={<Radio color='primary' />}
          onChange={() =>
            this.setState({
              selectedTab: ArchiveTab.EXPERIMENTS,
            })
          }
        />
      </div>

      {this.state.selectedTab === ArchiveTab.RUNS &&
      <div className={classes(commonCss.page, padding(20, 't'))}>
        <RunList
          namespaceMask={this.props.namespace}
          onError={this.showPageError.bind(this)}
          selectedIds={this.state.selectedIds}
          onSelectionChange={this._selectionChanged.bind(this)}
          ref={this._runlistRef}
          storageState={RunStorageState.ARCHIVED}
          {...this.props}
        />
      </div>}
      {this.state.selectedTab === ArchiveTab.EXPERIMENTS &&
      <div>
        <ExperimentListComponent
          ref={this._experimentlistRef}
          onError={this.showPageError.bind(this)}
          onSelectionChange={this._experimentSelectionChanged.bind(this)}
          storageState={ExperimentStorageState.ARCHIVED}
          {...this.props}
        />
      </div>}
      </div>
    );
  }

  public async refresh(): Promise<void> {
    // Tell run list to refresh
    if (this._runlistRef.current) {
      this.clearBanner();
      await this._runlistRef.current.refresh();
    }
  }

  private _selectionChanged(selectedIds: string[]): void {
    const toolbarActions = this.props.toolbarProps.actions;
    toolbarActions[ButtonKeys.RESTORE].disabled = !selectedIds.length;
    toolbarActions[ButtonKeys.DELETE_RUN].disabled = !selectedIds.length;
    this.props.updateToolbar({
      actions: toolbarActions,
      breadcrumbs: this.props.toolbarProps.breadcrumbs,
    });
    this.setState({ selectedIds });
  }

  private _experimentSelectionChanged(selectedIds: string[]): void {
    const toolbarActions = this.props.toolbarProps.actions;
    toolbarActions[ButtonKeys.RESTORE].disabled = !selectedIds.length;
    toolbarActions[ButtonKeys.DELETE_RUN].disabled = !selectedIds.length;
    this.props.updateToolbar({
      actions: toolbarActions,
      breadcrumbs: this.props.toolbarProps.breadcrumbs,
    });
    this.setState({ selectedIds });
  }
}

const EnhancedArchive = (props: PageProps) => {
  const namespace = React.useContext(NamespaceContext);
  return <Archive key={namespace} {...props} namespace={namespace} />;
};

export default EnhancedArchive;
