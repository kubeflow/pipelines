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
import AllRunsList from './AllRunsList';
import ExperimentList from './ExperimentList';
import MD2Tabs from '../atoms/MD2Tabs';
import { Page, PageProps } from './Page';
import { RoutePage } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';

export enum ExperimentsAndRunsTab {
  EXPERIMENTS = 0,
  RUNS = 1,
}

export interface ExperimentAndRunsProps extends PageProps {
  view: ExperimentsAndRunsTab;
}

interface ExperimentAndRunsState {
  selectedTab: ExperimentsAndRunsTab;
}

class ExperimentsAndRuns extends Page<ExperimentAndRunsProps, ExperimentAndRunsState> {

  public getInitialToolbarState(): ToolbarProps {
    return { actions: [], breadcrumbs: [] };
  }

  public render(): JSX.Element {
    return (
      <div className={classes(commonCss.page, padding(20, 't'))}>
        <MD2Tabs tabs={['All experiments', 'All runs']} selectedTab={this.props.view}
          onSwitch={this._tabSwitched.bind(this)} />
        {this.props.view === 0 && (
          <ExperimentList {...this.props} />
        )}

        {this.props.view === 1 && (
          <AllRunsList {...this.props} />
        )}
      </div>
    );
  }

  public async refresh(): Promise<void> {
    return;
  }

  private _tabSwitched(newTab: ExperimentsAndRunsTab): void {
    this.props.history.push(
      newTab === ExperimentsAndRunsTab.EXPERIMENTS ? RoutePage.EXPERIMENTS : RoutePage.RUNS);
  }
}

export default ExperimentsAndRuns;
