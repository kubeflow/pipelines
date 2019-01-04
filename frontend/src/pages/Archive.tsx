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
import RunList from './RunList';
import { Apis } from '../lib/Apis';
import { Page } from './Page';
import { RunStorageState } from '../apis/run';
import { ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { errorToMessage, s } from '../lib/Utils';

interface ArchiveState {
  selectedIds: string[];
}

export default class Archive extends Page<{}, ArchiveState> {
  private _runlistRef = React.createRef<RunList>();

  constructor(props: any) {
    super(props);

    this.state = {
      selectedIds: [],
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: [{
        action: () => this.props.updateDialog({
          buttons: [
            { onClick: async () => await this._unarchiveDialogClosed(true), text: 'Restore' },
            { onClick: async () => await this._unarchiveDialogClosed(false), text: 'Cancel' },
          ],
          onClose: async () => await this._unarchiveDialogClosed(false),
          title: `Restore ${this.state.selectedIds.length} resource${s(this.state.selectedIds.length)}?`,
        }),
        disabled: true,
        disabledTitle: 'Select at least one resource to restore',
        id: 'restoreBtn',
        title: 'Restore',
        tooltip: 'Restore',
      }, {
        action: this.refresh.bind(this),
        id: 'refreshBtn',
        title: 'Refresh',
        tooltip: 'Refresh the list of resources',
      }],
      breadcrumbs: [],
      pageTitle: 'Experiments',
    };
  }

  public render(): JSX.Element {
    return <div className={classes(commonCss.page, padding(20, 'lr'))}>
      <RunList onError={this.showPageError.bind(this)} selectedIds={this.state.selectedIds}
        onSelectionChange={this._selectionChanged.bind(this)} ref={this._runlistRef}
        storageState={RunStorageState.ARCHIVED} {...this.props} />
    </div>;
  }

  public async refresh(): Promise<void> {
    // Tell run list to refresh
    if (this._runlistRef.current) {
      this.clearBanner();
      await this._runlistRef.current.refresh();
    }
  }

  private _selectionChanged(selectedIds: string[]): void {
    const toolbarActions = [...this.props.toolbarProps.actions];
    // Unarchive button
    toolbarActions[0].disabled = !selectedIds.length;
    this.props.updateToolbar({ breadcrumbs: this.props.toolbarProps.breadcrumbs, actions: toolbarActions });
    this.setState({ selectedIds });
  }

  private async _unarchiveDialogClosed(confirmed: boolean): Promise<void> {
    if (confirmed) {
      const unsuccessfulIds: string[] = [];
      const errorMessages: string[] = [];
      await Promise.all(this.state.selectedIds.map(async (id) => {
        try {
          await Apis.runServiceApi.unarchiveRun(id);
        } catch (err) {
          unsuccessfulIds.push(id);
          const errorMessage = await errorToMessage(err);
          errorMessages.push(`Deleting run failed with error: "${errorMessage}"`);
        }
      }));

      const successfulObjects = this.state.selectedIds.length - unsuccessfulIds.length;
      if (successfulObjects > 0) {
        this.props.updateSnackbar({
          message: `Successfully restored ${successfulObjects} resource${s(successfulObjects)}!`,
          open: true,
        });
        this.refresh();
      }

      if (unsuccessfulIds.length > 0) {
        this.showErrorDialog(
          `Failed to restore ${unsuccessfulIds.length} resource${s(unsuccessfulIds)}`,
          errorMessages.join('\n\n'));
      }

      this._selectionChanged(unsuccessfulIds);
    }
  }

}
