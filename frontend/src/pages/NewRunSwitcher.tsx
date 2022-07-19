import React from 'react';
import { useQuery } from 'react-query';
import { QUERY_PARAMS } from 'src/components/Router';
import { isFeatureEnabled, FeatureKey } from 'src/features';
import { Apis } from 'src/lib/Apis';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { URLParser } from 'src/lib/URLParser';
import { NewRun } from './NewRun';
import NewRunV2 from './NewRunV2';
import { PageProps } from './Page';
import { isTemplateV2 } from 'src/lib/v2/WorkflowUtils';

function NewRunSwitcher(props: PageProps) {
  const namespace = React.useContext(NamespaceContext);

  const urlParser = new URLParser(props);
  const originalRunId = urlParser.get(QUERY_PARAMS.cloneFromRun);
  const pipelineId = urlParser.get(QUERY_PARAMS.pipelineId);
  const pipelineVersionIdParam = urlParser.get(QUERY_PARAMS.pipelineVersionId);

  const { isSuccess: isSuccessRun, data: runTemplateString} = useQuery(
    [originalRunId],
    async () => {
      if (!originalRunId) {
        return '';        
      }
      const originalRun = await Apis.runServiceApi.getRun(originalRunId);
      return originalRun.run?.pipeline_spec?.pipeline_manifest;
    },
    { staleTime: Infinity}
  )

  const { isSuccess: isSuccessPipeline, isFetching, data: pipelineTemplateString } = useQuery<string, Error>(
    [pipelineId, pipelineVersionIdParam],
    async () => {
      if (!pipelineId) {
        return '';
      }
      const pipeline = await Apis.pipelineServiceApi.getPipeline(pipelineId);

      const pipelineVersionId = pipelineVersionIdParam || pipeline?.default_version?.id;
      if (!pipelineVersionId) {
        return '';
      }

      await Apis.pipelineServiceApi.getPipelineVersion(pipelineVersionId);
      const template = await Apis.pipelineServiceApi.getPipelineVersionTemplate(pipelineVersionId);
      return template?.template || '';
    },
    { staleTime: Infinity },
  );

  const templateString = runTemplateString != '' ? runTemplateString : pipelineTemplateString;

  if (isFeatureEnabled(FeatureKey.V2_ALPHA)) {
    if ((isSuccessRun || isSuccessPipeline) && isTemplateV2(templateString || '')) {
      console.log('create v2 component');
      return <NewRunV2 {...props} namespace={namespace} />;
    }
  }

  if (isFetching) {
    return <div>Currently loading pipeline information</div>;
  }
  console.log('create v1 component');
  return <NewRun {...props} namespace={namespace} />;
}

export default NewRunSwitcher;
