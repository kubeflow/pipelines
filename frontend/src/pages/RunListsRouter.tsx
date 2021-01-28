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
  storageState: RunStorageState;
  onTabSwitch?: (tab: RunListsGroupTab, cb?: () => void) => void;
};

interface RunListsRouterState {
  storageState: RunStorageState;
}
/**
 * Contains two tab buttons which allows user to see Active or Archived runs list.
 */
class RunListsRouter extends React.PureComponent<RunListsRouterProps, RunListsRouterState> {
  protected _isMounted = true;
  private _runlistRef = React.createRef<RunList>();

  constructor(props: any) {
    super(props);

    this.state = {
      storageState: props.storageState,
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
          selectedTab={this._getSelectedTab()}
          onSwitch={this._switchTab.bind(this)}
        />

        {
          <RunList
            hideExperimentColumn={true}
            experimentIdMask={this.props.experimentIdMask}
            ref={this._runlistRef}
            onSelectionChange={this.props.onSelectionChange}
            selectedIds={this.props.selectedIds}
            noFilterBox={false}
            disablePaging={false}
            disableSorting={true}
            {...this.props}
          />
        }
      </div>
    );
  }

  public async refresh(): Promise<void> {
    if (this._runlistRef.current) {
      await this._runlistRef.current.refresh();
    }
    return;
  }

  private _switchTab(newTab: number): void {
    if (this._getSelectedTab() === newTab) {
      return;
    }
    this.setState({
      storageState: this._getStorageState(newTab),
    });
    if (this.props.onTabSwitch) {
      this.props.onTabSwitch(newTab, () => {
        this.refresh();
      });
    }
  }

  private _getSelectedTab(): number {
    return this.state.storageState === RunStorageState.ARCHIVED
      ? RunListsGroupTab.ARCHIVE
      : RunListsGroupTab.ACTIVE;
  }

  private _getStorageState(newTab: number): number {
    return newTab === RunListsGroupTab.ARCHIVE
      ? RunStorageState.ARCHIVED
      : RunStorageState.AVAILABLE;
  }
}

export default RunListsRouter;
