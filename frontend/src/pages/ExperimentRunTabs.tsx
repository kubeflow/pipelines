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

import * as React from 'react';
import { V2RunStorageState } from 'src/apisv2/run';
import { CollectionScopeToggle } from 'src/components/tables/CollectionScopeToggle';
import RunList, { RunListProps } from './RunList';

export enum RunListsGroupTab {
  ACTIVE = 0,
  ARCHIVE = 1,
}

export type ExperimentRunTabsProps = RunListProps & {
  storageState: V2RunStorageState;
  refreshCount: number;
  onTabSwitch?: (tab: RunListsGroupTab) => void;
};

/**
 * Shows active or archived experiment runs with an inline scope toggle.
 */
class ExperimentRunTabs extends React.PureComponent<ExperimentRunTabsProps> {
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

  public render(): React.JSX.Element {
    return (
      <div className='kfp-workflow-content'>
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
          scopeControl={
            <CollectionScopeToggle
              label='Experiment run scope'
              archived={this._getSelectedTab() === RunListsGroupTab.ARCHIVE}
              onChange={(archived) =>
                this.switchTab(archived ? RunListsGroupTab.ARCHIVE : RunListsGroupTab.ACTIVE)
              }
            />
          }
        />
      </div>
    );
  }

  private _getSelectedTab() {
    return this.props.storageState === V2RunStorageState.ARCHIVED
      ? RunListsGroupTab.ARCHIVE
      : RunListsGroupTab.ACTIVE;
  }
}

export default ExperimentRunTabs;
