/**
 * Copyright 2021 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react';
import { useQuery } from 'react-query';
import { ExternalLink } from 'src/atoms/ExternalLink';
import { color } from 'src/Css';
import { Apis } from 'src/lib/Apis';
import WorkflowParser, { StoragePath } from 'src/lib/WorkflowParser';
import { stylesheet } from 'typestyle';
import Banner from './Banner';
import { ValueComponentProps } from './DetailsTable';
import { logger } from 'src/lib/Utils';
import { URIToSessionInfo } from './tabs/InputOutputTab';

const css = stylesheet({
  root: {
    width: '100%',
  },
  preview: {
    maxHeight: 250,
    overflowY: 'auto',
    padding: 3,
    backgroundColor: color.lightGrey,
  },
  topDiv: {
    display: 'flex',
    justifyContent: 'space-between',
  },
  separater: {
    width: 20, // There's minimum 20px separation between URI and view button.
    display: 'inline-block',
  },
  viewLink: {
    whiteSpace: 'nowrap',
  },
});

export interface ArtifactPreviewProps extends ValueComponentProps<string> {
  namespace?: string;
  sessionMap?: URIToSessionInfo;
  maxbytes?: number;
  maxlines?: number;
}

/**
 * A component that renders a preview to an artifact with a link to the full content.
 */
const ArtifactPreview: React.FC<ArtifactPreviewProps> = ({
  value,
  namespace,
  sessionMap,
  maxbytes = 255,
  maxlines = 20,
}) => {
  let storage: StoragePath | undefined;
  let providerInfo: string | undefined;

  if (value) {
    try {
      providerInfo = sessionMap?.get(value);
      storage = WorkflowParser.parseStoragePath(value);
    } catch (error) {
      logger.error(error);
    }
  }

  const { isSuccess, isError, data, error } = useQuery<string, Error>(
    ['artifact_preview', { value, namespace, maxbytes, maxlines }],
    () => getPreview(storage, providerInfo, namespace, maxbytes, maxlines),
    { staleTime: Infinity },
  );

  if (!storage) {
    return (
      <Banner message={'Can not retrieve storage path from artifact uri: ' + value} mode='info' />
    );
  }

  const linkText = Apis.buildArtifactLinkText(storage);
  const artifactDownloadUrl = Apis.buildReadFileUrl({
    path: storage,
    namespace,
    isDownload: true,
  });
  const artifactViewUrl = Apis.buildReadFileUrl({ path: storage, namespace });

  return (
    <div className={css.root}>
      <div className={css.topDiv}>
        <ExternalLink download href={artifactDownloadUrl} title={linkText}>
          {linkText}
        </ExternalLink>
        <span className={css.separater} />
        <ExternalLink href={artifactViewUrl} className={css.viewLink}>
          View All
        </ExternalLink>
      </div>
      {isError && (
        <Banner
          message='Error in retrieving artifact preview.'
          mode='error'
          additionalInfo={error ? error.message : 'No error message'}
        />
      )}
      {isSuccess && data && (
        <div className={css.preview}>
          <small>
            <pre>{data}</pre>
          </small>
        </div>
      )}
    </div>
  );
};

export default ArtifactPreview;

async function getPreview(
  storagePath: StoragePath | undefined,
  providerInfo: string | undefined,
  namespace: string | undefined,
  maxbytes: number,
  maxlines?: number,
): Promise<string> {
  if (!storagePath) {
    return ``;
  }
  // TODO how to handle binary data (can probably use magic number to id common mime types)
  let data = await Apis.readFile({
    path: storagePath,
    providerInfo: providerInfo,
    namespace: namespace,
    peek: maxbytes + 1,
  });
  // is preview === data and no maxlines
  if (data.length <= maxbytes && (!maxlines || data.split('\n').length < maxlines)) {
    return data;
  }
  // remove extra byte at the end (we requested maxbytes +1)
  data = data.slice(0, maxbytes);
  // check num lines
  if (maxlines) {
    data = data
      .split('\n')
      .slice(0, maxlines)
      .join('\n')
      .trim();
  }
  return `${data}\n...`;
}
