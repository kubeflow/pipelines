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
  refreshCount: number;
  onTabSwitch?: (tab: RunListsGroupTab) => void;
};

/**
 * Contains two tab buttons which allows user to see Active or Archived runs list.
 */
class RunListsRouter extends React.PureComponent<RunListsRouterProps> {
  private _runlistRef = React.createRef<RunList>();

  switchTab = (newTab: number) => {
    if (this._getSelectedTab() === newTab) {
      return;
    }
    if (this.props.onTabSwitch) {
      this.props.onTabSwitch(newTab);
    }
  };

  componentDidUpdate(prevProps: { refreshCount: number }) {
    if (prevProps.refreshCount === this.props.refreshCount) {
      return;
    }
    this.refresh();
  }

  public async refresh(): Promise<void> {
    if (this._runlistRef.current) {
      await this._runlistRef.current.refresh();
    }
    return;
  }

  public render(): JSX.Element {
    return (
      <div className={classes(commonCss.page, padding(20, 't'))}>
        <MD2Tabs
          tabs={['Active', 'Archived']}
          selectedTab={this._getSelectedTab()}
          onSwitch={this.switchTab}
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

  private _getSelectedTab() {
    return this.props.storageState === RunStorageState.ARCHIVED
      ? RunListsGroupTab.ARCHIVE
      : RunListsGroupTab.ACTIVE;
  }
}

export default RunListsRouter;
