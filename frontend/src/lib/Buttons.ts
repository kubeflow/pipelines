/*
 * Copyright 2018 The Kubeflow Authors
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
import { QUERY_PARAMS, RoutePage } from '../components/Router';
import { ToolbarActionMap } from '../components/Toolbar';
import { PageProps } from '../pages/Page';
import { Apis } from './Apis';
import { URLParser } from './URLParser';
import { errorToMessage, s } from './Utils';

import i18n from "i18next";

export enum ButtonKeys {
  ARCHIVE = 'archive',
  CLONE_RUN = 'cloneRun',
  CLONE_RECURRING_RUN = 'cloneRecurringRun',
  RETRY = 'retry',
  COLLAPSE = 'collapse',
  COMPARE = 'compare',
  DELETE_RUN = 'deleteRun',
  DISABLE_RECURRING_RUN = 'disableRecurringRun',
  ENABLE_RECURRING_RUN = 'enableRecurringRun',
  EXPAND = 'expand',
  NEW_EXPERIMENT = 'newExperiment',
  NEW_PIPELINE_VERSION = 'newPipelineVersion',
  NEW_RUN = 'newRun',
  NEW_RECURRING_RUN = 'newRecurringRun',
  NEW_RUN_FROM_PIPELINE_VERSION = 'newRunFromPipelineVersion',
  REFRESH = 'refresh',
  RESTORE = 'restore',
  TERMINATE_RUN = 'terminateRun',
  UPLOAD_PIPELINE = 'uploadPipeline',
}

export default class Buttons {
  private _map: ToolbarActionMap;
  private _props: PageProps;
  private _refresh: () => void;
  private _urlParser: URLParser;

  constructor(pageProps: PageProps, refresh: () => void, map?: ToolbarActionMap) {
    this._props = pageProps;
    this._refresh = refresh;
    this._urlParser = new URLParser(pageProps);
    this._map = map || {};
  }

  public getToolbarActionMap(): ToolbarActionMap {
    return this._map;
  }

  public archive(
    resourceName: 'run' | 'experiment',
    getSelectedIds: () => string[],
    useCurrentResource: boolean,
    callback: (selectedIds: string[], success: boolean) => void,
  ): Buttons {
    this._map[ButtonKeys.ARCHIVE] = {
      action: () =>
        resourceName === 'run'
          ? this._archiveRun(getSelectedIds(), useCurrentResource, callback)
          : this._archiveExperiments(getSelectedIds(), useCurrentResource, callback),
      disabled: !useCurrentResource,
      disabledTitle: useCurrentResource ? undefined : 'Select at least one resource to archive',
      id: 'archiveBtn',
      title: i18n.t('Buttons.archive'),
      tooltip: i18n.t('Buttons.archive'),
    };
    return this;
  }

  public cloneRun(getSelectedIds: () => string[], useCurrentResource: boolean): Buttons {
    this._map[ButtonKeys.CLONE_RUN] = {
      action: () => this._cloneRun(getSelectedIds()),
      disabled: !useCurrentResource,
      disabledTitle: useCurrentResource ? undefined : 'Select a run to clone',
      id: 'cloneBtn',
      style: { minWidth: 100 },
      title: i18n.t('Buttons.cloneRun'),
      tooltip: i18n.t('Buttons.createACopyFromThisRunsInitialState'),
    };
    return this;
  }

  public cloneRecurringRun(getSelectedIds: () => string[], useCurrentResource: boolean): Buttons {
    this._map[ButtonKeys.CLONE_RECURRING_RUN] = {
      action: () => this._cloneRun(getSelectedIds(), true),
      disabled: !useCurrentResource,
      disabledTitle: useCurrentResource ? undefined : i18n.t('Buttons.selectARecurringRunToClone'),
      id: 'cloneBtn',
      title: i18n.t('Buttons.cloneRecurringRun'),
      tooltip: i18n.t('Buttons.createACopyFromThisRunsInitialState'),
    };
    return this;
  }

  public retryRun(
    getSelectedIds: () => string[],
    useCurrentResource: boolean,
    callback: (selectedIds: string[], success: boolean) => void,
  ): Buttons {
    this._map[ButtonKeys.RETRY] = {
      action: () => this._retryRun(getSelectedIds(), useCurrentResource, callback),
      disabled: !useCurrentResource,
      disabledTitle: useCurrentResource ? undefined : i18n.t('Buttons.selectAtLeastOneResourceToRetry'),
      id: 'retryBtn',
      title: i18n.t('Buttons.retry'),
      tooltip: i18n.t('Buttons.retryThisRun'),
    };
    return this;
  }

  public collapseSections(action: () => void): Buttons {
    this._map[ButtonKeys.COLLAPSE] = {
      action,
      icon: CollapseIcon,
      id: 'collapseBtn',
      title: i18n.t('Buttons.collapseAll'),
      tooltip: i18n.t('Buttons.collapseAllSections'),
    };
    return this;
  }

  public compareRuns(getSelectedIds: () => string[]): Buttons {
    this._map[ButtonKeys.COMPARE] = {
      action: () => this._compareRuns(getSelectedIds()),
      disabled: true,
      disabledTitle: i18n.t('Buttons.selectMultipleRunsToCompare'),
      id: 'compareBtn',
      style: { minWidth: 125 },
      title: i18n.t('Buttons.compareRuns'),
      tooltip: i18n.t('Buttons.compareUpTo10SelectedRuns'),
    };
    return this;
  }

  // Delete resources of the same type, which can be pipeline, pipeline version,
  // or recurring run config.
  public delete(
    getSelectedIds: () => string[],
    resourceName: 'pipeline' | 'recurring run config' | 'pipeline version' | 'run',
    callback: (selectedIds: string[], success: boolean) => void,
    useCurrentResource: boolean,
  ): Buttons {
    this._map[ButtonKeys.DELETE_RUN] = {
      action: () =>
        resourceName === 'pipeline'
          ? this._deletePipeline(getSelectedIds(), useCurrentResource, callback)
          : resourceName === 'pipeline version'
          ? this._deletePipelineVersion(getSelectedIds(), useCurrentResource, callback)
          : resourceName === 'run'
          ? this._deleteRun(getSelectedIds(), useCurrentResource, callback)
          : this._deleteRecurringRun(getSelectedIds()[0], useCurrentResource, callback),
      disabled: !useCurrentResource,
      disabledTitle: useCurrentResource
        ? undefined
        : `${i18n.t('Buttons.selectAtLeastOne')} ${resourceName} ${i18n.t('Buttons.toDelete')}`,
      id: 'deleteBtn',
      title: i18n.t('Buttons.delete'),
      tooltip: i18n.t('Buttons.delete'),
    };
    return this;
  }

  // Delete pipelines and pipeline versions simultaneously.
  public deletePipelinesAndPipelineVersions(
    getSelectedIds: () => string[],
    getSelectedVersionIds: () => { [pipelineId: string]: string[] },
    callback: (pipelineId: string | undefined, selectedIds: string[]) => void,
    useCurrentResource: boolean,
  ): Buttons {
    this._map[ButtonKeys.DELETE_RUN] = {
      action: () => {
        this._dialogDeletePipelinesAndPipelineVersions(
          getSelectedIds(),
          getSelectedVersionIds(),
          callback,
        );
      },
      disabled: !useCurrentResource,
      disabledTitle: useCurrentResource
        ? undefined
        : i18n.t('Buttons.selectAtLeastOnePipelineAndOrOnePipelineVersionToDelete'),
      id: 'deletePipelinesAndPipelineVersionsBtn',
      title: i18n.t('Buttons.delete'),
      tooltip: i18n.t('Buttons.delete'),
    };
    return this;
  }

  public disableRecurringRun(getId: () => string): Buttons {
    this._map[ButtonKeys.DISABLE_RECURRING_RUN] = {
      action: () => this._setRecurringRunEnabledState(getId(), false),
      disabled: true,
      disabledTitle: i18n.t('Buttons.runScheduleAlreadyDisabled'),
      id: 'disableBtn',
      title: i18n.t('Buttons.disable'),
      tooltip: i18n.t('Buttons.disableTheRunSTrigger'),
    };
    return this;
  }

  public enableRecurringRun(getId: () => string): Buttons {
    this._map[ButtonKeys.ENABLE_RECURRING_RUN] = {
      action: () => this._setRecurringRunEnabledState(getId(), true),
      disabled: true,
      disabledTitle: i18n.t('Buttons.runScheduleAlreadyEnabled'),
      id: 'enableBtn',
      title: i18n.t('Buttons.enable'),
      tooltip: i18n.t('Buttons.enableTheRunSTrigger'),
    };
    return this;
  }

  public expandSections(action: () => void): Buttons {
    this._map[ButtonKeys.EXPAND] = {
      action,
      icon: ExpandIcon,
      id: 'expandBtn',
      title: i18n.t('Buttons.expandAll'),
      tooltip: i18n.t('Buttons.expandAllSections'),
    };
    return this;
  }

  public newExperiment(getPipelineId?: () => string): Buttons {
    this._map[ButtonKeys.NEW_EXPERIMENT] = {
      action: () => this._createNewExperiment(getPipelineId ? getPipelineId() : ''),
      icon: AddIcon,
      id: 'newExperimentBtn',
      outlined: true,
      primary: true,
      style: { minWidth: 185 },
      title: i18n.t('Buttons.createExperiment'),
      tooltip: i18n.t('Buttons.createANewExperiment'),
    };
    return this;
  }

  public newRun(getExperimentId?: () => string): Buttons {
    this._map[ButtonKeys.NEW_RUN] = {
      action: () => this._createNewRun(false, getExperimentId ? getExperimentId() : undefined),
      icon: AddIcon,
      id: 'createNewRunBtn',
      outlined: true,
      primary: true,
      style: { minWidth: 130 },
      title: i18n.t('Buttons.createRun'),
      tooltip: i18n.t('Buttons.createANewRun'),
    };
    return this;
  }

  public newRunFromPipelineVersion(
    getPipelineId: () => string,
    getPipelineVersionId: () => string,
  ): Buttons {
    this._map[ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION] = {
      action: () => this._createNewRunFromPipelineVersion(getPipelineId(), getPipelineVersionId()),
      icon: AddIcon,
      id: 'createNewRunBtn',
      outlined: true,
      primary: true,
      style: { minWidth: 130 },
      title: i18n.t('Buttons.createRun'),
      tooltip: i18n.t('Buttons.createANewRun'),
    };
    return this;
  }

  public newRecurringRun(experimentId: string): Buttons {
    this._map[ButtonKeys.NEW_RECURRING_RUN] = {
      action: () => this._createNewRun(true, experimentId),
      icon: AddIcon,
      id: 'createNewRecurringRunBtn',
      outlined: true,
      style: { minWidth: 195 },
      title: i18n.t('Buttons.createRecurringRun'),
      tooltip: i18n.t('Buttons.createANewRecurringRun'),
    };
    return this;
  }

  public newPipelineVersion(label: string, getPipelineId?: () => string): Buttons {
    this._map[ButtonKeys.NEW_PIPELINE_VERSION] = {
      action: () => this._createNewPipelineVersion(getPipelineId ? getPipelineId() : ''),
      icon: AddIcon,
      id: 'createPipelineVersionBtn',
      outlined: true,
      style: { minWidth: 160 },
      title: label,
      tooltip: i18n.t('Buttons.uploadPipelineVersion'),
    };
    return this;
  }

  public refresh(action: () => void): Buttons {
    this._map[ButtonKeys.REFRESH] = {
      action,
      id: 'refreshBtn',
      title: i18n.t('Buttons.refresh'),
      tooltip: i18n.t('Buttons.refreshTheList'),
    };
    return this;
  }

  public restore(
    resourceName: 'run' | 'experiment',
    getSelectedIds: () => string[],
    useCurrentResource: boolean,
    callback: (selectedIds: string[], success: boolean) => void,
  ): Buttons {
    this._map[ButtonKeys.RESTORE] = {
      action: () =>
        resourceName === 'run'
          ? this._restore(getSelectedIds(), useCurrentResource, callback)
          : this._restoreExperiments(getSelectedIds(), useCurrentResource, callback),
      disabled: !useCurrentResource,
      disabledTitle: useCurrentResource ? undefined : i18n.t('Buttons.selectAtLeastOneResourceToRestore'),
      id: 'restoreBtn',
      title: i18n.t('Buttons.restore'),
      tooltip: i18n.t('Buttons.restoreTheArchivedRunSToOriginalLocation'),
    };
    return this;
  }

  public terminateRun(
    getSelectedIds: () => string[],
    useCurrentResource: boolean,
    callback: (selectedIds: string[], success: boolean) => void,
  ): Buttons {
    this._map[ButtonKeys.TERMINATE_RUN] = {
      action: () => this._terminateRun(getSelectedIds(), useCurrentResource, callback),
      disabled: !useCurrentResource,
      disabledTitle: useCurrentResource ? undefined : i18n.t('Buttons.selectAtLeastOneRunToTerminate'),
      id: 'terminateRunBtn',
      title: i18n.t('Buttons.terminate'),
      tooltip: i18n.t('Buttons.terminateExecutionOfARun'),
    };
    return this;
  }

  public upload(action: () => void): Buttons {
    this._map[ButtonKeys.UPLOAD_PIPELINE] = {
      action,
      icon: AddIcon,
      id: 'uploadBtn',
      outlined: true,
      style: { minWidth: 160 },
      title: i18n.t('Buttons.uploadPipeline'),
      tooltip: i18n.t('Buttons.uploadPipeline'),
    };
    return this;
  }

  private _cloneRun(selectedIds: string[], isRecurring?: boolean): void {
    if (selectedIds.length === 1) {
      const runId = selectedIds[0];
      let searchTerms;
      if (isRecurring) {
        searchTerms = {
          [QUERY_PARAMS.cloneFromRecurringRun]: runId || '',
          [QUERY_PARAMS.isRecurring]: '1',
        };
      } else {
        searchTerms = { [QUERY_PARAMS.cloneFromRun]: runId || '' };
      }
      const searchString = this._urlParser.build(searchTerms);
      this._props.history.push(RoutePage.NEW_RUN + searchString);
    }
  }

  private _retryRun(
    selectedIds: string[],
    useCurrent: boolean,
    callback: (selectedIds: string[], success: boolean) => void,
  ): void {
    this._dialogActionHandler(
      selectedIds,
        `${i18n.t('Buttons.retryThisRun')}?`,
      useCurrent,
      id => Apis.runServiceApi.retryRun(id),
      callback,
      'Retry',
      'run',
    );
  }

  private _archiveRun(
    selectedIds: string[],
    useCurrent: boolean,
    callback: (selectedIds: string[], success: boolean) => void,
  ): void {
    this._dialogActionHandler(
      selectedIds,
      `Run${s(selectedIds)} ${i18n.t('Buttons.willBeMovedToTheArchiveSectionWhereYouCanStillView')} ` +
        `${
          selectedIds.length === 1 ? i18n.t('Buttons.its') : i18n.t('Buttons.their')
        } ${i18n.t('Buttons.detailsPleaseNoteThatTheRunWillNot')} ` +
        `${i18n.t('Buttons.beStoppedIfItSRunningWhenItSArchivedUseTheRestoreActionToRestoreThe')} ` +
        `run${s(selectedIds)} ${i18n.t('Buttons.to')} ${selectedIds.length === 1 ? i18n.t('Buttons.its') : i18n.t('Buttons.their')} ${i18n.t('Buttons.originalLocation')}.`,
      useCurrent,
      id => Apis.runServiceApi.archiveRun(id),
      callback,
      'Archive',
      'run',
    );
  }

  private _restore(
    selectedIds: string[],
    useCurrent: boolean,
    callback: (selectedIds: string[], success: boolean) => void,
  ): void {
    this._dialogActionHandler(
      selectedIds,
      `${i18n.t('Buttons.doYouWantToRestore')} ${
        selectedIds.length === 1 ? i18n.t('Buttons.thisRunToIts') : i18n.t('Buttons.theseRunsToTheir')
      } ${i18n.t('Buttons.originalLocation')}?`,
      useCurrent,
      id => Apis.runServiceApi.unarchiveRun(id),
      callback,
      'Restore',
      'run',
    );
  }

  private _restoreExperiments(
    selectedIds: string[],
    useCurrent: boolean,
    callback: (selectedIds: string[], success: boolean) => void,
  ): void {
    this._dialogActionHandler(
      selectedIds,
      `${i18n.t('Buttons.doYouWantToRestore')} ${
        selectedIds.length === 1 ? i18n.t('Buttons.thisExperimentToIts') : i18n.t('Buttons.theseExperimentsToTheir')
      } ${i18n.t('Buttons.originalLocationAllRunsAndJobsIn')} ${
        selectedIds.length === 1 ? i18n.t('Buttons.thisExperiment') : i18n.t('Buttons.theseExperiments')
      } ${i18n.t('Buttons.willStayAtTheirCurrentLocationsInSpiteThat')} ${
        selectedIds.length === 1 ? i18n.t('Buttons.thisExperiment') : i18n.t('Buttons.theseExperiments')
      } ${i18n.t('Buttons.willBeMovedTo')} ${selectedIds.length === 1 ? i18n.t('Buttons.its') : i18n.t('Buttons.their')} ${i18n.t('Buttons.originalLocation')}${s(
        selectedIds,
      )}.`,
      useCurrent,
      id => Apis.experimentServiceApi.unarchiveExperiment(id),
      callback,
      'Restore',
      'experiment',
    );
  }

  private _deletePipeline(
    selectedIds: string[],
    useCurrentResource: boolean,
    callback: (selectedIds: string[], success: boolean) => void,
  ): void {
    this._dialogActionHandler(
      selectedIds,
      `${i18n.t('Buttons.doYouWantToDelete')} ${
        selectedIds.length === 1 ? i18n.t('Buttons.thisPipeline') : i18n.t('Buttons.thesePipelines')
      }? ${i18n.t('Buttons.thisActionCannotBeUndone')}`,
      useCurrentResource,
      id => Apis.pipelineServiceApi.deletePipeline(id),
      callback,
      'Delete',
      'pipeline',
    );
  }

  private _deletePipelineVersion(
    selectedIds: string[],
    useCurrentResource: boolean,
    callback: (selectedIds: string[], success: boolean) => void,
  ): void {
    this._dialogActionHandler(
      selectedIds,
      `${i18n.t('Buttons.doYouWantToDelete')} ${
        selectedIds.length === 1 ? i18n.t('Buttons.thisPipelineVersion') : i18n.t('Buttons.thesePipelineVersions')
      }? ${i18n.t('Buttons.thisActionCannotBeUndone')}`,
      useCurrentResource,
      id => Apis.pipelineServiceApi.deletePipelineVersion(id),
      callback,
      'Delete',
      'pipeline version',
    );
  }

  private _deleteRecurringRun(
    id: string,
    useCurrentResource: boolean,
    callback: (_: string[], success: boolean) => void,
  ): void {
    this._dialogActionHandler(
      [id],
        i18n.t('Buttons.doYouWantToDeleteThisRecurringRunConfigThisActionCannotBeUndone'),
      useCurrentResource,
      jobId => Apis.jobServiceApi.deleteJob(jobId),
      callback,
      'Delete',
      'recurring run config',
    );
  }

  private _terminateRun(
    ids: string[],
    useCurrentResource: boolean,
    callback: (_: string[], success: boolean) => void,
  ): void {
    this._dialogActionHandler(
      ids,
        i18n.t('Buttons.doYouWantToTerminateThisRunThisActionCannotBeUndoneThisWillTerminateAny') +
        i18n.t('Buttons.runningPodsButTheyWillNotBeDeleted'),
      useCurrentResource,
      id => Apis.runServiceApi.terminateRun(id),
      callback,
      'Terminate',
      'run',
    );
  }

  private _deleteRun(
    ids: string[],
    useCurrentResource: boolean,
    callback: (_: string[], success: boolean) => void,
  ): void {
    this._dialogActionHandler(
      ids,
        i18n.t('Buttons.doYouWantToDeleteTheSelectedRunsThisActionCannotBeUndone'),
      useCurrentResource,
      id => Apis.runServiceApi.deleteRun(id),
      callback,
      'Delete',
      'run',
    );
  }

  private _dialogActionHandler(
    selectedIds: string[],
    content: string,
    useCurrentResource: boolean,
    api: (id: string) => Promise<void>,
    callback: (selectedIds: string[], success: boolean) => void,
    actionName: string,
    resourceName: string,
  ): void {
    const dialogClosedHandler = (confirmed: boolean) =>
      this._dialogClosed(
        confirmed,
        selectedIds,
        actionName,
        resourceName,
        useCurrentResource,
        api,
        callback,
      );

    this._props.updateDialog({
      buttons: [
        {
          onClick: async () => await dialogClosedHandler(false),
          text: i18n.t('Buttons.cancel'),
        },
        {
          onClick: async () => await dialogClosedHandler(true),
          text: actionName,
        },
      ],
      content,
      onClose: async () => await dialogClosedHandler(false),
      title: `${actionName} ${useCurrentResource ? 'this' : selectedIds.length} ${resourceName}${
        useCurrentResource ? '' : s(selectedIds.length)
      }?`,
    });
  }

  private async _dialogClosed(
    confirmed: boolean,
    selectedIds: string[],
    actionName: string,
    resourceName: string,
    useCurrentResource: boolean,
    api: (id: string) => Promise<void>,
    callback: (selectedIds: string[], success: boolean) => void,
  ): Promise<void> {
    if (confirmed) {
      const unsuccessfulIds: string[] = [];
      const errorMessages: string[] = [];
      await Promise.all(
        selectedIds.map(async id => {
          try {
            await api(id);
          } catch (err) {
            unsuccessfulIds.push(id);
            const errorMessage = await errorToMessage(err);
            errorMessages.push(
              `${i18n.t('Buttons.failedTo')} ${actionName.toLowerCase()} ${resourceName}: ${id} ${i18n.t('Buttons.withError')}: "${errorMessage}"`,
            );
          }
        }),
      );

      const successfulOps = selectedIds.length - unsuccessfulIds.length;
      if (successfulOps > 0) {
        this._props.updateSnackbar({
          message: `${actionName} ${i18n.t('Buttons.succeededFor')} ${
            useCurrentResource ? 'this' : successfulOps
          } ${resourceName}${useCurrentResource ? '' : s(successfulOps)}`,
          open: true,
        });
        if (!useCurrentResource) {
          this._refresh();
        }
      }

      if (unsuccessfulIds.length > 0) {
        this._props.updateDialog({
          buttons: [{ text: i18n.t('Buttons.dismiss') }],
          content: errorMessages.join('\n\n'),
          title: `${i18n.t('Buttons.failedTo')} ${actionName.toLowerCase()} ${
            useCurrentResource ? '' : unsuccessfulIds.length + ' '
          }${resourceName}${useCurrentResource ? '' : s(unsuccessfulIds)}`,
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
    const searchString = pipelineId
      ? this._urlParser.build({
          [QUERY_PARAMS.pipelineId]: pipelineId,
        })
      : '';
    this._props.history.push(RoutePage.NEW_EXPERIMENT + searchString);
  }

  private _createNewRun(isRecurring: boolean, experimentId?: string): void {
    const searchString = this._urlParser.build({
      [QUERY_PARAMS.experimentId]: experimentId || '',
      ...(isRecurring ? { [QUERY_PARAMS.isRecurring]: '1' } : {}),
    });
    this._props.history.push(RoutePage.NEW_RUN + searchString);
  }

  private _createNewRunFromPipelineVersion(pipelineId?: string, pipelineVersionId?: string): void {
    let searchString = '';
    const fromRunId = this._urlParser.get(QUERY_PARAMS.fromRunId);

    if (fromRunId) {
      searchString = this._urlParser.build(Object.assign({ [QUERY_PARAMS.fromRunId]: fromRunId }));
    } else {
      searchString = this._urlParser.build({
        [QUERY_PARAMS.pipelineId]: pipelineId || '',
        [QUERY_PARAMS.pipelineVersionId]: pipelineVersionId || '',
      });
    }

    this._props.history.push(RoutePage.NEW_RUN + searchString);
  }

  private async _setRecurringRunEnabledState(id: string, enabled: boolean): Promise<void> {
    if (id) {
      const toolbarActions = this._props.toolbarProps.actions;

      // TODO(rileyjbauer): make sure this is working as expected
      const buttonKey = enabled
        ? ButtonKeys.ENABLE_RECURRING_RUN
        : ButtonKeys.DISABLE_RECURRING_RUN;

      toolbarActions[buttonKey].busy = true;
      this._props.updateToolbar({ actions: toolbarActions });
      try {
        await (enabled ? Apis.jobServiceApi.enableJob(id) : Apis.jobServiceApi.disableJob(id));
        this._refresh();
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        this._props.updateDialog({
          buttons: [{ text: 'Dismiss' }],
          content: errorMessage,
          title: `${i18n.t('Buttons.failedTo')} ${enabled ? i18n.t('Buttons.enable') : i18n.t('Buttons.disable')} ${i18n.t('Buttons.recurringRun')}}`,
        });
      } finally {
        toolbarActions[buttonKey].busy = false;
        this._props.updateToolbar({ actions: toolbarActions });
      }
    }
  }

  private _createNewPipelineVersion(pipelineId?: string): void {
    const searchString = pipelineId
      ? this._urlParser.build({
          [QUERY_PARAMS.pipelineId]: pipelineId,
        })
      : '';
    this._props.history.push(RoutePage.NEW_PIPELINE_VERSION + searchString);
  }

  private _dialogDeletePipelinesAndPipelineVersions(
    selectedIds: string[],
    selectedVersionIds: { [pipelineId: string]: string[] },
    callback: (pipelineId: string | undefined, selectedIds: string[]) => void,
  ): void {
    const numVersionIds = this._deepCountDictionary(selectedVersionIds);
    const pipelineMessage = this._nouns(selectedIds.length, `pipeline`, `pipelines`);
    const pipelineVersionMessage = this._nouns(
      numVersionIds,
      `pipeline version`,
      `pipeline versions`,
    );
    const andMessage = pipelineMessage !== `` && pipelineVersionMessage !== `` ? ` and ` : ``;
    this._props.updateDialog({
      buttons: [
        {
          onClick: async () =>
            await this._deletePipelinesAndPipelineVersions(
              false,
              selectedIds,
              selectedVersionIds,
              callback,
            ),
          text: i18n.t('Buttons.cancel'),
        },
        {
          onClick: async () =>
            await this._deletePipelinesAndPipelineVersions(
              true,
              selectedIds,
              selectedVersionIds,
              callback,
            ),
          text: i18n.t('Buttons.delete'),
        },
      ],
      onClose: async () =>
        await this._deletePipelinesAndPipelineVersions(
          false,
          selectedIds,
          selectedVersionIds,
          callback,
        ),
      title: `${i18n.t('Buttons.delete')} ` + pipelineMessage + andMessage + pipelineVersionMessage + `?`,
    });
  }

  private async _deletePipelinesAndPipelineVersions(
    confirmed: boolean,
    selectedIds: string[],
    selectedVersionIds: { [pipelineId: string]: string[] },
    callback: (pipelineId: string | undefined, selectedIds: string[]) => void,
  ): Promise<void> {
    if (!confirmed) {
      return;
    }

    // Since confirmed, delete pipelines first and then pipeline versions from
    // (other) pipelines.

    // Delete pipelines.
    const succeededfulIds: Set<string> = new Set<string>(selectedIds);
    const unsuccessfulIds: string[] = [];
    const errorMessages: string[] = [];
    await Promise.all(
      selectedIds.map(async id => {
        try {
          await Apis.pipelineServiceApi.deletePipeline(id);
        } catch (err) {
          unsuccessfulIds.push(id);
          succeededfulIds.delete(id);
          const errorMessage = await errorToMessage(err);
          errorMessages.push(`${i18n.t('Buttons.failedToDeletePipeline')}: ${id} ${i18n.t('Buttons.withError')}: "${errorMessage}"`);
        }
      }),
    );

    // Remove successfully deleted pipelines from selectedVersionIds if exists.
    const toBeDeletedVersionIds = Object.fromEntries(
      Object.entries(selectedVersionIds).filter(
        ([pipelineId, _]) => !succeededfulIds.has(pipelineId),
      ),
    );

    // Delete pipeline versions.
    const unsuccessfulVersionIds: { [pipelineId: string]: string[] } = {};
    await Promise.all(
      // TODO: fix the no no return value bug
      // eslint-disable-next-line array-callback-return
      Object.keys(toBeDeletedVersionIds).map(pipelineId => {
        toBeDeletedVersionIds[pipelineId].map(async versionId => {
          try {
            unsuccessfulVersionIds[pipelineId] = [];
            await Apis.pipelineServiceApi.deletePipelineVersion(versionId);
          } catch (err) {
            unsuccessfulVersionIds[pipelineId].push(versionId);
            const errorMessage = await errorToMessage(err);
            errorMessages.push(
              `${i18n.t('Buttons.failedToDeletePipelineVersion')}: ${versionId} ${i18n.t('Buttons.withError')}: "${errorMessage}"`,
            );
          }
        });
      }),
    );
    const selectedVersionIdsCt = this._deepCountDictionary(selectedVersionIds);
    const unsuccessfulVersionIdsCt = this._deepCountDictionary(unsuccessfulVersionIds);

    // Display successful and/or unsuccessful messages.
    const pipelineMessage = this._nouns(succeededfulIds.size, `pipeline`, `pipelines`);
    const pipelineVersionMessage = this._nouns(
      selectedVersionIdsCt - unsuccessfulVersionIdsCt,
      `pipeline version`,
      `pipeline versions`,
    );
    const andMessage = pipelineMessage !== `` && pipelineVersionMessage !== `` ? ` and ` : ``;
    if (pipelineMessage !== `` || pipelineVersionMessage !== ``) {
      this._props.updateSnackbar({
        message: `${i18n.t('Buttons.deletionSucceededFor')} ` + pipelineMessage + andMessage + pipelineVersionMessage,
        open: true,
      });
    }
    if (unsuccessfulIds.length > 0 || unsuccessfulVersionIdsCt > 0) {
      this._props.updateDialog({
        buttons: [{ text: 'Dismiss' }],
        content: errorMessages.join('\n\n'),
        title: `${i18n.t('Buttons.failedToDeleteSomePipelinesAndOrSomePipelineVersions')}`,
      });
    }

    // pipelines and pipeline versions that failed deletion will keep to be
    // checked.
    callback(undefined, unsuccessfulIds);
    Object.keys(selectedVersionIds).map(pipelineId =>
      callback(pipelineId, unsuccessfulVersionIds[pipelineId]),
    );

    // Refresh
    this._refresh();
  }

  private _nouns(count: number, singularNoun: string, pluralNoun: string): string {
    if (count <= 0) {
      return ``;
    } else if (count === 1) {
      return `${count} ` + singularNoun;
    } else {
      return `${count} ` + pluralNoun;
    }
  }

  private _deepCountDictionary(dict: { [pipelineId: string]: string[] }): number {
    return Object.keys(dict).reduce((count, pipelineId) => count + dict[pipelineId].length, 0);
  }

  private _archiveExperiments(
    selectedIds: string[],
    useCurrent: boolean,
    callback: (selectedIds: string[], success: boolean) => void,
  ): void {
    this._dialogActionHandler(
      selectedIds,
      `Experiment${s(selectedIds)} ${i18n.t('Buttons.willBeMovedToTheArchiveSectionWhereYouCanStillView')}${
        selectedIds.length === 1 ? i18n.t('Buttons.its') : i18n.t('Buttons.their')
      } ${i18n.t('Buttons.detailsAllRunsInThisArchivedExperiment')}${s(
        selectedIds,
      )} ${i18n.t('Buttons.to')} ${selectedIds.length === 1 ? i18n.t('Buttons.its') : i18n.t('Buttons.their')} ${i18n.t('Buttons.originalLocation')}.`,
      useCurrent,
      id => Apis.experimentServiceApi.archiveExperiment(id),
      callback,
      'Archive',
      'experiment',
    );
  }
}
