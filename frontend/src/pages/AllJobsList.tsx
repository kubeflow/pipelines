/*
 * Copyright 2021 Arrikto Inc.
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
import { Page, PageProps } from './Page';
import { ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import JobList from './JobList';

interface AllJobsListState {
  selectedIds: string[];
  jobListRefreshCount: number;
}

export class AllJobsList extends Page<{ namespace?: string }, AllJobsListState> {
  constructor(props: any) {
    super(props);

    this.state = {
      selectedIds: [],
      jobListRefreshCount: 0,
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    return {
      actions: buttons
        .newRun()
        .refresh(this.refresh.bind(this))
        .getToolbarActionMap(),
      breadcrumbs: [],
      pageTitle: 'Jobs',
    };
  }

  public render(): JSX.Element {
    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <JobList
          onError={this.showPageError.bind(this)}
          selectedIds={this.state.selectedIds}
          onSelectionChange={this._selectionChanged.bind(this)}
          refreshCount={this.state.jobListRefreshCount}
          namespaceMask={this.props.namespace}
          {...this.props}
        />
      </div>
    );
  }

  public async refresh(): Promise<void> {
    // Tell job list to refresh
    this.setState((prevState, _) => ({
      jobListRefreshCount: prevState.jobListRefreshCount + 1,
    }));
  }

  private _selectionChanged(selectedIds: string[]): void {
    const toolbarActions = this.props.toolbarProps.actions;
    this.props.updateToolbar({
      actions: toolbarActions,
      breadcrumbs: this.props.toolbarProps.breadcrumbs,
    });
    this.setState({ selectedIds });
  }
}

const EnhancedAllJobsList = (props: PageProps) => {
  const namespace = React.useContext(NamespaceContext);
  return <AllJobsList key={namespace} {...props} namespace={namespace} />;
};

export default EnhancedAllJobsList;
