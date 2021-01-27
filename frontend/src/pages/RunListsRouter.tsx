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
import { RunStorageState } from 'src/apis/run';
import MD2Tabs from 'src/atoms/MD2Tabs';
import { commonCss, padding } from 'src/Css';
import { classes } from 'typestyle';
import RunList, { RunListProps } from './RunList';

export enum RunListsGroupTab {
  ACTIVE = 0,
  ARCHIVE = 1,
}

export type RunListsRouterProps = RunListProps & {
  view: RunListsGroupTab;
  onTabSwitch?: (tab: RunListsGroupTab, cb?: () => void) => void;
};

interface RunListsRouterState {
  selectedTab: number;
  // experiment: Experiment
}

class RunListsRouter extends React.PureComponent<RunListsRouterProps, RunListsRouterState> {
  //   public getInitialToolbarState(): ToolbarProps {
  //       throw new Error('Method not implemented.');
  //   }
  protected _isMounted = true;
  private _runlistRef = React.createRef<RunList>();

  constructor(props: any) {
    super(props);

    this.state = {
      selectedTab: props.view,
    };
  }

  public componentWillUnmount(): void {
    this._isMounted = false;
  }

  public render(): JSX.Element {
    return (
      <div className={classes(commonCss.page, padding(20, 't'))}>
        <MD2Tabs
          tabs={['Active', 'Archived']}
          selectedTab={this.state.selectedTab}
          onSwitch={this._switchTab.bind(this)}
        />

        {
          <RunList
            hideExperimentColumn={true}
            experimentIdMask={this.props.experimentIdMask}
            ref={this._runlistRef}
            // onError={this.props.onError}
            onSelectionChange={this.props.onSelectionChange}
            selectedIds={this.props.selectedIds}
            storageState={
              this.state.selectedTab === RunListsGroupTab.ACTIVE
                ? RunStorageState.AVAILABLE
                : RunStorageState.ARCHIVED
            }
            noFilterBox={false}
            disablePaging={false}
            disableSorting={true}
            {...this.props}
          />
        }

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
    if (this.state.selectedTab === newTab) {
      return;
    }
    this.setState({
      selectedTab: newTab,
    });
    if (this.props.onTabSwitch) {
      this.props.onTabSwitch(newTab, () => {
        this.refresh();
      });
    }
  }

  public async refresh(): Promise<void> {
    if (this._runlistRef.current) {
      await this._runlistRef.current.refresh();
    }
    return;
  }
}

export default RunListsRouter;
