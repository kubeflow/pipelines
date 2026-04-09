import React, { useEffect, useState } from 'react';
import * as JsYaml from 'js-yaml';
import { useQuery } from '@tanstack/react-query';
import { CircularProgress } from '@mui/material';
import { QUERY_PARAMS } from 'src/components/Router';
import { queryKeys } from 'src/hooks/queryKeys';
import { Apis } from 'src/lib/Apis';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { URLParser } from 'src/lib/URLParser';
import { NewRun } from './NewRun';
import NewRunV2 from './NewRunV2';
import { PageProps } from './Page';
import { isTemplateV2 } from 'src/lib/v2/WorkflowUtils';
import { V2beta1Pipeline, V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import { V2beta1Run } from 'src/apisv2beta1/run';
import { V2beta1RecurringRun } from 'src/apisv2beta1/recurringrun';
import { V2beta1Experiment } from 'src/apisv2beta1/experiment';

function NewRunSwitcher(props: PageProps) {
  const namespace = React.useContext(NamespaceContext);

  const urlParser = new URLParser(props);
  // Currently using two query parameters to get Run ID.
  // because v1 has two different behavior with Run ID (clone a run / start a run)
  // Will keep clone run only in v2 if run ID is existing
  // runID query by cloneFromRun will be deprecated once v1 is deprecated.
  const originalRunId = urlParser.get(QUERY_PARAMS.cloneFromRun);
  const embeddedRunId = urlParser.get(QUERY_PARAMS.fromRunId);
  const originalRecurringRunId = urlParser.get(QUERY_PARAMS.cloneFromRecurringRun);
  const [pipelineIdFromPipeline, setPipelineIdFromPipeline] = useState(
    urlParser.get(QUERY_PARAMS.pipelineId),
  );
  const experimentId = urlParser.get(QUERY_PARAMS.experimentId);
  const [pipelineVersionIdParam, setPipelineVersionIdParam] = useState(
    urlParser.get(QUERY_PARAMS.pipelineVersionId),
  );
  const existingRunId = originalRunId ? originalRunId : embeddedRunId;
  const hasConflictingCloneSources = !!(originalRunId || embeddedRunId) && !!originalRecurringRunId;
  let pipelineIdFromRunOrRecurringRun;
  let pipelineVersionIdFromRunOrRecurringRun;

  // Retrieve v2 run details
  const {
    isSuccess: getV2RunSuccess,
    isFetching: v2RunIsFetching,
    isError: v2RunIsError,
    error: v2RunError,
    data: v2Run,
  } = useQuery<V2beta1Run, Error>({
    queryKey: queryKeys.v2RunDetailSingle(existingRunId),
    queryFn: () => {
      if (!existingRunId) {
        throw new Error('Run ID is missing');
      }
      return Apis.runServiceApiV2.getRun(existingRunId);
    },
    enabled: !!existingRunId && !hasConflictingCloneSources,
    staleTime: Infinity,
  });

  // Retrieve recurring run details
  const {
    isSuccess: getRecurringRunSuccess,
    isFetching: recurringRunIsFetching,
    isError: recurringRunIsError,
    error: recurringRunError,
    data: recurringRun,
  } = useQuery<V2beta1RecurringRun, Error>({
    queryKey: queryKeys.recurringRun(originalRecurringRunId),
    queryFn: () => {
      if (!originalRecurringRunId) {
        throw new Error('Recurring Run ID is missing');
      }
      return Apis.recurringRunServiceApi.getRecurringRun(originalRecurringRunId);
    },
    enabled: !!originalRecurringRunId && !hasConflictingCloneSources,
    staleTime: Infinity,
  });

  pipelineIdFromRunOrRecurringRun =
    v2Run?.pipeline_version_reference?.pipeline_id ||
    recurringRun?.pipeline_version_reference?.pipeline_id;
  pipelineVersionIdFromRunOrRecurringRun =
    v2Run?.pipeline_version_reference?.pipeline_version_id ||
    recurringRun?.pipeline_version_reference?.pipeline_version_id;

  // template string from cloned run / recurring run created by pipeline_spec
  let pipelineManifest: string | undefined;
  if (getV2RunSuccess && v2Run && v2Run.pipeline_spec) {
    pipelineManifest = JsYaml.safeDump(v2Run.pipeline_spec);
  }

  if (getRecurringRunSuccess && recurringRun && recurringRun.pipeline_spec) {
    pipelineManifest = JsYaml.safeDump(recurringRun.pipeline_spec);
  }

  const {
    isFetching: pipelineIsFetching,
    isError: pipelineIsError,
    error: pipelineError,
    data: pipeline,
  } = useQuery<V2beta1Pipeline, Error>({
    queryKey: queryKeys.pipeline(pipelineIdFromPipeline),
    queryFn: () => {
      if (!pipelineIdFromPipeline) {
        throw new Error('Pipeline ID is missing');
      }
      return Apis.pipelineServiceApiV2.getPipeline(pipelineIdFromPipeline);
    },
    enabled: !!pipelineIdFromPipeline,
    staleTime: Infinity,
    gcTime: 0,
  });

  const pipelineId = pipelineIdFromPipeline || pipelineIdFromRunOrRecurringRun;
  const pipelineVersionId = pipelineVersionIdParam || pipelineVersionIdFromRunOrRecurringRun;

  const {
    isFetching: pipelineVersionIsFetching,
    isError: pipelineVersionIsError,
    error: pipelineVersionError,
    data: pipelineVersion,
  } = useQuery<V2beta1PipelineVersion, Error>({
    queryKey: queryKeys.pipelineVersion(pipelineId, pipelineVersionId),
    queryFn: () => {
      if (!(pipelineId && pipelineVersionId)) {
        throw new Error('Pipeline id or pipeline Version ID is missing');
      }
      return Apis.pipelineServiceApiV2.getPipelineVersion(pipelineId, pipelineVersionId);
    },
    enabled: !!pipelineId && !!pipelineVersionId,
    staleTime: Infinity,
    gcTime: 0,
  });
  const pipelineSpecInVersion = pipelineVersion?.pipeline_spec;
  const templateStrFromSpec = pipelineSpecInVersion ? JsYaml.safeDump(pipelineSpecInVersion) : '';

  const {
    isFetching: v1TemplateStrIsFetching,
    isError: v1TemplateIsError,
    error: v1TemplateError,
    data: v1Template,
  } = useQuery<string, Error>({
    queryKey: queryKeys.v1PipelineVersionTemplate(pipelineId, pipelineVersionId),
    queryFn: async () => {
      if (!(pipelineId && pipelineVersionId)) {
        throw new Error('Pipeline id or pipeline Version ID is missing');
      }

      const v1TemplateResponse =
        await Apis.pipelineServiceApi.getPipelineVersionTemplate(pipelineVersionId);
      return v1TemplateResponse.template || '';
    },
    // Requires BOTH IDs: queryFn throws if either is missing. `&&` prevents avoidable fetch-then-throw.
    // (Previously enabled: !!pipelineId || !!pipelineVersionId would run with only one ID and then throw.)
    enabled: !!pipelineId && !!pipelineVersionId,
    staleTime: Infinity,
    gcTime: 0,
  });
  const v1TemplateStr = v1Template || '';

  const {
    isFetching: experimentIsFetching,
    isError: experimentIsError,
    error: experimentError,
    data: experiment,
  } = useQuery<V2beta1Experiment, Error>({
    queryKey: queryKeys.experiment(experimentId),
    queryFn: async () => {
      if (!experimentId) {
        throw new Error('Experiment ID is missing');
      }
      return Apis.experimentServiceApiV2.getExperiment(experimentId);
    },
    enabled: !!experimentId,
    staleTime: Infinity,
  });

  if (hasConflictingCloneSources) {
    throw new Error('The existence of run and recurring run should be exclusive.');
  }

  const firstQueryError =
    (v2RunIsError && v2RunError) ||
    (recurringRunIsError && recurringRunError) ||
    (pipelineIsError && pipelineError) ||
    (pipelineVersionIsError && pipelineVersionError) ||
    (v1TemplateIsError && v1TemplateError) ||
    (experimentIsError && experimentError) ||
    undefined;

  const { updateBanner } = props;
  useEffect(() => {
    if (firstQueryError) {
      updateBanner({
        message: 'Error: failed to retrieve run creation data. Click Details for more information.',
        additionalInfo: firstQueryError.message,
        mode: 'error',
      });
      return;
    }
    updateBanner({});
  }, [firstQueryError, updateBanner]);

  // Three possible sources for template string
  // 1. pipelineManifest: pipeline_spec stored in run or recurring run created by SDK
  // 2. templateStrFromSpec: pipeline_spec stored in pipeline_version
  // 3. v1TemplateStr: pipelines created by v1 API (no pipeline_spec field)
  const templateString = pipelineManifest ?? (templateStrFromSpec || v1TemplateStr);

  if (
    v2RunIsFetching ||
    recurringRunIsFetching ||
    pipelineIsFetching ||
    pipelineVersionIsFetching ||
    (!isTemplateV2(templateString) && v1TemplateStrIsFetching) ||
    experimentIsFetching
  ) {
    return <CircularProgress />;
  }

  if (templateString && !isTemplateV2(templateString)) {
    return (
      <NewRun
        {...props}
        namespace={namespace}
        existingPipelineId={pipelineIdFromPipeline}
        handlePipelineIdChange={setPipelineIdFromPipeline}
        existingPipelineVersionId={pipelineVersionIdParam}
        handlePipelineVersionIdChange={setPipelineVersionIdParam}
      />
    );
  }

  return (
    <NewRunV2
      {...props}
      namespace={namespace}
      existingRunId={existingRunId}
      existingRun={v2Run}
      existingRecurringRunId={originalRecurringRunId}
      existingRecurringRun={recurringRun}
      existingPipeline={pipeline}
      handlePipelineIdChange={setPipelineIdFromPipeline}
      existingPipelineVersion={pipelineVersion}
      handlePipelineVersionIdChange={setPipelineVersionIdParam}
      templateString={templateString}
      chosenExperiment={experiment}
    />
  );
}

export default NewRunSwitcher;
