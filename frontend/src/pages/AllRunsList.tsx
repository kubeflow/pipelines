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
import Buttons from '../lib/Buttons';
import RunList from './RunList';
import { Page } from './Page';
import { RoutePage, QUERY_PARAMS } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { URLParser } from '../lib/URLParser';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { s, errorToMessage } from '../lib/Utils';
import { Apis } from '../lib/Apis';

interface AllRunsListState {
  selectedIds: string[];
}

class AllRunsList extends Page<{}, AllRunsListState> {

  private _runlistRef = React.createRef<RunList>();

  constructor(props: any) {
    super(props);

    this.state = {
      selectedIds: [],
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: [
        Buttons.newExperiment(this._newExperimentClicked.bind(this)),
        Buttons.compareRuns(this._compareRuns.bind(this)),
        Buttons.cloneRun(this._cloneRun.bind(this)),
        Buttons.archive(() => this.props.updateDialog({
          buttons: [
            { onClick: async () => await this._archiveDialogClosed(true), text: 'Archive' },
            { onClick: async () => await this._archiveDialogClosed(false), text: 'Cancel' },
          ],
          onClose: async () => await this._archiveDialogClosed(false),
          title: `Archive ${this.state.selectedIds.length} run${s(this.state.selectedIds.length)}?`,
        })),
        Buttons.refresh(this.refresh.bind(this)),
      ],
      breadcrumbs: [],
      pageTitle: 'Experiments',
    };
  }

  public render(): JSX.Element {
    return <div className={classes(commonCss.page, padding(20, 'lr'))}>
      <RunList onError={this.showPageError.bind(this)} selectedIds={this.state.selectedIds}
        onSelectionChange={this._selectionChanged.bind(this)} ref={this._runlistRef} {...this.props} />
    </div>;
  }

  public async refresh(): Promise<void> {
    // Tell run list to refresh
    if (this._runlistRef.current) {
      this.clearBanner();
      await this._runlistRef.current.refresh();
    }
  }

  private _newExperimentClicked(): void {
    this.props.history.push(RoutePage.NEW_EXPERIMENT);
  }

  private _compareRuns(): void {
    const indices = this.state.selectedIds;
    if (indices.length > 1 && indices.length <= 10) {
      const runIds = this.state.selectedIds.join(',');
      const searchString = new URLParser(this.props).build({
        [QUERY_PARAMS.runlist]: runIds,
      });
      this.props.history.push(RoutePage.COMPARE + searchString);
    }
  }

  private _selectionChanged(selectedIds: string[]): void {
    const toolbarActions = [...this.props.toolbarProps.actions];
    // Compare runs button
    toolbarActions[1].disabled = selectedIds.length <= 1 || selectedIds.length > 10;
    // Clone run button
    toolbarActions[2].disabled = selectedIds.length !== 1;
    // Archive run button
    toolbarActions[3].disabled = !selectedIds.length;
    this.props.updateToolbar({ breadcrumbs: this.props.toolbarProps.breadcrumbs, actions: toolbarActions });
    this.setState({ selectedIds });
  }

  private _cloneRun(): void {
    if (this.state.selectedIds.length === 1) {
      const runId = this.state.selectedIds[0];
      const searchString = new URLParser(this.props).build({
        [QUERY_PARAMS.cloneFromRun]: runId || ''
      });
      this.props.history.push(RoutePage.NEW_RUN + searchString);
    }
  }

  private async _archiveDialogClosed(confirmed: boolean): Promise<void> {
    if (confirmed) {
      const unsuccessfulIds: string[] = [];
      const errorMessages: string[] = [];
      await Promise.all(this.state.selectedIds.map(async (id) => {
        try {
          await Apis.runServiceApi.archiveRun(id);
        } catch (err) {
          unsuccessfulIds.push(id);
          const errorMessage = await errorToMessage(err);
          errorMessages.push(`Deleting run failed with error: "${errorMessage}"`);
        }
      }));

      const successfulObjects = this.state.selectedIds.length - unsuccessfulIds.length;
      if (successfulObjects > 0) {
        this.props.updateSnackbar({
          message: `Successfully archived ${successfulObjects} run${s(successfulObjects)}!`,
          open: true,
        });
        this.refresh();
      }

      if (unsuccessfulIds.length > 0) {
        this.showErrorDialog(
          `Failed to archive ${unsuccessfulIds.length} run${s(unsuccessfulIds)}`,
          errorMessages.join('\n\n'));
      }

      this._selectionChanged(unsuccessfulIds);
    }
  }

}

export default AllRunsList;
