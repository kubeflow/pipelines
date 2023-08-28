/*
 * Copyright 2022 The Kubeflow Authors
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

import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  InputAdornment,
  FormControlLabel,
  Radio,
} from '@material-ui/core';
import React, { useEffect, useState } from 'react';
import * as JsYaml from 'js-yaml';
import { useMutation } from 'react-query';
import { Link } from 'react-router-dom';
import { V2beta1Experiment, V2beta1ExperimentStorageState } from 'src/apisv2beta1/experiment';
import { V2beta1Pipeline, V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import { V2beta1PipelineVersionReference, V2beta1Run } from 'src/apisv2beta1/run';
import { V2beta1Filter, V2beta1PredicateOperation } from 'src/apisv2beta1/filter';
import BusyButton from 'src/atoms/BusyButton';
import { ExternalLink } from 'src/atoms/ExternalLink';
import { HelpButton } from 'src/atoms/HelpButton';
import Input from 'src/atoms/Input';
import { CustomRendererProps } from 'src/components/CustomTable';
import { NameWithTooltip } from 'src/components/CustomTableNameColumn';
import { Description } from 'src/components/Description';
import NewRunParametersV2 from 'src/components/NewRunParametersV2';
import { QUERY_PARAMS, RoutePage, RouteParams } from 'src/components/Router';
import Trigger from 'src/components/Trigger';
import { color, commonCss, padding } from 'src/Css';
import { ComponentInputsSpec_ParameterSpec } from 'src/generated/pipeline_spec/pipeline_spec';
import { Apis, ExperimentSortKeys, PipelineSortKeys, PipelineVersionSortKeys } from 'src/lib/Apis';
import { URLParser } from 'src/lib/URLParser';
import { errorToMessage, generateRandomString, logger } from 'src/lib/Utils';
import { convertYamlToV2PipelineSpec } from 'src/lib/v2/WorkflowUtils';
import { classes, stylesheet } from 'typestyle';
import { PageProps } from './Page';
import PipelinesDialogV2 from 'src/components/PipelinesDialogV2';
import { V2beta1RecurringRun, RecurringRunMode } from 'src/apisv2beta1/recurringrun';
import ResourceSelector from 'src/pages/ResourceSelector';
import {
  convertExperimentToResource,
  convertPipelineVersionToResource,
} from 'src/lib/ResourceConverter';

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

interface RunV2Props {
  namespace?: string;
  existingRunId: string | null;
  existingRun?: V2beta1Run;
  existingRecurringRunId: string | null;
  existingRecurringRun?: V2beta1RecurringRun;
  existingPipeline?: V2beta1Pipeline;
  handlePipelineIdChange: (pipelineId: string) => void;
  existingPipelineVersion?: V2beta1PipelineVersion;
  handlePipelineVersionIdChange: (pipelineVersionId: string) => void;
  templateString?: string;
  chosenExperiment?: V2beta1Experiment;
}

type NewRunV2Props = RunV2Props & PageProps;

export type SpecParameters = { [key: string]: ComponentInputsSpec_ParameterSpec };
export type RuntimeParameters = { [key: string]: any };

type CloneOrigin = {
  isClone: boolean;
  isRecurring: boolean;
  run?: V2beta1Run;
  recurringRun?: V2beta1RecurringRun;
};

function getCloneOrigin(run?: V2beta1Run, recurringRun?: V2beta1RecurringRun) {
  let cloneOrigin: CloneOrigin = {
    isClone: run !== undefined || recurringRun !== undefined,
    isRecurring: recurringRun !== undefined,
    run: run,
    recurringRun: recurringRun,
  };
  return cloneOrigin;
}

function getPipelineDetailsUrl(
  props: NewRunV2Props,
  isRecurring: boolean,
  existingRunId: string | null,
  originalRecurringRunId: string | null,
): string {
  const urlParser = new URLParser(props);

  const pipelineDetailsUrlfromRun = existingRunId
    ? RoutePage.PIPELINE_DETAILS.replace(
        ':' + RouteParams.pipelineId + '/version/:' + RouteParams.pipelineVersionId + '?',
        '',
      ) + urlParser.build({ [QUERY_PARAMS.fromRunId]: existingRunId })
    : '';

  const pipelineDetailsUrlfromRecurringRun = originalRecurringRunId
    ? RoutePage.PIPELINE_DETAILS.replace(
        ':' + RouteParams.pipelineId + '/version/:' + RouteParams.pipelineVersionId + '?',
        '',
      ) + urlParser.build({ [QUERY_PARAMS.fromRecurringRunId]: originalRecurringRunId })
    : '';

  return isRecurring ? pipelineDetailsUrlfromRecurringRun : pipelineDetailsUrlfromRun;
}

export async function getLatestVersion(pipelineId: string) {
  try {
    const listVersionsResponse = await Apis.pipelineServiceApiV2.listPipelineVersions(
      pipelineId,
      undefined,
      1, // Only need the latest one
      'created_at desc',
    );
    return listVersionsResponse.pipeline_versions
      ? listVersionsResponse.pipeline_versions[0]
      : undefined;
  } catch (err) {
    logger.error('Cannot retrieve pipeline version list.', err);
    return;
  }
}

function NewRunV2(props: NewRunV2Props) {
  // List of elements we need to create Pipeline Run.
  const {
    existingRunId,
    existingRun,
    existingRecurringRunId,
    existingRecurringRun,
    existingPipeline,
    handlePipelineIdChange,
    existingPipelineVersion,
    handlePipelineVersionIdChange,
    templateString,
    chosenExperiment,
  } = props;
  const cloneOrigin = getCloneOrigin(existingRun, existingRecurringRun);
  const urlParser = new URLParser(props);
  const [runName, setRunName] = useState('');
  const [runDescription, setRunDescription] = useState('');
  const [pipelineName, setPipelineName] = useState('');
  const [pipelineVersionName, setPipelineVersionName] = useState('');
  const [experimentId, setExperimentId] = useState('');
  const [experiment, setExperiment] = useState(chosenExperiment);
  const [experimentName, setExperimentName] = useState('');
  const [serviceAccount, setServiceAccount] = useState('');
  const [specParameters, setSpecParameters] = useState<SpecParameters>({});
  const [runtimeParameters, setRuntimeParameters] = useState<RuntimeParameters>({});
  const [pipelineRoot, setPipelineRoot] = useState<string>();
  const [isStartButtonEnabled, setIsStartButtonEnabled] = useState(false);
  const [isStartingNewRun, setIsStartingNewRun] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [isParameterValid, setIsParameterValid] = useState(false);
  const [isRecurringRun, setIsRecurringRun] = useState(
    urlParser.get(QUERY_PARAMS.isRecurring) === '1' || cloneOrigin.isRecurring,
  );
  const initialTrigger = cloneOrigin.recurringRun?.trigger
    ? cloneOrigin.recurringRun.trigger
    : undefined;
  const [trigger, setTrigger] = useState(initialTrigger);
  const initialMaxConCurrentRuns =
    cloneOrigin.recurringRun?.max_concurrency !== undefined
      ? cloneOrigin.recurringRun.max_concurrency
      : '10';
  const [maxConcurrentRuns, setMaxConcurrentRuns] = useState(initialMaxConCurrentRuns);
  const [isMaxConcurrentRunValid, setIsMaxConcurrentRunValid] = useState(true);
  const initialCatchup =
    cloneOrigin.recurringRun?.no_catchup !== undefined
      ? !cloneOrigin.recurringRun.no_catchup
      : true;
  const [needCatchup, setNeedCatchup] = useState(initialCatchup);

  const clonedRuntimeConfig = cloneOrigin.isRecurring
    ? cloneOrigin.recurringRun?.runtime_config
    : cloneOrigin.run?.runtime_config;
  const labelTextAdjective = isRecurringRun ? 'recurring ' : '';
  const usePipelineFromRunLabel = `Using pipeline from existing ${labelTextAdjective} run.`;

  const isTemplatePullSuccess = templateString ? true : false;

  const titleVerb = cloneOrigin.isClone ? 'Clone' : 'Start';
  const titleAdjective = cloneOrigin.isClone ? '' : 'new';

  // Pipeline version reference from selected pipeline (version) when "creating" run
  const pipelineVersionRefNew: V2beta1PipelineVersionReference | undefined = cloneOrigin.isClone
    ? undefined
    : {
        pipeline_id: existingPipeline?.pipeline_id,
        pipeline_version_id: existingPipelineVersion?.pipeline_version_id,
      };

  // Pipeline version reference from existing run or recurring run when "cloning" run
  const pipelineVersionRefClone = cloneOrigin.isRecurring
    ? cloneOrigin.recurringRun?.pipeline_version_reference
    : cloneOrigin.run?.pipeline_version_reference;

  // Title and list of actions on the top of page.
  useEffect(() => {
    props.updateToolbar({
      actions: {},
      pageTitle: isRecurringRun
        ? `${titleVerb} a recurring run`
        : `${titleVerb} a ${titleAdjective} run`,
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isRecurringRun]);

  // Pre-fill names for pipeline, pipeline version and experiment.
  useEffect(() => {
    if (existingPipeline?.display_name) {
      setPipelineName(existingPipeline.display_name);
    }
    if (existingPipelineVersion?.display_name) {
      setPipelineVersionName(existingPipelineVersion.display_name);
    }
    if (experiment?.display_name) {
      setExperimentName(experiment.display_name);
    }
    if (experiment?.experiment_id) {
      setExperimentId(experiment.experiment_id);
    }
  }, [existingPipeline, existingPipelineVersion, experiment]);

  // When loading a pipeline version, automatically set the default run name.
  useEffect(() => {
    if (existingRun?.display_name) {
      const cloneRunName = 'Clone of ' + existingRun.display_name;
      setRunName(cloneRunName);
    } else if (existingRecurringRun?.display_name) {
      const cloneRecurringName = 'Clone of ' + existingRecurringRun.display_name;
      setRunName(cloneRecurringName);
    } else if (existingPipelineVersion?.display_name) {
      const initRunName =
        'Run of ' + existingPipelineVersion.display_name + ' (' + generateRandomString(5) + ')';
      setRunName(initRunName);
    }
  }, [existingRun, existingRecurringRun, existingPipelineVersion]);

  // Set pipeline spec, pipeline root and parameters fields on UI based on returned template.
  useEffect(() => {
    if (!templateString) {
      return;
    }

    const spec = convertYamlToV2PipelineSpec(templateString);

    const params = spec.root?.inputDefinitions?.parameters;
    if (params) {
      setSpecParameters(params);
    } else {
      setSpecParameters({});
    }

    const defaultRoot = spec.defaultPipelineRoot;
    const clonedRoot = clonedRuntimeConfig?.pipeline_root;
    if (clonedRoot) {
      setPipelineRoot(clonedRoot);
    } else if (defaultRoot) {
      setPipelineRoot(defaultRoot);
    }
  }, [templateString, clonedRuntimeConfig]);

  // Handle different change that can affect setIsStartButtonEnabled
  useEffect(() => {
    if (!templateString || errorMessage || !isParameterValid || !isMaxConcurrentRunValid) {
      setIsStartButtonEnabled(false);
    } else {
      setIsStartButtonEnabled(true);
    }
  }, [templateString, errorMessage, isParameterValid, isMaxConcurrentRunValid]);

  // Whenever any input value changes, validate and show error if needed.
  useEffect(() => {
    if (isTemplatePullSuccess) {
      if (runName) {
        setErrorMessage('');
        return;
      } else {
        setErrorMessage('Run name can not be empty.');
        return;
      }
    } else {
      if (!existingPipeline) {
        setErrorMessage('A pipeline must be selected');
        return;
      }
      if (!existingPipelineVersion) {
        setErrorMessage('A pipeline version must be selected');
        return;
      }
    }
  }, [runName, existingPipeline, existingPipelineVersion, isTemplatePullSuccess]);

  // Defines the behavior when user clicks `Start` button.
  const newRunMutation = useMutation((run: V2beta1Run) => {
    return Apis.runServiceApiV2.createRun(run);
  });
  const newRecurringRunMutation = useMutation((recurringRun: V2beta1RecurringRun) => {
    return Apis.recurringRunServiceApi.createRecurringRun(recurringRun);
  });

  const startRun = () => {
    let newRun: V2beta1Run = {
      description: runDescription,
      display_name: runName,
      experiment_id: experiment?.experiment_id,
      // pipeline_spec and pipeline_version_reference is exclusive.
      pipeline_spec: !(pipelineVersionRefClone || pipelineVersionRefNew)
        ? JsYaml.safeLoad(templateString || '')
        : undefined,
      pipeline_version_reference:
        pipelineVersionRefClone || pipelineVersionRefNew
          ? cloneOrigin.isClone
            ? pipelineVersionRefClone
            : pipelineVersionRefNew
          : undefined,
      runtime_config: {
        pipeline_root: pipelineRoot,
        parameters: runtimeParameters,
      },
      service_account: serviceAccount,
    };

    let newRecurringRun: V2beta1RecurringRun = Object.assign(
      newRun,
      isRecurringRun
        ? {
            max_concurrency: maxConcurrentRuns || '1',
            no_catchup: !needCatchup,
            mode: RecurringRunMode.ENABLE,
            trigger: trigger,
          }
        : {
            max_concurrency: undefined,
            no_catchup: undefined,
            mode: undefined,
            trigger: undefined,
          },
    );
    setIsStartingNewRun(true);

    const runCreation = () =>
      newRunMutation.mutate(newRun, {
        onSuccess: data => {
          setIsStartingNewRun(false);
          if (data.run_id) {
            props.history.push(RoutePage.RUN_DETAILS.replace(':' + RouteParams.runId, data.run_id));
          } else {
            props.history.push(RoutePage.RUNS);
          }

          props.updateSnackbar({
            message: `Successfully started new Run: ${data.display_name}`,
            open: true,
          });
        },
        onError: async error => {
          const errorMessage = await errorToMessage(error);
          props.updateDialog({
            buttons: [{ text: 'Dismiss' }],
            onClose: () => setIsStartingNewRun(false),
            content: errorMessage,
            title: 'Run creation failed',
          });
        },
      });

    const recurringRunCreation = () =>
      newRecurringRunMutation.mutate(newRecurringRun, {
        onSuccess: data => {
          setIsStartingNewRun(false);
          if (data.recurring_run_id) {
            props.history.push(
              RoutePage.RECURRING_RUN_DETAILS.replace(
                ':' + RouteParams.recurringRunId,
                data.recurring_run_id,
              ),
            );
          } else {
            props.history.push(RoutePage.RECURRING_RUNS);
          }

          props.updateSnackbar({
            message: `Successfully started new recurring Run: ${data.display_name}`,
            open: true,
          });
        },
        onError: async error => {
          const errorMessage = await errorToMessage(error);
          props.updateDialog({
            buttons: [{ text: 'Dismiss' }],
            onClose: () => setIsStartingNewRun(false),
            content: errorMessage,
            title: 'Recurring run creation failed',
          });
        },
      });

    isRecurringRun ? recurringRunCreation() : runCreation();
  };

  return (
    <div className={classes(commonCss.page, padding(20, 'lr'))}>
      <div className={commonCss.scrollContainer}>
        <div className={commonCss.header}>Run details</div>

        {cloneOrigin.isClone && (
          <div>
            <div>
              <span>{usePipelineFromRunLabel}</span>
            </div>
            <div className={classes(padding(10, 't'))}>
              {/* TODO(jlyaoyuli): View pipelineDetails from existing recurring run*/}
              {cloneOrigin.isClone && (
                <Link
                  className={classes(commonCss.link)}
                  to={getPipelineDetailsUrl(
                    props,
                    cloneOrigin.isRecurring,
                    existingRunId,
                    existingRecurringRunId,
                  )}
                >
                  [View pipeline]
                </Link>
              )}
            </div>
          </div>
        )}

        {!cloneOrigin.isClone && (
          <div>
            {/* Pipeline selection */}
            <PipelineSelector
              {...props}
              pipelineName={pipelineName}
              handlePipelineChange={async updatedPipeline => {
                if (updatedPipeline.display_name) {
                  setPipelineName(updatedPipeline.display_name);
                }
                if (updatedPipeline.pipeline_id) {
                  const latestVersion = await getLatestVersion(updatedPipeline.pipeline_id);
                  const searchString = urlParser.build({
                    [QUERY_PARAMS.experimentId]: experimentId || '',
                    [QUERY_PARAMS.pipelineId]: updatedPipeline.pipeline_id || '',
                    [QUERY_PARAMS.pipelineVersionId]: latestVersion?.pipeline_version_id || '',
                    [QUERY_PARAMS.isRecurring]: isRecurringRun ? '1' : '',
                  });
                  props.history.replace(searchString);
                  handlePipelineVersionIdChange(latestVersion?.pipeline_version_id!);
                  handlePipelineIdChange(updatedPipeline.pipeline_id);
                }
              }}
            />

            {/* Pipeline version selection */}
            <PipelineVersionSelector
              {...props}
              pipeline={existingPipeline}
              pipelineVersionName={pipelineVersionName}
              handlePipelineVersionChange={updatedPipelineVersion => {
                if (updatedPipelineVersion.display_name) {
                  setPipelineVersionName(updatedPipelineVersion.display_name);
                }
                if (existingPipeline?.pipeline_id && updatedPipelineVersion.pipeline_version_id) {
                  const searchString = urlParser.build({
                    [QUERY_PARAMS.experimentId]: experimentId || '',
                    [QUERY_PARAMS.pipelineId]: existingPipeline.pipeline_id || '',
                    [QUERY_PARAMS.pipelineVersionId]:
                      updatedPipelineVersion.pipeline_version_id || '',
                    [QUERY_PARAMS.isRecurring]: isRecurringRun ? '1' : '',
                  });
                  props.history.replace(searchString);
                  handlePipelineVersionIdChange(updatedPipelineVersion.pipeline_version_id);
                }
              }}
            />
          </div>
        )}

        {/* Run info inputs */}
        <Input
          label={isRecurringRun ? 'Recurring run config name' : 'Run name'}
          required={true}
          onChange={event => setRunName(event.target.value)}
          autoFocus={true}
          value={runName}
          variant='outlined'
        />
        <Input
          label='Description'
          multiline={true}
          onChange={event => setRunDescription(event.target.value)}
          required={false}
          value={runDescription}
          variant='outlined'
        />

        {/* Experiment selection */}
        <div>This run will be associated with the following experiment</div>
        <ExperimentSelector
          {...props}
          experimentName={experimentName}
          handleExperimentChange={experiment => {
            setExperiment(experiment);
            if (experiment.display_name) {
              setExperimentName(experiment.display_name);
            }
            if (experiment.experiment_id) {
              setExperimentId(experiment.experiment_id);
              let searchString;
              if (existingPipeline?.pipeline_id && existingPipelineVersion?.pipeline_version_id) {
                searchString = urlParser.build({
                  [QUERY_PARAMS.experimentId]: experiment.experiment_id || '',
                  [QUERY_PARAMS.pipelineId]: existingPipeline.pipeline_id || '',
                  [QUERY_PARAMS.pipelineVersionId]:
                    existingPipelineVersion.pipeline_version_id || '',
                });
              } else if (existingRunId) {
                searchString = urlParser.build({
                  [QUERY_PARAMS.experimentId]: experiment.experiment_id || '',
                  [QUERY_PARAMS.cloneFromRun]: existingRunId || '',
                });
              } else if (existingRecurringRunId) {
                searchString = urlParser.build({
                  [QUERY_PARAMS.experimentId]: experiment.experiment_id || '',
                  [QUERY_PARAMS.cloneFromRecurringRun]: existingRecurringRunId || '',
                });
              } else {
                // Enter new run page from run list (none of pipeline is selected)
                searchString = urlParser.build({
                  [QUERY_PARAMS.experimentId]: experiment.experiment_id || '',
                });
              }
              props.history.replace(searchString);
            }
          }}
        />

        {/* Service account selection */}
        <div>
          This run will use the following Kubernetes service account.{' '}
          <HelpButton
            helpText={
              <div>
                Note, the service account needs{' '}
                <ExternalLink href='https://argoproj.github.io/argo-workflows/workflow-rbac/'>
                  minimum permissions required by argo workflows
                </ExternalLink>{' '}
                and extra permissions the specific task requires.
              </div>
            }
          />
        </div>
        <Input
          value={serviceAccount}
          onChange={event => setServiceAccount(event.target.value)}
          required={false}
          label='Service Account'
          variant='outlined'
        />

        {/* One-off/Recurring Run Type */}
        {/* TODO(zijianjoy): Support Recurring Run */}
        <div className={commonCss.header}>Run Type</div>
        {cloneOrigin.isClone === true && <span>{isRecurringRun ? 'Recurring' : 'One-off'}</span>}
        {cloneOrigin.isClone === false && (
          <>
            <FormControlLabel
              id='oneOffToggle'
              label='One-off'
              control={<Radio color='primary' />}
              onChange={() => setIsRecurringRun(false)}
              checked={!isRecurringRun}
            />
            <FormControlLabel
              id='recurringToggle'
              label='Recurring'
              control={<Radio color='primary' />}
              onChange={() => setIsRecurringRun(true)}
              checked={isRecurringRun}
            />
          </>
        )}

        {/* Recurring run controls */}
        {isRecurringRun && (
          <>
            <div className={commonCss.header}>Run trigger</div>
            <div>Choose a method by which new runs will be triggered</div>

            <Trigger
              initialProps={{
                trigger: trigger,
                maxConcurrentRuns: maxConcurrentRuns,
                catchup: needCatchup,
              }}
              onChange={({ trigger, maxConcurrentRuns, catchup }) => {
                setTrigger(trigger);
                setMaxConcurrentRuns(maxConcurrentRuns!);
                setIsMaxConcurrentRunValid(
                  Number.isInteger(Number(maxConcurrentRuns)) && Number(maxConcurrentRuns) > 0,
                );
                setNeedCatchup(catchup);
              }}
            />
          </>
        )}

        {/* PipelineRoot and Run Parameters */}
        <NewRunParametersV2
          pipelineRoot={pipelineRoot}
          handlePipelineRootChange={setPipelineRoot}
          titleMessage={
            existingPipeline || cloneOrigin.isClone
              ? Object.keys(specParameters).length
                ? 'Specify parameters required by the pipeline'
                : 'This pipeline has no parameters'
              : 'Parameters will appear after you select a pipeline'
          }
          specParameters={specParameters}
          clonedRuntimeConfig={clonedRuntimeConfig}
          handleParameterChange={setRuntimeParameters}
          setIsValidInput={setIsParameterValid}
        />

        {/* Create/Cancel buttons */}
        <div className={classes(commonCss.flex, padding(20, 'tb'))}>
          <BusyButton
            id='startNewRunBtn'
            disabled={!isStartButtonEnabled}
            busy={isStartingNewRun}
            className={commonCss.buttonAction}
            title='Start'
            onClick={startRun}
          />
          <Button
            id='exitNewRunPageBtn'
            onClick={() => {
              // TODO(zijianjoy): Return to previous page instead of defaulting to RUNS page.
              props.history.push(RoutePage.RUNS);
            }}
          >
            {'Cancel'}
          </Button>
          <div className={classes(padding(20, 'r'))} style={{ color: 'red' }}>
            {errorMessage}
          </div>
          {/* TODO(zijianjoy): Show error when custom pipelineRoot or parameters are missing. */}
        </div>
      </div>
    </div>
  );
}

export default NewRunV2;

const PIPELINE_SELECTOR_COLUMNS = [
  {
    customRenderer: NameWithTooltip,
    flex: 1,
    label: 'Pipeline name',
    sortKey: PipelineSortKeys.NAME,
  },
  { label: 'Description', flex: 2, customRenderer: descriptionCustomRenderer },
  { label: 'Uploaded on', flex: 1, sortKey: PipelineSortKeys.CREATED_AT },
];

const PIPELINE_VERSION_SELECTOR_COLUMNS = [
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

const EXPERIMENT_SELECTOR_COLUMNS = [
  {
    customRenderer: NameWithTooltip,
    flex: 1,
    label: 'Experiment name',
    sortKey: ExperimentSortKeys.NAME,
  },
  { label: 'Description', flex: 2 },
  { label: 'Created at', flex: 1, sortKey: ExperimentSortKeys.CREATED_AT },
];

interface PipelineSelectorSpecificProps {
  namespace?: string;
  pipelineName: string | undefined;
  handlePipelineChange: (pipeline: V2beta1Pipeline) => void;
}
type PipelineSelectorProps = PageProps & PipelineSelectorSpecificProps;

function PipelineSelector(props: PipelineSelectorProps) {
  const [pipelineSelectorOpen, setPipelineSelectorOpen] = useState(false);

  return (
    <>
      <Input
        value={props.pipelineName}
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
                onClick={() => setPipelineSelectorOpen(true)}
                style={{ padding: '3px 5px', margin: 0 }}
              >
                Choose
              </Button>
            </InputAdornment>
          ),
          readOnly: true,
        }}
      />

      {/* Pipeline selector dialog */}
      <PipelinesDialogV2
        {...props}
        open={pipelineSelectorOpen}
        selectorDialog={css.selectorDialog}
        onClose={(confirmed, selectedPipeline?: V2beta1Pipeline) => {
          if (confirmed && selectedPipeline) {
            props.handlePipelineChange(selectedPipeline);
          }
          setPipelineSelectorOpen(false);
        }}
        namespace={props.namespace}
        pipelineSelectorColumns={PIPELINE_SELECTOR_COLUMNS}
        // TODO(jlyaoyuli): enable pipeline upload function in the selector dialog
      ></PipelinesDialogV2>
    </>
  );
}

interface PipelineVersionSelectorSpecificProps {
  namespace?: string;
  pipeline: V2beta1Pipeline | undefined;
  pipelineVersionName: string | undefined;
  handlePipelineVersionChange: (pipelineVersion: V2beta1PipelineVersion) => void;
}
type PipelineVersionSelectorProps = PageProps & PipelineVersionSelectorSpecificProps;

function PipelineVersionSelector(props: PipelineVersionSelectorProps) {
  const [pipelineVersionSelectorOpen, setPipelineVersionSelectorOpen] = useState(false);
  const [pendingPipelineVersion, setPendingPipelineVersion] = useState<V2beta1PipelineVersion>();

  return (
    <>
      <Input
        value={props.pipelineVersionName}
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
                onClick={() => setPipelineVersionSelectorOpen(true)}
                style={{ padding: '3px 5px', margin: 0 }}
                disabled={!props.pipeline}
              >
                Choose
              </Button>
            </InputAdornment>
          ),
          readOnly: true,
        }}
      />

      {/* Pipeline version selector dialog */}
      <Dialog
        open={pipelineVersionSelectorOpen}
        classes={{ paper: css.selectorDialog }}
        onClose={() => setPipelineVersionSelectorOpen(false)}
        PaperProps={{ id: 'pipelineVersionSelectorDialog' }}
      >
        <DialogContent>
          <ResourceSelector
            {...props}
            title='Choose a pipeline version'
            filterLabel='Filter pipeline versions'
            listApi={async (
              page_token?: string,
              page_size?: number,
              sort_by?: string,
              filter?: string,
            ) => {
              const response = await Apis.pipelineServiceApiV2.listPipelineVersions(
                props.pipeline ? props.pipeline.pipeline_id! : '',
                page_token,
                page_size,
                sort_by,
                filter,
              );
              return {
                nextPageToken: response.next_page_token || '',
                resources:
                  response.pipeline_versions?.map(v => convertPipelineVersionToResource(v)) || [],
              };
            }}
            columns={PIPELINE_VERSION_SELECTOR_COLUMNS}
            emptyMessage='No pipeline versions found. Select or upload a pipeline then try again.'
            initialSortColumn={PipelineVersionSortKeys.CREATED_AT}
            selectionChanged={async (selectedVersionId: string) => {
              const selectedPipelineVersion = await Apis.pipelineServiceApiV2.getPipelineVersion(
                props.pipeline?.pipeline_id!,
                selectedVersionId,
              );
              setPendingPipelineVersion(selectedPipelineVersion);
            }}
            // TODO(jlyaoyuli): enable pipeline upload function in the selector dialog
          />
        </DialogContent>
        <DialogActions>
          <Button
            id='cancelPipelineVersionSelectionBtn'
            onClick={() => setPipelineVersionSelectorOpen(false)}
            color='secondary'
          >
            Cancel
          </Button>
          <Button
            id='usePipelineVersionBtn'
            onClick={() => {
              if (pendingPipelineVersion) {
                props.handlePipelineVersionChange(pendingPipelineVersion);
              }
              setPipelineVersionSelectorOpen(false);
            }}
            color='secondary'
            disabled={!pendingPipelineVersion}
          >
            Use this pipeline version
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}

interface ExperimentSelectorSpecificProps {
  namespace?: string;
  experimentName: string | undefined;
  handleExperimentChange: (experiment: V2beta1Experiment) => void;
}
type ExperimentSelectorProps = PageProps & ExperimentSelectorSpecificProps;

function ExperimentSelector(props: ExperimentSelectorProps) {
  const [experimentSelectorOpen, setExperimentSelectorOpen] = useState(false);
  const [pendingExperiment, setPendingExperiment] = useState<V2beta1Experiment>();

  return (
    <>
      <Input
        value={props.experimentName}
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
                onClick={() => setExperimentSelectorOpen(true)}
                style={{ padding: '3px 5px', margin: 0 }}
              >
                Choose
              </Button>
            </InputAdornment>
          ),
          readOnly: true,
        }}
      />

      {/* Experiment selector dialog */}
      <Dialog
        open={experimentSelectorOpen}
        classes={{ paper: css.selectorDialog }}
        onClose={() => setExperimentSelectorOpen(false)}
        PaperProps={{ id: 'experimentSelectorDialog' }}
      >
        <DialogContent>
          <ResourceSelector
            {...props}
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
              ) as V2beta1Filter;
              new_filter.predicates = (new_filter.predicates || []).concat([
                {
                  key: 'storage_state',
                  operation: V2beta1PredicateOperation.NOTEQUALS,
                  string_value: V2beta1ExperimentStorageState.ARCHIVED.toString(),
                },
              ]);
              const response = await Apis.experimentServiceApiV2.listExperiments(
                page_token,
                page_size,
                sort_by,
                encodeURIComponent(JSON.stringify(new_filter)),
                props.namespace,
              );
              return {
                nextPageToken: response.next_page_token || '',
                resources: response.experiments?.map(e => convertExperimentToResource(e)) || [],
              };
            }}
            columns={EXPERIMENT_SELECTOR_COLUMNS}
            emptyMessage='No experiments found. Create an experiment and then try again.'
            initialSortColumn={ExperimentSortKeys.CREATED_AT}
            selectionChanged={async (selectedExperimentId: string) => {
              const selectedExperiment = await Apis.experimentServiceApiV2.getExperiment(
                selectedExperimentId,
              );
              setPendingExperiment(selectedExperiment);
            }}
          />
        </DialogContent>
        <DialogActions>
          <Button
            id='cancelExperimentSelectionBtn'
            onClick={() => setExperimentSelectorOpen(false)}
            color='secondary'
          >
            Cancel
          </Button>
          <Button
            id='useExperimentBtn'
            onClick={() => {
              if (pendingExperiment) {
                props.handleExperimentChange(pendingExperiment);
              }
              setExperimentSelectorOpen(false);
            }}
            color='secondary'
            disabled={!pendingExperiment}
          >
            Use this experiment
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
