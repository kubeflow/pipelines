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
import { CollectionScopeToggle } from '../components/tables/CollectionScopeToggle';
import AllRunsListPage from './AllRunsList';
import '../components/tables/RunsTable.css';
import { Page, PageProps } from './Page';
import { RoutePage } from '../components/Router';
import { ToolbarProps } from 'src/lib/PageChromeTypes';
import ArchivedRunsPage from './ArchivedRuns';

export enum RunsPageTab {
  RUNS = 0,
  ARCHIVE = 1,
}

export interface RunsPageProps extends PageProps {
  view: RunsPageTab;
}

interface RunsPageState {
  selectedTab: RunsPageTab;
}

class RunsPage extends Page<RunsPageProps, RunsPageState> {
  public getInitialToolbarState(): ToolbarProps {
    return { actions: {}, breadcrumbs: [], pageTitle: '' };
  }

  public render(): React.JSX.Element {
    const scopeControl = (
      <CollectionScopeToggle
        label='Run scope'
        archived={this.props.view === 1}
        onChange={(archived) =>
          this.props.navigate(archived ? RoutePage.ARCHIVED_RUNS : RoutePage.RUNS)
        }
      />
    );
    return (
      <div className='kfp-runs-page'>
        {this.props.view === 0 ? (
          <AllRunsListPage {...this.props} scopeControl={scopeControl} />
        ) : (
          <ArchivedRunsPage {...this.props} scopeControl={scopeControl} />
        )}
      </div>
    );
  }

  public async refresh(): Promise<void> {
    return;
  }
}

export default RunsPage;
