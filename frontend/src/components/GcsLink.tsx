import * as React from 'react';
import { generateGcsConsoleUri } from '../lib/Utils';

/**
 * A component that renders an artifact URL as clickable link if URL is correct
  */
export const GcsLink: React.FC<{ gcsUri?: string }> = ({ gcsUri }) => {
  let clickableUrl: string | undefined = undefined;
  if (gcsUri) {
    if (gcsUri.startsWith('gs:')) {
      const gcsConsoleUrl = generateGcsConsoleUri(gcsUri);
      if (gcsConsoleUrl) {
        clickableUrl = gcsConsoleUrl;
      }
    } else if (gcsUri.startsWith('http:') || gcsUri.startsWith('https:')) {
      clickableUrl = gcsUri;
    }
  }

  if (clickableUrl) {
    // Opens in new window safely
    return (
      <a href={clickableUrl} target={'_blank'} rel={'noreferrer noopener'}>
        {gcsUri}
      </a>
    );
  } else {
    return <>{gcsUri}</>;
  }
};
