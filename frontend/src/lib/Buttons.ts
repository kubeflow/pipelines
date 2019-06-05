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

import AddIcon from '@material-ui/icons/Add';
import CollapseIcon from '@material-ui/icons/UnfoldLess';
import ExpandIcon from '@material-ui/icons/UnfoldMore';
import { ToolbarActionConfig } from '../components/Toolbar';
import { PageProps } from '../pages/Page';
import { URLParser } from './URLParser';
import { RoutePage, QUERY_PARAMS } from '../components/Router';
import { Apis } from './Apis';
import { errorToMessage, s } from './Utils';

export default class Buttons {
  private _props: PageProps;
  private _refresh: () => void;
  private _urlParser: URLParser;

  constructor(pageProps: PageProps, refresh: () => void) {
    this._props = pageProps;
    this._refresh = refresh;
    this._urlParser = new URLParser(pageProps);
  }

  public archive(getSelectedIds: () => string[], useCurrentResource: boolean,
    callback: (selectedIds: string[], success: boolean) => void): ToolbarActionConfig {
    return {
      action: () => this._archive(getSelectedIds(), useCurrentResource, callback),
      disabled: !useCurrentResource,
      disabledTitle: useCurrentResource ? undefined : 'Select at least one resource to archive',
      id: 'archiveBtn',
      title: 'Archive',
      tooltip: 'Archive',
    };
  }

  public cloneRun(getSelectedIds: () => string[], useCurrentResource: boolean): ToolbarActionConfig {
    return {
      action: () => this._cloneRun(getSelectedIds()),
      disabled: !useCurrentResource,
      disabledTitle: useCurrentResource ? undefined : 'Select a run to clone',
      id: 'cloneBtn',
      title: 'Clone run',
      tooltip: 'Create a copy from this run\s initial state',
    };
  }

  public collapseSections(action: () => void): ToolbarActionConfig {
    return {
      action,
      icon: CollapseIcon,
      id: 'collapseBtn',
      title: 'Collapse all',
      tooltip: 'Collapse all sections',
    };
  }

  public compareRuns(getSelectedIds: () => string[]): ToolbarActionConfig {
    return {
      action: () => this._compareRuns(getSelectedIds()),
      disabled: true,
      disabledTitle: 'Select multiple runs to compare',
      id: 'compareBtn',
      title: 'Compare runs',
      tooltip: 'Compare up to 10 selected runs',
    };
  }

  public delete(getSelectedIds: () => string[], resourceName: 'pipeline' | 'recurring run config',
    callback: (selectedIds: string[], success: boolean) => void, useCurrentResource: boolean): ToolbarActionConfig {
    return {
      action: () => resourceName === 'pipeline' ?
        this._deletePipeline(getSelectedIds(), useCurrentResource, callback) :
        this._deleteRecurringRun(getSelectedIds()[0], useCurrentResource, callback),
      disabled: !useCurrentResource,
      disabledTitle: useCurrentResource ? undefined : `Select at least one ${resourceName} to delete`,
      id: 'deleteBtn',
      title: 'Delete',
      tooltip: 'Delete',
    };
  }

  public disableRecurringRun(getId: () => string): ToolbarActionConfig {
    return {
      action: () => this._setRecurringRunEnabledState(getId(), false),
      disabled: true,
      disabledTitle: 'Run schedule already disabled',
      id: 'disableBtn',
      title: 'Disable',
      tooltip: 'Disable the run\'s trigger',
    };
  }

  public enableRecurringRun(getId: () => string): ToolbarActionConfig {
    return {
      action: () => this._setRecurringRunEnabledState(getId(), true),
      disabled: true,
      disabledTitle: 'Run schedule already enabled',
      id: 'enableBtn',
      title: 'Enable',
      tooltip: 'Enable the run\'s trigger',
    };
  }

  public expandSections(action: () => void): ToolbarActionConfig {
    return {
      action,
      icon: ExpandIcon,
      id: 'expandBtn',
      title: 'Expand all',
      tooltip: 'Expand all sections',
    };
  }

  public newExperiment(getPipelineId?: () => string): ToolbarActionConfig {
    return {
      action: () => this._createNewExperiment(getPipelineId ? getPipelineId() : ''),
      icon: AddIcon,
      id: 'newExperimentBtn',
      outlined: true,
      title: 'Create experiment',
      tooltip: 'Create a new experiment',
    };
  }

  public newRun(getExperimentId?: () => string): ToolbarActionConfig {
    return {
      action: () => this._createNewRun(false, getExperimentId ? getExperimentId() : undefined),
      icon: AddIcon,
      id: 'createNewRunBtn',
      outlined: true,
      primary: true,
      title: 'Create run',
      tooltip: 'Create a new run',
    };
  }

  public newRunFromPipeline(getPipelineId: () => string): ToolbarActionConfig {
    return {
      action: () => this._createNewRunFromPipeline(getPipelineId()),
      icon: AddIcon,
      id: 'createNewRunBtn',
      outlined: true,
      primary: true,
      title: 'Create run',
      tooltip: 'Create a new run',
    };
  }

  public newRecurringRun(experimentId: string): ToolbarActionConfig {
    return {
      action: () => this._createNewRun(true, experimentId),
      icon: AddIcon,
      id: 'createNewRecurringRunBtn',
      outlined: true,
      title: 'Create recurring run',
      tooltip: 'Create a new recurring run',
    };
  }

  public refresh(action: () => void): ToolbarActionConfig {
    return {
      action,
      id: 'refreshBtn',
      title: 'Refresh',
      tooltip: 'Refresh the list',
    };
  }

  public restore(getSelectedIds: () => string[], useCurrentResource: boolean,
    callback: (selectedIds: string[], success: boolean) => void): ToolbarActionConfig {
    return {
      action: () => this._restore(getSelectedIds(), useCurrentResource, callback),
      disabled: !useCurrentResource,
      disabledTitle: useCurrentResource ? undefined : 'Select at least one resource to restore',
      id: 'restoreBtn',
      title: 'Restore',
      tooltip: 'Restore',
    };
  }

  public terminateRun(getSelectedIds: () => string[], useCurrentResource: boolean,
      callback: (selectedIds: string[], success: boolean) => void): ToolbarActionConfig {
    return {
      action: () => this._terminateRun(getSelectedIds(), useCurrentResource, callback),
      disabled: !useCurrentResource,
      disabledTitle: useCurrentResource ? undefined : 'Select at least one run to terminate',
      id: 'terminateRunBtn',
      title: 'Terminate',
      tooltip: 'Terminate execution of a run',
    };
  }

  public upload(action: () => void): ToolbarActionConfig {
    return {
      action,
      icon: AddIcon,
      id: 'uploadBtn',
      outlined: true,
      title: 'Upload pipeline',
      tooltip: 'Upload pipeline',
    };
  }

  private _cloneRun(selectedIds: string[]): void {
    if (selectedIds.length === 1) {
      const runId = selectedIds[0];
      const searchString = this._urlParser.build({ [QUERY_PARAMS.cloneFromRun]: runId || '' });
      this._props.history.push(RoutePage.NEW_RUN + searchString);
    }
  }

  private _archive(selectedIds: string[], useCurrent: boolean,
    callback: (selectedIds: string[], success: boolean) => void): void {
    this._dialogActionHandler(
      selectedIds,
      `Run${s(selectedIds)} will be moved to the Archive section, where you can still view ` +
      `${selectedIds.length === 1 ? 'its' : 'their'} details. Please note that the run will not ` +
      `be stopped if it's running when it's archived. Use the Restore action to restore the ` +
      `run${s(selectedIds)} to ${selectedIds.length === 1 ? 'its' : 'their'} original location.`,
      useCurrent,
      id => Apis.runServiceApi.archiveRun(id),
      callback,
      'Archive',
      'run',
    );
  }

  private _restore(selectedIds: string[], useCurrent: boolean,
    callback: (selectedIds: string[], success: boolean) => void): void {
    this._dialogActionHandler(
      selectedIds,
      `Do you want to restore ${selectedIds.length === 1 ? 'this run to its' : 'these runs to their'} original location?`,
      useCurrent,
      id => Apis.runServiceApi.unarchiveRun(id),
      callback,
      'Restore',
      'run',
    );
  }

  private _deletePipeline(selectedIds: string[], useCurrentResource: boolean,
    callback: (selectedIds: string[], success: boolean) => void): void {
    this._dialogActionHandler(
      selectedIds,
      'Do you want to delete this Pipeline? This action cannot be undone.',
      useCurrentResource,
      id => Apis.pipelineServiceApi.deletePipeline(id),
      callback,
      'Delete',
      'pipeline',
    );
  }

  private _deleteRecurringRun(id: string, useCurrentResource: boolean,
    callback: (_: string[], success: boolean) => void): void {
    this._dialogActionHandler(
      [id],
      'Do you want to delete this recurring run config? This action cannot be undone.',
      useCurrentResource,
      jobId => Apis.jobServiceApi.deleteJob(jobId),
      callback,
      'Delete',
      'recurring run config',
    );
  }

  private _terminateRun(ids: string[], useCurrentResource: boolean,
    callback: (_: string[], success: boolean) => void): void {
    this._dialogActionHandler(
      ids,
      'Do you want to terminate this run? This action cannot be undone. This will terminate any'
      + ' running pods, but they will not be deleted.',
      useCurrentResource,
      id => Apis.runServiceApi.terminateRun(id),
      callback,
      'Terminate',
      'run',
    );
  }

  private _dialogActionHandler(selectedIds: string[], content: string, useCurrentResource: boolean,
    api: (id: string) => Promise<void>, callback: (selectedIds: string[], success: boolean) => void,
    actionName: string, resourceName: string): void {

    const dialogClosedHandler = (confirmed: boolean) =>
      this._dialogClosed(confirmed, selectedIds, actionName, resourceName, useCurrentResource, api, callback);

    this._props.updateDialog({
      buttons: [{
        onClick: async () => await dialogClosedHandler(false),
        text: 'Cancel',
      }, {
        onClick: async () => await dialogClosedHandler(true),
        text: actionName,
      }],
      content,
      onClose: async () => await dialogClosedHandler(false),
      title: `${actionName} ${useCurrentResource ? 'this' : selectedIds.length} ${resourceName}${useCurrentResource ? '' : s(selectedIds.length)}?`,
    });
  }

  private async _dialogClosed(confirmed: boolean, selectedIds: string[], actionName: string,
    resourceName: string, useCurrentResource: boolean, api: (id: string) => Promise<void>,
    callback: (selectedIds: string[], success: boolean) => void): Promise<void> {
    if (confirmed) {
      const unsuccessfulIds: string[] = [];
      const errorMessages: string[] = [];
      await Promise.all(selectedIds.map(async (id) => {
        try {
          await api(id);
        } catch (err) {
          unsuccessfulIds.push(id);
          const errorMessage = await errorToMessage(err);
          errorMessages.push(`Failed to ${actionName.toLowerCase()} ${resourceName}: ${id} with error: "${errorMessage}"`);
        }
      }));

      const successfulOps = selectedIds.length - unsuccessfulIds.length;
      if (useCurrentResource || successfulOps > 0) {
        this._props.updateSnackbar({
          message: `${actionName} succeeded for ${useCurrentResource ? 'this' : successfulOps} ${resourceName}${useCurrentResource ? '' : s(successfulOps)}`,
          open: true,
        });
        if (!useCurrentResource) {
          this._refresh();
        }
      }

      if (unsuccessfulIds.length > 0) {
        this._props.updateDialog({
          buttons: [{ text: 'Dismiss' }],
          content: errorMessages.join('\n\n'),
          title: `Failed to ${actionName.toLowerCase()} ${useCurrentResource ? '' : unsuccessfulIds.length + ' '}${resourceName}${useCurrentResource ? '' : s(unsuccessfulIds)}`,
        });
      }

      callback(unsuccessfulIds, !unsuccessfulIds.length);
    }
  }
  private _compareRuns(selectedIds: string[]): void {
    const indices = selectedIds;
    if (indices.length > 1 && indices.length <= 10) {
      const runIds = selectedIds.join(',');
      const searchString = this._urlParser.build({ [QUERY_PARAMS.runlist]: runIds });
      this._props.history.push(RoutePage.COMPARE + searchString);
    }
  }

  private _createNewExperiment(pipelineId: string): void {
    const searchString = pipelineId ? this._urlParser.build({
      [QUERY_PARAMS.pipelineId]: pipelineId
    }) : '';
    this._props.history.push(RoutePage.NEW_EXPERIMENT + searchString);
  }

  private _createNewRun(isRecurring: boolean, experimentId?: string): void {
    const searchString = this._urlParser.build(Object.assign(
      { [QUERY_PARAMS.experimentId]: experimentId || '' },
      isRecurring ? { [QUERY_PARAMS.isRecurring]: '1' } : {}));
    this._props.history.push(RoutePage.NEW_RUN + searchString);
  }

  private _createNewRunFromPipeline(pipelineId?: string): void {
    let searchString = '';
    const fromRunId = this._urlParser.get(QUERY_PARAMS.fromRunId);

    if (fromRunId) {
      searchString = this._urlParser.build(Object.assign(
        { [QUERY_PARAMS.fromRunId]: fromRunId }
      ));
    } else {
      searchString = this._urlParser.build(Object.assign(
        { [QUERY_PARAMS.pipelineId]: pipelineId || '' }
      ));
    }

    this._props.history.push(RoutePage.NEW_RUN + searchString);
  }

  private async _setRecurringRunEnabledState(id: string, enabled: boolean): Promise<void> {
    if (id) {
      const toolbarActions = [...this._props.toolbarProps.actions];

      const buttonIndex = enabled ? 1 : 2;

      toolbarActions[buttonIndex].busy = true;
      this._props.updateToolbar({ actions: toolbarActions });
      try {
        await (enabled ? Apis.jobServiceApi.enableJob(id) : Apis.jobServiceApi.disableJob(id));
        this._refresh();
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        this._props.updateDialog({
          buttons: [{ text: 'Dismiss' }],
          content: errorMessage,
          title: `Failed to ${enabled ? 'enable' : 'disable'} recurring run`,
        });
      } finally {
        toolbarActions[buttonIndex].busy = false;
        this._props.updateToolbar({ actions: toolbarActions });
      }
    }
  }

}
