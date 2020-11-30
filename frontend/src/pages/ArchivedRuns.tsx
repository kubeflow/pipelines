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
import { ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { NamespaceContext } from 'src/lib/KubeflowClient';

interface ArchivedRunsState {
  selectedIds: string[];
}

export class ArchivedRuns extends Page<{ namespace?: string }, ArchivedRunsState> {
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
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <RunList
          namespaceMask={this.props.namespace}
          onError={this.showPageError.bind(this)}
          selectedIds={this.state.selectedIds}
          onSelectionChange={this._selectionChanged.bind(this)}
          ref={this._runlistRef}
          storageState={RunStorageState.ARCHIVED}
          {...this.props}
        />
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
}

const EnhancedArchivedRuns = (props: PageProps) => {
  const namespace = React.useContext(NamespaceContext);
  return <ArchivedRuns key={namespace} {...props} namespace={namespace} />;
};

export default EnhancedArchivedRuns;
