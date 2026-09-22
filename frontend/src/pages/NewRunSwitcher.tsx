import React, { useEffect, useState } from 'react';
import * as JsYaml from 'js-yaml';
import { useQuery } from '@tanstack/react-query';
import { CircularProgress } from '@mui/material';
import { QUERY_PARAMS } from 'src/components/Router';
import { queryKeys } from 'src/hooks/queryKeys';
import { Apis } from 'src/lib/Apis';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { URLParser } from 'src/lib/URLParser';
import NewRunV2 from './NewRunV2';
import { PageProps } from './Page';
import { V2beta1Pipeline, V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import { V2beta1Run } from 'src/apisv2beta1/run';
import { V2beta1RecurringRun } from 'src/apisv2beta1/recurringrun';
import { V2beta1Experiment } from 'src/apisv2beta1/experiment';

function NewRunSwitcher(props: PageProps) {
  const namespace = React.useContext(NamespaceContext);

  const urlParser = new URLParser(props);
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
    pipelineManifest = JsYaml.dump(v2Run.pipeline_spec);
  }

  if (getRecurringRunSuccess && recurringRun && recurringRun.pipeline_spec) {
    pipelineManifest = JsYaml.dump(recurringRun.pipeline_spec);
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
  const templateStrFromSpec = pipelineSpecInVersion ? JsYaml.dump(pipelineSpecInVersion) : '';

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

  const templateString = pipelineManifest ?? templateStrFromSpec;

  if (
    v2RunIsFetching ||
    recurringRunIsFetching ||
    pipelineIsFetching ||
    pipelineVersionIsFetching ||
    experimentIsFetching
  ) {
    return (
      <div style={{ textAlign: 'center', paddingTop: 40 }}>
        <CircularProgress />
        <div>Currently loading pipeline information</div>
      </div>
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
