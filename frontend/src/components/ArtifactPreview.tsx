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
import { Button } from '@mui/material';
import { useQuery } from '@tanstack/react-query';
import { ExternalLink } from 'src/atoms/ExternalLink';
import { color } from 'src/Css';
import { queryKeys } from 'src/hooks/queryKeys';
import { Apis } from 'src/lib/Apis';
import { StoragePath } from 'src/lib/WorkflowParser';
import { parseArtifactFileLocation } from 'src/lib/v2/ArtifactFileUtils';
import { stylesheet } from 'typestyle';
import Banner from './Banner';
import { ValueComponentProps } from './DetailsTable';
import { logger } from 'src/lib/Utils';

export type ArtifactPreviewValue =
  | string
  | {
      uri: string;
      providerInfo?: string;
    };

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

export interface ArtifactPreviewProps extends ValueComponentProps<ArtifactPreviewValue> {
  namespace?: string;
  maxbytes?: number;
  maxlines?: number;
}

/**
 * A component that renders a preview to an artifact with a link to the full content.
 */
const ArtifactPreview: React.FC<ArtifactPreviewProps> = ({
  value,
  namespace,
  maxbytes = 255,
  maxlines = 20,
}) => {
  const [previewRequested, setPreviewRequested] = React.useState(false);
  const rawUri = typeof value === 'object' && value !== null ? value.uri : value;
  const uri = typeof rawUri === 'string' ? rawUri : undefined;
  let storage: StoragePath | undefined;
  let artifactUriQuery: string | undefined;
  let providerInfo = typeof value === 'object' && value !== null ? value.providerInfo : undefined;

  if (uri) {
    try {
      const location = parseArtifactFileLocation(uri);
      storage = location.path;
      artifactUriQuery = location.artifactUriQuery;
    } catch (error) {
      logger.error(error);
    }
  }

  const { isSuccess, isError, data, error, refetch } = useQuery<string, Error>({
    queryKey: queryKeys.artifactPreview(
      uri,
      namespace,
      artifactUriQuery,
      providerInfo,
      maxbytes,
      maxlines,
    ),
    queryFn: () =>
      getPreview(storage, artifactUriQuery, providerInfo, namespace, maxbytes, maxlines),
    enabled: previewRequested && !!storage,
    retry: false,
    staleTime: Infinity,
  });

  if (!storage) {
    return (
      <Banner message={'Can not retrieve storage path from artifact uri: ' + rawUri} mode='info' />
    );
  }

  const linkText = Apis.buildArtifactLinkText(storage);
  const artifactDownloadUrl = Apis.buildReadFileUrl({
    path: storage,
    namespace,
    artifactUriQuery,
    providerInfo,
    isDownload: true,
  });
  const artifactViewUrl = Apis.buildReadFileUrl({
    path: storage,
    namespace,
    artifactUriQuery,
    providerInfo,
  });

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
      {!previewRequested && (
        <Button size='small' onClick={() => setPreviewRequested(true)}>
          Load preview
        </Button>
      )}
      {isError && (
        <>
          <Banner
            message='Error in retrieving artifact preview.'
            mode='error'
            additionalInfo={error ? error.message : 'No error message'}
          />
          <Button size='small' onClick={() => void refetch()}>
            Retry preview
          </Button>
        </>
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
  artifactUriQuery: string | undefined,
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
    artifactUriQuery,
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
    data = data.split('\n').slice(0, maxlines).join('\n').trim();
  }
  return `${data}\n...`;
}
