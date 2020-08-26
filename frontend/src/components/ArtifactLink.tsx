import * as React from 'react';
import {
  generateGcsConsoleUri,
  generateS3ArtifactUrl,
  generateMinioArtifactUrl,
} from '../lib/Utils';

/**
 * A component that renders an artifact URL as clickable link if URL is correct
 */
export const ArtifactLink: React.FC<{ artifactUri?: string }> = ({ artifactUri }) => {
  let clickableUrl: string | undefined;
  if (artifactUri) {
    if (artifactUri.startsWith('gs:')) {
      const gcsConsoleUrl = generateGcsConsoleUri(artifactUri);
      if (gcsConsoleUrl) {
        clickableUrl = gcsConsoleUrl;
      }
    }
    if (artifactUri.startsWith('s3:')) {
      clickableUrl = generateS3ArtifactUrl(artifactUri);
    } else if (artifactUri.startsWith('http:') || artifactUri.startsWith('https:')) {
      clickableUrl = artifactUri;
    } else if (artifactUri.startsWith('minio:')) {
      clickableUrl = generateMinioArtifactUrl(artifactUri);
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
