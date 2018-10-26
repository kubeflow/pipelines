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
import JobList from './JobList';
import MD2Tabs from '../atoms/MD2Tabs';
import { Page, PageProps } from './Page';
import { RoutePage } from '../components/Router';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';

export enum JobsAndRunsTab {
  JOBS = 0,
  RUNS = 1,
}

export interface JobsAndRunsProps extends PageProps {
  view: JobsAndRunsTab;
}

interface JobAndRunsState {
  selectedTab: JobsAndRunsTab;
}

class JobsAndRuns extends Page<JobsAndRunsProps, JobAndRunsState> {

  public getInitialToolbarState() {
    return { actions: [], breadcrumbs: [] };
  }

  public render() {
    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <MD2Tabs tabs={['Group by job', 'Show all runs']} selectedTab={this.props.view}
          onSwitch={this._tabSwitched.bind(this)} />
        {this.props.view === 0 && (
          <JobList {...this.props} />
        )}

        {this.props.view === 1 && (
          <AllRunsList {...this.props} />
        )}
      </div>
    );
  }

  public async load(): Promise<void> {
    return;
  }

  private _tabSwitched(newTab: JobsAndRunsTab) {
    this.props.history.push(newTab === JobsAndRunsTab.JOBS ? RoutePage.JOBS : RoutePage.RUNS);
  }
}

export default JobsAndRuns;
