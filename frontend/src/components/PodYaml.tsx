import * as JsYaml from 'js-yaml';
import React from 'react';
import { Apis, JSONObject } from 'src/lib/Apis';
import { serviceErrorToString } from 'src/lib/Utils';
import Banner from './Banner';
import Editor from './Editor';
import { TFunction } from 'i18next';

async function getPodYaml(name: string, namespace: string): Promise<string> {
  const response = await Apis.getPodInfo(name, namespace);
  return JsYaml.safeDump(reorderPodJson(response), { skipInvalid: true });
}
export const PodInfo: React.FC<{ name: string; namespace: string; t: TFunction }> = ({
  name,
  namespace,
  t,
}) => {
  return (
    <PodYaml
      name={name}
      namespace={namespace}
      errorMessage={t('common:retrievePodInfoFailed')}
      getYaml={getPodYaml}
    />
  );
};

async function getPodEventsYaml(name: string, namespace: string): Promise<string> {
  const response = await Apis.getPodEvents(name, namespace);
  return JsYaml.safeDump(response, { skipInvalid: true });
}
export const PodEvents: React.FC<{
  name: string;
  namespace: string;
  t: TFunction;
}> = ({ name, namespace, t }) => {
  return (
    <PodYaml
      name={name}
      namespace={namespace}
      errorMessage={t('common:retrievePodEventsFailed')}
      getYaml={getPodEventsYaml}
    />
  );
};

const PodYaml: React.FC<{
  name: string;
  namespace: string;
  errorMessage: string;
  getYaml: (name: string, namespace: string) => Promise<string>;
}> = ({ name, namespace, getYaml, errorMessage }) => {
  const [yaml, setYaml] = React.useState<string | undefined>(undefined);
  const [error, setError] = React.useState<
    | {
        message: string;
        additionalInfo: string;
      }
    | undefined
  >(undefined);
  const [refreshes, setRefresh] = React.useState(0);

  React.useEffect(() => {
    let aborted = false;
    async function load() {
      setYaml(undefined);
      setError(undefined);
      try {
        const yaml = await getYaml(name, namespace);
        if (!aborted) {
          setYaml(yaml);
        }
      } catch (err) {
        if (!aborted) {
          setError({
            message: errorMessage,
            additionalInfo: await serviceErrorToString(err),
          });
        }
      }
    }
    load();
    return () => {
      aborted = true;
    };
  }, [name, namespace, refreshes, getYaml, errorMessage]); // When refreshes change, request is fetched again.

  return (
    <>
      {error && (
        <Banner
          message={error.message}
          mode='warning'
          additionalInfo={error.additionalInfo}
          // Increases refresh counter, so it will automatically trigger a refetch.
          refresh={() => setRefresh(refreshes => refreshes + 1)}
        />
      )}
      {!error && yaml && (
        <Editor
          value={yaml || ''}
          height='100%'
          width='100%'
          mode='yaml'
          theme='github'
          editorProps={{ $blockScrolling: true }}
          readOnly={true}
          highlightActiveLine={true}
          showGutter={true}
        />
      )}
    </>
  );
};

function reorderPodJson(jsonData: JSONObject): JSONObject {
  const orderedData = { ...jsonData };
  if (jsonData.spec) {
    // Deleting spec property and add it again moves spec to the last property in the JSON object.
    delete orderedData.spec;
    orderedData.spec = jsonData.spec;
  }
  return orderedData;
}

export const TEST_ONLY = {
  PodYaml,
};
