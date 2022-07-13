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
import RunUtils from 'src/lib/RunUtils';

function NewRunSwitcher(props: PageProps) {
  const namespace = React.useContext(NamespaceContext);
  //console.log(props);

  const urlParser = new URLParser(props);
  const originalRunId = urlParser.get(QUERY_PARAMS.cloneFromRun);
  console.log(originalRunId);

  /* 
  runID
  if(runID) {
    templateStr = Apis.runServiceApi.getRun(originalRunId)... pipelineManfest
  }
  */




  const pipelineId = urlParser.get(QUERY_PARAMS.pipelineId);
  const pipelineVersionIdParam = urlParser.get(QUERY_PARAMS.pipelineVersionId);
  //console.log(urlParser)
  //console.log(pipelineId);
  // console.log(pipelineVersionIdParam);

  const { isSuccess, isFetching, data: templateString } = useQuery<string, Error>(
    [pipelineId, pipelineVersionIdParam],
    async () => {
      // if (originalRunId) {
      //     const originalRun = await (await Apis.runServiceApi.getRun(originalRunId)).run?.pipeline_spec.;
      //     const pipelineID = RunUtils.getPipelineId(originalRun.run);
      // }



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
      console.log(template);
      return template?.template || '';
    },
    { staleTime: Infinity },
  );

  if (isFeatureEnabled(FeatureKey.V2_ALPHA)) {
    if (isSuccess && isTemplateV2(templateString || '')) {
      return <NewRunV2 {...props} namespace={namespace} />;
    }
  }

  if (isFetching) {
    return <div>Currently loading pipeline information</div>;
  }
  //console.log(namespace);

  return <NewRun {...props} namespace={namespace} />;
}

export default NewRunSwitcher;
