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

import type * as React from 'react';
import ExperimentsPage from './ExperimentList';
import ArchivedExperimentsPage from './ArchivedExperiments';
import { InspectionTabs } from '../components/modernization/InspectionTabs';
import '../components/modernization/ExperimentWorkflows.css';
import { Page, PageProps } from './Page';
import { RoutePage } from '../components/Router';
import { ToolbarProps } from 'src/lib/PageChromeTypes';

export enum AllExperimentsAndArchiveTab {
  EXPERIMENTS = 0,
  ARCHIVE = 1,
}

export interface AllExperimentsAndArchiveProps extends PageProps {
  view: AllExperimentsAndArchiveTab;
}

interface AllExperimentsAndArchiveState {
  selectedTab: AllExperimentsAndArchiveTab;
}

class AllExperimentsAndArchive extends Page<
  AllExperimentsAndArchiveProps,
  AllExperimentsAndArchiveState
> {
  public getInitialToolbarState(): ToolbarProps {
    return { actions: {}, breadcrumbs: [], pageTitle: '' };
  }

  public render(): React.JSX.Element {
    return (
      <div className='kfp-workflow-page'>
        <InspectionTabs
          tabs={['Active', 'Archived']}
          selectedTab={this.props.view}
          onSwitch={this._tabSwitched.bind(this)}
          ariaLabel='Experiments'
        >
          {this.props.view === 0 && <ExperimentsPage {...this.props} />}

          {this.props.view === 1 && <ArchivedExperimentsPage {...this.props} />}
        </InspectionTabs>
      </div>
    );
  }

  public async refresh(): Promise<void> {
    return;
  }

  private _tabSwitched(newTab: AllExperimentsAndArchiveTab): void {
    this.props.navigate(
      newTab === AllExperimentsAndArchiveTab.EXPERIMENTS
        ? RoutePage.EXPERIMENTS
        : RoutePage.ARCHIVED_EXPERIMENTS,
    );
  }
}

export default AllExperimentsAndArchive;
