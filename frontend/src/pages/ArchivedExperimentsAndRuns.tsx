/*
 * Copyright 2020 Google LLC
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
import ArchivedRuns from './ArchivedRuns';
import ArchivedExperiments from './ArchivedExperiments';
import MD2Tabs from '../atoms/MD2Tabs';
import { Page, PageProps } from './Page';
import { RoutePage } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { TFunction } from 'i18next';
import { withTranslation } from 'react-i18next';

export enum ArchivedExperimentsAndRunsTab {
  RUNS = 0,
  EXPERIMENTS = 1,
}

export interface ArchivedExperimentAndRunsProps extends PageProps {
  view: ArchivedExperimentsAndRunsTab;
  t: TFunction;
}

interface ArchivedExperimentAndRunsState {
  selectedTab: ArchivedExperimentsAndRunsTab;
}

class ArchivedExperimentsAndRuns extends Page<
  ArchivedExperimentAndRunsProps,
  ArchivedExperimentAndRunsState
> {
  public getInitialToolbarState(): ToolbarProps {
    const { t } = this.props;
    return { actions: {}, breadcrumbs: [], pageTitle: '', t };
  }

  public render(): JSX.Element {
    const { t } = this.props;
    return (
      <div className={classes(commonCss.page, padding(20, 't'))}>
        <MD2Tabs
          tabs={[t('runs'), t('common:experiments')]}
          selectedTab={this.props.view}
          onSwitch={this._tabSwitched.bind(this)}
        />
        {this.props.view === ArchivedExperimentsAndRunsTab.RUNS && <ArchivedRuns {...this.props} />}

        {this.props.view === ArchivedExperimentsAndRunsTab.EXPERIMENTS && (
          <ArchivedExperiments {...this.props} />
        )}
      </div>
    );
  }

  public async refresh(): Promise<void> {
    return;
  }

  private _tabSwitched(newTab: ArchivedExperimentsAndRunsTab): void {
    this.props.history.push(
      newTab === ArchivedExperimentsAndRunsTab.EXPERIMENTS
        ? RoutePage.ARCHIVED_EXPERIMENTS
        : RoutePage.ARCHIVED_RUNS,
    );
  }
}

export default withTranslation(['experiments', 'common'])(ArchivedExperimentsAndRuns);
