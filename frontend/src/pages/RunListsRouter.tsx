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
import { RouteComponentProps } from 'react-router-dom';
import { RunStorageState } from 'src/apis/run';
import MD2Tabs from 'src/atoms/MD2Tabs';
import { RoutePage } from 'src/components/Router';
import { ToolbarProps } from 'src/components/Toolbar';
import { commonCss, padding } from 'src/Css';
import { classes } from 'typestyle';
import AllRunsList from './AllRunsList';
import ArchivedRuns from './ArchivedRuns';
import { Page, PageProps } from './Page';
import RunList, { RunListProps } from './RunList';

export enum RunListsGroupTab {
  ACTIVE = 0,
  ARCHIVE = 1,
}

export type RunListsRouterProps = RunListProps & {
  view: RunListsGroupTab;
};

interface RunListsRouterState {
  selectedTab: number;
  // experiment: Experiment
}

class RunListsRouter extends React.PureComponent<RunListsRouterProps, RunListsRouterState> {
  //   public getInitialToolbarState(): ToolbarProps {
  //       throw new Error('Method not implemented.');
  //   }
  private _runlistRef = React.createRef<RunList>();

  constructor(props: any) {
    super(props);

    this.state = {
      selectedTab: props.view,
    };
  }

  public render(): JSX.Element {
    return (
      <div className={classes(commonCss.header, padding(10, 't'))}>
        <MD2Tabs
          tabs={['Active', 'Archived']}
          selectedTab={this.state.selectedTab}
          onSwitch={this._switchTab.bind(this)}
        />

        {this.state.selectedTab === RunListsGroupTab.ACTIVE && (
          <RunList
            hideExperimentColumn={true}
            experimentIdMask={this.props.experimentIdMask}
            ref={this._runlistRef}
            // onError={this.props.onError}
            onSelectionChange={this.props.onSelectionChange}
            selectedIds={this.props.selectedIds}
            storageState={RunStorageState.AVAILABLE}
            noFilterBox={false}
            disablePaging={false}
            disableSorting={true}
            {...this.props}
          />
        )}

        {this.state.selectedTab === RunListsGroupTab.ARCHIVE && (
          <RunList
            hideExperimentColumn={true}
            experimentIdMask={this.props.experimentIdMask}
            ref={this._runlistRef}
            // onError={this.props.onError}
            onSelectionChange={this.props.onSelectionChange}
            selectedIds={this.props.selectedIds}
            storageState={RunStorageState.ARCHIVED}
            noFilterBox={false}
            disablePaging={false}
            disableSorting={true}
            {...this.props}
          />
        )}
        {/* this should be RunList */}
        {/* {this.props.view === RunListsGroupTab.ACTIVE && <AllRunsList {...this.props} />}

        {this.props.view === RunListsGroupTab.ARCHIVE && <ArchivedRuns {...this.props} />} */}
      </div>
    );
  }

  private _switchTab(newTab: number): void {
    // this.props.history.push(
    //   newTab === RunListsGroupTab.ACTIVE ? RoutePage.RUNS : RoutePage.ARCHIVED_RUNS,
    // );
    this.setState({
      selectedTab: newTab,
    });
    this.refresh();
  }

  public async refresh(): Promise<void> {
    if (this._runlistRef.current) {
      await this._runlistRef.current.refresh();
    }
    return;
  }
}

export default RunListsRouter;
