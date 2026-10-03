/*
 * Copyright 2020 The Kubeflow Authors
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
import AllRunsListPage from './AllRunsList';
import { Link } from 'react-router';
import '../components/modernization/RunsTable.css';
import { Page, PageProps } from './Page';
import { RoutePage } from '../components/Router';
import { ToolbarProps } from 'src/lib/PageChromeTypes';
import ArchivedRunsPage from './ArchivedRuns';

export enum AllRunsAndArchiveTab {
  RUNS = 0,
  ARCHIVE = 1,
}

export interface AllRunsAndArchiveProps extends PageProps {
  view: AllRunsAndArchiveTab;
}

interface AllRunsAndArchiveState {
  selectedTab: AllRunsAndArchiveTab;
}

class AllRunsAndArchive extends Page<AllRunsAndArchiveProps, AllRunsAndArchiveState> {
  public getInitialToolbarState(): ToolbarProps {
    return { actions: {}, breadcrumbs: [], pageTitle: '' };
  }

  public render(): React.JSX.Element {
    return (
      <div className='kfp-runs-page'>
        <nav className='kfp-runs-tabs' aria-label='Run views'>
          <Link
            to={RoutePage.RUNS}
            aria-current={this.props.view === AllRunsAndArchiveTab.RUNS ? 'page' : undefined}
          >
            Active
          </Link>
          <Link
            to={RoutePage.ARCHIVED_RUNS}
            aria-current={this.props.view === AllRunsAndArchiveTab.ARCHIVE ? 'page' : undefined}
          >
            Archived
          </Link>
        </nav>
        {this.props.view === 0 && <AllRunsListPage {...this.props} />}

        {this.props.view === 1 && <ArchivedRunsPage {...this.props} />}
      </div>
    );
  }

  public async refresh(): Promise<void> {
    return;
  }
}

export default AllRunsAndArchive;
