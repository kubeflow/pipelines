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
import Buttons, { ButtonKeys } from '../lib/Buttons';
import RunList from './RunList';
import { Page, PageProps } from './Page';
import { ApiRunStorageState } from '../apis/run';
import { V2beta1RunStorageState } from 'src/apisv2beta1/run';
import { ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { NamespaceContext } from 'src/lib/KubeflowClient';

interface AllRunsListState {
  selectedIds: string[];
  parentIds: string[];
}

export class AllRunsList extends Page<{ namespace?: string }, AllRunsListState> {
  private _runlistRef = React.createRef<RunList>();

  constructor(props: any) {
    super(props);

    this.state = {
      selectedIds: [],
      parentIds:[],
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    return {
      actions: buttons
        .newRun()
        .compareRuns(() => this.state.selectedIds)
        .cloneRun(() => this.state.selectedIds, false)
        .archiveRunV2(
          () => this.state.selectedIds,
          () => this.state.parentIds,
          false,
          (selectedIds, parentIds) => this._selectionChanged(selectedIds, parentIds),
        )
        .refresh(this.refresh.bind(this))
        .getToolbarActionMap(),
      breadcrumbs: [],
      pageTitle: 'Runs',
    };
  }

  public render(): JSX.Element {
    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <RunList
          isAllRunsList={true}
          onError={this.showPageError.bind(this)}
          selectedIds={this.state.selectedIds}
          parentIds={this.state.parentIds}
          onSelectionChange={this._selectionChanged.bind(this)}
          ref={this._runlistRef}
          storageState={V2beta1RunStorageState.AVAILABLE}
          hideMetricMetadata={true}
          namespaceMask={this.props.namespace}
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

  private _selectionChanged(selectedIds: string[], parentIds: string[]): void {
    const toolbarActions = this.props.toolbarProps.actions;
    toolbarActions[ButtonKeys.COMPARE].disabled =
      selectedIds.length <= 1 || selectedIds.length > 10;
    toolbarActions[ButtonKeys.CLONE_RUN].disabled = selectedIds.length !== 1;
    toolbarActions[ButtonKeys.ARCHIVE].disabled = !selectedIds.length;
    this.props.updateToolbar({
      actions: toolbarActions,
      breadcrumbs: this.props.toolbarProps.breadcrumbs,
    });
    this.setState({ selectedIds, parentIds });
  }
}

const EnhancedAllRunsList = (props: PageProps) => {
  const namespace = React.useContext(NamespaceContext);
  return <AllRunsList key={namespace} {...props} namespace={namespace} />;
};

export default EnhancedAllRunsList;
