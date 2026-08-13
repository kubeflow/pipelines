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

import * as React from 'react';
import { Apis } from '../lib/Apis';
import { generateGcsConsoleUri } from '../lib/Utils';
import { parseArtifactFileLocation } from '../lib/v2/ArtifactFileUtils';
import { StorageService } from '../lib/WorkflowParser';

interface ArtifactLinkProps {
  artifactUri?: string;
  namespace?: string;
}

/**
 * A component that renders an artifact URL as clickable link if URL is correct
 */
export const ArtifactLink: React.FC<ArtifactLinkProps> = ({ artifactUri, namespace }) => {
  let clickableUrl: string | undefined;
  if (artifactUri) {
    if (artifactUri.startsWith('gs:')) {
      const gcsConsoleUrl = generateGcsConsoleUri(artifactUri.split('?', 1)[0]);
      if (gcsConsoleUrl) {
        clickableUrl = gcsConsoleUrl;
      }
    }
    if (artifactUri.startsWith('s3:')) {
      clickableUrl = generateArtifactDownloadUrl(artifactUri, namespace, StorageService.S3);
    } else if (artifactUri.startsWith('http:') || artifactUri.startsWith('https:')) {
      clickableUrl = artifactUri;
    } else if (artifactUri.startsWith('minio:')) {
      clickableUrl = generateArtifactDownloadUrl(artifactUri, namespace, StorageService.MINIO);
    }
  }

  if (clickableUrl) {
    // Opens in new window safely
    return (
      <a href={clickableUrl} target={'_blank'} rel='noreferrer noopener'>
        {artifactUri}
      </a>
    );
  } else {
    return <>{artifactUri}</>;
  }
};

function generateArtifactDownloadUrl(
  artifactUri: string,
  namespace: string | undefined,
  expectedSource: StorageService,
): string | undefined {
  let location: ReturnType<typeof parseArtifactFileLocation>;
  try {
    location = parseArtifactFileLocation(artifactUri);
  } catch {
    return undefined;
  }
  if (location.path.source !== expectedSource || !location.path.bucket || !location.path.key) {
    return undefined;
  }
  return Apis.buildReadFileUrl({
    path: location.path,
    namespace,
    providerInfo: location.providerInfo,
    isDownload: true,
  });
}
