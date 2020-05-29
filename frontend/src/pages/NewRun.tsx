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
import BusyButton from '../atoms/BusyButton';
import Button from '@material-ui/core/Button';
import Buttons from '../lib/Buttons';
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogContent from '@material-ui/core/DialogContent';
import FormControlLabel from '@material-ui/core/FormControlLabel';
import Input from '../atoms/Input';
import InputAdornment from '@material-ui/core/InputAdornment';
import NewRunParameters from '../components/NewRunParameters';
import Radio from '@material-ui/core/Radio';
import ResourceSelector from './ResourceSelector';
import RunUtils from '../lib/RunUtils';
import { TextFieldProps } from '@material-ui/core/TextField';
import Trigger from '../components/Trigger';
import { ApiExperiment, ExperimentStorageState } from '../apis/experiment';
import { ApiPipeline, ApiParameter, ApiPipelineVersion } from '../apis/pipeline';
import {
  ApiRun,
  ApiResourceReference,
  ApiRelationship,
  ApiResourceType,
  ApiRunDetail,
  ApiPipelineRuntime,
} from '../apis/run';
import { ApiTrigger, ApiJob } from '../apis/job';
import { Apis, PipelineSortKeys, PipelineVersionSortKeys, ExperimentSortKeys } from '../lib/Apis';
import { Link } from 'react-router-dom';
import { Page, PageProps } from './Page';
import { RoutePage, RouteParams, QUERY_PARAMS } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { URLParser } from '../lib/URLParser';
import { Workflow } from '../../../frontend/third_party/argo-ui/argo_template';
import { classes, stylesheet } from 'typestyle';
import { commonCss, padding, color } from '../Css';
import { logger, errorToMessage } from '../lib/Utils';
import UploadPipelineDialog, { ImportMethod } from '../components/UploadPipelineDialog';
import { CustomRendererProps } from '../components/CustomTable';
import { Description } from '../components/Description';
import { NamespaceContext } from '../lib/KubeflowClient';
import { NameWithTooltip } from '../components/CustomTableNameColumn';
import { PredicateOp, ApiFilter } from '../apis/filter';
import { HelpButton } from 'src/atoms/HelpButton';
import { ExternalLink } from 'src/atoms/ExternalLink';

interface NewRunState {
  description: string;
  errorMessage: string;
  experiment?: ApiExperiment;
  experimentName: string;
  serviceAccount: string;
  experimentSelectorOpen: boolean;
  isBeingStarted: boolean;
  isClone: boolean;
  isFirstRunInExperiment: boolean;
  isRecurringRun: boolean;
  maxConcurrentRuns?: string;
  catchup: boolean;
  parameters: ApiParameter[];
  pipeline?: ApiPipeline;
  pipelineVersion?: ApiPipelineVersion;
  // This represents a pipeline from a run that is being cloned, or if a user is creating a run from
  // a pipeline that was not uploaded to the system (as in the case of runs created from notebooks).
  workflowFromRun?: Workflow;
  // TODO: this is only here to properly display the name in the text field.
  // There is definitely a way to do this that doesn't necessitate this being in state.
  // Note: this cannot be undefined/optional or the label animation for the input field will not
  // work properly.
  pipelineName: string;
  pipelineVersionName: string;
  pipelineSelectorOpen: boolean;
  pipelineVersionSelectorOpen: boolean;
  runName: string;
  trigger?: ApiTrigger;
  unconfirmedSelectedExperiment?: ApiExperiment;
  unconfirmedSelectedPipeline?: ApiPipeline;
  unconfirmedSelectedPipelineVersion?: ApiPipelineVersion;
  useWorkflowFromRun: boolean;
  uploadDialogOpen: boolean;
  usePipelineFromRunLabel: string;
}

const css = stylesheet({
  nonEditableInput: {
    color: color.secondaryText,
  },
  selectorDialog: {
    // If screen is small, use calc(100% - 120px). If screen is big, use 1200px.
    maxWidth: 1200, // override default maxWidth to expand this dialog further
    minWidth: 680,
    width: 'calc(100% - 120px)',
  },
});

const descriptionCustomRenderer: React.FC<CustomRendererProps<string>> = props => {
  return <Description description={props.value || ''} forceInline={true} />;
};

export class NewRun extends Page<{ namespace?: string }, NewRunState> {
  public state: NewRunState = {
    catchup: true,
    description: '',
    errorMessage: '',
    experimentName: '',
    serviceAccount: '',
    experimentSelectorOpen: false,
    isBeingStarted: false,
    isClone: false,
    isFirstRunInExperiment: false,
    isRecurringRun: false,
    parameters: [],
    pipelineName: '',
    pipelineSelectorOpen: false,
    pipelineVersionName: '',
    pipelineVersionSelectorOpen: false,
    runName: '',
    uploadDialogOpen: false,
    usePipelineFromRunLabel: 'Using pipeline from cloned run',
    useWorkflowFromRun: false,
  };

  private pipelineSelectorColumns = [
    {
      customRenderer: NameWithTooltip,
      flex: 1,
      label: 'Pipeline name',
      sortKey: PipelineSortKeys.NAME,
    },
    { label: 'Description', flex: 2, customRenderer: descriptionCustomRenderer },
    { label: 'Uploaded on', flex: 1, sortKey: PipelineSortKeys.CREATED_AT },
  ];

  private pipelineVersionSelectorColumns = [
    {
      customRenderer: NameWithTooltip,
      flex: 2,
      label: 'Version name',
      sortKey: PipelineVersionSortKeys.NAME,
    },
    // TODO(jingzhang36): version doesn't have description field; remove it and
    // fix the rendering.
    { label: 'Description', flex: 1, customRenderer: descriptionCustomRenderer },
    { label: 'Uploaded on', flex: 1, sortKey: PipelineVersionSortKeys.CREATED_AT },
  ];

  private experimentSelectorColumns = [
    {
      customRenderer: NameWithTooltip,
      flex: 1,
      label: 'Experiment name',
      sortKey: ExperimentSortKeys.NAME,
    },
    { label: 'Description', flex: 2 },
    { label: 'Created at', flex: 1, sortKey: ExperimentSortKeys.CREATED_AT },
  ];

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: {},
      breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
      pageTitle: 'Start a new run',
    };
  }

  public render(): JSX.Element {
    const {
      workflowFromRun,
      description,
      errorMessage,
      experimentName,
      serviceAccount,
      experimentSelectorOpen,
      isClone,
      isFirstRunInExperiment,
      isRecurringRun,
      parameters,
      pipelineName,
      pipelineVersionName,
      pipelineSelectorOpen,
      pipelineVersionSelectorOpen,
      runName,
      unconfirmedSelectedExperiment,
      unconfirmedSelectedPipeline,
      unconfirmedSelectedPipelineVersion,
      usePipelineFromRunLabel,
      useWorkflowFromRun,
    } = this.state;

    const urlParser = new URLParser(this.props);
    const originalRunId =
      urlParser.get(QUERY_PARAMS.cloneFromRun) || urlParser.get(QUERY_PARAMS.fromRunId);
    const pipelineDetailsUrl = originalRunId
      ? RoutePage.PIPELINE_DETAILS.replace(
          ':' + RouteParams.pipelineId + '/version/:' + RouteParams.pipelineVersionId + '?',
          '',
        ) + urlParser.build({ [QUERY_PARAMS.fromRunId]: originalRunId })
      : '';

    const buttons = new Buttons(this.props, this.refresh.bind(this));

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <div className={commonCss.scrollContainer}>
          <div className={commonCss.header}>Run details</div>

          {/* Pipeline selection */}
          {!!workflowFromRun && (
            <div>
              <span>{usePipelineFromRunLabel}</span>
              {!!originalRunId && <Link to={pipelineDetailsUrl}>[View pipeline]</Link>}
            </div>
          )}
          {!useWorkflowFromRun && (
            <Input
              value={pipelineName}
              required={true}
              label='Pipeline'
              disabled={true}
              variant='outlined'
              InputProps={{
                classes: { disabled: css.nonEditableInput },
                endAdornment: (
                  <InputAdornment position='end'>
                    <Button
                      color='secondary'
                      id='choosePipelineBtn'
                      onClick={() => this.setStateSafe({ pipelineSelectorOpen: true })}
                      style={{ padding: '3px 5px', margin: 0 }}
                    >
                      Choose
                    </Button>
                  </InputAdornment>
                ),
                readOnly: true,
              }}
            />
          )}
          {!useWorkflowFromRun && (
            <Input
              value={pipelineVersionName}
              required={true}
              label='Pipeline Version'
              disabled={true}
              variant='outlined'
              InputProps={{
                classes: { disabled: css.nonEditableInput },
                endAdornment: (
                  <InputAdornment position='end'>
                    <Button
                      color='secondary'
                      id='choosePipelineVersionBtn'
                      onClick={() => this.setStateSafe({ pipelineVersionSelectorOpen: true })}
                      style={{ padding: '3px 5px', margin: 0 }}
                    >
                      Choose
                    </Button>
                  </InputAdornment>
                ),
                readOnly: true,
              }}
            />
          )}

          {/* Pipeline selector dialog */}
          <Dialog
            open={pipelineSelectorOpen}
            classes={{ paper: css.selectorDialog }}
            onClose={() => this._pipelineSelectorClosed(false)}
            PaperProps={{ id: 'pipelineSelectorDialog' }}
          >
            <DialogContent>
              <ResourceSelector
                {...this.props}
                title='Choose a pipeline'
                filterLabel='Filter pipelines'
                listApi={async (...args) => {
                  const response = await Apis.pipelineServiceApi.listPipelines(...args);
                  return {
                    nextPageToken: response.next_page_token || '',
                    resources: response.pipelines || [],
                  };
                }}
                columns={this.pipelineSelectorColumns}
                emptyMessage='No pipelines found. Upload a pipeline and then try again.'
                initialSortColumn={PipelineSortKeys.CREATED_AT}
                selectionChanged={(selectedPipeline: ApiPipeline) =>
                  this.setStateSafe({ unconfirmedSelectedPipeline: selectedPipeline })
                }
                toolbarActionMap={buttons
                  .upload(() =>
                    this.setStateSafe({ pipelineSelectorOpen: false, uploadDialogOpen: true }),
                  )
                  .getToolbarActionMap()}
              />
            </DialogContent>
            <DialogActions>
              <Button
                id='cancelPipelineSelectionBtn'
                onClick={() => this._pipelineSelectorClosed(false)}
                color='secondary'
              >
                Cancel
              </Button>
              <Button
                id='usePipelineBtn'
                onClick={() => this._pipelineSelectorClosed(true)}
                color='secondary'
                disabled={!unconfirmedSelectedPipeline}
              >
                Use this pipeline
              </Button>
            </DialogActions>
          </Dialog>

          {/* Pipeline version selector dialog */}
          <Dialog
            open={pipelineVersionSelectorOpen}
            classes={{ paper: css.selectorDialog }}
            onClose={() => this._pipelineVersionSelectorClosed(false)}
            PaperProps={{ id: 'pipelineVersionSelectorDialog' }}
          >
            <DialogContent>
              <ResourceSelector
                {...this.props}
                title='Choose a pipeline version'
                filterLabel='Filter pipeline versions'
                listApi={async (...args) => {
                  const response = await Apis.pipelineServiceApi.listPipelineVersions(
                    'PIPELINE',
                    this.state.pipeline ? this.state.pipeline!.id! : '',
                    args[1] /* page size */,
                    args[0] /* page token*/,
                    args[2] /* sort by */,
                    args[3] /* filter */,
                  );
                  return {
                    nextPageToken: response.next_page_token || '',
                    resources: response.versions || [],
                  };
                }}
                columns={this.pipelineVersionSelectorColumns}
                emptyMessage='No pipeline versions found. Select or upload a pipeline then try again.'
                initialSortColumn={PipelineVersionSortKeys.CREATED_AT}
                selectionChanged={(selectedPipelineVersion: ApiPipelineVersion) =>
                  this.setStateSafe({ unconfirmedSelectedPipelineVersion: selectedPipelineVersion })
                }
                toolbarActionMap={buttons
                  .upload(() =>
                    this.setStateSafe({
                      pipelineVersionSelectorOpen: false,
                      uploadDialogOpen: true,
                    }),
                  )
                  .getToolbarActionMap()}
              />
            </DialogContent>
            <DialogActions>
              <Button
                id='cancelPipelineVersionSelectionBtn'
                onClick={() => this._pipelineVersionSelectorClosed(false)}
                color='secondary'
              >
                Cancel
              </Button>
              <Button
                id='usePipelineVersionBtn'
                onClick={() => this._pipelineVersionSelectorClosed(true)}
                color='secondary'
                disabled={!unconfirmedSelectedPipelineVersion}
              >
                Use this pipeline version
              </Button>
            </DialogActions>
          </Dialog>

          <UploadPipelineDialog
            open={this.state.uploadDialogOpen}
            onClose={this._uploadDialogClosed.bind(this)}
          />

          {/* Experiment selector dialog */}
          <Dialog
            open={experimentSelectorOpen}
            classes={{ paper: css.selectorDialog }}
            onClose={() => this._experimentSelectorClosed(false)}
            PaperProps={{ id: 'experimentSelectorDialog' }}
          >
            <DialogContent>
              <ResourceSelector
                {...this.props}
                title='Choose an experiment'
                filterLabel='Filter experiments'
                listApi={async (
                  page_token?: string,
                  page_size?: number,
                  sort_by?: string,
                  filter?: string,
                ) => {
                  // A new run can only be created in an unarchived experiment.
                  // Therefore, when listing experiments here for selection, we
                  // only list unarchived experiments.
                  const new_filter = JSON.parse(
                    decodeURIComponent(filter || '{"predicates": []}'),
                  ) as ApiFilter;
                  new_filter.predicates = (new_filter.predicates || []).concat([
                    {
                      key: 'storage_state',
                      op: PredicateOp.NOTEQUALS,
                      string_value: ExperimentStorageState.ARCHIVED.toString(),
                    },
                  ]);
                  const response = await Apis.experimentServiceApi.listExperiment(
                    page_token,
                    page_size,
                    sort_by,
                    encodeURIComponent(JSON.stringify(new_filter)),
                    this.props.namespace ? 'NAMESPACE' : undefined,
                    this.props.namespace,
                  );
                  return {
                    nextPageToken: response.next_page_token || '',
                    resources: response.experiments || [],
                  };
                }}
                columns={this.experimentSelectorColumns}
                emptyMessage='No experiments found. Create an experiment and then try again.'
                initialSortColumn={ExperimentSortKeys.CREATED_AT}
                selectionChanged={(selectedExperiment: ApiExperiment) =>
                  this.setStateSafe({ unconfirmedSelectedExperiment: selectedExperiment })
                }
              />
            </DialogContent>
            <DialogActions>
              <Button
                id='cancelExperimentSelectionBtn'
                onClick={() => this._experimentSelectorClosed(false)}
                color='secondary'
              >
                Cancel
              </Button>
              <Button
                id='useExperimentBtn'
                onClick={() => this._experimentSelectorClosed(true)}
                color='secondary'
                disabled={!unconfirmedSelectedExperiment}
              >
                Use this experiment
              </Button>
            </DialogActions>
          </Dialog>

          {/* Run metadata inputs */}
          <Input
            label={isRecurringRun ? 'Recurring run config name' : 'Run name'}
            required={true}
            onChange={this.handleChange('runName')}
            autoFocus={true}
            value={runName}
            variant='outlined'
          />
          <Input
            label='Description (optional)'
            multiline={true}
            onChange={this.handleChange('description')}
            value={description}
            variant='outlined'
          />

          {/* Experiment selection */}
          <div>This run will be associated with the following experiment</div>
          <Input
            value={experimentName}
            required={true}
            label='Experiment'
            disabled={true}
            variant='outlined'
            InputProps={{
              classes: { disabled: css.nonEditableInput },
              endAdornment: (
                <InputAdornment position='end'>
                  <Button
                    color='secondary'
                    id='chooseExperimentBtn'
                    onClick={() => this.setStateSafe({ experimentSelectorOpen: true })}
                    style={{ padding: '3px 5px', margin: 0 }}
                  >
                    Choose
                  </Button>
                </InputAdornment>
              ),
              readOnly: true,
            }}
          />

          <div>
            This run will use the following Kubernetes service account.{' '}
            <HelpButton
              helpText={
                <div>
                  Note, the service account needs{' '}
                  <ExternalLink href='https://github.com/argoproj/argo/blob/v2.3.0/docs/workflow-rbac.md'>
                    minimum permissions required by argo workflows
                  </ExternalLink>{' '}
                  and extra permissions the specific task requires.
                </div>
              }
            />
          </div>
          <Input
            value={serviceAccount}
            onChange={this.handleChange('serviceAccount')}
            label='Service Account (Optional)'
            variant='outlined'
          />

          {/* One-off/Recurring Run Type */}
          <div className={commonCss.header}>Run Type</div>
          {isClone && <span>{isRecurringRun ? 'Recurring' : 'One-off'}</span>}
          {!isClone && (
            <React.Fragment>
              <FormControlLabel
                id='oneOffToggle'
                label='One-off'
                control={<Radio color='primary' />}
                onChange={() => this._updateRecurringRunState(false)}
                checked={!isRecurringRun}
              />
              <FormControlLabel
                id='recurringToggle'
                label='Recurring'
                control={<Radio color='primary' />}
                onChange={() => this._updateRecurringRunState(true)}
                checked={isRecurringRun}
              />
            </React.Fragment>
          )}

          {/* Recurring run controls */}
          {isRecurringRun && (
            <React.Fragment>
              <div className={commonCss.header}>Run trigger</div>
              <div>Choose a method by which new runs will be triggered</div>

              <Trigger
                initialProps={{
                  trigger: this.state.trigger,
                  maxConcurrentRuns: this.state.maxConcurrentRuns,
                  catchup: this.state.catchup,
                }}
                onChange={({ trigger, maxConcurrentRuns, catchup }) =>
                  this.setStateSafe(
                    {
                      catchup,
                      maxConcurrentRuns,
                      trigger,
                    },
                    this._validate.bind(this),
                  )
                }
              />
            </React.Fragment>
          )}

          {/* Run parameters form */}
          <NewRunParameters
            initialParams={parameters}
            titleMessage={this._runParametersMessage()}
            handleParamChange={this._handleParamChange.bind(this)}
          />

          {/* Create/Cancel buttons */}
          <div className={classes(commonCss.flex, padding(20, 'tb'))}>
            <BusyButton
              id='startNewRunBtn'
              disabled={!!errorMessage}
              busy={this.state.isBeingStarted}
              className={commonCss.buttonAction}
              title='Start'
              onClick={this._start.bind(this)}
            />
            <Button
              id='exitNewRunPageBtn'
              onClick={() => {
                this.props.history.push(
                  !!this.state.experiment
                    ? RoutePage.EXPERIMENT_DETAILS.replace(
                        ':' + RouteParams.experimentId,
                        this.state.experiment.id!,
                      )
                    : RoutePage.RUNS,
                );
              }}
            >
              {isFirstRunInExperiment ? 'Skip this step' : 'Cancel'}
            </Button>
            <div className={classes(padding(20, 'r'))} style={{ color: 'red' }}>
              {errorMessage}
            </div>
            {this._areParametersMissing() && (
              <div id='missing-parameters-message' style={{ color: 'orange' }}>
                Some parameters are missing values
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }

  public async refresh(): Promise<void> {
    return this.load();
  }

  public async componentDidMount(): Promise<void> {
    return this.load();
  }

  public async load(): Promise<void> {
    this.clearBanner();
    const urlParser = new URLParser(this.props);
    let experimentId: string | null = urlParser.get(QUERY_PARAMS.experimentId);

    // Get clone run id from querystring if any
    const originalRunId = urlParser.get(QUERY_PARAMS.cloneFromRun);
    const originalRecurringRunId = urlParser.get(QUERY_PARAMS.cloneFromRecurringRun);
    // If we are not cloning from an existing run, we may have an embedded pipeline from a run from
    // a notebook. This is a somewhat hidden path that can be reached via the following steps:
    // 1. Create a pipeline and run it from a notebook
    // 2. Click [View Pipeline Version] for this run from one of the list pages
    //    (Now you will be viewing a pipeline details page for a pipeline version that hasn't been uploaded)
    // 3. Click Create run
    const embeddedRunId = urlParser.get(QUERY_PARAMS.fromRunId);
    if (originalRunId) {
      // If we are cloning a run, fetch the original
      try {
        const originalRun = await Apis.runServiceApi.getRun(originalRunId);
        await this._prepareFormFromClone(originalRun.run, originalRun.pipeline_runtime);
        // If the querystring did not contain an experiment ID, try to get one from the run.
        if (!experimentId) {
          experimentId = RunUtils.getFirstExperimentReferenceId(originalRun.run);
        }
      } catch (err) {
        await this.showPageError(`Error: failed to retrieve original run: ${originalRunId}.`, err);
        logger.error(`Failed to retrieve original run: ${originalRunId}`, err);
      }
    } else if (originalRecurringRunId) {
      // If we are cloning a recurring run, fetch the original
      try {
        const originalJob = await Apis.jobServiceApi.getJob(originalRecurringRunId);
        await this._prepareFormFromClone(originalJob);
        this.setStateSafe({
          trigger: originalJob.trigger,
          maxConcurrentRuns: originalJob.max_concurrency,
          catchup: !originalJob.no_catchup,
        });
        if (!experimentId) {
          experimentId = RunUtils.getFirstExperimentReferenceId(originalJob);
        }
      } catch (err) {
        await this.showPageError(
          `Error: failed to retrieve original recurring run: ${originalRunId}.`,
          err,
        );
        logger.error(`Failed to retrieve original recurring run: ${originalRunId}`, err);
      }
    } else if (embeddedRunId) {
      // If we create run from a workflow manifest that is acquried from an existing run.
      this._prepareFormFromEmbeddedPipeline(embeddedRunId);
    } else {
      // If we create a run from an existing pipeline version.
      // Get pipeline and pipeline version id from querystring if any
      const possiblePipelineId = urlParser.get(QUERY_PARAMS.pipelineId);
      if (possiblePipelineId) {
        try {
          const pipeline = await Apis.pipelineServiceApi.getPipeline(possiblePipelineId);
          this.setStateSafe({
            parameters: pipeline.parameters || [],
            pipeline,
            pipelineName: (pipeline && pipeline.name) || '',
          });
          const possiblePipelineVersionId =
            urlParser.get(QUERY_PARAMS.pipelineVersionId) ||
            (pipeline.default_version && pipeline.default_version.id);
          if (possiblePipelineVersionId) {
            try {
              const pipelineVersion = await Apis.pipelineServiceApi.getPipelineVersion(
                possiblePipelineVersionId,
              );
              this.setStateSafe({
                parameters: pipelineVersion.parameters || [],
                pipelineVersion,
                pipelineVersionName: (pipelineVersion && pipelineVersion.name) || '',
                runName: this._getRunNameFromPipelineVersion(
                  (pipelineVersion && pipelineVersion.name) || '',
                ),
              });
            } catch (err) {
              urlParser.clear(QUERY_PARAMS.pipelineVersionId);
              await this.showPageError(
                `Error: failed to retrieve pipeline version: ${possiblePipelineVersionId}.`,
                err,
              );
              logger.error(
                `Failed to retrieve pipeline version: ${possiblePipelineVersionId}`,
                err,
              );
            }
          } else {
            this.setStateSafe({
              runName: this._getRunNameFromPipelineVersion((pipeline && pipeline.name) || ''),
            });
          }
        } catch (err) {
          urlParser.clear(QUERY_PARAMS.pipelineId);
          await this.showPageError(
            `Error: failed to retrieve pipeline: ${possiblePipelineId}.`,
            err,
          );
          logger.error(`Failed to retrieve pipeline: ${possiblePipelineId}`, err);
        }
      }
    }

    let experiment: ApiExperiment | undefined;
    let experimentName = '';
    const breadcrumbs = [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }];
    if (experimentId) {
      try {
        experiment = await Apis.experimentServiceApi.getExperiment(experimentId);
        experimentName = experiment.name || '';
        breadcrumbs.push({
          displayName: experimentName!,
          href: RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, experimentId),
        });
      } catch (err) {
        await this.showPageError(
          `Error: failed to retrieve associated experiment: ${experimentId}.`,
          err,
        );
        logger.error(`Failed to retrieve associated experiment: ${experimentId}`, err);
      }
    }

    const isRecurringRun = urlParser.get(QUERY_PARAMS.isRecurring) === '1';
    const titleVerb = originalRunId ? 'Clone' : 'Start';
    const pageTitle = isRecurringRun ? `${titleVerb} a recurring run` : `${titleVerb} a run`;

    this.props.updateToolbar({ actions: this.props.toolbarProps.actions, breadcrumbs, pageTitle });

    this.setStateSafe({
      experiment,
      experimentName,
      isFirstRunInExperiment: urlParser.get(QUERY_PARAMS.firstRunInExperiment) === '1',
      isRecurringRun,
    });

    this._validate();
  }

  public handleChange = (name: string) => (event: any) => {
    const value = (event.target as TextFieldProps).value;
    this.setStateSafe({ [name]: value } as any, () => {
      this._validate();
    });
  };

  protected async _experimentSelectorClosed(confirmed: boolean): Promise<void> {
    let { experiment } = this.state;
    if (confirmed && this.state.unconfirmedSelectedExperiment) {
      experiment = this.state.unconfirmedSelectedExperiment;
    }

    this.setStateSafe({
      experiment,
      experimentName: (experiment && experiment.name) || '',
      experimentSelectorOpen: false,
    });
  }

  protected async _pipelineSelectorClosed(confirmed: boolean): Promise<void> {
    let { parameters, pipeline, pipelineVersion } = this.state;
    if (confirmed && this.state.unconfirmedSelectedPipeline) {
      pipeline = this.state.unconfirmedSelectedPipeline;
      // Get the default version of selected pipeline to auto-fill the version
      // input field.
      if (pipeline.default_version) {
        pipelineVersion = await Apis.pipelineServiceApi.getPipelineVersion(
          pipeline.default_version.id!,
        );
        parameters = pipelineVersion.parameters || [];
      }
    }

    this.setStateSafe(
      {
        parameters,
        pipeline,
        pipelineName: (pipeline && pipeline.name) || '',
        pipelineSelectorOpen: false,
        pipelineVersion,
        pipelineVersionName: (pipelineVersion && pipelineVersion.name) || '',
      },
      () => this._validate(),
    );
  }

  protected async _pipelineVersionSelectorClosed(confirmed: boolean): Promise<void> {
    let { parameters, pipelineVersion } = this.state;
    if (confirmed && this.state.unconfirmedSelectedPipelineVersion) {
      pipelineVersion = this.state.unconfirmedSelectedPipelineVersion;
      parameters = pipelineVersion.parameters || [];
    }

    this.setStateSafe(
      {
        parameters,
        pipelineVersion,
        pipelineVersionName: (pipelineVersion && pipelineVersion.name) || '',
        pipelineVersionSelectorOpen: false,
      },
      () => this._validate(),
    );
  }

  protected _updateRecurringRunState(isRecurringRun: boolean): void {
    this.props.updateToolbar({
      pageTitle: isRecurringRun ? 'Start a recurring run' : 'Start a new run',
    });
    this.setStateSafe({ isRecurringRun });
  }

  protected _handleParamChange(index: number, value: string): void {
    const { parameters } = this.state;
    parameters[index].value = value;
    this.setStateSafe({ parameters });
  }

  private async _uploadDialogClosed(
    confirmed: boolean,
    name: string,
    file: File | null,
    url: string,
    method: ImportMethod,
    description?: string,
  ): Promise<boolean> {
    if (
      !confirmed ||
      (method === ImportMethod.LOCAL && !file) ||
      (method === ImportMethod.URL && !url)
    ) {
      this.setStateSafe({ pipelineSelectorOpen: true, uploadDialogOpen: false });
      return false;
    }

    try {
      const uploadedPipeline =
        method === ImportMethod.LOCAL
          ? await Apis.uploadPipeline(name, description || '', file!)
          : await Apis.pipelineServiceApi.createPipeline({ name, url: { pipeline_url: url } });
      this.setStateSafe(
        {
          pipeline: uploadedPipeline,
          pipelineName: (uploadedPipeline && uploadedPipeline.name) || '',
          pipelineSelectorOpen: false,
          uploadDialogOpen: false,
        },
        () => this._validate(),
      );
      return true;
    } catch (err) {
      const errorMessage = await errorToMessage(err);
      this.showErrorDialog('Failed to upload pipeline', errorMessage);
      return false;
    }
  }

  private async _prepareFormFromEmbeddedPipeline(embeddedRunId: string): Promise<void> {
    let embeddedPipelineSpec: string | null;
    let runWithEmbeddedPipeline: ApiRunDetail;

    try {
      runWithEmbeddedPipeline = await Apis.runServiceApi.getRun(embeddedRunId);
      embeddedPipelineSpec = RunUtils.getWorkflowManifest(runWithEmbeddedPipeline.run);
    } catch (err) {
      await this.showPageError(
        `Error: failed to retrieve the specified run: ${embeddedRunId}.`,
        err,
      );
      logger.error(`Failed to retrieve the specified run: ${embeddedRunId}`, err);
      return;
    }

    if (!embeddedPipelineSpec) {
      await this.showPageError(
        `Error: somehow the run provided in the query params: ${embeddedRunId} had no embedded pipeline.`,
      );
      return;
    }

    try {
      const workflow: Workflow = JSON.parse(embeddedPipelineSpec);
      const parameters = RunUtils.getParametersFromRun(runWithEmbeddedPipeline);
      this.setStateSafe({
        parameters,
        usePipelineFromRunLabel: 'Using pipeline from previous page',
        useWorkflowFromRun: true,
        workflowFromRun: workflow,
      });
    } catch (err) {
      await this.showPageError(
        `Error: failed to parse the embedded pipeline's spec: ${embeddedPipelineSpec}.`,
        err,
      );
      logger.error(`Failed to parse the embedded pipeline's spec from run: ${embeddedRunId}`, err);
      return;
    }

    this._validate();
  }

  private async _prepareFormFromClone(
    originalRun?: ApiRun | ApiJob,
    runtime?: ApiPipelineRuntime,
  ): Promise<void> {
    if (!originalRun) {
      logger.error('Could not get cloned run details');
      return;
    }

    let pipeline: ApiPipeline | undefined;
    let pipelineVersion: ApiPipelineVersion | undefined;
    let workflowFromRun: Workflow | undefined;
    let useWorkflowFromRun = false;
    let usePipelineFromRunLabel = '';
    let name = '';
    let pipelineVersionName = '';
    const serviceAccount = originalRun.service_account || '';

    // Case 1: a legacy run refers to a pipeline without specifying version.
    const referencePipelineId = RunUtils.getPipelineId(originalRun);
    // Case 2: a run refers to a pipeline version.
    const referencePipelineVersionId = RunUtils.getPipelineVersionId(originalRun);
    // Case 3: a run whose pipeline (version) has not been uploaded, such as runs started from
    // the CLI or notebooks
    const embeddedPipelineSpec = RunUtils.getWorkflowManifest(originalRun);
    if (referencePipelineVersionId) {
      try {
        // TODO(jingzhang36): optimize this part to make only one api call.
        pipelineVersion = await Apis.pipelineServiceApi.getPipelineVersion(
          referencePipelineVersionId,
        );
        pipelineVersionName = pipelineVersion && pipelineVersion.name ? pipelineVersion.name : '';
        pipeline = await Apis.pipelineServiceApi.getPipeline(
          RunUtils.getPipelineIdFromApiPipelineVersion(pipelineVersion)!,
        );
        name = pipeline.name || '';
      } catch (err) {
        await this.showPageError(
          'Error: failed to find a pipeline version corresponding to that of the original run:' +
            ` ${originalRun.id}.`,
          err,
        );
        return;
      }
    } else if (referencePipelineId) {
      try {
        pipeline = await Apis.pipelineServiceApi.getPipeline(referencePipelineId);
        name = pipeline.name || '';
      } catch (err) {
        await this.showPageError(
          'Error: failed to find a pipeline corresponding to that of the original run:' +
            ` ${originalRun.id}.`,
          err,
        );
        return;
      }
    } else if (embeddedPipelineSpec) {
      try {
        workflowFromRun = JSON.parse(embeddedPipelineSpec);
        name = workflowFromRun!.metadata.name || '';
      } catch (err) {
        await this.showPageError("Error: failed to read the clone run's pipeline definition.", err);
        return;
      }
      useWorkflowFromRun = true;
      usePipelineFromRunLabel = 'Using pipeline from cloned run';
    } else {
      await this.showPageError("Could not find the cloned run's pipeline definition.");
      return;
    }

    if (!originalRun.pipeline_spec || !originalRun.pipeline_spec.workflow_manifest) {
      await this.showPageError(`Error: run ${originalRun.id} had no workflow manifest`);
      return;
    }

    const parameters = runtime
      ? await RunUtils.getParametersFromRuntime(runtime) // cloned from run
      : originalRun.pipeline_spec.parameters || []; // cloned from recurring run

    this.setStateSafe({
      isClone: true,
      parameters,
      pipeline,
      pipelineName: name,
      pipelineVersion,
      pipelineVersionName,
      runName: this._getCloneName(originalRun.name!),
      usePipelineFromRunLabel,
      useWorkflowFromRun,
      workflowFromRun,
      serviceAccount,
    });

    this._validate();
  }

  private _runParametersMessage(): string {
    if (this.state.pipeline || this.state.workflowFromRun) {
      if (this.state.parameters.length) {
        return 'Specify parameters required by the pipeline';
      } else {
        return 'This pipeline has no parameters';
      }
    }
    return 'Parameters will appear after you select a pipeline';
  }

  private _start(): void {
    if (!this.state.pipelineVersion && !this.state.workflowFromRun) {
      this.showErrorDialog('Run creation failed', 'Cannot start run without pipeline version');
      logger.error('Cannot start run without pipeline version');
      return;
    }
    const references: ApiResourceReference[] = [];
    if (this.state.experiment) {
      references.push({
        key: {
          id: this.state.experiment.id,
          type: ApiResourceType.EXPERIMENT,
        },
        relationship: ApiRelationship.OWNER,
      });
    }
    if (this.state.pipelineVersion) {
      references.push({
        key: {
          id: this.state.pipelineVersion!.id,
          type: ApiResourceType.PIPELINEVERSION,
        },
        relationship: ApiRelationship.CREATOR,
      });
    }

    let newRun: ApiRun | ApiJob = {
      description: this.state.description,
      name: this.state.runName,
      pipeline_spec: {
        parameters: (this.state.parameters || []).map(p => {
          p.value = (p.value || '').trim();
          return p;
        }),
        workflow_manifest: this.state.useWorkflowFromRun
          ? JSON.stringify(this.state.workflowFromRun)
          : undefined,
      },
      resource_references: references,
      service_account: this.state.serviceAccount,
    };
    if (this.state.isRecurringRun) {
      newRun = Object.assign(newRun, {
        enabled: true,
        max_concurrency: this.state.maxConcurrentRuns || '1',
        no_catchup: !this.state.catchup,
        trigger: this.state.trigger,
      });
    }

    this.setStateSafe({ isBeingStarted: true }, async () => {
      // TODO: there was previously a bug here where the await wasn't being applied to the API
      // calls, so a run creation could fail, and the success path would still be taken. We need
      // tests for this and other similar situations.
      try {
        this.state.isRecurringRun
          ? await Apis.jobServiceApi.createJob(newRun)
          : await Apis.runServiceApi.createRun(newRun);
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        this.showErrorDialog('Run creation failed', errorMessage);
        logger.error('Error creating Run:', err);
        return;
      } finally {
        this.setStateSafe({ isBeingStarted: false });
      }

      if (this.state.experiment) {
        this.props.history.push(
          RoutePage.EXPERIMENT_DETAILS.replace(
            ':' + RouteParams.experimentId,
            this.state.experiment.id!,
          ),
        );
      } else {
        this.props.history.push(RoutePage.RUNS);
      }
      this.props.updateSnackbar({
        message: `Successfully started new Run: ${newRun.name}`,
        open: true,
      });
    });
  }

  private _getCloneName(oldName: string): string {
    const numberRegex = /Clone(?: \(([0-9]*)\))? of (.*)/;
    const match = oldName.match(numberRegex);
    if (!match) {
      // No match, add Clone prefix
      return 'Clone of ' + oldName;
    } else {
      const cloneNumber = match[1] ? +match[1] : 1;
      return `Clone (${cloneNumber + 1}) of ${match[2]}`;
    }
  }

  private _generateRandomString(length: number): string {
    let d = 0;
    function randomChar(): string {
      const r = Math.trunc((d + Math.random() * 16) % 16);
      d = Math.floor(d / 16);
      return r.toString(16);
    }
    let str = '';
    for (let i = 0; i < length; ++i) {
      str += randomChar();
    }
    return str;
  }

  private _getRunNameFromPipelineVersion(pipelineVersionName: string): string {
    return 'Run of ' + pipelineVersionName + ' (' + this._generateRandomString(5) + ')';
  }

  private _validate(): void {
    // Validate state
    const { pipelineVersion, workflowFromRun, maxConcurrentRuns, runName, trigger } = this.state;
    try {
      if (!pipelineVersion && !workflowFromRun) {
        throw new Error('A pipeline version must be selected');
      }
      if (!runName) {
        throw new Error('Run name is required');
      }

      const hasTrigger = trigger && (!!trigger.cron_schedule || !!trigger.periodic_schedule);
      if (hasTrigger) {
        const startDate = !!trigger!.cron_schedule
          ? trigger!.cron_schedule!.start_time
          : trigger!.periodic_schedule!.start_time;
        const endDate = !!trigger!.cron_schedule
          ? trigger!.cron_schedule!.end_time
          : trigger!.periodic_schedule!.end_time;
        if (startDate && endDate && startDate > endDate) {
          throw new Error('End date/time cannot be earlier than start date/time');
        }
        const validMaxConcurrentRuns = (input: string) =>
          !isNaN(Number.parseInt(input, 10)) && +input > 0;

        if (maxConcurrentRuns !== undefined && !validMaxConcurrentRuns(maxConcurrentRuns)) {
          throw new Error('For triggered runs, maximum concurrent runs must be a positive number');
        }
      }

      this.setStateSafe({ errorMessage: '' });
    } catch (err) {
      this.setStateSafe({ errorMessage: err.message });
    }
  }

  private _areParametersMissing(): boolean {
    const { parameters } = this.state;
    return parameters.some(parameter => !parameter.value);
  }
}

const EnhancedNewRun: React.FC<PageProps> = props => {
  const namespace = React.useContext(NamespaceContext);
  return <NewRun {...props} namespace={namespace} />;
};

export default EnhancedNewRun;
